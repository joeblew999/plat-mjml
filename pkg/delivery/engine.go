// Package delivery provides the email delivery engine with retry support.
package delivery

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/joeblew999/plat-mjml/pkg/mail"
	"github.com/joeblew999/plat-mjml/pkg/mjml"
	"github.com/joeblew999/plat-mjml/pkg/queue"
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
	queue       *queue.Queue
	renderer    *mjml.Renderer
	smtpConfig  mail.Config
	rateLimiter *rate.Limiter
	running     *syncx.AtomicBool

	ctx    context.Context
	cancel context.CancelFunc
	group  *threading.RoutineGroup
}

// NewEngine creates a new delivery engine.
func NewEngine(q *queue.Queue, r *mjml.Renderer, smtp mail.Config, cfg Config) *Engine {
	// Rate limiter: N emails per minute
	limiter := rate.NewLimiter(rate.Every(time.Minute/time.Duration(cfg.RateLimit)), 1)

	ctx, cancel := context.WithCancel(context.Background())

	return &Engine{
		config:      cfg,
		queue:       q,
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
		e.group.RunSafe(func() { e.worker(workerID) })
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

func (e *Engine) worker(id int) {
	backoff := 100 * time.Millisecond
	const maxBackoff = 5 * time.Second

	for {
		select {
		case <-e.ctx.Done():
			return
		default:
			job, err := e.queue.Receive(e.ctx)
			if err != nil {
				time.Sleep(backoff)
				if backoff < maxBackoff {
					backoff = min(backoff*2, maxBackoff)
				}
				continue
			}
			if job == nil {
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
			e.processJob(job)
		}
	}
}

func (e *Engine) processJob(job *queue.EmailJob) {
	// Enrich context with per-job fields — all logx calls with ctx include these automatically
	ctx := logx.ContextWithFields(e.ctx,
		logx.Field("job_id", job.ID),
		logx.Field("template", job.TemplateSlug),
		logx.Field("recipients", len(job.Recipients)),
	)

	// Panic recovery: mark job failed and record metric if processJob panics
	defer rescue.RecoverCtx(ctx, func() {
		emailsFailed.Inc(job.TemplateSlug, "panic")
		e.queue.MarkFailed(ctx, job.ID, fmt.Errorf("panic during delivery"))
	})

	logx.WithContext(ctx).Info("Processing email")

	start := time.Now()

	// Apply rate limiting
	if err := e.rateLimiter.Wait(ctx); err != nil {
		e.handleError(ctx, job, err)
		return
	}

	// Render template
	html, err := e.renderer.RenderTemplate(job.TemplateSlug, job.Data)
	if err != nil {
		e.handleError(ctx, job, fmt.Errorf("render template: %w", err))
		return
	}

	// Send email to each recipient, collecting failures
	var sendErrors []string
	for _, recipient := range job.Recipients {
		if err := mail.Send(e.smtpConfig, recipient, job.Subject, html); err != nil {
			sendErrors = append(sendErrors, fmt.Sprintf("send to %s: %v", recipient, err))
		}
	}

	if len(sendErrors) > 0 {
		e.handleError(ctx, job, fmt.Errorf("%s", strings.Join(sendErrors, "; ")))
		return
	}

	// Success
	e.queue.MarkSent(ctx, job.ID, "")
	emailsSent.Inc(job.TemplateSlug)
	deliveryDuration.ObserveFloat(time.Since(start).Seconds(), job.TemplateSlug)
	e.recordEvent(job.ID, "sent", "")

	logx.WithContext(ctx).Info("Email sent")
}

func (e *Engine) handleError(ctx context.Context, job *queue.EmailJob, err error) {
	job.Attempts++
	job.Error = err.Error()

	// Classify failure reason for metrics
	reason := "transient"
	if isPermanentFailure(err) {
		reason = "permanent"
	}

	// Check if permanent failure
	if isPermanentFailure(err) || job.Attempts >= job.MaxAttempts {
		e.queue.MarkFailed(ctx, job.ID, err)
		emailsFailed.Inc(job.TemplateSlug, reason)
		e.recordEvent(job.ID, "failed", err.Error())
		logx.WithContext(ctx).Errorf("Email delivery failed permanently: %v", err)
		return
	}

	// Schedule retry with backoff
	backoff := e.calculateBackoff(job.Attempts)
	e.queue.MarkRetry(ctx, job.ID, backoff, err)
	emailsRetried.Inc(job.TemplateSlug)
	e.recordEvent(job.ID, "retry", fmt.Sprintf("attempt %d, backoff %s: %v", job.Attempts, backoff, err))

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

// recordEvent writes an event to the queue's BulkInserter if available.
func (e *Engine) recordEvent(emailID, eventType, details string) {
	if e.queue.Events != nil {
		e.queue.Events.RecordEvent(emailID, eventType, details)
	}
}

// updateQueueDepth refreshes the queue depth gauge from current stats.
func (e *Engine) updateQueueDepth() {
	stats, err := e.queue.Stats(e.ctx)
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
