package model

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ EmailsModel = (*customEmailsModel)(nil)

type (
	// EmailsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customEmailsModel.
	EmailsModel interface {
		emailsModel
		withSession(session sqlx.Session) EmailsModel
		ListByStatus(ctx context.Context, status string, limit int) ([]*Emails, error)
		Stats(ctx context.Context) (map[string]int, error)
		UpdateStatus(ctx context.Context, id, status string, errMsg string) error
		MarkSent(ctx context.Context, id, messageID string) error
		Enqueue(ctx context.Context, templateSlug string, recipients []string, subject string, data map[string]any, priority int) (string, error)
		Receive(ctx context.Context) (*Emails, error)
		MarkRetry(ctx context.Context, id string, retryAt time.Time, errMsg string) error
		MarkFailed(ctx context.Context, id string, errMsg string) error
	}

	customEmailsModel struct {
		*defaultEmailsModel
	}
)

// NewEmailsModel returns a model for the database table.
func NewEmailsModel(conn sqlx.SqlConn) EmailsModel {
	return &customEmailsModel{
		defaultEmailsModel: newEmailsModel(conn),
	}
}

func (m *customEmailsModel) withSession(session sqlx.Session) EmailsModel {
	return NewEmailsModel(sqlx.NewSqlConnFromSession(session))
}

// ListByStatus returns emails filtered by status with a limit.
func (m *customEmailsModel) ListByStatus(ctx context.Context, status string, limit int) ([]*Emails, error) {
	var resp []*Emails
	var query string
	var args []any

	if status != "" && status != "all" {
		query = fmt.Sprintf("select %s from %s where `status` = ? order by `created_at` desc limit ?", emailsRows, m.table)
		args = []any{status, limit}
	} else {
		query = fmt.Sprintf("select %s from %s order by `created_at` desc limit ?", emailsRows, m.table)
		args = []any{limit}
	}

	err := m.conn.QueryRowsCtx(ctx, &resp, query, args...)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// Stats returns email counts grouped by status.
func (m *customEmailsModel) Stats(ctx context.Context) (map[string]int, error) {
	type statusCount struct {
		Status string `db:"status"`
		Count  int    `db:"count"`
	}

	var rows []statusCount
	query := fmt.Sprintf("select `status`, count(*) as `count` from %s group by `status`", m.table)
	err := m.conn.QueryRowsCtx(ctx, &rows, query)
	if err != nil {
		return nil, err
	}

	stats := make(map[string]int)
	for _, r := range rows {
		stats[r.Status] = r.Count
	}
	return stats, nil
}

// UpdateStatus updates the status and error of an email, incrementing attempts.
func (m *customEmailsModel) UpdateStatus(ctx context.Context, id, status string, errMsg string) error {
	var query string
	var args []any

	if errMsg != "" {
		query = fmt.Sprintf("update %s set `status` = ?, `error` = ?, `attempts` = `attempts` + 1, `updated_at` = CURRENT_TIMESTAMP where `id` = ?", m.table)
		args = []any{status, errMsg, id}
	} else {
		query = fmt.Sprintf("update %s set `status` = ?, `attempts` = `attempts` + 1, `updated_at` = CURRENT_TIMESTAMP where `id` = ?", m.table)
		args = []any{status, id}
	}

	_, err := m.conn.ExecCtx(ctx, query, args...)
	return err
}

// MarkSent marks an email as successfully sent.
func (m *customEmailsModel) MarkSent(ctx context.Context, id, messageID string) error {
	query := fmt.Sprintf("update %s set `status` = 'sent', `sent_at` = CURRENT_TIMESTAMP, `message_id` = ?, `updated_at` = CURRENT_TIMESTAMP where `id` = ?", m.table)
	_, err := m.conn.ExecCtx(ctx, query, messageID, id)
	return err
}

// Enqueue adds an email with status 'pending'. Generates a UUID and marshals
// recipients/data to JSON for storage. Returns the generated ID.
func (m *customEmailsModel) Enqueue(ctx context.Context, templateSlug string, recipients []string, subject string, data map[string]any, priority int) (string, error) {
	id := uuid.New().String()
	if priority == 0 {
		priority = PriorityNormal
	}

	recipientsJSON, err := json.Marshal(recipients)
	if err != nil {
		return "", fmt.Errorf("marshal recipients: %w", err)
	}

	var dataStr sql.NullString
	if len(data) > 0 {
		b, err := json.Marshal(data)
		if err != nil {
			return "", fmt.Errorf("marshal data: %w", err)
		}
		dataStr = sql.NullString{String: string(b), Valid: true}
	}

	query := fmt.Sprintf("insert into %s (`id`, `template_slug`, `recipients`, `subject`, `data`, `status`, `priority`, `attempts`, `max_attempts`, `created_at`, `updated_at`) values (?, ?, ?, ?, ?, 'pending', ?, 0, 3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)", m.table)
	_, err = m.conn.ExecCtx(ctx, query, id, templateSlug, string(recipientsJSON), subject, dataStr, priority)
	if err != nil {
		return "", fmt.Errorf("enqueue email: %w", err)
	}

	return id, nil
}

// Receive atomically finds the next pending/retry job and marks it as 'processing'.
// Uses TransactCtx for atomic SELECT + UPDATE. Returns nil if no jobs available.
func (m *customEmailsModel) Receive(ctx context.Context) (*Emails, error) {
	var row Emails
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		query := fmt.Sprintf("select %s from %s where `status` in ('pending', 'retry') and (`scheduled_at` is null or `scheduled_at` <= datetime('now')) order by `priority` desc, `created_at` asc limit 1", emailsRows, m.table)
		if err := session.QueryRowCtx(ctx, &row, query); err != nil {
			return err
		}
		_, err := session.ExecCtx(ctx,
			fmt.Sprintf("update %s set `status` = 'processing', `updated_at` = CURRENT_TIMESTAMP where `id` = ?", m.table),
			row.Id)
		return err
	})

	switch err {
	case nil:
		return &row, nil
	case sqlx.ErrNotFound:
		return nil, nil
	default:
		return nil, err
	}
}

// MarkRetry schedules an email for retry at the given time.
func (m *customEmailsModel) MarkRetry(ctx context.Context, id string, retryAt time.Time, errMsg string) error {
	var errStr sql.NullString
	if errMsg != "" {
		errStr = sql.NullString{String: errMsg, Valid: true}
	}
	query := fmt.Sprintf("update %s set `status` = 'retry', `error` = ?, `attempts` = `attempts` + 1, `scheduled_at` = ?, `updated_at` = CURRENT_TIMESTAMP where `id` = ?", m.table)
	_, err := m.conn.ExecCtx(ctx, query, errStr, retryAt, id)
	return err
}

// MarkFailed marks an email as permanently failed.
func (m *customEmailsModel) MarkFailed(ctx context.Context, id string, errMsg string) error {
	var errStr sql.NullString
	if errMsg != "" {
		errStr = sql.NullString{String: errMsg, Valid: true}
	}
	query := fmt.Sprintf("update %s set `status` = 'failed', `error` = ?, `attempts` = `attempts` + 1, `updated_at` = CURRENT_TIMESTAMP where `id` = ?", m.table)
	_, err := m.conn.ExecCtx(ctx, query, errStr, id)
	return err
}
