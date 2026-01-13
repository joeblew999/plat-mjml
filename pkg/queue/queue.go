// Package queue provides email queue operations using goqite.
package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"maragu.dev/goqite"
)

// Priority levels for email delivery.
const (
	PriorityLow    = 0 // Marketing, newsletters
	PriorityNormal = 1 // Transactional
	PriorityHigh   = 2 // Password reset, security alerts
)

// EmailJob represents an email to be sent.
type EmailJob struct {
	ID           string         `json:"id"`
	TemplateSlug string         `json:"template_slug"`
	Recipients   []string       `json:"recipients"`
	Subject      string         `json:"subject"`
	Data         map[string]any `json:"data,omitempty"`
	Priority     int            `json:"priority"`
	Attempts     int            `json:"attempts"`
	MaxAttempts  int            `json:"max_attempts"`
	ScheduledAt  *time.Time     `json:"scheduled_at,omitempty"`
	Error        string         `json:"error,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

// Queue manages email jobs using goqite.
type Queue struct {
	db      *sql.DB
	queue   *goqite.Queue
	name    string
	workers int
}

// NewQueue creates a new email queue.
func NewQueue(db *sql.DB, name string, workers int) (*Queue, error) {
	// Setup goqite schema
	if err := goqite.Setup(context.Background(), db); err != nil {
		return nil, fmt.Errorf("setup goqite: %w", err)
	}

	q := goqite.New(goqite.NewOpts{
		DB:   db,
		Name: name,
	})

	return &Queue{
		db:      db,
		queue:   q,
		name:    name,
		workers: workers,
	}, nil
}

// Enqueue adds an email job to the queue.
func (q *Queue) Enqueue(ctx context.Context, job EmailJob) (string, error) {
	if job.ID == "" {
		job.ID = uuid.New().String()
	}
	if job.MaxAttempts == 0 {
		job.MaxAttempts = 3
	}
	if job.Priority == 0 {
		job.Priority = PriorityNormal
	}
	job.CreatedAt = time.Now()

	body, err := json.Marshal(job)
	if err != nil {
		return "", fmt.Errorf("marshal job: %w", err)
	}

	// Calculate delay based on scheduled time
	var delay time.Duration
	if job.ScheduledAt != nil && job.ScheduledAt.After(time.Now()) {
		delay = time.Until(*job.ScheduledAt)
	}

	if err := q.queue.Send(ctx, goqite.Message{
		Body:  body,
		Delay: delay,
	}); err != nil {
		return "", fmt.Errorf("send to queue: %w", err)
	}

	// Also store in emails table for tracking
	if err := q.storeEmail(ctx, job); err != nil {
		return "", fmt.Errorf("store email: %w", err)
	}

	return job.ID, nil
}

// Schedule adds an email job to be sent at a specific time.
func (q *Queue) Schedule(ctx context.Context, job EmailJob, at time.Time) (string, error) {
	job.ScheduledAt = &at
	return q.Enqueue(ctx, job)
}

// Receive gets the next job from the queue.
func (q *Queue) Receive(ctx context.Context) (*EmailJob, *goqite.Message, error) {
	msg, err := q.queue.Receive(ctx)
	if err != nil {
		return nil, nil, err
	}
	if msg == nil {
		return nil, nil, nil
	}

	var job EmailJob
	if err := json.Unmarshal(msg.Body, &job); err != nil {
		return nil, msg, fmt.Errorf("unmarshal job: %w", err)
	}

	return &job, msg, nil
}

// Extend extends the timeout for a message being processed.
func (q *Queue) Extend(ctx context.Context, msg *goqite.Message, d time.Duration) error {
	return q.queue.Extend(ctx, msg.ID, d)
}

// Delete removes a message from the queue (job completed).
func (q *Queue) Delete(ctx context.Context, msg *goqite.Message) error {
	return q.queue.Delete(ctx, msg.ID)
}

// GetStatus returns the status of an email by ID.
func (q *Queue) GetStatus(ctx context.Context, id string) (*EmailJob, error) {
	row := q.db.QueryRowContext(ctx, `
		SELECT id, template_slug, recipients, subject, data, status,
		       priority, attempts, max_attempts, scheduled_at, sent_at, error, created_at
		FROM emails WHERE id = ?
	`, id)

	var job EmailJob
	var recipients, data, status string
	var scheduledAt, sentAt sql.NullTime
	var errStr sql.NullString

	err := row.Scan(
		&job.ID, &job.TemplateSlug, &recipients, &job.Subject, &data,
		&status, &job.Priority, &job.Attempts, &job.MaxAttempts,
		&scheduledAt, &sentAt, &errStr, &job.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	json.Unmarshal([]byte(recipients), &job.Recipients)
	json.Unmarshal([]byte(data), &job.Data)
	if errStr.Valid {
		job.Error = errStr.String
	}
	if scheduledAt.Valid {
		job.ScheduledAt = &scheduledAt.Time
	}

	return &job, nil
}

// UpdateStatus updates the status of an email.
func (q *Queue) UpdateStatus(ctx context.Context, id, status string, err error) error {
	var errStr sql.NullString
	if err != nil {
		errStr = sql.NullString{String: err.Error(), Valid: true}
	}

	_, dbErr := q.db.ExecContext(ctx, `
		UPDATE emails
		SET status = ?, error = ?, updated_at = CURRENT_TIMESTAMP,
		    attempts = attempts + 1
		WHERE id = ?
	`, status, errStr, id)
	return dbErr
}

// MarkSent marks an email as successfully sent.
func (q *Queue) MarkSent(ctx context.Context, id, messageID string) error {
	_, err := q.db.ExecContext(ctx, `
		UPDATE emails
		SET status = 'sent', sent_at = CURRENT_TIMESTAMP,
		    message_id = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, messageID, id)
	return err
}

// List returns jobs from the queue with optional status filter.
func (q *Queue) List(ctx context.Context, status string, limit int) ([]*EmailJob, error) {
	query := `
		SELECT id, template_slug, recipients, subject, data, status,
		       priority, attempts, max_attempts, scheduled_at, sent_at, error, created_at
		FROM emails
	`
	args := []any{}

	if status != "" && status != "all" {
		query += " WHERE status = ?"
		args = append(args, status)
	}

	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*EmailJob
	for rows.Next() {
		var job EmailJob
		var recipients, data, jobStatus string
		var scheduledAt, sentAt sql.NullTime
		var errStr sql.NullString

		err := rows.Scan(
			&job.ID, &job.TemplateSlug, &recipients, &job.Subject, &data,
			&jobStatus, &job.Priority, &job.Attempts, &job.MaxAttempts,
			&scheduledAt, &sentAt, &errStr, &job.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		json.Unmarshal([]byte(recipients), &job.Recipients)
		json.Unmarshal([]byte(data), &job.Data)
		if errStr.Valid {
			job.Error = errStr.String
		}
		if scheduledAt.Valid {
			job.ScheduledAt = &scheduledAt.Time
		}

		jobs = append(jobs, &job)
	}

	return jobs, nil
}

// Stats returns queue statistics.
func (q *Queue) Stats(ctx context.Context) (map[string]int, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT status, COUNT(*) as count FROM emails GROUP BY status
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats[status] = count
	}

	return stats, nil
}

func (q *Queue) storeEmail(ctx context.Context, job EmailJob) error {
	recipients, _ := json.Marshal(job.Recipients)
	data, _ := json.Marshal(job.Data)

	var scheduledAt sql.NullTime
	if job.ScheduledAt != nil {
		scheduledAt = sql.NullTime{Time: *job.ScheduledAt, Valid: true}
	}

	_, err := q.db.ExecContext(ctx, `
		INSERT INTO emails (id, template_slug, recipients, subject, data, status,
		                    priority, attempts, max_attempts, scheduled_at, created_at)
		VALUES (?, ?, ?, ?, ?, 'pending', ?, 0, ?, ?, CURRENT_TIMESTAMP)
	`, job.ID, job.TemplateSlug, string(recipients), job.Subject, string(data),
		job.Priority, job.MaxAttempts, scheduledAt)

	return err
}
