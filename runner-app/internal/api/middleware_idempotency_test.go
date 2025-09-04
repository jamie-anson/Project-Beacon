package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestIdempotencyKeyMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		headerValue    string
		expectedKey    string
		expectedExists bool
	}{
		{
			name:           "valid idempotency key",
			headerValue:    "test-key-123",
			expectedKey:    "test-key-123",
			expectedExists: true,
		},
		{
			name:           "idempotency key with whitespace",
			headerValue:    "  test-key-456  ",
			expectedKey:    "test-key-456",
			expectedExists: true,
		},
		{
			name:           "empty idempotency key",
			headerValue:    "",
			expectedKey:    "",
			expectedExists: false,
		},
		{
			name:           "whitespace only idempotency key",
			headerValue:    "   ",
			expectedKey:    "",
			expectedExists: false,
		},
		{
			name:           "no header",
			headerValue:    "",
			expectedKey:    "",
			expectedExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup router with middleware
			r := gin.New()
			r.Use(IdempotencyKeyMiddleware())
			
			var capturedKey string
			var capturedExists bool
			
			r.POST("/test", func(c *gin.Context) {
				capturedKey, capturedExists = GetIdempotencyKey(c)
				c.JSON(http.StatusOK, gin.H{"success": true})
			})

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/test", nil)
			if tt.headerValue != "" || tt.name == "no header" {
				if tt.name != "no header" {
					req.Header.Set("Idempotency-Key", tt.headerValue)
				}
			}

			// Execute request
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			// Assertions
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, tt.expectedKey, capturedKey)
			assert.Equal(t, tt.expectedExists, capturedExists)
		})
	}
}

func TestGetIdempotencyKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("key exists in context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set(IdempotencyKeyContextKey, "test-key")

		key, exists := GetIdempotencyKey(c)
		assert.True(t, exists)
		assert.Equal(t, "test-key", key)
	})

	t.Run("key does not exist in context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())

		key, exists := GetIdempotencyKey(c)
		assert.False(t, exists)
		assert.Equal(t, "", key)
	})

	t.Run("key exists but wrong type", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set(IdempotencyKeyContextKey, 123) // Wrong type

		key, exists := GetIdempotencyKey(c)
		assert.False(t, exists)
		assert.Equal(t, "", key)
	})
}
