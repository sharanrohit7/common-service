package httpservice

import (
	"github.com/gin-gonic/gin"
	"github.com/yourorg/go-service-kit/pkg/errors"
	"github.com/yourorg/go-service-kit/pkg/logging"
)

// GetLogger retrieves the contextual logger from the request.
func GetLogger(c *gin.Context) logging.Logger {
	return logging.FromContext(c.Request.Context())
}

// LogInfo logs an info message using the contextual logger.
func LogInfo(c *gin.Context, msg string, fields ...logging.Field) {
	GetLogger(c).Info(msg, fields...)
}

// LogWarn logs a warning message using the contextual logger.
func LogWarn(c *gin.Context, msg string, fields ...logging.Field) {
	GetLogger(c).Warn(msg, fields...)
}

// LogError logs an error message using the contextual logger.
func LogError(c *gin.Context, msg string, err error, fields ...logging.Field) {
	if err != nil {
		fields = append(fields, logging.NewField("error", err))
	}
	GetLogger(c).Error(msg, fields...)
}

// LogDebug logs a debug message using the contextual logger.
func LogDebug(c *gin.Context, msg string, fields ...logging.Field) {
	GetLogger(c).Debug(msg, fields...)
}

// RespondSuccess sends a standard success response.
func RespondSuccess(c *gin.Context, data interface{}) {
	SuccessResponse(c, data)
}

// RespondCreated sends a standard created response.
func RespondCreated(c *gin.Context, data interface{}) {
	CreatedResponse(c, data)
}

// RespondError handles an error and sends a standard error response.
func RespondError(c *gin.Context, err error) {
	HandleError(c, err)
}

// RespondErrorWithLog logs the error and then sends a standard error response.
func RespondErrorWithLog(c *gin.Context, msg string, err error, fields ...logging.Field) {
	LogError(c, msg, err, fields...)
	HandleError(c, err)
}

// RespondValidationError sends a validation error response.
func RespondValidationError(c *gin.Context, msg string) {
	appErr := errors.NewValidationError(msg)
	c.JSON(appErr.HTTPStatus, gin.H{
		"error": appErr.Message,
		"code":  appErr.Code,
	})
}
