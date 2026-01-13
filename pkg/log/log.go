// Package log provides structured logging utilities for the MJML package.
package log

import (
	"log/slog"
	"os"
)

var logger *slog.Logger

func init() {
	// Initialize with default JSON handler for structured logging
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger = slog.New(handler)
}

// SetLogger allows setting a custom logger.
func SetLogger(l *slog.Logger) {
	logger = l
}

// GetLogger returns the current logger instance.
func GetLogger() *slog.Logger {
	return logger
}

// Debug logs a debug message with optional key-value pairs.
func Debug(msg string, args ...any) {
	logger.Debug(msg, args...)
}

// Info logs an info message with optional key-value pairs.
func Info(msg string, args ...any) {
	logger.Info(msg, args...)
}

// Warn logs a warning message with optional key-value pairs.
func Warn(msg string, args ...any) {
	logger.Warn(msg, args...)
}

// Error logs an error message with optional key-value pairs.
func Error(msg string, args ...any) {
	logger.Error(msg, args...)
}
