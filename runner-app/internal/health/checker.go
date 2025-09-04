package health

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jamie-anson/project-beacon-runner/internal/circuitbreaker"
	"github.com/jamie-anson/project-beacon-runner/internal/logging"
)

// ServiceStatus represents the health status of a service
type ServiceStatus string

const (
	StatusHealthy   ServiceStatus = "healthy"
	StatusDegraded  ServiceStatus = "degraded"
	StatusUnhealthy ServiceStatus = "unhealthy"
	StatusUnknown   ServiceStatus = "unknown"
)

// ServiceHealth represents health information for a service
type ServiceHealth struct {
	Name               string                     `json:"name"`
	Status             ServiceStatus              `json:"status"`
	LastCheck          time.Time                  `json:"last_check"`
	ResponseTime       time.Duration              `json:"response_time_ms"`
	Error              string                     `json:"error,omitempty"`
	CircuitBreakerStats circuitbreaker.Stats      `json:"circuit_breaker_stats"`
	Details            map[string]interface{}     `json:"details,omitempty"`
}

// HealthChecker performs health checks on various services
type HealthChecker struct {
	services map[string]HealthCheckFunc
	cbManager *circuitbreaker.Manager
	logger   *logging.StructuredLogger
	mutex    sync.RWMutex
}

// HealthCheckFunc defines the signature for health check functions
type HealthCheckFunc func(context.Context) (ServiceStatus, time.Duration, error, map[string]interface{})

// NewHealthChecker creates a new health checker
func NewHealthChecker(cbManager *circuitbreaker.Manager) *HealthChecker {
	return &HealthChecker{
		services:  make(map[string]HealthCheckFunc),
		cbManager: cbManager,
		logger:    logging.NewStructuredLogger("health-checker"),
	}
}

// RegisterService registers a health check function for a service
func (hc *HealthChecker) RegisterService(name string, checkFunc HealthCheckFunc) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()
	hc.services[name] = checkFunc
}

// CheckService performs a health check on a specific service
func (hc *HealthChecker) CheckService(ctx context.Context, serviceName string) ServiceHealth {
	hc.mutex.RLock()
	checkFunc, exists := hc.services[serviceName]
	hc.mutex.RUnlock()

	if !exists {
		return ServiceHealth{
			Name:        serviceName,
			Status:      StatusUnknown,
			LastCheck:   time.Now(),
			Error:       "service not registered",
		}
	}

	start := time.Now()
	status, responseTime, err, details := checkFunc(ctx)
	
	health := ServiceHealth{
		Name:         serviceName,
		Status:       status,
		LastCheck:    start,
		ResponseTime: responseTime,
		Details:      details,
	}

	if err != nil {
		health.Error = err.Error()
	}

	// Get circuit breaker stats if available
	if cb, exists := hc.cbManager.Get(serviceName); exists {
		health.CircuitBreakerStats = cb.Stats()
	}

	return health
}

// CheckAllServices performs health checks on all registered services
func (hc *HealthChecker) CheckAllServices(ctx context.Context) []ServiceHealth {
	hc.mutex.RLock()
	serviceNames := make([]string, 0, len(hc.services))
	for name := range hc.services {
		serviceNames = append(serviceNames, name)
	}
	hc.mutex.RUnlock()

	results := make([]ServiceHealth, len(serviceNames))
	var wg sync.WaitGroup

	for i, name := range serviceNames {
		wg.Add(1)
		go func(index int, serviceName string) {
			defer wg.Done()
			results[index] = hc.CheckService(ctx, serviceName)
		}(i, name)
	}

	wg.Wait()
	return results
}

// GetOverallHealth determines overall system health
func (hc *HealthChecker) GetOverallHealth(ctx context.Context) (ServiceStatus, []ServiceHealth) {
	services := hc.CheckAllServices(ctx)
	
	overallStatus := StatusHealthy
	criticalServices := []string{"database", "redis"}
	
	for _, service := range services {
		// Critical services affect overall health more severely
		isCritical := false
		for _, critical := range criticalServices {
			if service.Name == critical {
				isCritical = true
				break
			}
		}
		
		if isCritical && service.Status == StatusUnhealthy {
			overallStatus = StatusUnhealthy
			break
		} else if service.Status == StatusUnhealthy {
			if overallStatus == StatusHealthy {
				overallStatus = StatusDegraded
			}
		} else if service.Status == StatusDegraded && overallStatus == StatusHealthy {
			overallStatus = StatusDegraded
		}
	}
	
	return overallStatus, services
}

// DatabaseHealthCheck creates a health check function for database
func DatabaseHealthCheck(db *sql.DB) HealthCheckFunc {
	return func(ctx context.Context) (ServiceStatus, time.Duration, error, map[string]interface{}) {
		start := time.Now()
		
		// Simple ping
		err := db.PingContext(ctx)
		responseTime := time.Since(start)
		
		if err != nil {
			return StatusUnhealthy, responseTime, err, nil
		}
		
		// Check connection stats
		stats := db.Stats()
		details := map[string]interface{}{
			"open_connections":     stats.OpenConnections,
			"in_use":              stats.InUse,
			"idle":                stats.Idle,
			"wait_count":          stats.WaitCount,
			"wait_duration_ms":    stats.WaitDuration.Milliseconds(),
		}
		
		// Determine status based on connection health
		status := StatusHealthy
		if stats.OpenConnections > 80 { // Assuming max 100 connections
			status = StatusDegraded
		}
		if responseTime > 1*time.Second {
			status = StatusDegraded
		}
		
		return status, responseTime, nil, details
	}
}

// RedisHealthCheck creates a health check function for Redis
func RedisHealthCheck(client *redis.Client) HealthCheckFunc {
	return func(ctx context.Context) (ServiceStatus, time.Duration, error, map[string]interface{}) {
		start := time.Now()
		
		// Simple ping
		result := client.Ping(ctx)
		responseTime := time.Since(start)
		
		if result.Err() != nil {
			return StatusUnhealthy, responseTime, result.Err(), nil
		}
		
		// Get Redis info
		info := client.Info(ctx, "memory", "stats")
		details := map[string]interface{}{
			"ping_response": result.Val(),
		}
		
		if info.Err() == nil {
			details["redis_info"] = "available"
		}
		
		// Determine status based on response time
		status := StatusHealthy
		if responseTime > 500*time.Millisecond {
			status = StatusDegraded
		}
		
		return status, responseTime, nil, details
	}
}

// HTTPHealthCheck creates a health check function for HTTP services
func HTTPHealthCheck(url string, timeout time.Duration) HealthCheckFunc {
	client := &http.Client{Timeout: timeout}
	
	return func(ctx context.Context) (ServiceStatus, time.Duration, error, map[string]interface{}) {
		start := time.Now()
		
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return StatusUnhealthy, 0, err, nil
		}
		
		resp, err := client.Do(req)
		responseTime := time.Since(start)
		
		if err != nil {
			return StatusUnhealthy, responseTime, err, nil
		}
		defer resp.Body.Close()
		
		details := map[string]interface{}{
			"status_code":    resp.StatusCode,
			"content_length": resp.ContentLength,
		}
		
		// Determine status based on HTTP status code and response time
		status := StatusHealthy
		if resp.StatusCode >= 500 {
			status = StatusUnhealthy
		} else if resp.StatusCode >= 400 {
			status = StatusDegraded
		}
		
		if responseTime > 2*time.Second {
			status = StatusDegraded
		}
		
		return status, responseTime, nil, details
	}
}

// YagnaHealthCheck creates a health check function for Yagna service
func YagnaHealthCheck(baseURL string) HealthCheckFunc {
	return HTTPHealthCheck(fmt.Sprintf("%s/version", baseURL), 5*time.Second)
}

// IPFSHealthCheck creates a health check function for IPFS service
func IPFSHealthCheck(baseURL string) HealthCheckFunc {
	return HTTPHealthCheck(fmt.Sprintf("%s/api/v0/version", baseURL), 5*time.Second)
}

// CustomHealthCheck creates a health check function with custom logic
func CustomHealthCheck(name string, checkFn func(context.Context) error) HealthCheckFunc {
	return func(ctx context.Context) (ServiceStatus, time.Duration, error, map[string]interface{}) {
		start := time.Now()
		err := checkFn(ctx)
		responseTime := time.Since(start)
		
		if err != nil {
			return StatusUnhealthy, responseTime, err, nil
		}
		
		status := StatusHealthy
		if responseTime > 1*time.Second {
			status = StatusDegraded
		}
		
		return status, responseTime, nil, map[string]interface{}{
			"check_type": "custom",
		}
	}
}
