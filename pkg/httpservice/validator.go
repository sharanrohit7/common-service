package httpservice

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/yourorg/go-service-kit/pkg/errors"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// ValidateRequest validates a request struct using go-playground/validator.
func ValidateRequest(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBind(req); err != nil {
		appErr := errors.NewValidationError("Invalid request: " + err.Error())
		c.JSON(appErr.HTTPStatus, gin.H{
			"error": appErr.Message,
			"code":  appErr.Code,
		})
		return false
	}
	
	if err := validate.Struct(req); err != nil {
		appErr := errors.NewValidationError("Validation failed: " + err.Error())
		c.JSON(appErr.HTTPStatus, gin.H{
			"error": appErr.Message,
			"code":  appErr.Code,
		})
		return false
	}
	
	return true
}

// ValidateJSON validates JSON request body.
func ValidateJSON(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		appErr := errors.NewValidationError("Invalid JSON: " + err.Error())
		c.JSON(appErr.HTTPStatus, gin.H{
			"error": appErr.Message,
			"code":  appErr.Code,
		})
		return false
	}
	
	if err := validate.Struct(req); err != nil {
		appErr := errors.NewValidationError("Validation failed: " + err.Error())
		c.JSON(appErr.HTTPStatus, gin.H{
			"error": appErr.Message,
			"code":  appErr.Code,
		})
		return false
	}
	
	return true
}

// ValidateQuery validates query parameters.
func ValidateQuery(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBindQuery(req); err != nil {
		appErr := errors.NewValidationError("Invalid query parameters: " + err.Error())
		c.JSON(appErr.HTTPStatus, gin.H{
			"error": appErr.Message,
			"code":  appErr.Code,
		})
		return false
	}
	
	if err := validate.Struct(req); err != nil {
		appErr := errors.NewValidationError("Validation failed: " + err.Error())
		c.JSON(appErr.HTTPStatus, gin.H{
			"error": appErr.Message,
			"code":  appErr.Code,
		})
		return false
	}
	
	return true
}

// ErrorHandler handles errors and returns appropriate HTTP responses.
func ErrorHandler(c *gin.Context) {
	c.Next()
	
	// Check if there are any errors
	if len(c.Errors) > 0 {
		err := c.Errors.Last()
		
		// Check if it's an AppError
		if appErr, ok := err.Err.(*errors.AppError); ok {
			c.JSON(appErr.HTTPStatus, gin.H{
				"error": appErr.Message,
				"code":  appErr.Code,
			})
			return
		}
		
		// Convert to AppError
		appErr := errors.FromError(err.Err)
		c.JSON(appErr.HTTPStatus, gin.H{
			"error": appErr.Message,
			"code":  appErr.Code,
		})
		return
	}
}

// HandleError is a helper to handle errors in handlers.
func HandleError(c *gin.Context, err error) {
	appErr := errors.FromError(err)
	c.JSON(appErr.HTTPStatus, gin.H{
		"error": appErr.Message,
		"code":  appErr.Code,
	})
	c.Abort()
}

// SuccessResponse sends a success response.
func SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"data": data,
	})
}

// CreatedResponse sends a created response.
func CreatedResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, gin.H{
		"data": data,
	})
}

