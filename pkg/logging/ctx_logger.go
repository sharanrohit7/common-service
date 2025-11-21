package logging

import (
	"context"
)

type contextKey string

const loggerKey contextKey = "logger"

// WithLogger attaches a logger to the context.
func WithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext retrieves the logger from the context.
// Returns a no-op logger if not found.
func FromContext(ctx context.Context) Logger {
	if logger, ok := ctx.Value(loggerKey).(Logger); ok {
		return logger
	}
	return &noOpLogger{}
}

// noOpLogger is a logger that does nothing (useful for tests or when logger is not available).
type noOpLogger struct{}

func (n *noOpLogger) Debug(msg string, fields ...Field)                {}
func (n *noOpLogger) Info(msg string, fields ...Field)                 {}
func (n *noOpLogger) Warn(msg string, fields ...Field)                {}
func (n *noOpLogger) Error(msg string, fields ...Field)               {}
func (n *noOpLogger) Fatal(msg string, fields ...Field)               {}
func (n *noOpLogger) With(fields ...Field) Logger                      { return n }
func (n *noOpLogger) WithError(err error) Logger                       { return n }

