package httpservice

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/yourorg/go-service-kit/pkg/logging"
)

// MockLogger implements logging.Logger for testing
type MockLogger struct{}

func (m *MockLogger) Info(msg string, fields ...logging.Field)    {}
func (m *MockLogger) Error(msg string, fields ...logging.Field)   {}
func (m *MockLogger) Debug(msg string, fields ...logging.Field)   {}
func (m *MockLogger) Warn(msg string, fields ...logging.Field)    {}
func (m *MockLogger) Fatal(msg string, fields ...logging.Field)   {}
func (m *MockLogger) With(fields ...logging.Field) logging.Logger { return m }
func (m *MockLogger) WithError(err error) logging.Logger          { return m }

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Allow 2 requests per second with burst of 2
	cfg := RateLimitConfig{RPS: 2, Burst: 2}
	router := gin.New()
	router.Use(RateLimitMiddleware(cfg))
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// 3rd request should fail (too fast)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// Wait for token refill
	time.Sleep(600 * time.Millisecond)

	// Should succeed again
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SecurityHeadersMiddleware())
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "default-src 'self'", w.Header().Get("Content-Security-Policy"))
}

func TestSQLInjectionCheckMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Mock logger
	logger := &MockLogger{}
	router := gin.New()
	router.Use(SQLInjectionCheckMiddleware(logger))
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	tests := []struct {
		name   string
		query  string
		status int
	}{
		{"Valid query", "search", http.StatusOK},
		{"SQL Injection OR", "admin' OR 1=1", http.StatusBadRequest},
		{"SQL Injection UNION", "UNION SELECT *", http.StatusBadRequest},
		{"SQL Injection Comment", "admin --", http.StatusBadRequest},
		{"SQL Injection Drop", "; DROP TABLE users", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			// Properly encode query parameters
			req, _ := http.NewRequest("GET", "/?q="+tt.query, nil)
			q := req.URL.Query()
			q.Set("q", tt.query)
			req.URL.RawQuery = q.Encode()

			router.ServeHTTP(w, req)
			assert.Equal(t, tt.status, w.Code)
		})
	}
}

func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		cfg            CORSConfig
		origin         string
		expectedOrigin string
	}{
		{
			name:           "Allow All",
			cfg:            CORSConfig{AllowedOrigins: []string{"*"}},
			origin:         "http://example.com",
			expectedOrigin: "*",
		},
		{
			name:           "Allow Specific",
			cfg:            CORSConfig{AllowedOrigins: []string{"http://example.com"}},
			origin:         "http://example.com",
			expectedOrigin: "http://example.com",
		},
		{
			name:           "Disallow Specific",
			cfg:            CORSConfig{AllowedOrigins: []string{"http://example.com"}},
			origin:         "http://evil.com",
			expectedOrigin: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(CORSMiddleware(tt.cfg))
			router.GET("/", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/", nil)
			req.Header.Set("Origin", tt.origin)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedOrigin, w.Header().Get("Access-Control-Allow-Origin"))
		})
	}
}
