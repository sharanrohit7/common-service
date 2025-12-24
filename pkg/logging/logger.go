package logging

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger defines the interface for structured logging.
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)

	DebugWithContext(ctx context.Context, msg string, fields ...Field)
	InfoWithContext(ctx context.Context, msg string, fields ...Field)
	WarnWithContext(ctx context.Context, msg string, fields ...Field)
	ErrorWithContext(ctx context.Context, msg string, fields ...Field)
	FatalWithContext(ctx context.Context, msg string, fields ...Field)

	DebugfWithContext(ctx context.Context, format string, args ...interface{})
	InfofWithContext(ctx context.Context, format string, args ...interface{})
	WarnfWithContext(ctx context.Context, format string, args ...interface{})
	ErrorfWithContext(ctx context.Context, format string, args ...interface{})
	FatalfWithContext(ctx context.Context, format string, args ...interface{})

	With(fields ...Field) Logger
	WithError(err error) Logger
}

// Field represents a key-value pair for structured logging.
type Field struct {
	Key   string
	Value interface{}
}

// NewField creates a new log field.
func NewField(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// zapLogger is the zap-based implementation of Logger.
type zapLogger struct {
	logger *zap.Logger
}

// NewLogger creates a new logger with the specified level and format.
// level: debug, info, warn, error
// format: json, text
func NewLogger(level, format string) (Logger, error) {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	// Create encoder config
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zapLevel)
	config.Encoding = format
	config.EncoderConfig = encoderConfig

	// Disable caller and stacktrace (too verbose)
	config.DisableCaller = true
	config.DisableStacktrace = true

	// Build logger
	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return &zapLogger{logger: logger}, nil
}

// NewLoggerFromConfig creates a logger from a config source.
func NewLoggerFromConfig(level, format string) Logger {
	logger, err := NewLogger(level, format)
	if err != nil {
		// Fallback to default logger if config is invalid
		logger, _ = NewLogger("info", "json")
	}
	return logger
}

// Debug logs a debug message.
func (z *zapLogger) Debug(msg string, fields ...Field) {
	z.logger.Debug(msg, z.fieldsToZap(fields)...)
}

// Info logs an info message.
func (z *zapLogger) Info(msg string, fields ...Field) {
	z.logger.Info(msg, z.fieldsToZap(fields)...)
}

// Warn logs a warning message.
func (z *zapLogger) Warn(msg string, fields ...Field) {
	z.logger.Warn(msg, z.fieldsToZap(fields)...)
}

// Error logs an error message.
func (z *zapLogger) Error(msg string, fields ...Field) {
	z.logger.Error(msg, z.fieldsToZap(fields)...)
}

// Fatal logs a fatal message and exits.
func (z *zapLogger) Fatal(msg string, fields ...Field) {
	z.logger.Fatal(msg, z.fieldsToZap(fields)...)
}

// DebugWithContext logs a debug message with context.
func (z *zapLogger) DebugWithContext(ctx context.Context, msg string, fields ...Field) {
	z.logger.Debug(msg, z.fieldsToZap(z.enrichFields(ctx, fields))...)
}

// InfoWithContext logs an info message with context.
func (z *zapLogger) InfoWithContext(ctx context.Context, msg string, fields ...Field) {
	z.logger.Info(msg, z.fieldsToZap(z.enrichFields(ctx, fields))...)
}

// WarnWithContext logs a warning message with context.
func (z *zapLogger) WarnWithContext(ctx context.Context, msg string, fields ...Field) {
	z.logger.Warn(msg, z.fieldsToZap(z.enrichFields(ctx, fields))...)
}

// ErrorWithContext logs an error message with context.
func (z *zapLogger) ErrorWithContext(ctx context.Context, msg string, fields ...Field) {
	z.logger.Error(msg, z.fieldsToZap(z.enrichFields(ctx, fields))...)
}

// FatalWithContext logs a fatal message with context and exits.
func (z *zapLogger) FatalWithContext(ctx context.Context, msg string, fields ...Field) {
	z.logger.Fatal(msg, z.fieldsToZap(z.enrichFields(ctx, fields))...)
}

// DebugfWithContext logs a formatted debug message with context.
func (z *zapLogger) DebugfWithContext(ctx context.Context, format string, args ...interface{}) {
	z.logger.Debug(fmt.Sprintf(format, args...), z.fieldsToZap(z.enrichFields(ctx, nil))...)
}

// InfofWithContext logs a formatted info message with context.
func (z *zapLogger) InfofWithContext(ctx context.Context, format string, args ...interface{}) {
	z.logger.Info(fmt.Sprintf(format, args...), z.fieldsToZap(z.enrichFields(ctx, nil))...)
}

// WarnfWithContext logs a formatted warning message with context.
func (z *zapLogger) WarnfWithContext(ctx context.Context, format string, args ...interface{}) {
	z.logger.Warn(fmt.Sprintf(format, args...), z.fieldsToZap(z.enrichFields(ctx, nil))...)
}

// ErrorfWithContext logs a formatted error message with context.
func (z *zapLogger) ErrorfWithContext(ctx context.Context, format string, args ...interface{}) {
	z.logger.Error(fmt.Sprintf(format, args...), z.fieldsToZap(z.enrichFields(ctx, nil))...)
}

// FatalfWithContext logs a formatted fatal message with context and exits.
func (z *zapLogger) FatalfWithContext(ctx context.Context, format string, args ...interface{}) {
	z.logger.Fatal(fmt.Sprintf(format, args...), z.fieldsToZap(z.enrichFields(ctx, nil))...)
}

// enrichFields adds context fields to the log fields.
func (z *zapLogger) enrichFields(ctx context.Context, fields []Field) []Field {
	if ctx == nil {
		return fields
	}
	// Try standard "request_id" key and "x-request-id"
	if reqID, ok := ctx.Value("request_id").(string); ok && reqID != "" {
		fields = append(fields, NewField("request_id", reqID))
	} else if reqID, ok := ctx.Value("x-request-id").(string); ok && reqID != "" {
		fields = append(fields, NewField("x-request-id", reqID))
	}
	return fields
}

// With creates a new logger with additional fields.
func (z *zapLogger) With(fields ...Field) Logger {
	return &zapLogger{logger: z.logger.With(z.fieldsToZap(fields)...)}
}

// WithError creates a new logger with an error field.
func (z *zapLogger) WithError(err error) Logger {
	return &zapLogger{logger: z.logger.With(zap.Error(err))}
}

// fieldsToZap converts Field slice to zap fields.
func (z *zapLogger) fieldsToZap(fields []Field) []zap.Field {
	zapFields := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		switch v := f.Value.(type) {
		case string:
			zapFields = append(zapFields, zap.String(f.Key, v))
		case int:
			zapFields = append(zapFields, zap.Int(f.Key, v))
		case int64:
			zapFields = append(zapFields, zap.Int64(f.Key, v))
		case float64:
			zapFields = append(zapFields, zap.Float64(f.Key, v))
		case bool:
			zapFields = append(zapFields, zap.Bool(f.Key, v))
		case error:
			zapFields = append(zapFields, zap.Error(v))
		default:
			zapFields = append(zapFields, zap.Any(f.Key, v))
		}
	}
	return zapFields
}

// Sync flushes any buffered log entries. Should be called before application exit.
func Sync(logger Logger) {
	if zl, ok := logger.(*zapLogger); ok {
		_ = zl.logger.Sync()
	}
}
