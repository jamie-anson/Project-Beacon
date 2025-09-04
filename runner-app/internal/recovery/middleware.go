package recovery

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	apperrors "github.com/jamie-anson/project-beacon-runner/internal/errors"
)

// PanicRecoveryMiddleware recovers from panics and converts them to structured errors
func PanicRecoveryMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic with stack trace
				stack := debug.Stack()
				logger.Error("panic recovered",
					"error", err,
					"path", c.Request.URL.Path,
					"method", c.Request.Method,
					"stack", string(stack))

				// Convert panic to structured error
				appErr := apperrors.NewInternalError("internal server error").
					WithCode("panic_recovered").
					WithDetails("An unexpected error occurred")

				c.JSON(http.StatusInternalServerError, gin.H{
					"error":      appErr.Message,
					"error_code": appErr.Code,
					"type":       appErr.Type,
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}

// TimeoutMiddleware adds request timeout handling
func TimeoutMiddleware(timeout time.Duration, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		done := make(chan struct{})
		go func() {
			defer close(done)
			c.Next()
		}()

		select {
		case <-done:
			// Request completed normally
		case <-ctx.Done():
			// Request timed out
			logger.Warn("request timeout",
				"path", c.Request.URL.Path,
				"method", c.Request.Method,
				"timeout", timeout)

			if !c.Writer.Written() {
				appErr := apperrors.NewTimeoutError("request").
					WithCode("request_timeout")

				c.JSON(http.StatusRequestTimeout, gin.H{
					"error":      appErr.Message,
					"error_code": appErr.Code,
					"type":       appErr.Type,
				})
			}
			c.Abort()
		}
	}
}

// ErrorHandlingMiddleware converts structured errors to appropriate HTTP responses
func ErrorHandlingMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there are any errors to handle
		if len(c.Errors) == 0 {
			return
		}

		// Get the last error (most recent)
		err := c.Errors.Last().Err

		// Convert to AppError if possible
		var appErr *apperrors.AppError
		if !errors.As(err, &appErr) {
			// Create a generic internal error
			appErr = apperrors.NewInternalError("an unexpected error occurred")
		}

		// Log the error
		logger.Error("request error",
			"error", err,
			"type", appErr.Type,
			"code", appErr.Code,
			"path", c.Request.URL.Path,
			"method", c.Request.Method)

		// Map error type to HTTP status code
		statusCode := mapErrorTypeToHTTPStatus(appErr.Type)

		// Return structured error response
		c.JSON(statusCode, gin.H{
			"error":      appErr.Message,
			"error_code": appErr.Code,
			"type":       appErr.Type,
			"details":    appErr.Details,
		})
	}
}

// mapErrorTypeToHTTPStatus maps error types to HTTP status codes
func mapErrorTypeToHTTPStatus(errorType apperrors.ErrorType) int {
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

// RateLimitingMiddleware provides basic rate limiting
func RateLimitingMiddleware(requestsPerMinute int, logger *slog.Logger) gin.HandlerFunc {
	// Simple in-memory rate limiter (for production, use Redis-based limiter)
	clients := make(map[string][]time.Time)

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()

		// Clean old entries
		if requests, exists := clients[clientIP]; exists {
			var validRequests []time.Time
			cutoff := now.Add(-time.Minute)

			for _, reqTime := range requests {
				if reqTime.After(cutoff) {
					validRequests = append(validRequests, reqTime)
				}
			}
			clients[clientIP] = validRequests
		}

		// Check rate limit
		if len(clients[clientIP]) >= requestsPerMinute {
			logger.Warn("rate limit exceeded",
				"client_ip", clientIP,
				"requests_per_minute", requestsPerMinute)

			appErr := apperrors.NewValidationError("rate limit exceeded").
				WithCode("rate_limit_exceeded")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":      appErr.Message,
				"error_code": appErr.Code,
				"type":       appErr.Type,
			})
			c.Abort()
			return
		}

		// Record this request
		clients[clientIP] = append(clients[clientIP], now)
		c.Next()
	}
}
