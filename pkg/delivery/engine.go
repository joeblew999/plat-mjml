// Package delivery provides the email delivery engine with retry support.
package delivery

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/joeblew999/plat-mjml/pkg/mail"
	"github.com/joeblew999/plat-mjml/pkg/mjml"
	"github.com/joeblew999/plat-mjml/pkg/queue"
	"golang.org/x/time/rate"
	"maragu.dev/goqite"
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

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
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
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start starts the delivery engine with the specified number of workers.
func (e *Engine) Start(workers int) {
	for i := 0; i < workers; i++ {
		e.wg.Add(1)
		go e.worker(i)
	}
}

// Stop gracefully stops the delivery engine.
func (e *Engine) Stop() {
	e.cancel()
	e.wg.Wait()
}

func (e *Engine) worker(id int) {
	defer e.wg.Done()

	for {
		select {
		case <-e.ctx.Done():
			return
		default:
			job, msg, err := e.queue.Receive(e.ctx)
			if err != nil {
				time.Sleep(time.Second)
				continue
			}
			if job == nil {
				time.Sleep(time.Second)
				continue
			}

			e.processJob(job, msg)
		}
	}
}

func (e *Engine) processJob(job *queue.EmailJob, msg *goqite.Message) {
	ctx := e.ctx

	// Update status to processing
	e.queue.UpdateStatus(ctx, job.ID, "processing", nil)

	// Apply rate limiting
	if err := e.rateLimiter.Wait(ctx); err != nil {
		e.handleError(ctx, job, msg, err)
		return
	}

	// Render template
	html, err := e.renderer.RenderTemplate(job.TemplateSlug, job.Data)
	if err != nil {
		e.handleError(ctx, job, msg, fmt.Errorf("render template: %w", err))
		return
	}

	// Send email to each recipient
	for _, recipient := range job.Recipients {
		if err := mail.Send(e.smtpConfig, recipient, job.Subject, html); err != nil {
			e.handleError(ctx, job, msg, fmt.Errorf("send to %s: %w", recipient, err))
			return
		}
	}

	// Success - mark as sent and delete from queue
	e.queue.MarkSent(ctx, job.ID, "")
	e.queue.Delete(ctx, msg)
}

func (e *Engine) handleError(ctx context.Context, job *queue.EmailJob, msg *goqite.Message, err error) {
	job.Attempts++
	job.Error = err.Error()

	// Check if permanent failure
	if isPermanentFailure(err) || job.Attempts >= job.MaxAttempts {
		e.queue.UpdateStatus(ctx, job.ID, "failed", err)
		e.queue.Delete(ctx, msg)
		return
	}

	// Schedule retry with backoff
	backoff := e.calculateBackoff(job.Attempts)
	e.queue.UpdateStatus(ctx, job.ID, "retry", err)

	// Extend the message timeout to retry later
	e.queue.Extend(ctx, msg, backoff)
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
