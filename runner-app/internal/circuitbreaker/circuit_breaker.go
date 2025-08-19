package circuitbreaker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// State represents the circuit breaker state
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Config holds circuit breaker configuration
type Config struct {
	Name               string        // Circuit breaker name for logging
	MaxFailures        int           // Maximum failures before opening
	Timeout            time.Duration // Time to wait before transitioning to half-open
	MaxRequests        int           // Maximum requests allowed in half-open state
	SuccessThreshold   int           // Successful requests needed to close from half-open
	IsFailure          func(error) bool // Function to determine if error should count as failure
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig(name string) Config {
	return Config{
		Name:             name,
		MaxFailures:      5,
		Timeout:          30 * time.Second,
		MaxRequests:      3,
		SuccessThreshold: 2,
		IsFailure: func(err error) bool {
			return err != nil
		},
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	config       Config
	state        State
	failures     int
	successes    int
	requests     int
	lastFailTime time.Time
	mutex        sync.RWMutex
}

// New creates a new circuit breaker with the given configuration
func New(config Config) *CircuitBreaker {
	if config.IsFailure == nil {
		config.IsFailure = func(err error) bool {
			return err != nil
		}
	}
	
	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

// ErrCircuitOpen is returned when the circuit breaker is open
var ErrCircuitOpen = errors.New("circuit breaker is open")

// ErrTooManyRequests is returned when too many requests are made in half-open state
var ErrTooManyRequests = errors.New("too many requests in half-open state")

// Execute runs the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	if err := cb.beforeRequest(); err != nil {
		return err
	}
	
	err := fn(ctx)
	cb.afterRequest(err)
	return err
}

// ExecuteWithFallback runs the function with circuit breaker protection and fallback
func (cb *CircuitBreaker) ExecuteWithFallback(ctx context.Context, fn func(context.Context) error, fallback func(context.Context) error) error {
	err := cb.Execute(ctx, fn)
	if err == ErrCircuitOpen || err == ErrTooManyRequests {
		if fallback != nil {
			return fallback(ctx)
		}
	}
	return err
}

// beforeRequest checks if the request should be allowed
func (cb *CircuitBreaker) beforeRequest() error {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	switch cb.state {
	case StateClosed:
		return nil
	case StateOpen:
		if time.Since(cb.lastFailTime) > cb.config.Timeout {
			cb.state = StateHalfOpen
			cb.requests = 1 // Count this request
			cb.successes = 0
			return nil
		}
		return ErrCircuitOpen
	case StateHalfOpen:
		if cb.requests >= cb.config.MaxRequests {
			return ErrTooManyRequests
		}
		cb.requests++
		return nil
	default:
		return ErrCircuitOpen
	}
}

// afterRequest processes the result of the request
func (cb *CircuitBreaker) afterRequest(err error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	if cb.config.IsFailure(err) {
		cb.onFailure()
	} else {
		cb.onSuccess()
	}
}

// onFailure handles a failed request
func (cb *CircuitBreaker) onFailure() {
	cb.failures++
	cb.lastFailTime = time.Now()
	
	switch cb.state {
	case StateClosed:
		if cb.failures >= cb.config.MaxFailures {
			cb.state = StateOpen
		}
	case StateHalfOpen:
		cb.state = StateOpen
		cb.requests = 0
		cb.successes = 0
	}
}

// onSuccess handles a successful request
func (cb *CircuitBreaker) onSuccess() {
	switch cb.state {
	case StateClosed:
		cb.failures = 0
	case StateHalfOpen:
		cb.successes++
		if cb.successes >= cb.config.SuccessThreshold {
			cb.state = StateClosed
			cb.failures = 0
			cb.requests = 0
			cb.successes = 0
		}
	}
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() State {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// Stats returns current statistics
func (cb *CircuitBreaker) Stats() Stats {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	
	return Stats{
		Name:         cb.config.Name,
		State:        cb.state,
		Failures:     cb.failures,
		Successes:    cb.successes,
		Requests:     cb.requests,
		LastFailTime: cb.lastFailTime,
	}
}

// Stats holds circuit breaker statistics
type Stats struct {
	Name         string
	State        State
	Failures     int
	Successes    int
	Requests     int
	LastFailTime time.Time
}

// String returns a string representation of the stats
func (s Stats) String() string {
	return fmt.Sprintf("CircuitBreaker[%s]: state=%s, failures=%d, successes=%d, requests=%d",
		s.Name, s.State, s.Failures, s.Successes, s.Requests)
}

// Manager manages multiple circuit breakers
type Manager struct {
	breakers map[string]*CircuitBreaker
	mutex    sync.RWMutex
}

// NewManager creates a new circuit breaker manager
func NewManager() *Manager {
	return &Manager{
		breakers: make(map[string]*CircuitBreaker),
	}
}

// GetOrCreate gets an existing circuit breaker or creates a new one
func (m *Manager) GetOrCreate(name string, config Config) *CircuitBreaker {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if cb, exists := m.breakers[name]; exists {
		return cb
	}
	
	config.Name = name
	cb := New(config)
	m.breakers[name] = cb
	return cb
}

// Get retrieves a circuit breaker by name
func (m *Manager) Get(name string) (*CircuitBreaker, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	cb, exists := m.breakers[name]
	return cb, exists
}

// AllStats returns statistics for all circuit breakers
func (m *Manager) AllStats() []Stats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	stats := make([]Stats, 0, len(m.breakers))
	for _, cb := range m.breakers {
		stats = append(stats, cb.Stats())
	}
	return stats
}

// Reset resets all circuit breakers to closed state
func (m *Manager) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	for _, cb := range m.breakers {
		cb.mutex.Lock()
		cb.state = StateClosed
		cb.failures = 0
		cb.successes = 0
		cb.requests = 0
		cb.mutex.Unlock()
	}
}
