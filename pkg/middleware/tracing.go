package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/go-service-kit/pkg/logging"
	"github.com/yourorg/go-service-kit/pkg/utils"
)

const (
	TraceIDKey    = "trace_id"
	TraceIDHeader = "X-Trace-ID"
)

// TracingMiddleware generates or extracts trace ID and attaches it to context.
// If X-Trace-ID header is present, it uses that; otherwise generates a new UUID.
func TracingMiddleware(logger logging.Logger, serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract or generate Trace ID
		traceID := c.GetHeader(TraceIDHeader)
		if traceID == "" {
			traceID = utils.GenerateUUID()
			logger.Debug("Trace ID missing, generated new one",
				logging.NewField("service", serviceName),
				logging.NewField("trace_id", traceID),
			)
		}

		// Store in context
		ctx := context.WithValue(c.Request.Context(), TraceIDKey, traceID)
		c.Request = c.Request.WithContext(ctx)

		// Set in Gin context for easy access
		c.Set(TraceIDKey, traceID)

		// Add header to response
		c.Header(TraceIDHeader, traceID)

		c.Next()
	}
}

// GetTraceID retrieves the trace ID from context.
func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// GetTraceIDFromGin retrieves the trace ID from Gin context.
func GetTraceIDFromGin(c *gin.Context) string {
	if traceID, exists := c.Get(TraceIDKey); exists {
		if id, ok := traceID.(string); ok {
			return id
		}
	}
	return ""
}

