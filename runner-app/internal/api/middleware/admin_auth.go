package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/config"
	"github.com/jamie-anson/project-beacon-runner/internal/logging"
)

// AdminAuthMiddleware provides authentication for admin endpoints
func AdminAuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		l := logging.FromContext(c.Request.Context())
		
		// Check if admin authentication is enabled
		adminToken := cfg.AdminToken
		if adminToken == "" {
			l.Warn().Msg("admin endpoints accessed without authentication configured")
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "admin_auth_not_configured",
				"message": "Admin authentication not configured",
			})
			c.Abort()
			return
		}

		// Extract Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			l.Warn().
				Str("client_ip", c.ClientIP()).
				Str("user_agent", c.Request.UserAgent()).
				Msg("admin endpoint accessed without authorization header")
			
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
				"message": "Authorization header required for admin endpoints",
			})
			c.Abort()
			return
		}

		// Check Bearer token format
		if !strings.HasPrefix(authHeader, "Bearer ") {
			l.Warn().
				Str("client_ip", c.ClientIP()).
				Msg("admin endpoint accessed with invalid authorization format")
			
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid_auth_format",
				"message": "Authorization must use Bearer token format",
			})
			c.Abort()
			return
		}

		// Extract and validate token
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token != adminToken {
			l.Warn().
				Str("client_ip", c.ClientIP()).
				Str("user_agent", c.Request.UserAgent()).
				Str("endpoint", c.Request.URL.Path).
				Msg("admin endpoint accessed with invalid token")
			
			c.JSON(http.StatusForbidden, gin.H{
				"error": "invalid_token",
				"message": "Invalid admin token",
			})
			c.Abort()
			return
		}

		// Log successful admin access for audit trail
		l.Info().
			Str("client_ip", c.ClientIP()).
			Str("user_agent", c.Request.UserAgent()).
			Str("endpoint", c.Request.URL.Path).
			Str("method", c.Request.Method).
			Time("timestamp", time.Now().UTC()).
			Msg("admin endpoint accessed successfully")

		// Add admin context for downstream handlers
		c.Set("admin_authenticated", true)
		c.Set("admin_access_time", time.Now().UTC())
		
		c.Next()
	}
}

// AdminRateLimitMiddleware provides rate limiting for admin operations
func AdminRateLimitMiddleware() gin.HandlerFunc {
	// Simple in-memory rate limiter for admin operations
	// In production, consider using Redis-based rate limiting
	lastAccess := make(map[string]time.Time)
	const adminRateLimit = 10 * time.Second // Max 1 admin operation per 10 seconds per IP
	
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()
		
		if lastTime, exists := lastAccess[clientIP]; exists {
			if now.Sub(lastTime) < adminRateLimit {
				l := logging.FromContext(c.Request.Context())
				l.Warn().
					Str("client_ip", clientIP).
					Str("endpoint", c.Request.URL.Path).
					Msg("admin endpoint rate limit exceeded")
				
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error": "rate_limit_exceeded",
					"message": "Admin operations are rate limited",
					"retry_after": int(adminRateLimit.Seconds()),
				})
				c.Abort()
				return
			}
		}
		
		lastAccess[clientIP] = now
		c.Next()
	}
}
