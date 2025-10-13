package middleware

import (
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

  // CORS adds Cross-Origin Resource Sharing headers
  func CORS() gin.HandlerFunc {
      return func(c *gin.Context) {
          origin := c.Request.Header.Get("Origin")
          
          // Allow specific origins or localhost for development
          allowedOrigins := []string{
              "http://localhost:3000",
              "http://localhost:8080",
              "http://127.0.0.1:3000",
              "http://127.0.0.1:8080",
              "https://projectbeacon.netlify.app",
              "https://preview--projectbeacon.netlify.app",
          }
          // Also allow any Netlify subdomain (e.g., preview deploys)
          allowedSuffixes := []string{
              ".netlify.app",
          }
          
          // Check if origin is allowed
          allowed := false
          for _, allowedOrigin := range allowedOrigins {
              if origin == allowedOrigin {
                  allowed = true
                  break
              }
          }
          if !allowed && origin != "" {
              for _, sfx := range allowedSuffixes {
                  if strings.HasSuffix(origin, sfx) {
                      allowed = true
                      break
                  }
              }
          }
          
          if allowed {
              c.Header("Access-Control-Allow-Origin", origin)
          }
          
          c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
          c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Request-ID, Idempotency-Key")
          c.Header("Access-Control-Expose-Headers", "X-Request-ID, X-Response-Time")
          c.Header("Access-Control-Allow-Credentials", "true")
          c.Header("Access-Control-Max-Age", "86400") // 24 hours
		
		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		c.Next()
	}
}

// SecurityHeaders adds security-related HTTP headers
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")
		
		// Enable XSS protection
		c.Header("X-XSS-Protection", "1; mode=block")
		
		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")
		
		// Strict Transport Security (HTTPS only)
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		
		// Content Security Policy
		csp := strings.Join([]string{
			"default-src 'self'",
			"script-src 'self' 'unsafe-inline'",
			"style-src 'self' 'unsafe-inline'",
			"img-src 'self' data: https:",
			"connect-src 'self' ws: wss:",
			"font-src 'self'",
			"object-src 'none'",
			"base-uri 'self'",
			"form-action 'self'",
		}, "; ")
		c.Header("Content-Security-Policy", csp)
		
		// Referrer Policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// Permissions Policy (formerly Feature Policy)
		permissions := strings.Join([]string{
			"camera=()",
			"microphone=()",
			"geolocation=()",
			"payment=()",
		}, ", ")
		c.Header("Permissions-Policy", permissions)
		
		c.Next()
	}
}

// RateLimiting adds basic rate limiting headers and logic
func RateLimiting() gin.HandlerFunc {
    // Simple in-memory token bucket per client IP
    // Defaults: 60 req/minute per IP
    const maxTokens = 60.0
    const refillPerSecond = maxTokens / 60.0

    type bucket struct {
        tokens    float64
        lastRefill time.Time
    }

    var (
        mu     sync.Mutex
        buckets = map[string]*bucket{}
    )

    return func(c *gin.Context) {
        ip := c.ClientIP()
        now := time.Now()

        mu.Lock()
        b, ok := buckets[ip]
        if !ok {
            b = &bucket{tokens: maxTokens, lastRefill: now}
            buckets[ip] = b
        }
        // Refill based on elapsed time
        elapsed := now.Sub(b.lastRefill).Seconds()
        if elapsed > 0 {
            b.tokens = math.Min(maxTokens, b.tokens+elapsed*refillPerSecond)
            b.lastRefill = now
        }

        // Consume one token
        if b.tokens >= 1.0 {
            b.tokens -= 1.0
        } else {
            // Calculate reset (seconds until next full token)
            var resetSec int64 = 1
            if refillPerSecond > 0 {
                resetSec = int64(math.Ceil((1.0 - b.tokens) / refillPerSecond))
            }
            mu.Unlock()

            // Headers
            c.Header("X-RateLimit-Limit", "60")
            c.Header("X-RateLimit-Remaining", "0")
            c.Header("X-RateLimit-Reset", time.Now().Add(time.Duration(resetSec)*time.Second).Format(time.RFC3339))

            c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
                "error": gin.H{
                    "type":    "rate_limited",
                    "message": "Too many requests. Please slow down.",
                    "code":    "RATE_LIMIT_EXCEEDED",
                },
                "request_id": c.GetString("request_id"),
            })
            return
        }
        // Compute remaining tokens after consumption
        remaining := int64(math.Floor(b.tokens))
        mu.Unlock()

        c.Header("X-RateLimit-Limit", "60")
        c.Header("X-RateLimit-Remaining", itoa(remaining))
        // reset to when bucket would be full (approximate)
        // time to full = (maxTokens - tokens) / refillPerSecond
        secsToFull := 0.0
        if refillPerSecond > 0 {
            mu.Lock()
            // briefly lock to read bucket state safely
            btokens := b.tokens
            mu.Unlock()
            secsToFull = (maxTokens - btokens) / refillPerSecond
        }
        c.Header("X-RateLimit-Reset", time.Now().Add(time.Duration(secsToFull)*time.Second).Format(time.RFC3339))

        c.Next()
    }
}

// small int64 -> string helper without fmt to avoid import
func itoa(n int64) string {
    if n == 0 { return "0" }
    sign := ""
    if n < 0 { sign = "-"; n = -n }
    var buf [20]byte
    i := len(buf)
    for n > 0 {
        i--
        buf[i] = byte('0' + n%10)
        n /= 10
    }
    return sign + string(buf[i:])
}

// AuditLogging logs security-relevant events
func AuditLogging() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Log security-relevant information
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()
		path := c.Request.URL.Path
		
		// TODO: Implement structured audit logging
		// For now, add headers for traceability
		c.Header("X-Client-IP", clientIP)
		
		// Log suspicious patterns
		if strings.Contains(userAgent, "bot") || 
		   strings.Contains(userAgent, "crawler") ||
		   strings.Contains(path, "..") ||
		   strings.Contains(path, "<script>") {
			// TODO: Log to security audit system
		}
		
		c.Next()
	}
}
