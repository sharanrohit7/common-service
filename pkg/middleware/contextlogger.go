package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/yourorg/go-service-kit/pkg/logging"
)

// ContextLoggerMiddleware attaches a contextual logger to the request context.
// The logger is pre-populated with service, trace_id, and request_id fields.
func ContextLoggerMiddleware(baseLogger logging.Logger, serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get trace ID and request ID from context (set by previous middleware)
		traceID := GetTraceIDFromGin(c)
		requestID := GetRequestIDFromGin(c)

		// Create contextual logger with service, trace_id, and request_id
		fields := []logging.Field{
			logging.NewField("service", serviceName),
		}

		if traceID != "" {
			fields = append(fields, logging.NewField("trace_id", traceID))
		}

		if requestID != "" {
			fields = append(fields, logging.NewField("request_id", requestID))
		}

		ctxLogger := baseLogger.With(fields...)

		// Attach to context
		ctx := logging.WithLogger(c.Request.Context(), ctxLogger)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

