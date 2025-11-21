package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/yourorg/go-service-kit/pkg/logging"
)

func TestTracingMiddleware_GeneratesTraceID(t *testing.T) {
	logger, _ := logging.NewLogger("info", "json")
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(TracingMiddleware(logger, "test-service"))
	router.GET("/test", func(c *gin.Context) {
		traceID := GetTraceIDFromGin(c)
		c.JSON(http.StatusOK, gin.H{"trace_id": traceID})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Trace-ID"))
}

func TestTracingMiddleware_UsesExistingTraceID(t *testing.T) {
	logger, _ := logging.NewLogger("info", "json")
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(TracingMiddleware(logger, "test-service"))
	router.GET("/test", func(c *gin.Context) {
		traceID := GetTraceIDFromGin(c)
		c.JSON(http.StatusOK, gin.H{"trace_id": traceID})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Trace-ID", "existing-trace-id")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "existing-trace-id", w.Header().Get("X-Trace-ID"))
}

