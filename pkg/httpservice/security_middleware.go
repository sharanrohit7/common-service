package httpservice

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"html"
	"io"
	"net/http"
	"regexp"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/go-service-kit/pkg/logging"
)

// RequestSizeLimitMiddleware limits the maximum size of request bodies.
func RequestSizeLimitMiddleware(maxBytes int64, logger logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			logger.Warn("Request body too large",
				logging.NewField("content_length", c.Request.ContentLength),
				logging.NewField("max_bytes", maxBytes),
				logging.NewField("ip", c.ClientIP()),
			)
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "Request body too large",
			})
			return
		}

		// Wrap the body reader with a LimitReader
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

// CSRFMiddleware provides CSRF protection using the Double Submit Cookie pattern.
type CSRFMiddleware struct {
	tokens sync.Map
	logger logging.Logger
}

// NewCSRFMiddleware creates a new CSRF middleware.
func NewCSRFMiddleware(logger logging.Logger) *CSRFMiddleware {
	return &CSRFMiddleware{
		logger: logger,
	}
}

// Middleware returns the Gin middleware function.
func (csrf *CSRFMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip CSRF check for safe methods
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// Get CSRF token from header
		token := c.GetHeader("X-CSRF-Token")
		if token == "" {
			csrf.logger.Warn("Missing CSRF token", logging.NewField("ip", c.ClientIP()))
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "CSRF token required",
			})
			return
		}

		// Validate token (in production, use session-based validation)
		// This is a simplified implementation
		if _, ok := csrf.tokens.Load(token); !ok {
			csrf.logger.Warn("Invalid CSRF token", logging.NewField("ip", c.ClientIP()))
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Invalid CSRF token",
			})
			return
		}

		c.Next()
	}
}

// GenerateToken generates a new CSRF token.
func (csrf *CSRFMiddleware) GenerateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	token := base64.URLEncoding.EncodeToString(b)
	csrf.tokens.Store(token, true)
	return token
}

// XSSProtectionMiddleware provides XSS protection by encoding HTML in responses.
// Note: This is primarily for APIs that return HTML. JSON APIs are naturally protected.
func XSSProtectionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Wrap the response writer
		blw := &xssResponseWriter{ResponseWriter: c.Writer, context: c}
		c.Writer = blw
		c.Next()
	}
}

type xssResponseWriter struct {
	gin.ResponseWriter
	context *gin.Context
}

func (w *xssResponseWriter) Write(data []byte) (int, error) {
	contentType := w.Header().Get("Content-Type")

	// Only encode HTML responses
	if contentType == "text/html" || contentType == "text/html; charset=utf-8" {
		escaped := html.EscapeString(string(data))
		return w.ResponseWriter.Write([]byte(escaped))
	}

	return w.ResponseWriter.Write(data)
}

// EnhancedSQLInjectionCheckMiddleware checks for SQL injection in query params AND request body.
func EnhancedSQLInjectionCheckMiddleware(logger logging.Logger) gin.HandlerFunc {
	sqlInjectionPattern := regexp.MustCompile(`(?i)(union\s+select|or\s+1=1|--|;\s*drop|;\s*delete|;\s*update|;\s*insert|exec\s*\(|script\s*>|<\s*script)`)

	return func(c *gin.Context) {
		// Check query parameters
		for _, values := range c.Request.URL.Query() {
			for _, value := range values {
				if sqlInjectionPattern.MatchString(value) {
					logger.Warn("Potential SQL injection detected in query",
						logging.NewField("value", value),
						logging.NewField("ip", c.ClientIP()),
					)
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid input detected"})
					return
				}
			}
		}

		// Check JSON body
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			if c.ContentType() == "application/json" {
				// Read body
				bodyBytes, err := io.ReadAll(c.Request.Body)
				if err == nil && len(bodyBytes) > 0 {
					// Restore body for handlers
					c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

					// Parse JSON and check all string values
					var jsonData interface{}
					if err := json.Unmarshal(bodyBytes, &jsonData); err == nil {
						if checkJSONForSQLInjection(jsonData, sqlInjectionPattern) {
							logger.Warn("Potential SQL injection detected in request body",
								logging.NewField("ip", c.ClientIP()),
							)
							c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid input detected"})
							return
						}
					}
				}
			}
		}

		c.Next()
	}
}

// checkJSONForSQLInjection recursively checks JSON data for SQL injection patterns.
func checkJSONForSQLInjection(data interface{}, pattern *regexp.Regexp) bool {
	switch v := data.(type) {
	case string:
		return pattern.MatchString(v)
	case map[string]interface{}:
		for _, value := range v {
			if checkJSONForSQLInjection(value, pattern) {
				return true
			}
		}
	case []interface{}:
		for _, item := range v {
			if checkJSONForSQLInjection(item, pattern) {
				return true
			}
		}
	}
	return false
}

// HTTPMethodWhitelistMiddleware restricts HTTP methods to an allowed list.
// This provides defense-in-depth by explicitly blocking unwanted HTTP methods.
func HTTPMethodWhitelistMiddleware(allowedMethods []string, logger logging.Logger) gin.HandlerFunc {
	// Convert to map for O(1) lookup
	allowed := make(map[string]bool)
	for _, method := range allowedMethods {
		allowed[method] = true
	}

	return func(c *gin.Context) {
		if !allowed[c.Request.Method] {
			logger.Warn("HTTP method not allowed",
				logging.NewField("method", c.Request.Method),
				logging.NewField("path", c.Request.URL.Path),
				logging.NewField("ip", c.ClientIP()),
			)
			c.AbortWithStatusJSON(http.StatusMethodNotAllowed, gin.H{
				"error": "Method not allowed",
			})
			return
		}
		c.Next()
	}
}
