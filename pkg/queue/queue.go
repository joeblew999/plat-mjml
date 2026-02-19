// Package queue provides email queue operations using go-zero sqlx.
// All queue state is managed through the emails table directly — no external
// queue dependency required. Circuit breaking and OpenTelemetry tracing are
// automatic via sqlx.SqlConn.
package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
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
	Status       string         `json:"status"`
	Priority     int            `json:"priority"`
	Attempts     int            `json:"attempts"`
	MaxAttempts  int            `json:"max_attempts"`
	ScheduledAt  *time.Time     `json:"scheduled_at,omitempty"`
	Error        string         `json:"error,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

// Queue manages email jobs using go-zero sqlx directly on the emails table.
type Queue struct {
	conn   sqlx.SqlConn
	Events *EventRecorder
}

// NewQueue creates a new email queue backed by go-zero sqlx.
func NewQueue(conn sqlx.SqlConn) *Queue {
	events, _ := NewEventRecorder(conn)
	return &Queue{conn: conn, Events: events}
}

// Enqueue adds an email job with status 'pending'.
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

	recipients, err := json.Marshal(job.Recipients)
	if err != nil {
		return "", fmt.Errorf("marshal recipients: %w", err)
	}
	data, err := json.Marshal(job.Data)
	if err != nil {
		return "", fmt.Errorf("marshal data: %w", err)
	}

	var scheduledAt sql.NullTime
	if job.ScheduledAt != nil {
		scheduledAt = sql.NullTime{Time: *job.ScheduledAt, Valid: true}
	}

	_, err = q.conn.ExecCtx(ctx,
		"insert into `emails` (`id`, `template_slug`, `recipients`, `subject`, `data`, `status`, `priority`, `attempts`, `max_attempts`, `scheduled_at`, `created_at`, `updated_at`) values (?, ?, ?, ?, ?, 'pending', ?, 0, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
		job.ID, job.TemplateSlug, string(recipients), job.Subject, string(data),
		job.Priority, job.MaxAttempts, scheduledAt)
	if err != nil {
		return "", fmt.Errorf("enqueue email: %w", err)
	}

	return job.ID, nil
}

// Schedule adds an email job to be sent at a specific time.
func (q *Queue) Schedule(ctx context.Context, job EmailJob, at time.Time) (string, error) {
	job.ScheduledAt = &at
	return q.Enqueue(ctx, job)
}

// emailRow is used for sqlx struct scanning.
type emailRow struct {
	ID           string         `db:"id"`
	TemplateSlug string         `db:"template_slug"`
	Recipients   string         `db:"recipients"`
	Subject      string         `db:"subject"`
	Data         sql.NullString `db:"data"`
	Status       string         `db:"status"`
	Priority     int            `db:"priority"`
	Attempts     int            `db:"attempts"`
	MaxAttempts  int            `db:"max_attempts"`
	ScheduledAt  sql.NullTime   `db:"scheduled_at"`
	SentAt       sql.NullTime   `db:"sent_at"`
	Error        sql.NullString `db:"error"`
	CreatedAt    time.Time      `db:"created_at"`
}

const emailColumns = "`id`, `template_slug`, `recipients`, `subject`, `data`, `status`, `priority`, `attempts`, `max_attempts`, `scheduled_at`, `sent_at`, `error`, `created_at`"

func (r *emailRow) toJob() (*EmailJob, error) {
	job := &EmailJob{
		ID:           r.ID,
		TemplateSlug: r.TemplateSlug,
		Subject:      r.Subject,
		Status:       r.Status,
		Priority:     r.Priority,
		Attempts:     r.Attempts,
		MaxAttempts:  r.MaxAttempts,
		CreatedAt:    r.CreatedAt,
	}

	if err := json.Unmarshal([]byte(r.Recipients), &job.Recipients); err != nil {
		return nil, fmt.Errorf("unmarshal recipients: %w", err)
	}
	if r.Data.Valid {
		if err := json.Unmarshal([]byte(r.Data.String), &job.Data); err != nil {
			return nil, fmt.Errorf("unmarshal data: %w", err)
		}
	}
	if r.Error.Valid {
		job.Error = r.Error.String
	}
	if r.ScheduledAt.Valid {
		job.ScheduledAt = &r.ScheduledAt.Time
	}

	return job, nil
}

// Receive atomically finds the next pending/retry job and marks it as 'processing'.
// Uses TransactCtx for atomic SELECT + UPDATE. Returns nil if no jobs available.
func (q *Queue) Receive(ctx context.Context) (*EmailJob, error) {
	var row emailRow
	err := q.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		query := fmt.Sprintf("select %s from `emails` where `status` in ('pending', 'retry') and (`scheduled_at` is null or `scheduled_at` <= datetime('now')) order by `priority` desc, `created_at` asc limit 1", emailColumns)
		if err := session.QueryRowCtx(ctx, &row, query); err != nil {
			return err
		}
		_, err := session.ExecCtx(ctx,
			"update `emails` set `status` = 'processing', `updated_at` = CURRENT_TIMESTAMP where `id` = ?",
			row.ID)
		return err
	})

	switch err {
	case nil:
		return row.toJob()
	case sqlx.ErrNotFound:
		return nil, nil
	default:
		return nil, err
	}
}

// GetStatus returns the status of an email by ID.
func (q *Queue) GetStatus(ctx context.Context, id string) (*EmailJob, error) {
	var row emailRow
	query := fmt.Sprintf("select %s from `emails` where `id` = ? limit 1", emailColumns)
	err := q.conn.QueryRowCtx(ctx, &row, query, id)
	switch err {
	case nil:
		return row.toJob()
	case sqlx.ErrNotFound:
		return nil, nil
	default:
		return nil, err
	}
}

// UpdateStatus updates the status of an email, incrementing attempts.
func (q *Queue) UpdateStatus(ctx context.Context, id, status string, err error) error {
	var errStr sql.NullString
	if err != nil {
		errStr = sql.NullString{String: err.Error(), Valid: true}
	}
	_, dbErr := q.conn.ExecCtx(ctx,
		"update `emails` set `status` = ?, `error` = ?, `attempts` = `attempts` + 1, `updated_at` = CURRENT_TIMESTAMP where `id` = ?",
		status, errStr, id)
	return dbErr
}

// MarkSent marks an email as successfully sent.
func (q *Queue) MarkSent(ctx context.Context, id, messageID string) error {
	_, err := q.conn.ExecCtx(ctx,
		"update `emails` set `status` = 'sent', `sent_at` = CURRENT_TIMESTAMP, `message_id` = ?, `updated_at` = CURRENT_TIMESTAMP where `id` = ?",
		messageID, id)
	return err
}

// MarkRetry schedules an email for retry after a backoff delay.
func (q *Queue) MarkRetry(ctx context.Context, id string, backoff time.Duration, err error) error {
	retryAt := time.Now().Add(backoff)
	var errStr sql.NullString
	if err != nil {
		errStr = sql.NullString{String: err.Error(), Valid: true}
	}
	_, dbErr := q.conn.ExecCtx(ctx,
		"update `emails` set `status` = 'retry', `error` = ?, `attempts` = `attempts` + 1, `scheduled_at` = ?, `updated_at` = CURRENT_TIMESTAMP where `id` = ?",
		errStr, retryAt, id)
	return dbErr
}

// MarkFailed marks an email as permanently failed.
func (q *Queue) MarkFailed(ctx context.Context, id string, err error) error {
	var errStr sql.NullString
	if err != nil {
		errStr = sql.NullString{String: err.Error(), Valid: true}
	}
	_, dbErr := q.conn.ExecCtx(ctx,
		"update `emails` set `status` = 'failed', `error` = ?, `attempts` = `attempts` + 1, `updated_at` = CURRENT_TIMESTAMP where `id` = ?",
		errStr, id)
	return dbErr
}

// List returns jobs with optional status filter.
func (q *Queue) List(ctx context.Context, status string, limit int) ([]*EmailJob, error) {
	var rows []*emailRow
	var query string
	var args []any

	if status != "" && status != "all" {
		query = fmt.Sprintf("select %s from `emails` where `status` = ? order by `created_at` desc limit ?", emailColumns)
		args = []any{status, limit}
	} else {
		query = fmt.Sprintf("select %s from `emails` order by `created_at` desc limit ?", emailColumns)
		args = []any{limit}
	}

	err := q.conn.QueryRowsCtx(ctx, &rows, query, args...)
	if err != nil {
		return nil, err
	}

	jobs := make([]*EmailJob, 0, len(rows))
	for _, r := range rows {
		job, err := r.toJob()
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// Stats returns email counts grouped by status.
func (q *Queue) Stats(ctx context.Context) (map[string]int, error) {
	type statusCount struct {
		Status string `db:"status"`
		Count  int    `db:"count"`
	}

	var rows []statusCount
	err := q.conn.QueryRowsCtx(ctx, &rows,
		"select `status`, count(*) as `count` from `emails` group by `status`")
	if err != nil {
		return nil, err
	}

	stats := make(map[string]int)
	for _, r := range rows {
		stats[r.Status] = r.Count
	}
	return stats, nil
}
