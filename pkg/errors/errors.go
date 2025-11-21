package errors

import (
	"fmt"
	"net/http"
)

// ErrorCode represents a typed error code.
type ErrorCode string

const (
	// ErrorCodeInternal represents an internal server error.
	ErrorCodeInternal ErrorCode = "INTERNAL_ERROR"
	// ErrorCodeNotFound represents a resource not found error.
	ErrorCodeNotFound ErrorCode = "NOT_FOUND"
	// ErrorCodeBadRequest represents a bad request error.
	ErrorCodeBadRequest ErrorCode = "BAD_REQUEST"
	// ErrorCodeUnauthorized represents an unauthorized error.
	ErrorCodeUnauthorized ErrorCode = "UNAUTHORIZED"
	// ErrorCodeForbidden represents a forbidden error.
	ErrorCodeForbidden ErrorCode = "FORBIDDEN"
	// ErrorCodeConflict represents a conflict error.
	ErrorCodeConflict ErrorCode = "CONFLICT"
	// ErrorCodeValidation represents a validation error.
	ErrorCodeValidation ErrorCode = "VALIDATION_ERROR"
	// ErrorCodeTimeout represents a timeout error.
	ErrorCodeTimeout ErrorCode = "TIMEOUT"
	// ErrorCodeServiceUnavailable represents a service unavailable error.
	ErrorCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
)

// AppError represents an application error with code, message, and HTTP status.
type AppError struct {
	Code             ErrorCode
	Message          string
	HTTPStatus       int
	Err              error
	Details          map[string]interface{}
	HandledByService bool // Indicates if the service has already handled/alerted on this error
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error.
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError creates a new application error.
func NewAppError(code ErrorCode, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

// NewAppErrorWithErr creates a new application error with an underlying error.
func NewAppErrorWithErr(code ErrorCode, message string, httpStatus int, err error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Err:        err,
	}
}

// WithDetails adds details to the error.
func (e *AppError) WithDetails(details map[string]interface{}) *AppError {
	e.Details = details
	return e
}

// SetHandledByService marks the error as handled by the service.
func (e *AppError) SetHandledByService(handled bool) *AppError {
	e.HandledByService = handled
	return e
}

// ErrorResponse represents the JSON error response format.
type ErrorResponse struct {
	Code             ErrorCode              `json:"code"`
	Message          string                  `json:"message"`
	Details          map[string]interface{}  `json:"details,omitempty"`
	HandledByService bool                    `json:"handled_by_service,omitempty"`
}

// ToErrorResponse converts an AppError to an ErrorResponse for JSON serialization.
func (e *AppError) ToErrorResponse() ErrorResponse {
	return ErrorResponse{
		Code:             e.Code,
		Message:          e.Message,
		Details:          e.Details,
		HandledByService: e.HandledByService,
	}
}

// ToHTTPStatus maps an error code to HTTP status code.
func ToHTTPStatus(code ErrorCode) int {
	switch code {
	case ErrorCodeBadRequest, ErrorCodeValidation:
		return http.StatusBadRequest
	case ErrorCodeUnauthorized:
		return http.StatusUnauthorized
	case ErrorCodeForbidden:
		return http.StatusForbidden
	case ErrorCodeNotFound:
		return http.StatusNotFound
	case ErrorCodeConflict:
		return http.StatusConflict
	case ErrorCodeTimeout:
		return http.StatusRequestTimeout
	case ErrorCodeServiceUnavailable:
		return http.StatusServiceUnavailable
	case ErrorCodeInternal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// FromError converts a standard error to an AppError.
// If the error is already an AppError, it returns it as-is.
// Otherwise, it wraps it as an internal error.
func FromError(err error) *AppError {
	if err == nil {
		return nil
	}
	
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	
	return NewAppErrorWithErr(
		ErrorCodeInternal,
		"An internal error occurred",
		http.StatusInternalServerError,
		err,
	)
}

// Common error constructors

// NewBadRequestError creates a bad request error.
func NewBadRequestError(message string) *AppError {
	return NewAppError(ErrorCodeBadRequest, message, http.StatusBadRequest)
}

// NewNotFoundError creates a not found error.
func NewNotFoundError(message string) *AppError {
	return NewAppError(ErrorCodeNotFound, message, http.StatusNotFound)
}

// NewUnauthorizedError creates an unauthorized error.
func NewUnauthorizedError(message string) *AppError {
	return NewAppError(ErrorCodeUnauthorized, message, http.StatusUnauthorized)
}

// NewForbiddenError creates a forbidden error.
func NewForbiddenError(message string) *AppError {
	return NewAppError(ErrorCodeForbidden, message, http.StatusForbidden)
}

// NewInternalError creates an internal error.
func NewInternalError(message string) *AppError {
	return NewAppError(ErrorCodeInternal, message, http.StatusInternalServerError)
}

// NewValidationError creates a validation error.
func NewValidationError(message string) *AppError {
	return NewAppError(ErrorCodeValidation, message, http.StatusBadRequest)
}

// NewConflictError creates a conflict error.
func NewConflictError(message string) *AppError {
	return NewAppError(ErrorCodeConflict, message, http.StatusConflict)
}

// NewTimeoutError creates a timeout error.
func NewTimeoutError(message string) *AppError {
	return NewAppError(ErrorCodeTimeout, message, http.StatusRequestTimeout)
}

// NewServiceUnavailableError creates a service unavailable error.
func NewServiceUnavailableError(message string) *AppError {
	return NewAppError(ErrorCodeServiceUnavailable, message, http.StatusServiceUnavailable)
}

