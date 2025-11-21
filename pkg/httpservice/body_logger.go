package httpservice

import (
	"bytes"
	"encoding/json"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/go-service-kit/pkg/logging"
)

// responseWriter wraps gin.ResponseWriter to capture response body
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// BodyLoggingMiddleware logs full request and response bodies for complete observability.
// This provides comprehensive logging for debugging and monitoring.
func BodyLoggingMiddleware(logger logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Read and log request body
		var requestBody []byte
		var requestBodyJSON interface{}

		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			// Restore the body for handlers to read
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))

			// Try to parse as JSON for pretty logging
			if len(requestBody) > 0 {
				json.Unmarshal(requestBody, &requestBodyJSON)
			}
		}

		// Capture response body
		responseBodyWriter := &responseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = responseBodyWriter

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Parse response body as JSON if possible
		var responseBodyJSON interface{}
		if responseBodyWriter.body.Len() > 0 {
			json.Unmarshal(responseBodyWriter.body.Bytes(), &responseBodyJSON)
		}

		// Build comprehensive log fields
		fields := []logging.Field{
			logging.NewField("method", c.Request.Method),
			logging.NewField("path", c.Request.URL.Path),
			logging.NewField("status", c.Writer.Status()),
			logging.NewField("latency_ms", latency.Milliseconds()),
			logging.NewField("ip", c.ClientIP()),
			logging.NewField("user_agent", c.Request.UserAgent()),
		}

		// Add request ID
		if requestID, exists := c.Get("request_id"); exists {
			fields = append(fields, logging.NewField("request_id", requestID))
		}

		// Add query parameters if present
		if len(c.Request.URL.RawQuery) > 0 {
			fields = append(fields, logging.NewField("query_params", c.Request.URL.RawQuery))
		}

		// Add path parameters if present
		if len(c.Params) > 0 {
			params := make(map[string]string)
			for _, param := range c.Params {
				params[param.Key] = param.Value
			}
			fields = append(fields, logging.NewField("path_params", params))
		}

		// Add request headers (excluding sensitive ones)
		headers := make(map[string]string)
		for key, values := range c.Request.Header {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}
		fields = append(fields, logging.NewField("request_headers", headers))

		// Add request body
		if requestBodyJSON != nil {
			fields = append(fields, logging.NewField("request_body", requestBodyJSON))
		} else if len(requestBody) > 0 {
			fields = append(fields, logging.NewField("request_body_raw", string(requestBody)))
		}

		// Add response body
		if responseBodyJSON != nil {
			fields = append(fields, logging.NewField("response_body", responseBodyJSON))
		} else if responseBodyWriter.body.Len() > 0 {
			fields = append(fields, logging.NewField("response_body_raw", responseBodyWriter.body.String()))
		}

		// Add response size
		fields = append(fields, logging.NewField("response_size_bytes", responseBodyWriter.body.Len()))

		// Log based on status code
		if c.Writer.Status() >= 500 {
			logger.Error("HTTP Request/Response", fields...)
		} else if c.Writer.Status() >= 400 {
			logger.Warn("HTTP Request/Response", fields...)
		} else {
			logger.Info("HTTP Request/Response", fields...)
		}
	}
}
