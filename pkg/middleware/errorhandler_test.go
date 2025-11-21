package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/yourorg/go-service-kit/pkg/errors"
	"github.com/yourorg/go-service-kit/pkg/logging"
)

func TestErrorHandlerMiddleware_HandlesAppError(t *testing.T) {
	logger, _ := logging.NewLogger("info", "json")
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(ErrorHandlerMiddleware(logger))
	router.GET("/test", func(c *gin.Context) {
		SetError(c, errors.NewNotFoundError("resource not found"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestErrorHandlerMiddleware_SetsServiceHandledHeader(t *testing.T) {
	logger, _ := logging.NewLogger("info", "json")
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(ErrorHandlerMiddleware(logger))
	router.GET("/test", func(c *gin.Context) {
		err := errors.NewInternalError("internal error").SetHandledByService(true)
		SetError(c, err)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "true", w.Header().Get("X-Service-Handled"))
}

