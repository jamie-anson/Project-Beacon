package ipfs

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/circuitbreaker"
	apperrors "github.com/jamie-anson/project-beacon-runner/internal/errors"
)

// ConnectionPool manages a pool of IPFS HTTP connections
type ConnectionPool struct {
	baseURL     string
	pool        chan *http.Client
	maxConns    int
	connTimeout time.Duration
	idleTimeout time.Duration
	cbManager   *circuitbreaker.Manager
	mu          sync.RWMutex
	closed      bool
}

// PoolConfig configures the connection pool
type PoolConfig struct {
	BaseURL     string
	MaxConns    int
	ConnTimeout time.Duration
	IdleTimeout time.Duration
}

// DefaultPoolConfig returns sensible defaults for IPFS connection pooling
func DefaultPoolConfig(baseURL string) PoolConfig {
	return PoolConfig{
		BaseURL:     baseURL,
		MaxConns:    10,
		ConnTimeout: 30 * time.Second,
		IdleTimeout: 90 * time.Second,
	}
}

// NewConnectionPool creates a new IPFS connection pool
func NewConnectionPool(config PoolConfig) *ConnectionPool {
	pool := &ConnectionPool{
		baseURL:     config.BaseURL,
		pool:        make(chan *http.Client, config.MaxConns),
		maxConns:    config.MaxConns,
		connTimeout: config.ConnTimeout,
		idleTimeout: config.IdleTimeout,
		cbManager:   circuitbreaker.NewManager(),
	}
	
	// Pre-populate the pool with connections
	for i := 0; i < config.MaxConns; i++ {
		client := pool.createClient()
		pool.pool <- client
	}
	
	return pool
}

// createClient creates a new HTTP client optimized for IPFS
func (p *ConnectionPool) createClient() *http.Client {
	transport := &http.Transport{
		MaxIdleConns:        p.maxConns,
		MaxIdleConnsPerHost: p.maxConns,
		IdleConnTimeout:     p.idleTimeout,
		DisableCompression:  false,
		ForceAttemptHTTP2:   true,
	}
	
	return &http.Client{
		Transport: transport,
		Timeout:   p.connTimeout,
	}
}

// Get retrieves a connection from the pool
func (p *ConnectionPool) Get() (*http.Client, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, apperrors.NewInternalError("connection pool is closed")
	}
	p.mu.RUnlock()
	
	select {
	case client := <-p.pool:
		return client, nil
	case <-time.After(5 * time.Second):
		// If pool is empty, create a new client
		return p.createClient(), nil
	}
}

// Put returns a connection to the pool
func (p *ConnectionPool) Put(client *http.Client) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return
	}
	p.mu.RUnlock()
	
	select {
	case p.pool <- client:
		// Successfully returned to pool
	default:
		// Pool is full, let the client be garbage collected
	}
}

// Close closes the connection pool and all connections
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.closed {
		return nil
	}
	
	p.closed = true
	close(p.pool)
	
	// Close all connections in the pool
	for client := range p.pool {
		if transport, ok := client.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
	}
	
	return nil
}

// Stats returns connection pool statistics
func (p *ConnectionPool) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	return PoolStats{
		MaxConnections:     p.maxConns,
		AvailableConns:     len(p.pool),
		ActiveConnections:  p.maxConns - len(p.pool),
		ConnectionTimeout:  p.connTimeout,
		IdleTimeout:        p.idleTimeout,
		Closed:            p.closed,
	}
}

// PoolStats represents connection pool statistics
type PoolStats struct {
	MaxConnections     int           `json:"max_connections"`
	AvailableConns     int           `json:"available_connections"`
	ActiveConnections  int           `json:"active_connections"`
	ConnectionTimeout  time.Duration `json:"connection_timeout"`
	IdleTimeout        time.Duration `json:"idle_timeout"`
	Closed            bool          `json:"closed"`
}

// PooledClient wraps the connection pool with IPFS-specific operations
type PooledClient struct {
	pool      *ConnectionPool
	cbManager *circuitbreaker.Manager
}

// NewPooledClient creates a new pooled IPFS client
func NewPooledClient(config PoolConfig) *PooledClient {
	return &PooledClient{
		pool:      NewConnectionPool(config),
		cbManager: circuitbreaker.NewManager(),
	}
}

// Add stores data in IPFS using a pooled connection
func (pc *PooledClient) Add(ctx context.Context, data []byte) (string, error) {
	cbConfig := circuitbreaker.Config{
		Name:             "ipfs-add",
		MaxFailures:      5,
		Timeout:          10 * time.Second,
		MaxRequests:      3,
		SuccessThreshold: 2,
		IsFailure: func(err error) bool {
			return err != nil && !apperrors.IsType(err, apperrors.ValidationError)
		},
	}
	
	cb := pc.cbManager.GetOrCreate("ipfs-add", cbConfig)
	
	var hash string
	err := cb.Execute(ctx, func(ctx context.Context) error {
		client, err := pc.pool.Get()
		if err != nil {
			return err
		}
		defer pc.pool.Put(client)
		
		// Simulate IPFS add operation
		hash = fmt.Sprintf("Qm%x", data[:min(len(data), 32)])
		return nil
	})
	
	if err != nil {
		return "", apperrors.Wrap(err, apperrors.ExternalServiceError, "IPFS add operation failed")
	}
	
	return hash, nil
}

// Get retrieves data from IPFS using a pooled connection
func (pc *PooledClient) Get(ctx context.Context, hash string) ([]byte, error) {
	cbConfig := circuitbreaker.DefaultConfig("ipfs-get")
	cb := pc.cbManager.GetOrCreate("ipfs-get", cbConfig)
	
	var data []byte
	err := cb.Execute(ctx, func(ctx context.Context) error {
		client, err := pc.pool.Get()
		if err != nil {
			return err
		}
		defer pc.pool.Put(client)
		
		// Simulate IPFS get operation
		data = []byte(fmt.Sprintf("data-for-%s", hash))
		return nil
	})
	
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ExternalServiceError, "IPFS get operation failed")
	}
	
	return data, nil
}

// Pin pins content in IPFS using a pooled connection
func (pc *PooledClient) Pin(ctx context.Context, hash string) error {
	cbConfig := circuitbreaker.DefaultConfig("ipfs-pin")
	cb := pc.cbManager.GetOrCreate("ipfs-pin", cbConfig)
	
	return cb.Execute(ctx, func(ctx context.Context) error {
		client, err := pc.pool.Get()
		if err != nil {
			return err
		}
		defer pc.pool.Put(client)
		
		// Simulate IPFS pin operation
		return nil
	})
}

// Close closes the pooled client and all connections
func (pc *PooledClient) Close() error {
	return pc.pool.Close()
}

// Stats returns both pool and circuit breaker statistics
func (pc *PooledClient) Stats() ClientStats {
	poolStats := pc.pool.Stats()
	
	// Get circuit breaker stats
	cbStats := make(map[string]circuitbreaker.Stats)
	for _, name := range []string{"ipfs-add", "ipfs-get", "ipfs-pin"} {
		if cb, exists := pc.cbManager.Get(name); exists {
			cbStats[name] = cb.Stats()
		}
	}
	
	return ClientStats{
		Pool:           poolStats,
		CircuitBreakers: cbStats,
	}
}

// ClientStats represents comprehensive client statistics
type ClientStats struct {
	Pool           PoolStats                      `json:"pool"`
	CircuitBreakers map[string]circuitbreaker.Stats `json:"circuit_breakers"`
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
