package recovery

import (
	"context"
	"log/slog"
	"math/rand"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/circuitbreaker"
	"github.com/jamie-anson/project-beacon-runner/internal/errors"
)

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	Jitter       bool
}

// DefaultRetryConfig returns sensible defaults
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
	}
}

// RecoveryManager handles error recovery and retry logic
type RecoveryManager struct {
	circuitManager *circuitbreaker.Manager
	logger         *slog.Logger
}

// NewRecoveryManager creates a new recovery manager
func NewRecoveryManager(logger *slog.Logger) *RecoveryManager {
	return &RecoveryManager{
		circuitManager: circuitbreaker.NewManager(),
		logger:         logger,
	}
}

// ExecuteWithRecovery executes a function with comprehensive error handling
func (rm *RecoveryManager) ExecuteWithRecovery(
	ctx context.Context,
	operation string,
	fn func(context.Context) error,
	config RetryConfig,
) error {
	// Get or create circuit breaker for this operation
	cb := rm.circuitManager.GetOrCreate(operation, circuitbreaker.Config{
		Name:             operation,
		MaxFailures:      5,
		Timeout:          30 * time.Second,
		MaxRequests:      3,
		SuccessThreshold: 2,
		IsFailure: func(err error) bool {
			// Don't count validation errors as circuit breaker failures
			return err != nil && !errors.IsType(err, errors.ValidationError)
		},
	})

	// Execute with circuit breaker and retry
	return rm.executeWithRetry(ctx, operation, func(ctx context.Context) error {
		return cb.Execute(ctx, fn)
	}, config)
}

// ExecuteWithFallback executes with fallback when circuit breaker is open
func (rm *RecoveryManager) ExecuteWithFallback(
	ctx context.Context,
	operation string,
	primary func(context.Context) error,
	fallback func(context.Context) error,
	config RetryConfig,
) error {
	cb := rm.circuitManager.GetOrCreate(operation, circuitbreaker.DefaultConfig(operation))
	
	return rm.executeWithRetry(ctx, operation, func(ctx context.Context) error {
		return cb.ExecuteWithFallback(ctx, primary, fallback)
	}, config)
}

// executeWithRetry implements exponential backoff retry logic
func (rm *RecoveryManager) executeWithRetry(
	ctx context.Context,
	operation string,
	fn func(context.Context) error,
	config RetryConfig,
) error {
	var lastErr error
	delay := config.InitialDelay

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		// Check context cancellation
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Execute the function
		err := fn(ctx)
		if err == nil {
			if attempt > 1 {
				rm.logger.Info("operation recovered after retry",
					"operation", operation,
					"attempt", attempt)
			}
			return nil
		}

		lastErr = err

		// Don't retry certain error types
		if !rm.shouldRetry(err) {
			rm.logger.Warn("operation failed with non-retryable error",
				"operation", operation,
				"error", err,
				"attempt", attempt)
			return err
		}

		// Don't retry on last attempt
		if attempt == config.MaxAttempts {
			break
		}

		// Calculate next delay with exponential backoff
		nextDelay := rm.calculateDelay(delay, config)
		
		rm.logger.Warn("operation failed, retrying",
			"operation", operation,
			"error", err,
			"attempt", attempt,
			"max_attempts", config.MaxAttempts,
			"retry_delay", nextDelay)

		// Wait before retry
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(nextDelay):
			delay = nextDelay
		}
	}

	rm.logger.Error("operation failed after all retries",
		"operation", operation,
		"error", lastErr,
		"attempts", config.MaxAttempts)

	return errors.Wrapf(lastErr, errors.InternalError,
		"operation %s failed after %d attempts", operation, config.MaxAttempts)
}

// shouldRetry determines if an error should trigger a retry
func (rm *RecoveryManager) shouldRetry(err error) bool {
	// Don't retry validation errors
	if errors.IsType(err, errors.ValidationError) {
		return false
	}
	
	// Don't retry authentication/authorization errors
	if errors.IsType(err, errors.AuthenticationError) ||
		errors.IsType(err, errors.AuthorizationError) {
		return false
	}
	
	// Don't retry not found errors
	if errors.IsType(err, errors.NotFoundError) {
		return false
	}
	
	// Don't retry circuit breaker errors (circuit breaker handles its own recovery)
	if errors.IsType(err, errors.CircuitBreakerError) {
		return false
	}
	
	// Retry external service errors, database errors, timeouts, and internal errors
	return errors.IsType(err, errors.ExternalServiceError) ||
		errors.IsType(err, errors.DatabaseError) ||
		errors.IsType(err, errors.TimeoutError) ||
		errors.IsType(err, errors.InternalError)
}

// calculateDelay calculates the next retry delay with exponential backoff
func (rm *RecoveryManager) calculateDelay(currentDelay time.Duration, config RetryConfig) time.Duration {
	nextDelay := time.Duration(float64(currentDelay) * config.Multiplier)
	
	// Apply maximum delay cap
	if nextDelay > config.MaxDelay {
		nextDelay = config.MaxDelay
	}
	
	// Apply jitter to prevent thundering herd
	if config.Jitter {
		jitterRange := nextDelay / 4 // 25% jitter
		if jitterRange > 0 {
			// Random delta in [-jitterRange, +jitterRange]
			delta := rand.Int63n(int64(2*jitterRange)+1) - int64(jitterRange)
			nextDelay += time.Duration(delta)
		}
		
		// Ensure delay doesn't go negative
		if nextDelay < 0 {
			nextDelay = config.InitialDelay
		}
	}
	
	return nextDelay
}

// GetCircuitBreakerStats returns stats for all circuit breakers
func (rm *RecoveryManager) GetCircuitBreakerStats() []circuitbreaker.Stats {
	return rm.circuitManager.AllStats()
}

// ResetCircuitBreakers resets all circuit breakers to closed state
func (rm *RecoveryManager) ResetCircuitBreakers() {
	rm.circuitManager.Reset()
	rm.logger.Info("all circuit breakers reset to closed state")
}

// HealthCheck performs a health check on all circuit breakers
func (rm *RecoveryManager) HealthCheck() map[string]string {
	stats := rm.circuitManager.AllStats()
	health := make(map[string]string)
	
	for _, stat := range stats {
		switch stat.State {
		case circuitbreaker.StateClosed:
			health[stat.Name] = "healthy"
		case circuitbreaker.StateHalfOpen:
			health[stat.Name] = "recovering"
		case circuitbreaker.StateOpen:
			health[stat.Name] = "unhealthy"
		default:
			health[stat.Name] = "unknown"
		}
	}
	
	return health
}
