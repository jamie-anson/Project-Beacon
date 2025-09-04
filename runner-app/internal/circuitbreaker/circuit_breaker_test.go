package circuitbreaker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCircuitBreakerStates(t *testing.T) {
	config := Config{
		Name:             "test",
		MaxFailures:      2,
		Timeout:          100 * time.Millisecond,
		MaxRequests:      2,
		SuccessThreshold: 2,
	}
	
	cb := New(config)
	ctx := context.Background()
	
	// Initially closed
	assert.Equal(t, StateClosed, cb.State())
	
	// First failure - should remain closed
	err := cb.Execute(ctx, func(ctx context.Context) error {
		return errors.New("failure")
	})
	assert.Error(t, err)
	assert.Equal(t, StateClosed, cb.State())
	
	// Second failure - should open
	err = cb.Execute(ctx, func(ctx context.Context) error {
		return errors.New("failure")
	})
	assert.Error(t, err)
	assert.Equal(t, StateOpen, cb.State())
	
	// Should reject requests when open
	err = cb.Execute(ctx, func(ctx context.Context) error {
		return nil
	})
	assert.Equal(t, ErrCircuitOpen, err)
	
	// Wait for timeout and try again - should transition to half-open
	time.Sleep(150 * time.Millisecond)
	
	err = cb.Execute(ctx, func(ctx context.Context) error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateHalfOpen, cb.State())
	
	// Another success should close the circuit
	err = cb.Execute(ctx, func(ctx context.Context) error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.State())
}

func TestCircuitBreakerHalfOpenLimits(t *testing.T) {
	config := Config{
		Name:             "test-half-open",
		MaxFailures:      1,
		Timeout:          50 * time.Millisecond,
		MaxRequests:      2,
		SuccessThreshold: 2, // Need 2 successes to close
	}
	
	cb := New(config)
	ctx := context.Background()
	
	// Trigger open state
	cb.Execute(ctx, func(ctx context.Context) error {
		return errors.New("failure")
	})
	assert.Equal(t, StateOpen, cb.State())
	
	// Wait for timeout
	time.Sleep(60 * time.Millisecond)
	
	// First request in half-open (success)
	err := cb.Execute(ctx, func(ctx context.Context) error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateHalfOpen, cb.State())
	
	// Second request in half-open (success) - should close circuit
	err = cb.Execute(ctx, func(ctx context.Context) error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.State())
}

func TestCircuitBreakerWithFallback(t *testing.T) {
	config := Config{
		Name:        "test-fallback",
		MaxFailures: 1,
		Timeout:     50 * time.Millisecond,
	}
	
	cb := New(config)
	ctx := context.Background()
	
	// Trigger open state
	cb.Execute(ctx, func(ctx context.Context) error {
		return errors.New("failure")
	})
	
	fallbackCalled := false
	err := cb.ExecuteWithFallback(
		ctx,
		func(ctx context.Context) error {
			return nil
		},
		func(ctx context.Context) error {
			fallbackCalled = true
			return nil
		},
	)
	
	assert.NoError(t, err)
	assert.True(t, fallbackCalled, "Fallback should be called when circuit is open")
}

func TestCircuitBreakerCustomFailureFunction(t *testing.T) {
	config := Config{
		Name:        "test-custom-failure",
		MaxFailures: 2,
		IsFailure: func(err error) bool {
			// Only count specific errors as failures
			return err != nil && err.Error() == "critical"
		},
	}
	
	cb := New(config)
	ctx := context.Background()
	
	// Non-critical error should not count as failure
	err := cb.Execute(ctx, func(ctx context.Context) error {
		return errors.New("non-critical")
	})
	assert.Error(t, err)
	assert.Equal(t, StateClosed, cb.State())
	
	// Critical error should count as failure
	err = cb.Execute(ctx, func(ctx context.Context) error {
		return errors.New("critical")
	})
	assert.Error(t, err)
	assert.Equal(t, StateClosed, cb.State()) // Still closed, need 2 failures
	
	// Second critical error should open circuit
	err = cb.Execute(ctx, func(ctx context.Context) error {
		return errors.New("critical")
	})
	assert.Error(t, err)
	assert.Equal(t, StateOpen, cb.State())
}

func TestCircuitBreakerStats(t *testing.T) {
	config := DefaultConfig("test-stats")
	cb := New(config)
	ctx := context.Background()
	
	// Execute some operations
	cb.Execute(ctx, func(ctx context.Context) error { return nil })
	cb.Execute(ctx, func(ctx context.Context) error { return errors.New("failure") })
	
	stats := cb.Stats()
	assert.Equal(t, "test-stats", stats.Name)
	assert.Equal(t, StateClosed, stats.State)
	assert.Equal(t, 1, stats.Failures)
}

func TestCircuitBreakerManager(t *testing.T) {
	manager := NewManager()
	
	// Create circuit breakers
	config1 := DefaultConfig("service1")
	cb1 := manager.GetOrCreate("service1", config1)
	
	config2 := DefaultConfig("service2")
	cb2 := manager.GetOrCreate("service2", config2)
	
	// Verify they are different instances
	assert.NotEqual(t, cb1, cb2)
	
	// Verify we get the same instance on subsequent calls
	cb1Again := manager.GetOrCreate("service1", config1)
	assert.Equal(t, cb1, cb1Again)
	
	// Test Get method
	retrieved, exists := manager.Get("service1")
	assert.True(t, exists)
	assert.Equal(t, cb1, retrieved)
	
	_, exists = manager.Get("nonexistent")
	assert.False(t, exists)
	
	// Test AllStats
	stats := manager.AllStats()
	assert.Len(t, stats, 2)
	
	// Test Reset
	ctx := context.Background()
	cb1.Execute(ctx, func(ctx context.Context) error { return errors.New("failure") })
	assert.Equal(t, 1, cb1.Stats().Failures)
	
	manager.Reset()
	assert.Equal(t, 0, cb1.Stats().Failures)
	assert.Equal(t, StateClosed, cb1.State())
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig("test")
	
	assert.Equal(t, "test", config.Name)
	assert.Equal(t, 5, config.MaxFailures)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, 3, config.MaxRequests)
	assert.Equal(t, 2, config.SuccessThreshold)
	assert.NotNil(t, config.IsFailure)
	
	// Test default IsFailure function
	assert.True(t, config.IsFailure(errors.New("error")))
	assert.False(t, config.IsFailure(nil))
}

func TestStateString(t *testing.T) {
	assert.Equal(t, "closed", StateClosed.String())
	assert.Equal(t, "open", StateOpen.String())
	assert.Equal(t, "half-open", StateHalfOpen.String())
	assert.Equal(t, "unknown", State(999).String())
}

func TestStatsString(t *testing.T) {
	stats := Stats{
		Name:      "test",
		State:     StateOpen,
		Failures:  3,
		Successes: 1,
		Requests:  2,
	}
	
	expected := "CircuitBreaker[test]: state=open, failures=3, successes=1, requests=2"
	assert.Equal(t, expected, stats.String())
}

func BenchmarkCircuitBreakerExecute(b *testing.B) {
	config := DefaultConfig("benchmark")
	cb := New(config)
	ctx := context.Background()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.Execute(ctx, func(ctx context.Context) error {
				return nil
			})
		}
	})
}
