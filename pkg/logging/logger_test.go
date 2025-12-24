package logging

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestEnrichFields(t *testing.T) {
	// Setup observer to capture logs
	core, _ := observer.New(zapcore.DebugLevel)
	logger := zap.New(core)
	zl := &zapLogger{logger: logger}

	t.Run("Extract request_id from context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "request_id", "req-123")
		fields := zl.enrichFields(ctx, []Field{})

		assert.Len(t, fields, 1)
		assert.Equal(t, "request_id", fields[0].Key)
		assert.Equal(t, "req-123", fields[0].Value)
	})

	t.Run("Extract x-request-id from context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "x-request-id", "x-req-456")
		fields := zl.enrichFields(ctx, []Field{})

		assert.Len(t, fields, 1)
		assert.Equal(t, "x-request-id", fields[0].Key)
		assert.Equal(t, "x-req-456", fields[0].Value)
	})

	t.Run("Prioritize request_id over x-request-id", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "request_id", "req-primary")
		ctx = context.WithValue(ctx, "x-request-id", "req-secondary")
		fields := zl.enrichFields(ctx, []Field{})

		assert.Len(t, fields, 1)
		assert.Equal(t, "request_id", fields[0].Key)
		assert.Equal(t, "req-primary", fields[0].Value)
	})

	t.Run("No request ID in context", func(t *testing.T) {
		ctx := context.Background()
		fields := zl.enrichFields(ctx, []Field{})
		assert.Empty(t, fields)
	})
}

func TestWithContextMethods(t *testing.T) {
	core, observedLogs := observer.New(zapcore.DebugLevel)
	logger := zap.New(core)
	zl := &zapLogger{logger: logger}

	ctx := context.WithValue(context.Background(), "x-request-id", "test-req-id")

	t.Run("InfoWithContext", func(t *testing.T) {
		zl.InfoWithContext(ctx, "test message", NewField("key", "value"))

		logs := observedLogs.All()
		assert.Equal(t, 1, len(logs))
		assert.Equal(t, "test message", logs[0].Message)

		contextFieldFound := false
		for _, f := range logs[0].Context {
			if f.Key == "x-request-id" && f.String == "test-req-id" {
				contextFieldFound = true
			}
		}
		assert.True(t, contextFieldFound, "x-request-id field not found in logs")
	})
}

func TestFWithContextMethods(t *testing.T) {
	core, observedLogs := observer.New(zapcore.DebugLevel)
	logger := zap.New(core)
	zl := &zapLogger{logger: logger}

	ctx := context.WithValue(context.Background(), "x-request-id", "fmt-req-id")

	t.Run("InfofWithContext", func(t *testing.T) {
		// Clear previous logs
		observedLogs.TakeAll()

		zl.InfofWithContext(ctx, "hello %s", "world")

		logs := observedLogs.All()
		assert.Equal(t, 1, len(logs))
		assert.Equal(t, "hello world", logs[0].Message)

		contextFieldFound := false
		for _, f := range logs[0].Context {
			if f.Key == "x-request-id" && f.String == "fmt-req-id" {
				contextFieldFound = true
			}
		}
		assert.True(t, contextFieldFound, "x-request-id field not found in logs")
	})
}
