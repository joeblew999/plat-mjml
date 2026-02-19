package model

import (
	"context"
	"fmt"

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

