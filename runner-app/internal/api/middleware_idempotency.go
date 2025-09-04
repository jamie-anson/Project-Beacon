package api

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const IdempotencyKeyContextKey = "idempotency_key"

// IdempotencyKeyMiddleware extracts the Idempotency-Key header and attaches it to the request context.
// This provides cleaner separation of concerns by centralizing header extraction.
func IdempotencyKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		idemKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
		if idemKey != "" {
			c.Set(IdempotencyKeyContextKey, idemKey)
		}
		c.Next()
	}
}

// GetIdempotencyKey retrieves the idempotency key from the Gin context.
// Returns the key and true if present, empty string and false otherwise.
func GetIdempotencyKey(c *gin.Context) (string, bool) {
	if key, exists := c.Get(IdempotencyKeyContextKey); exists {
		if keyStr, ok := key.(string); ok {
			return keyStr, true
		}
	}
	return "", false
}
