package httpservice

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/go-service-kit/pkg/logging"
)

// HandlerFunc is a handler function that returns an error.
// This allows for cleaner error handling and automatic logging.
type HandlerFunc func(c *gin.Context) error

// Wrap wraps a HandlerFunc with automatic logging and error handling.
// This eliminates boilerplate from handlers - they only need to contain business logic.
//
// Features:
// - Logs handler entry with handler name
// - Logs handler exit with latency
// - Automatically handles errors (logs and converts to HTTP response)
// - No manual logging needed in handlers
func Wrap(handlerName string, fn HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := logging.FromContext(c.Request.Context())
		start := time.Now()

		// Log entry
		logger.Debug("Handler started",
			logging.NewField("handler", handlerName),
			logging.NewField("method", c.Request.Method),
			logging.NewField("path", c.Request.URL.Path),
		)

		// Execute handler
		err := fn(c)

		// Calculate latency
		latency := time.Since(start)

		// Handle error if any
		if err != nil {
			logger.Error("Handler failed",
				logging.NewField("handler", handlerName),
				logging.NewField("latency_ms", latency.Milliseconds()),
				logging.NewField("error", err),
			)
			HandleError(c, err)
			return
		}

		// Log successful completion
		logger.Debug("Handler completed",
			logging.NewField("handler", handlerName),
			logging.NewField("latency_ms", latency.Milliseconds()),
		)
	}
}
