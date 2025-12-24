package httputils

import (
	"errors"

	"github.com/gin-gonic/gin"
)

// Common error types
var (
	ErrInvalidRequest     = errors.New("invalid request")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrNotFound           = errors.New("resource not found")
	ErrConflict           = errors.New("resource conflict")
	ErrInternalServer     = errors.New("internal server error")
	ErrValidation         = errors.New("validation error")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// HandleError handles errors and sends appropriate response
func HandleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidRequest):
		BadRequest(c, "Invalid request", err)
	case errors.Is(err, ErrUnauthorized), errors.Is(err, ErrInvalidCredentials):
		Unauthorized(c, "Unauthorized", err)
	case errors.Is(err, ErrForbidden):
		Forbidden(c, "Forbidden", err)
	case errors.Is(err, ErrNotFound):
		NotFound(c, "Resource not found", err)
	case errors.Is(err, ErrConflict):
		Conflict(c, "Resource conflict", err)
	case errors.Is(err, ErrValidation):
		ValidationError(c, "Validation error", err)
	default:
		InternalServerError(c, "Internal server error", err)
	}
}

// BindJSON binds JSON request and handles errors
func BindJSON(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		BadRequest(c, "Invalid JSON request", err)
		return false
	}
	return true
}

// BindQuery binds query parameters and handles errors
func BindQuery(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindQuery(obj); err != nil {
		BadRequest(c, "Invalid query parameters", err)
		return false
	}
	return true
}

// BindURI binds URI parameters and handles errors
func BindURI(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindUri(obj); err != nil {
		BadRequest(c, "Invalid URI parameters", err)
		return false
	}
	return true
}
