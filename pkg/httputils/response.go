package httputils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// StandardResponse represents a standard API response
type StandardResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo represents error details
type ErrorInfo struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// RespondSuccess sends a successful response
func RespondSuccess(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, StandardResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// RespondError sends an error response
func RespondError(c *gin.Context, statusCode int, message string, err error) {
	errorInfo := &ErrorInfo{
		Message: message,
	}

	if err != nil {
		errorInfo.Details = err.Error()
	}

	c.JSON(statusCode, StandardResponse{
		Success: false,
		Error:   errorInfo,
	})
}

// RespondErrorWithCode sends an error response with error code
func RespondErrorWithCode(c *gin.Context, statusCode int, errorCode, message string, err error) {
	errorInfo := &ErrorInfo{
		Code:    errorCode,
		Message: message,
	}

	if err != nil {
		errorInfo.Details = err.Error()
	}

	c.JSON(statusCode, StandardResponse{
		Success: false,
		Error:   errorInfo,
	})
}

// Common response helpers

// OK sends 200 OK response
func OK(c *gin.Context, data interface{}) {
	RespondSuccess(c, http.StatusOK, "Success", data)
}

// Created sends 201 Created response
func Created(c *gin.Context, message string, data interface{}) {
	RespondSuccess(c, http.StatusCreated, message, data)
}

// BadRequest sends 400 Bad Request response
func BadRequest(c *gin.Context, message string, err error) {
	RespondError(c, http.StatusBadRequest, message, err)
}

// Unauthorized sends 401 Unauthorized response
func Unauthorized(c *gin.Context, message string, err error) {
	RespondError(c, http.StatusUnauthorized, message, err)
}

// Forbidden sends 403 Forbidden response
func Forbidden(c *gin.Context, message string, err error) {
	RespondError(c, http.StatusForbidden, message, err)
}

// NotFound sends 404 Not Found response
func NotFound(c *gin.Context, message string, err error) {
	RespondError(c, http.StatusNotFound, message, err)
}

// Conflict sends 409 Conflict response
func Conflict(c *gin.Context, message string, err error) {
	RespondError(c, http.StatusConflict, message, err)
}

// InternalServerError sends 500 Internal Server Error response
func InternalServerError(c *gin.Context, message string, err error) {
	RespondError(c, http.StatusInternalServerError, message, err)
}

// ValidationError sends 422 Unprocessable Entity response
func ValidationError(c *gin.Context, message string, err error) {
	RespondError(c, http.StatusUnprocessableEntity, message, err)
}
