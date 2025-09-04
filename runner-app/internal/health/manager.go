package health

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jamie-anson/project-beacon-runner/internal/circuitbreaker"
	"github.com/jamie-anson/project-beacon-runner/internal/logging"
)

// HealthManager manages health checks and circuit breakers
type HealthManager struct {
	checker       *HealthChecker
	cbManager     *circuitbreaker.Manager
	logger        *logging.StructuredLogger
	checkInterval time.Duration
	stopChan      chan struct{}
	wg            sync.WaitGroup
}

// NewHealthManager creates a new health manager
func NewHealthManager(checkInterval time.Duration) *HealthManager {
	cbManager := circuitbreaker.NewManager()
	
	return &HealthManager{
		checker:       NewHealthChecker(cbManager),
		cbManager:     cbManager,
		logger:        logging.NewStructuredLogger("health-manager"),
		checkInterval: checkInterval,
		stopChan:      make(chan struct{}),
	}
}

// RegisterDatabaseHealth registers database health check with circuit breaker
func (hm *HealthManager) RegisterDatabaseHealth(db *sql.DB) {
	// Register health check
	hm.checker.RegisterService("database", DatabaseHealthCheck(db))
	
	// Create circuit breaker for database operations
	hm.cbManager.GetOrCreate("database", circuitbreaker.Config{
		Name:             "database",
		MaxFailures:      3,
		Timeout:          10 * time.Second,
		MaxRequests:      2,
		SuccessThreshold: 2,
		IsFailure: func(err error) bool {
			return err != nil
		},
	})
}

// RegisterRedisHealth registers Redis health check with circuit breaker
func (hm *HealthManager) RegisterRedisHealth(client *redis.Client) {
	// Register health check
	hm.checker.RegisterService("redis", RedisHealthCheck(client))
	
	// Create circuit breaker for Redis operations
	hm.cbManager.GetOrCreate("redis", circuitbreaker.Config{
		Name:             "redis",
		MaxFailures:      3,
		Timeout:          15 * time.Second,
		MaxRequests:      3,
		SuccessThreshold: 2,
		IsFailure: func(err error) bool {
			return err != nil
		},
	})
}

// RegisterYagnaHealth registers Yagna health check with circuit breaker
func (hm *HealthManager) RegisterYagnaHealth(baseURL string) {
	// Register health check
	hm.checker.RegisterService("yagna", YagnaHealthCheck(baseURL))
	
	// Create circuit breaker for Yagna operations
	hm.cbManager.GetOrCreate("yagna", circuitbreaker.Config{
		Name:             "yagna",
		MaxFailures:      5,
		Timeout:          30 * time.Second,
		MaxRequests:      3,
		SuccessThreshold: 2,
		IsFailure: func(err error) bool {
			return err != nil
		},
	})
}

// RegisterIPFSHealth registers IPFS health check with circuit breaker
func (hm *HealthManager) RegisterIPFSHealth(baseURL string) {
	// Register health check
	hm.checker.RegisterService("ipfs", IPFSHealthCheck(baseURL))
	
	// Create circuit breaker for IPFS operations
	hm.cbManager.GetOrCreate("ipfs", circuitbreaker.Config{
		Name:             "ipfs",
		MaxFailures:      4,
		Timeout:          20 * time.Second,
		MaxRequests:      2,
		SuccessThreshold: 2,
		IsFailure: func(err error) bool {
			return err != nil
		},
	})
}

// RegisterCustomHealth registers a custom health check
func (hm *HealthManager) RegisterCustomHealth(name string, checkFn func(context.Context) error, cbConfig circuitbreaker.Config) {
	// Register health check
	hm.checker.RegisterService(name, CustomHealthCheck(name, checkFn))
	
	// Create circuit breaker
	cbConfig.Name = name
	hm.cbManager.GetOrCreate(name, cbConfig)
}

// Start begins periodic health checks
func (hm *HealthManager) Start(ctx context.Context) {
	hm.wg.Add(1)
	go hm.healthCheckLoop(ctx)
	
	hm.logger.Info("health manager started",
		"check_interval", hm.checkInterval)
}

// Stop stops the health check loop
func (hm *HealthManager) Stop() {
	close(hm.stopChan)
	hm.wg.Wait()
	
	hm.logger.Info("health manager stopped")
}

// healthCheckLoop runs periodic health checks
func (hm *HealthManager) healthCheckLoop(ctx context.Context) {
	defer hm.wg.Done()
	
	ticker := time.NewTicker(hm.checkInterval)
	defer ticker.Stop()
	
	// Run initial health check
	hm.performHealthChecks(ctx)
	
	for {
		select {
		case <-ticker.C:
			hm.performHealthChecks(ctx)
		case <-hm.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// performHealthChecks executes health checks and updates circuit breakers
func (hm *HealthManager) performHealthChecks(ctx context.Context) {
	services := hm.checker.CheckAllServices(ctx)
	
	for _, service := range services {
		// Log health status changes
		if service.Status != StatusHealthy {
			hm.logger.Warn("service health degraded",
				"service", service.Name,
				"status", service.Status,
				"error", service.Error,
				"response_time_ms", service.ResponseTime.Milliseconds())
		}
		
		// Update circuit breaker state based on health
		if cb, exists := hm.cbManager.Get(service.Name); exists {
			switch service.Status {
			case StatusUnhealthy:
				// Circuit breaker will handle failures automatically
				hm.logger.Debug("service unhealthy, circuit breaker will protect",
					"service", service.Name,
					"cb_state", cb.State())
			case StatusHealthy:
				// Service is healthy, circuit breaker should allow requests
				hm.logger.Debug("service healthy",
					"service", service.Name,
					"cb_state", cb.State())
			}
		}
	}
	
	// Log overall system health
	overallStatus, _ := hm.checker.GetOverallHealth(ctx)
	if overallStatus != StatusHealthy {
		hm.logger.Warn("system health degraded",
			"overall_status", overallStatus)
	}
}

// GetHealthChecker returns the health checker instance
func (hm *HealthManager) GetHealthChecker() *HealthChecker {
	return hm.checker
}

// GetCircuitBreakerManager returns the circuit breaker manager
func (hm *HealthManager) GetCircuitBreakerManager() *circuitbreaker.Manager {
	return hm.cbManager
}

// GetHealthSummary returns a summary of all health checks
func (hm *HealthManager) GetHealthSummary(ctx context.Context) map[string]interface{} {
	overallStatus, services := hm.checker.GetOverallHealth(ctx)
	cbStats := hm.cbManager.AllStats()
	
	summary := map[string]interface{}{
		"overall_status":    overallStatus,
		"timestamp":         time.Now(),
		"services":          services,
		"circuit_breakers":  make(map[string]interface{}),
	}
	
	// Add circuit breaker stats
	cbData := make(map[string]interface{})
	for _, stat := range cbStats {
		cbData[stat.Name] = map[string]interface{}{
			"state":          stat.State.String(),
			"failures":       stat.Failures,
			"successes":      stat.Successes,
			"requests":       stat.Requests,
			"last_fail_time": stat.LastFailTime,
		}
	}
	summary["circuit_breakers"] = cbData
	
	return summary
}

// ResetCircuitBreakers resets all circuit breakers
func (hm *HealthManager) ResetCircuitBreakers() {
	hm.cbManager.Reset()
	hm.logger.Info("all circuit breakers reset")
}

// ExecuteWithCircuitBreaker executes a function with circuit breaker protection
func (hm *HealthManager) ExecuteWithCircuitBreaker(ctx context.Context, serviceName string, fn func(context.Context) error) error {
	cb := hm.cbManager.GetOrCreate(serviceName, circuitbreaker.DefaultConfig(serviceName))
	return cb.Execute(ctx, fn)
}

// ExecuteWithFallback executes a function with circuit breaker and fallback
func (hm *HealthManager) ExecuteWithFallback(ctx context.Context, serviceName string, primary func(context.Context) error, fallback func(context.Context) error) error {
	cb := hm.cbManager.GetOrCreate(serviceName, circuitbreaker.DefaultConfig(serviceName))
	return cb.ExecuteWithFallback(ctx, primary, fallback)
}
