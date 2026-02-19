// Package delivery provides the email delivery engine with retry support.
package delivery

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/joeblew999/plat-mjml/internal/model"
	"github.com/joeblew999/plat-mjml/internal/events"
	"github.com/joeblew999/plat-mjml/pkg/mail"
	"github.com/joeblew999/plat-mjml/internal/mjml"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/rescue"
	"github.com/zeromicro/go-zero/core/syncx"
	"github.com/zeromicro/go-zero/core/threading"
	"golang.org/x/time/rate"
)

// Config holds delivery engine configuration.
type Config struct {
	MaxRetries   int
	RetryBackoff time.Duration
	MaxBackoff   time.Duration
	RateLimit    int // emails per minute
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		MaxRetries:   3,
		RetryBackoff: 5 * time.Minute,
		MaxBackoff:   4 * time.Hour,
		RateLimit:    60,
	}
}

// Engine handles email delivery with retry logic.
type Engine struct {
	config      Config
	emailsModel model.EmailsModel
	events      *events.EventRecorder
	renderer    *mjml.Renderer
	smtpConfig  mail.Config
	rateLimiter *rate.Limiter
	running     *syncx.AtomicBool

	ctx    context.Context
	cancel context.CancelFunc
	group  *threading.RoutineGroup
}

// NewEngine creates a new delivery engine.
func NewEngine(emailsModel model.EmailsModel, ev *events.EventRecorder, r *mjml.Renderer, smtp mail.Config, cfg Config) *Engine {
	// Rate limiter: N emails per minute
	limiter := rate.NewLimiter(rate.Every(time.Minute/time.Duration(cfg.RateLimit)), 1)

	ctx, cancel := context.WithCancel(context.Background())

	return &Engine{
		config:      cfg,
		emailsModel: emailsModel,
		events:      ev,
		renderer:    r,
		smtpConfig:  smtp,
		rateLimiter: limiter,
		running:     syncx.NewAtomicBool(),
		ctx:         ctx,
		cancel:      cancel,
		group:       threading.NewRoutineGroup(),
	}
}

// Start starts the delivery engine with the specified number of workers.
func (e *Engine) Start(workers int) {
	if !e.running.CompareAndSwap(false, true) {
		return // Already running
	}

	logx.Infow("Delivery engine started", logx.Field("workers", workers))
	for i := 0; i < workers; i++ {
		workerID := i
		_ = workerID
		e.group.RunSafe(func() { e.worker() })
	}
}

// Stop gracefully stops the delivery engine.
func (e *Engine) Stop() {
	if !e.running.CompareAndSwap(true, false) {
		return // Already stopped
	}

	logx.Info("Delivery engine stopping, waiting for workers")
	e.cancel()
	e.group.Wait()
	logx.Info("Delivery engine stopped")
}

func (e *Engine) worker() {
	backoff := 100 * time.Millisecond
	const maxBackoff = 5 * time.Second

	for {
		select {
		case <-e.ctx.Done():
			return
		default:
			email, err := e.emailsModel.Receive(e.ctx)
			if err != nil {
				time.Sleep(backoff)
				if backoff < maxBackoff {
					backoff = min(backoff*2, maxBackoff)
				}
				continue
			}
			if email == nil {
				// No work available — adaptive backoff
				time.Sleep(backoff)
				if backoff < maxBackoff {
					backoff = min(backoff*2, maxBackoff)
				}

				// Periodically update queue depth gauge
				e.updateQueueDepth()
				continue
			}

			backoff = 100 * time.Millisecond // Reset on work found
			e.processJob(email)
		}
	}
}

func (e *Engine) processJob(email *model.Emails) {
	recipients := model.ParseRecipients(email.Recipients)
	data := model.ParseData(email.Data)

	// Enrich context with per-job fields — all logx calls with ctx include these automatically
	ctx := logx.ContextWithFields(e.ctx,
		logx.Field("job_id", email.Id),
		logx.Field("template", email.TemplateSlug),
		logx.Field("recipients", len(recipients)),
	)

	// Panic recovery: mark job failed and record metric if processJob panics
	defer rescue.RecoverCtx(ctx, func() {
		emailsFailed.Inc(email.TemplateSlug, "panic")
		e.emailsModel.MarkFailed(ctx, email.Id, "panic during delivery")
	})

	logx.WithContext(ctx).Info("Processing email")

	start := time.Now()

	// Apply rate limiting
	if err := e.rateLimiter.Wait(ctx); err != nil {
		e.handleError(ctx, email, recipients, err)
		return
	}

	// Render template
	html, err := e.renderer.RenderTemplate(email.TemplateSlug, data)
	if err != nil {
		e.handleError(ctx, email, recipients, fmt.Errorf("render template: %w", err))
		return
	}

	// Send email to each recipient, collecting failures
	var sendErrors []string
	for _, recipient := range recipients {
		if err := mail.Send(e.smtpConfig, recipient, email.Subject, html); err != nil {
			sendErrors = append(sendErrors, fmt.Sprintf("send to %s: %v", recipient, err))
		}
	}

	if len(sendErrors) > 0 {
		e.handleError(ctx, email, recipients, fmt.Errorf("%s", strings.Join(sendErrors, "; ")))
		return
	}

	// Success
	e.emailsModel.MarkSent(ctx, email.Id, "")
	emailsSent.Inc(email.TemplateSlug)
	deliveryDuration.ObserveFloat(time.Since(start).Seconds(), email.TemplateSlug)
	e.recordEvent(email.Id, "sent", "")

	logx.WithContext(ctx).Info("Email sent")
}

func (e *Engine) handleError(ctx context.Context, email *model.Emails, recipients []string, err error) {
	attempts := int(email.Attempts) + 1
	maxAttempts := int(email.MaxAttempts)

	// Classify failure reason for metrics
	reason := "transient"
	if isPermanentFailure(err) {
		reason = "permanent"
	}

	// Check if permanent failure
	if isPermanentFailure(err) || attempts >= maxAttempts {
		e.emailsModel.MarkFailed(ctx, email.Id, err.Error())
		emailsFailed.Inc(email.TemplateSlug, reason)
		e.recordEvent(email.Id, "failed", err.Error())
		logx.WithContext(ctx).Errorf("Email delivery failed permanently: %v", err)
		return
	}

	// Schedule retry with backoff
	backoff := e.calculateBackoff(attempts)
	retryAt := time.Now().Add(backoff)
	e.emailsModel.MarkRetry(ctx, email.Id, retryAt, err.Error())
	emailsRetried.Inc(email.TemplateSlug)
	e.recordEvent(email.Id, "retry", fmt.Sprintf("attempt %d, backoff %s: %v", attempts, backoff, err))

	logx.WithContext(ctx).Infof("Email delivery retrying in %s: %v", backoff, err)
}

func (e *Engine) calculateBackoff(attempts int) time.Duration {
	backoff := e.config.RetryBackoff * time.Duration(math.Pow(2, float64(attempts-1)))
	if backoff > e.config.MaxBackoff {
		return e.config.MaxBackoff
	}
	return backoff
}

// isPermanentFailure checks if the error indicates a permanent failure.
func isPermanentFailure(err error) bool {
	msg := err.Error()
	// SMTP 5xx codes are permanent failures
	permanentCodes := []string{"550", "551", "552", "553", "554"}
	for _, code := range permanentCodes {
		if strings.Contains(msg, code) {
			return true
		}
	}
	return false
}

// recordEvent writes an event to the BulkInserter if available.
func (e *Engine) recordEvent(emailID, eventType, details string) {
	if e.events != nil {
		e.events.RecordEvent(emailID, eventType, details)
	}
}

// updateQueueDepth refreshes the queue depth gauge from current stats.
func (e *Engine) updateQueueDepth() {
	stats, err := e.emailsModel.Stats(e.ctx)
	if err != nil {
		return
	}
	for status, count := range stats {
		queueDepth.Set(float64(count), status)
	}
}

// SendNow sends an email immediately without queueing.
func (e *Engine) SendNow(ctx context.Context, templateSlug string, recipients []string, subject string, data map[string]any) error {
	// Apply rate limiting
	if err := e.rateLimiter.Wait(ctx); err != nil {
		return err
	}

	// Render template
	html, err := e.renderer.RenderTemplate(templateSlug, data)
	if err != nil {
		return fmt.Errorf("render template: %w", err)
	}

	// Send to each recipient
	for _, recipient := range recipients {
		if err := mail.Send(e.smtpConfig, recipient, subject, html); err != nil {
			return fmt.Errorf("send to %s: %w", recipient, err)
		}
	}

	return nil
}
