package middleware

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error ErrorDetails `json:"error"`
	Status string      `json:"status"`
}

// ErrorDetails contains detailed error information
type ErrorDetails struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	Details     string `json:"details,omitempty"`
	Timestamp   string `json:"timestamp"`
	RequestID   string `json:"request_id,omitempty"`
	RetryAfter  int    `json:"retry_after,omitempty"`
	UserMessage string `json:"user_message"`
}

// Error codes for different failure types
const (
	ErrorCodeDatabaseConnectionFailed = "DATABASE_CONNECTION_FAILED"
	ErrorCodeCrossRegionExecutionFailed = "CROSS_REGION_EXECUTION_FAILED"
	ErrorCodeCrossRegionSubmissionFailed = "CROSS_REGION_SUBMISSION_FAILED"
	ErrorCodeAPIInternalError = "API_INTERNAL_ERROR"
	ErrorCodeAPIEndpointNotFound = "API_ENDPOINT_NOT_FOUND"
	ErrorCodeProviderDiscoveryFailed = "PROVIDER_DISCOVERY_FAILED"
	ErrorCodeInfrastructureUnavailable = "INFRASTRUCTURE_UNAVAILABLE"
	ErrorCodeServiceDegraded = "SERVICE_DEGRADED"
	ErrorCodeJobTrackingFailed = "JOB_TRACKING_FAILED"
)

// ErrorHandler middleware for standardized error responses
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there are any errors
		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			
			// Log the error
			log.Printf("Error in request %s %s: %v", c.Request.Method, c.Request.URL.Path, err.Err)

			// Don't override if response already written
			if c.Writer.Written() {
				return
			}

			// Create standardized error response
			errorResp := createErrorResponse(err.Err, c.Request.URL.Path)
			
			// Determine HTTP status code
			statusCode := http.StatusInternalServerError
			if c.Writer.Status() != 200 {
				statusCode = c.Writer.Status()
			}

			c.JSON(statusCode, errorResp)
		}
	}
}

// createErrorResponse creates a standardized error response
func createErrorResponse(err error, path string) ErrorResponse {
	code := ErrorCodeAPIInternalError
	message := "An internal server error occurred"
	userMessage := "Something went wrong. Please try again later."
	retryAfter := 30

	// Categorize error based on error message
	errStr := err.Error()
	switch {
	case contains(errStr, "database", "connection", "postgres"):
		code = ErrorCodeDatabaseConnectionFailed
		message = "Database connection failed"
		userMessage = "The service is temporarily unavailable. Please try again in a few moments."
		retryAfter = 60
	case contains(errStr, "cross-region", "execution"):
		code = ErrorCodeCrossRegionExecutionFailed
		message = "Failed to retrieve cross-region execution results"
		userMessage = "Cross-region analysis is temporarily unavailable. Please try again later."
	case contains(errStr, "provider", "discovery"):
		code = ErrorCodeProviderDiscoveryFailed
		message = "Provider discovery failed"
		userMessage = "Unable to discover available providers. Please try again."
	}

	return ErrorResponse{
		Error: ErrorDetails{
			Code:        code,
			Message:     message,
			Details:     errStr,
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
			RetryAfter:  retryAfter,
			UserMessage: userMessage,
		},
		Status: "error",
	}
}

// contains checks if any of the keywords exist in the string (case-insensitive)
func contains(str string, keywords ...string) bool {
	strLower := strings.ToLower(str)
	for _, keyword := range keywords {
		if strings.Contains(strLower, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}
