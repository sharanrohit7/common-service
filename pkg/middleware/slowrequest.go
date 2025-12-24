package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/go-service-kit/pkg/logging"
)

// TelemetryClient defines the interface for telemetry operations.
type TelemetryClient interface {
	RecordSlowRequest(ctx interface{}, path string, durationMs int64, traceID, requestID string)
	RecordError(ctx interface{}, path, errorMsg string, statusCode int, traceID, requestID string)
}

// SlackClient defines the interface for Slack notifications.
type SlackClient interface {
	SendSlowRequestAlert(ctx interface{}, path string, durationMs int64, traceID, requestID string) error
	SendErrorAlert(ctx interface{}, path, errorMsg string, statusCode int, traceID, requestID string) error
}

// SlowRequestMiddleware detects slow requests and triggers alerts.
// It respects the X-Service-Handled header to avoid duplicate alerts.
func SlowRequestMiddleware(
	slowThresholdMs int64,
	telemetryClient TelemetryClient,
	slackClient SlackClient,
	logger logging.Logger,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)
		latencyMs := latency.Milliseconds()
		statusCode := c.Writer.Status()

		// Get trace and request IDs
		traceID := GetTraceIDFromGin(c)
		requestID := GetRequestIDFromGin(c)

		// Check if downstream service already handled the error
		serviceHandled := c.GetHeader(ServiceHandledHeader) == "true"

		// Check for slow request
		if latencyMs > slowThresholdMs && !serviceHandled {
			logger.Warn("Slow request detected",
				logging.NewField("path", path),
				logging.NewField("duration_ms", latencyMs),
				logging.NewField("threshold_ms", slowThresholdMs),
			)

			if telemetryClient != nil {
				telemetryClient.RecordSlowRequest(c.Request.Context(), path, latencyMs, traceID, requestID)
			}

			if slackClient != nil {
				if err := slackClient.SendSlowRequestAlert(c.Request.Context(), path, latencyMs, traceID, requestID); err != nil {
					logger.Error("Failed to send Slack alert", logging.NewField("error", err))
				}
			}
		}

		// Check for errors (5xx) that weren't handled by service
		if statusCode >= 500 && !serviceHandled {
			errorMsg := "Internal server error"
			if c.Errors != nil && len(c.Errors) > 0 {
				errorMsg = c.Errors.String()
			}

			// Only send telemetry, don't log (ErrorHandlerMiddleware already logged it)
			if telemetryClient != nil {
				telemetryClient.RecordError(c.Request.Context(), path, errorMsg, statusCode, traceID, requestID)
			}

			if slackClient != nil {
				if err := slackClient.SendErrorAlert(c.Request.Context(), path, errorMsg, statusCode, traceID, requestID); err != nil {
					logger.Error("Failed to send Slack alert", logging.NewField("error", err))
				}
			}
		}
	}
}
