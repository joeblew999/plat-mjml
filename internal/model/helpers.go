package model

import (
	"database/sql"
	"encoding/json"
)

// Priority levels for email delivery.
const (
	PriorityLow    = 0 // Marketing, newsletters
	PriorityNormal = 1 // Transactional
	PriorityHigh   = 2 // Password reset, security alerts
)

// ParseRecipients parses a JSON array string into a string slice.
func ParseRecipients(raw string) []string {
	var recipients []string
	_ = json.Unmarshal([]byte(raw), &recipients)
	return recipients
}

// ParseData parses a NullString containing JSON into a map.
func ParseData(raw sql.NullString) map[string]any {
	if !raw.Valid {
		return nil
	}
	var data map[string]any
	_ = json.Unmarshal([]byte(raw.String), &data)
	return data
}

// NullStringValue returns the string value or empty string.
func NullStringValue(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}
