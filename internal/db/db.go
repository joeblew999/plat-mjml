// Package db provides SQLite database operations for the email platform.
package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
	_ "modernc.org/sqlite"
)

// DB wraps a SQLite database connection.
type DB struct {
	*sql.DB
	path string
}

// Open opens or creates a SQLite database at the given path.
func Open(path string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Enable foreign keys and WAL mode for better performance
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA busy_timeout = 5000",
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("execute pragma %q: %w", pragma, err)
		}
	}

	d := &DB{DB: db, path: path}

	// Run migrations
	if err := d.Migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return d, nil
}

// Path returns the database file path.
func (d *DB) Path() string {
	return d.path
}

// Migrate runs database migrations.
func (d *DB) Migrate() error {
	schema := `
	-- Templates with versioning
	CREATE TABLE IF NOT EXISTS templates (
		id TEXT PRIMARY KEY,
		slug TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		content TEXT NOT NULL,
		version INTEGER DEFAULT 1,
		status TEXT DEFAULT 'draft',
		category TEXT,
		metadata TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		published_at DATETIME
	);

	CREATE INDEX IF NOT EXISTS idx_templates_slug ON templates(slug);
	CREATE INDEX IF NOT EXISTS idx_templates_status ON templates(status);

	-- Template versions (history)
	CREATE TABLE IF NOT EXISTS template_versions (
		id TEXT PRIMARY KEY,
		template_id TEXT NOT NULL,
		version INTEGER NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (template_id) REFERENCES templates(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_versions_template ON template_versions(template_id);

	-- Email queue
	CREATE TABLE IF NOT EXISTS emails (
		id TEXT PRIMARY KEY,
		template_slug TEXT NOT NULL,
		recipients TEXT NOT NULL,
		subject TEXT NOT NULL,
		data TEXT,
		status TEXT DEFAULT 'pending',
		priority INTEGER DEFAULT 1,
		attempts INTEGER DEFAULT 0,
		max_attempts INTEGER DEFAULT 3,
		scheduled_at DATETIME,
		sent_at DATETIME,
		message_id TEXT,
		error TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_emails_status ON emails(status);
	CREATE INDEX IF NOT EXISTS idx_emails_scheduled ON emails(scheduled_at);
	CREATE INDEX IF NOT EXISTS idx_emails_template ON emails(template_slug);

	-- Email events (for tracking)
	CREATE TABLE IF NOT EXISTS email_events (
		id TEXT PRIMARY KEY,
		email_id TEXT NOT NULL,
		event_type TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		details TEXT,
		FOREIGN KEY (email_id) REFERENCES emails(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_events_email ON email_events(email_id);
	CREATE INDEX IF NOT EXISTS idx_events_type ON email_events(event_type);

	-- SMTP providers
	CREATE TABLE IF NOT EXISTS smtp_providers (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		host TEXT NOT NULL,
		port INTEGER NOT NULL,
		username TEXT,
		password TEXT,
		from_email TEXT NOT NULL,
		from_name TEXT,
		is_default INTEGER DEFAULT 0,
		rate_limit INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := d.Exec(schema)
	return err
}

// SqlConn returns a go-zero sqlx.SqlConn wrapping the underlying database.
// This provides automatic circuit breaking and OpenTelemetry tracing on every query.
func (d *DB) SqlConn() sqlx.SqlConn {
	return sqlx.NewSqlConnFromDB(d.DB, sqlx.WithAcceptable(sqliteAcceptable))
}

// sqliteAcceptable tells the circuit breaker that "database is locked" errors
// are transient (SQLite WAL contention) and should not trip the breaker.
func sqliteAcceptable(err error) bool {
	return err == nil || strings.Contains(err.Error(), "database is locked")
}

