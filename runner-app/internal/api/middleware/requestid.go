package middleware

import (
    "crypto/rand"
    "encoding/hex"

    "github.com/gin-gonic/gin"
)

// RequestID ensures every request has a correlation ID.
// If the client supplies X-Request-ID, it is trusted (validated for length) and echoed.
// Otherwise a random 16-byte ID is generated.
func RequestID() gin.HandlerFunc {
    return func(c *gin.Context) {
        rid := c.GetHeader("X-Request-ID")
        if len(rid) == 0 || len(rid) > 128 {
            // generate 16 random bytes as hex (32 chars)
            var b [16]byte
            _, _ = rand.Read(b[:])
            rid = hex.EncodeToString(b[:])
        }
        // store in context and header for downstream handlers and clients
        c.Set("request_id", rid)
        c.Writer.Header().Set("X-Request-ID", rid)

        // continue chain
        c.Next()

        // ensure header still present on error paths
        if c.Writer.Header().Get("X-Request-ID") == "" {
            c.Writer.Header().Set("X-Request-ID", rid)
        }
    }
}
