package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/yourorg/go-service-kit/pkg/errors"
	"github.com/yourorg/go-service-kit/pkg/logging"
)

const ServiceHandledHeader = "X-Service-Handled"

// ErrorHandlerMiddleware provides centralized error handling for HTTP handlers.
// It catches AppError from context and converts them to proper HTTP responses.
func ErrorHandlerMiddleware(logger logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there's an error in the context
		if len(c.Errors) > 0 {
			// Get the last error
			err := c.Errors.Last()

			// Try to convert to AppError
			var appErr *errors.AppError
			if appErrVal, ok := err.Err.(*errors.AppError); ok {
				appErr = appErrVal
			} else {
				// Wrap unknown errors as internal errors
				appErr = errors.FromError(err.Err)
			}

			// Get contextual logger (already has trace_id and request_id)
			ctxLogger := logging.FromContext(c.Request.Context())

			// Log the error
			ctxLogger.Error("Request failed",
				logging.NewField("error", appErr.Error()),
				logging.NewField("status_code", appErr.HTTPStatus),
				logging.NewField("handled_by_service", appErr.HandledByService),
			)

			// Set X-Service-Handled header if service handled the error
			if appErr.HandledByService {
				c.Header(ServiceHandledHeader, "true")
			}

			// Return error response (only if not already written)
			if !c.Writer.Written() {
				c.JSON(appErr.HTTPStatus, appErr.ToErrorResponse())
			}
		}
	}
}

// SetError sets an error in the Gin context to be handled by ErrorHandlerMiddleware.
func SetError(c *gin.Context, err *errors.AppError) {
	c.Error(err)
	c.Abort()
}

