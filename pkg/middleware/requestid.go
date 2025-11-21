package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/go-service-kit/pkg/utils"
)

const (
	RequestIDKey    = "request_id"
	RequestIDHeader = "X-Request-ID"
)

// ServiceRequestIDMiddleware generates a service-specific request ID and attaches it to context.
// This is different from the gateway request ID - this is for internal service operations.
func ServiceRequestIDMiddleware(headerName string) gin.HandlerFunc {
	if headerName == "" {
		headerName = "X-Service-Request-ID"
	}

	return func(c *gin.Context) {
		// Generate service request ID
		requestID := utils.GenerateRequestID()

		// Store in context
		ctx := context.WithValue(c.Request.Context(), RequestIDKey, requestID)
		c.Request = c.Request.WithContext(ctx)

		// Set in Gin context
		c.Set(RequestIDKey, requestID)

		// Add header to response
		c.Header(headerName, requestID)

		c.Next()
	}
}

// GetRequestID retrieves the request ID from context.
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// GetRequestIDFromGin retrieves the request ID from Gin context.
func GetRequestIDFromGin(c *gin.Context) string {
	if requestID, exists := c.Get(RequestIDKey); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}

