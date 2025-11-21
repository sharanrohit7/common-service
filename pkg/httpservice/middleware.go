package httpservice

import (
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/go-service-kit/pkg/logging"
	"github.com/yourorg/go-service-kit/pkg/utils"
	"golang.org/x/time/rate"
)

// RateLimitConfig holds configuration for rate limiting.
type RateLimitConfig struct {
	RPS   float64 // Requests per second
	Burst int     // Maximum burst size
}

// RateLimitMiddleware limits the number of requests per second per IP.
func RateLimitMiddleware(cfg RateLimitConfig) gin.HandlerFunc {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}
	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	// Background goroutine to clean up old clients
	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			for ip, c := range clients {
				if time.Since(c.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		mu.Lock()
		if _, found := clients[ip]; !found {
			clients[ip] = &client{
				limiter: rate.NewLimiter(rate.Limit(cfg.RPS), cfg.Burst),
			}
		}
		clients[ip].lastSeen = time.Now()
		if !clients[ip].limiter.Allow() {
			mu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests",
			})
			return
		}
		mu.Unlock()
		c.Next()
	}
}

// SecurityHeadersMiddleware adds security-related headers to responses.
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Next()
	}
}

// SQLInjectionCheckMiddleware checks for common SQL injection patterns in query params and JSON body.
// NOTE: This is a secondary defense. Parameterized queries are the primary defense.
func SQLInjectionCheckMiddleware(logger logging.Logger) gin.HandlerFunc {
	// Regex for common SQL injection patterns (simplified)
	// Matches: UNION SELECT, OR 1=1, --, ; DROP, etc.
	sqlInjectionPattern := regexp.MustCompile(`(?i)(union\s+select|or\s+1=1|--|;\s*drop|;\s*delete|;\s*update|;\s*insert|exec\s*\()`)

	return func(c *gin.Context) {
		// Check query parameters
		for _, values := range c.Request.URL.Query() {
			for _, value := range values {
				if sqlInjectionPattern.MatchString(value) {
					logger.Warn("Potential SQL injection detected in query", logging.NewField("value", value), logging.NewField("ip", c.ClientIP()))
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid input detected"})
					return
				}
			}
		}

		// Check JSON body (if applicable)
		// Note: This requires reading the body, which might be expensive or interfere with binding.
		// For now, we'll skip body check to avoid reading the stream twice without a proper buffer wrapper.
		// A more robust implementation would use a body dumper or similar approach.

		c.Next()
	}
}

// LoggingMiddleware logs HTTP requests with structured logging.
func LoggingMiddleware(logger logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Build log fields
		fields := []logging.Field{
			logging.NewField("method", c.Request.Method),
			logging.NewField("path", path),
			logging.NewField("status", c.Writer.Status()),
			logging.NewField("latency_ms", latency.Milliseconds()),
			logging.NewField("ip", c.ClientIP()),
			logging.NewField("user_agent", c.Request.UserAgent()),
		}

		if raw != "" {
			fields = append(fields, logging.NewField("query", raw))
		}

		// Get request ID from context
		if requestID, exists := c.Get("request_id"); exists {
			fields = append(fields, logging.NewField("request_id", requestID))
		}

		// Log based on status code
		if c.Writer.Status() >= 500 {
			logger.Error("HTTP request", fields...)
		} else if c.Writer.Status() >= 400 {
			logger.Warn("HTTP request", fields...)
		} else {
			logger.Info("HTTP request", fields...)
		}
	}
}

// RequestIDMiddleware adds a request ID to each request.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = utils.GenerateRequestID()
		}

		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// RecoveryMiddleware recovers from panics and logs the error.
func RecoveryMiddleware(logger logging.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logger.Error("Panic recovered",
			logging.NewField("error", recovered),
			logging.NewField("path", c.Request.URL.Path),
			logging.NewField("method", c.Request.Method),
		)

		c.JSON(500, gin.H{
			"error": "Internal server error",
		})
		c.Abort()
	})
}

// MetricsMiddleware is a placeholder for metrics collection.
// TODO: Implement with your metrics backend (Prometheus, DataDog, etc.)
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Example metrics collection:
		// metrics.IncrementCounter("http_requests_total", "method", c.Request.Method, "path", c.Request.URL.Path)
		// metrics.RecordHistogram("http_request_duration", latency, "method", c.Request.Method)

		c.Next()

		// Record status code
		// metrics.IncrementCounter("http_responses_total", "status", strconv.Itoa(c.Writer.Status()))
	}
}

// AuthMiddleware is a placeholder for authentication.
// TODO: Implement with your auth provider (JWT, OAuth, etc.)
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Example JWT validation:
		// token := c.GetHeader("Authorization")
		// if token == "" {
		//   c.JSON(401, gin.H{"error": "Unauthorized"})
		//   c.Abort()
		//   return
		// }
		// ... validate token ...
		c.Next()
	}
}

// CORSConfig holds configuration for CORS.
type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

// CORSMiddleware adds CORS headers with configuration.
func CORSMiddleware(cfg CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allowed := false

		// Check if origin is allowed
		if len(cfg.AllowedOrigins) == 0 || (len(cfg.AllowedOrigins) == 1 && cfg.AllowedOrigins[0] == "*") {
			allowed = true
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		} else {
			for _, o := range cfg.AllowedOrigins {
				if o == origin {
					allowed = true
					c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}
		}

		if allowed {
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

			headers := "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With"
			if len(cfg.AllowedHeaders) > 0 {
				headers = strings.Join(cfg.AllowedHeaders, ", ")
			}
			c.Writer.Header().Set("Access-Control-Allow-Headers", headers)

			methods := "POST, OPTIONS, GET, PUT, DELETE"
			if len(cfg.AllowedMethods) > 0 {
				methods = strings.Join(cfg.AllowedMethods, ", ")
			}
			c.Writer.Header().Set("Access-Control-Allow-Methods", methods)
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
