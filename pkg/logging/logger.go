package logging

import (
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
