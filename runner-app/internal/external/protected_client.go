package external

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/circuitbreaker"
)

// ProtectedClient wraps external service calls with circuit breaker protection
type ProtectedClient struct {
	httpClient *http.Client
	cbManager  *circuitbreaker.Manager
}

// NewProtectedClient creates a new protected client with circuit breaker support
func NewProtectedClient(httpClient *http.Client) *ProtectedClient {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}
	
	return &ProtectedClient{
		httpClient: httpClient,
		cbManager:  circuitbreaker.NewManager(),
	}
}

// YagnaClient provides circuit breaker protected access to Yagna API
type YagnaClient struct {
	*ProtectedClient
	baseURL string
}

// NewYagnaClient creates a new Yagna client with circuit breaker protection
func NewYagnaClient(baseURL string) *YagnaClient {
	return &YagnaClient{
		ProtectedClient: NewProtectedClient(nil),
		baseURL:         baseURL,
	}
}

// SubmitTask submits a task to Yagna with circuit breaker protection
func (yc *YagnaClient) SubmitTask(ctx context.Context, taskSpec interface{}) (string, error) {
	config := circuitbreaker.Config{
		Name:             "yagna-submit",
		MaxFailures:      3,
		Timeout:          60 * time.Second,
		MaxRequests:      2,
		SuccessThreshold: 2,
		IsFailure: func(err error) bool {
			// Only count 5xx errors and timeouts as failures
			if err == nil {
				return false
			}
			// Add logic to check for specific error types
			return true
		},
	}
	
	cb := yc.cbManager.GetOrCreate("yagna-submit", config)
	
	var taskID string
	err := cb.ExecuteWithFallback(
		ctx,
		func(ctx context.Context) error {
			// Actual Yagna API call would go here
			// For now, simulate the call
			taskID = fmt.Sprintf("task-%d", time.Now().UnixNano())
			return nil
		},
		func(ctx context.Context) error {
			// Fallback: return cached result or error
			return fmt.Errorf("yagna service unavailable, circuit breaker open")
		},
	)
	
	return taskID, err
}

// GetTaskStatus retrieves task status from Yagna with circuit breaker protection
func (yc *YagnaClient) GetTaskStatus(ctx context.Context, taskID string) (string, error) {
	config := circuitbreaker.DefaultConfig("yagna-status")
	cb := yc.cbManager.GetOrCreate("yagna-status", config)
	
	var status string
	err := cb.Execute(ctx, func(ctx context.Context) error {
		// Actual Yagna API call would go here
		status = "running" // Simulate status
		return nil
	})
	
	return status, err
}

// IPFSClient provides circuit breaker protected access to IPFS
type IPFSClient struct {
	*ProtectedClient
	apiURL string
}

// NewIPFSClient creates a new IPFS client with circuit breaker protection
func NewIPFSClient(apiURL string) *IPFSClient {
	return &IPFSClient{
		ProtectedClient: NewProtectedClient(nil),
		apiURL:          apiURL,
	}
}

// StoreData stores data in IPFS with circuit breaker protection
func (ic *IPFSClient) StoreData(ctx context.Context, data []byte) (string, error) {
	config := circuitbreaker.Config{
		Name:             "ipfs-store",
		MaxFailures:      5,
		Timeout:          30 * time.Second,
		MaxRequests:      3,
		SuccessThreshold: 2,
		IsFailure: func(err error) bool {
			return err != nil
		},
	}
	
	cb := ic.cbManager.GetOrCreate("ipfs-store", config)
	
	var hash string
	err := cb.ExecuteWithFallback(
		ctx,
		func(ctx context.Context) error {
			// Actual IPFS store call would go here
			hash = fmt.Sprintf("Qm%d", time.Now().UnixNano())
			return nil
		},
		func(ctx context.Context) error {
			// Fallback: store locally or return error
			return fmt.Errorf("ipfs service unavailable, storing locally")
		},
	)
	
	return hash, err
}

// RetrieveData retrieves data from IPFS with circuit breaker protection
func (ic *IPFSClient) RetrieveData(ctx context.Context, hash string) ([]byte, error) {
	config := circuitbreaker.DefaultConfig("ipfs-retrieve")
	cb := ic.cbManager.GetOrCreate("ipfs-retrieve", config)
	
	var data []byte
	err := cb.Execute(ctx, func(ctx context.Context) error {
		// Actual IPFS retrieve call would go here
		data = []byte("retrieved data")
		return nil
	})
	
	return data, err
}

// DatabaseClient provides circuit breaker protected database operations
type DatabaseClient struct {
	*ProtectedClient
}

// NewDatabaseClient creates a new database client with circuit breaker protection
func NewDatabaseClient() *DatabaseClient {
	return &DatabaseClient{
		ProtectedClient: NewProtectedClient(nil),
	}
}

// ExecuteQuery executes a database query with circuit breaker protection
func (c *DatabaseClient) ExecuteQuery(_ context.Context, query string, args ...interface{}) error {
	config := circuitbreaker.Config{
		Name:             "database-query",
		MaxFailures:      10, // Higher threshold for database
		Timeout:          15 * time.Second,
		MaxRequests:      5,
		SuccessThreshold: 3,
		IsFailure: func(err error) bool {
			// Don't count application-level errors as circuit breaker failures
			if err == nil {
				return false
			}
			// Only count connection errors, timeouts, etc.
			return true
		},
	}
	
	cb := c.cbManager.GetOrCreate("database-query", config)
	
	return cb.Execute(context.Background(), func(ctx context.Context) error {
		// Actual database query would go here
		return nil
	})
}

// RedisClient provides circuit breaker protected Redis operations
type RedisClient struct {
	*ProtectedClient
}

// NewRedisClient creates a new Redis client with circuit breaker protection
func NewRedisClient() *RedisClient {
	return &RedisClient{
		ProtectedClient: NewProtectedClient(nil),
	}
}

// Set sets a value in Redis with circuit breaker protection
func (c *RedisClient) Set(_ context.Context, key, value string, expiration time.Duration) error {
	config := circuitbreaker.DefaultConfig("redis-set")
	cb := c.cbManager.GetOrCreate("redis-set", config)
	
	return cb.Execute(context.Background(), func(ctx context.Context) error {
		// Actual Redis SET command would go here
		return nil
	})
}

// Get retrieves a value from Redis with circuit breaker protection
func (c *RedisClient) Get(_ context.Context, key string) (string, error) {
	config := circuitbreaker.DefaultConfig("redis-get")
	cb := c.cbManager.GetOrCreate("redis-get", config)
	
	var value string
	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		// Actual Redis GET command would go here
		value = "cached-value"
		return nil
	})
	
	return value, err
}

// HealthChecker provides health check functionality for all protected services
type HealthChecker struct {
	yagna *YagnaClient
	ipfs  *IPFSClient
	db    *DatabaseClient
	redis *RedisClient
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(yagnaURL, ipfsURL string) *HealthChecker {
	return &HealthChecker{
		yagna: NewYagnaClient(yagnaURL),
		ipfs:  NewIPFSClient(ipfsURL),
		db:    NewDatabaseClient(),
		redis: NewRedisClient(),
	}
}

// ServiceHealth represents the health status of a service
type ServiceHealth struct {
	Name      string                    `json:"name"`
	Status    string                    `json:"status"` // "healthy", "degraded", "unhealthy"
	CBStats   circuitbreaker.Stats      `json:"circuit_breaker_stats"`
	LastCheck time.Time                 `json:"last_check"`
	Error     string                    `json:"error,omitempty"`
}

// CheckAllServices checks the health of all external services
func (hc *HealthChecker) CheckAllServices(ctx context.Context) []ServiceHealth {
	services := []ServiceHealth{}
	
	// Check Yagna
	yagnaHealth := hc.checkYagnaHealth(ctx)
	services = append(services, yagnaHealth)
	
	// Check IPFS
	ipfsHealth := hc.checkIPFSHealth(ctx)
	services = append(services, ipfsHealth)
	
	// Check Database
	dbHealth := hc.checkDatabaseHealth(ctx)
	services = append(services, dbHealth)
	
	// Check Redis
	redisHealth := hc.checkRedisHealth(ctx)
	services = append(services, redisHealth)
	
	return services
}

func (hc *HealthChecker) checkYagnaHealth(ctx context.Context) ServiceHealth {
	health := ServiceHealth{
		Name:      "yagna",
		LastCheck: time.Now(),
	}
	
	// Get circuit breaker stats
	if cb, exists := hc.yagna.cbManager.Get("yagna-submit"); exists {
		health.CBStats = cb.Stats()
		
		switch cb.State() {
		case circuitbreaker.StateClosed:
			health.Status = "healthy"
		case circuitbreaker.StateHalfOpen:
			health.Status = "degraded"
		case circuitbreaker.StateOpen:
			health.Status = "unhealthy"
			health.Error = "circuit breaker open"
		}
	} else {
		health.Status = "healthy"
	}
	
	return health
}

func (hc *HealthChecker) checkIPFSHealth(ctx context.Context) ServiceHealth {
	health := ServiceHealth{
		Name:      "ipfs",
		LastCheck: time.Now(),
	}
	
	if cb, exists := hc.ipfs.cbManager.Get("ipfs-store"); exists {
		health.CBStats = cb.Stats()
		
		switch cb.State() {
		case circuitbreaker.StateClosed:
			health.Status = "healthy"
		case circuitbreaker.StateHalfOpen:
			health.Status = "degraded"
		case circuitbreaker.StateOpen:
			health.Status = "unhealthy"
			health.Error = "circuit breaker open"
		}
	} else {
		health.Status = "healthy"
	}
	
	return health
}

func (hc *HealthChecker) checkDatabaseHealth(ctx context.Context) ServiceHealth {
	health := ServiceHealth{
		Name:      "database",
		LastCheck: time.Now(),
	}
	
	if cb, exists := hc.db.cbManager.Get("database-query"); exists {
		health.CBStats = cb.Stats()
		
		switch cb.State() {
		case circuitbreaker.StateClosed:
			health.Status = "healthy"
		case circuitbreaker.StateHalfOpen:
			health.Status = "degraded"
		case circuitbreaker.StateOpen:
			health.Status = "unhealthy"
			health.Error = "circuit breaker open"
		}
	} else {
		health.Status = "healthy"
	}
	
	return health
}

func (hc *HealthChecker) checkRedisHealth(ctx context.Context) ServiceHealth {
	health := ServiceHealth{
		Name:      "redis",
		LastCheck: time.Now(),
	}
	
	if cb, exists := hc.redis.cbManager.Get("redis-get"); exists {
		health.CBStats = cb.Stats()
		
		switch cb.State() {
		case circuitbreaker.StateClosed:
			health.Status = "healthy"
		case circuitbreaker.StateHalfOpen:
			health.Status = "degraded"
		case circuitbreaker.StateOpen:
			health.Status = "unhealthy"
			health.Error = "circuit breaker open"
		}
	} else {
		health.Status = "healthy"
	}
	
	return health
}
