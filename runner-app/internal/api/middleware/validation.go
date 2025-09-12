package middleware

import (
	"encoding/json"
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// ErrorResponse represents a structured API error response
type ErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// ValidateJSON middleware ensures request body is valid JSON for POST/PUT requests
func ValidateJSON() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "POST" || c.Request.Method == "PUT" {
			contentType := c.GetHeader("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				c.JSON(http.StatusBadRequest, ErrorResponse{
					Error:   "invalid_content_type",
					Message: "Content-Type must be application/json",
				})
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

// ValidateJobSpec middleware validates job specification structure
func ValidateJobSpec() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPost && c.Request.Method != http.MethodPut {
			c.Next()
			return
		}

		// Read body safely and restore it afterward
		var bodyBytes []byte
		if c.Request.Body != nil {
			var err error
			// Limit read to ~1MB to avoid abuse; handlers should enforce own limits as well
			bodyBytes, err = io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
			if err != nil {
				c.JSON(http.StatusBadRequest, ErrorResponse{
					Error:   "invalid_body",
					Message: "Failed to read request body",
				})
				c.Abort()
				return
			}
		}
		// Restore body for downstream handlers regardless of outcome
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Empty body is invalid for JobSpec creation
		if len(bodyBytes) == 0 {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_body",
				Message: "Request body cannot be empty",
			})
			c.Abort()
			return
		}

		// Parse JSON into JobSpec
		var spec models.JobSpec
		if err := json.Unmarshal(bodyBytes, &spec); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_json",
				Message: "Malformed JSON in body",
			})
			c.Abort()
			return
		}

		// Handle jobspec_id field mapping for portal compatibility
		var rawMap map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &rawMap); err == nil {
			if jobspecID, exists := rawMap["jobspec_id"]; exists && spec.ID == "" {
				if idStr, ok := jobspecID.(string); ok {
					spec.ID = idStr
				}
			}
		}

		// Structural validation
		if err := spec.Validate(); err != nil {
			// Provide a simple field-oriented message if recognizable
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "validation_error",
				Message: err.Error(),
			})
			c.Abort()
			return
		}

		// Store parsed spec for handlers (optional usage)
		c.Set("jobspec.parsed", &spec)

		// Restore body again so handlers can bind
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		c.Next()
	}
}

// validateJobSpecFields checks required job specification fields
func validateJobSpecFields(spec map[string]interface{}) error { return nil }

// ValidationError represents a field validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
