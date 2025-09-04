package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	apperrors "github.com/jamie-anson/project-beacon-runner/internal/errors"
)

// ErrorHandler converts AppErrors to appropriate HTTP responses
func ErrorHandler() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(error); ok {
			handleError(c, err)
		} else {
			// Handle non-error panics
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"type":    "internal",
					"message": "Internal server error",
					"code":    "INTERNAL_ERROR",
				},
				"request_id": c.GetString("request_id"),
			})
		}
		c.Abort()
	})
}

// HandleError processes errors and returns appropriate HTTP responses
func HandleError(c *gin.Context, err error) {
	handleError(c, err)
}

func handleError(c *gin.Context, err error) {
	var appErr *apperrors.AppError
	
	// Convert to AppError if not already
	if !errors.As(err, &appErr) {
		appErr = apperrors.NewInternalError(err.Error())
	}
	
	// Map error types to HTTP status codes
	statusCode := getHTTPStatusCode(appErr.Type)
	
	// Create error response
	errorResponse := gin.H{
		"error": gin.H{
			"type":    string(appErr.Type),
			"message": appErr.Message,
		},
		"request_id": c.GetString("request_id"),
	}
	
	// Add optional fields
	if appErr.Code != "" {
		errorResponse["error"].(gin.H)["code"] = appErr.Code
	}
	
	if appErr.Details != "" {
		errorResponse["error"].(gin.H)["details"] = appErr.Details
	}
	
	// Log error for internal tracking
	if appErr.Type == apperrors.InternalError || appErr.Type == apperrors.DatabaseError {
		// TODO: Add structured logging
		c.Header("X-Error-Logged", "true")
	}
	
	c.JSON(statusCode, errorResponse)
}

func getHTTPStatusCode(errorType apperrors.ErrorType) int {
	switch errorType {
	case apperrors.ValidationError:
		return http.StatusBadRequest
	case apperrors.NotFoundError:
		return http.StatusNotFound
	case apperrors.ConflictError:
		return http.StatusConflict
	case apperrors.AuthenticationError:
		return http.StatusUnauthorized
	case apperrors.AuthorizationError:
		return http.StatusForbidden
	case apperrors.ExternalServiceError:
		return http.StatusBadGateway
	case apperrors.CircuitBreakerError:
		return http.StatusServiceUnavailable
	case apperrors.TimeoutError:
		return http.StatusRequestTimeout
	case apperrors.DatabaseError:
		return http.StatusInternalServerError
	case apperrors.InternalError:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
