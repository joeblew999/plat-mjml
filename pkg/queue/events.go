package queue

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// EventRecorder batches email event writes using go-zero's BulkInserter.
type EventRecorder struct {
	inserter *sqlx.BulkInserter
}

// NewEventRecorder creates a new event recorder that batches inserts.
func NewEventRecorder(conn sqlx.SqlConn) (*EventRecorder, error) {
	inserter, err := sqlx.NewBulkInserter(conn,
		"insert into `email_events` (`id`, `email_id`, `event_type`, `timestamp`, `details`) values (?, ?, ?, ?, ?)")
	if err != nil {
		return nil, err
	}

	inserter.SetResultHandler(func(_ sql.Result, err error) {
		if err != nil {
			logx.Errorf("BulkInserter email_events error: %v", err)
		}
	})

	return &EventRecorder{inserter: inserter}, nil
}

// RecordEvent batches an email event insert.
func (r *EventRecorder) RecordEvent(emailID, eventType, details string) {
	if err := r.inserter.Insert(
		uuid.New().String(),
		emailID,
		eventType,
		time.Now().Format(time.RFC3339),
		details,
	); err != nil {
		logx.Errorf("Failed to record event: %v", err)
	}
}

// Flush forces all pending events to be written.
func (r *EventRecorder) Flush() {
	r.inserter.Flush()
}
