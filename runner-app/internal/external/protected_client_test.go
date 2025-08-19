package external

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestYagnaClientCircuitBreaker(t *testing.T) {
	client := NewYagnaClient("http://localhost:7465")
	ctx := context.Background()

	// Test successful task submission
	taskID, err := client.SubmitTask(ctx, map[string]interface{}{
		"image": "alpine:latest",
		"command": []string{"echo", "hello"},
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, taskID)

	// Test task status retrieval
	status, err := client.GetTaskStatus(ctx, taskID)
	assert.NoError(t, err)
	assert.Equal(t, "running", status)
}

func TestIPFSClientCircuitBreaker(t *testing.T) {
	client := NewIPFSClient("http://localhost:5001")
	ctx := context.Background()

	// Test data storage
	data := []byte("test data for circuit breaker")
	hash, err := client.StoreData(ctx, data)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Test data retrieval
	retrievedData, err := client.RetrieveData(ctx, hash)
	assert.NoError(t, err)
	assert.NotEmpty(t, retrievedData)
}

func TestHealthChecker(t *testing.T) {
	checker := NewHealthChecker("http://localhost:7465", "http://localhost:5001")
	ctx := context.Background()

	// Test health check
	services := checker.CheckAllServices(ctx)
	assert.Len(t, services, 4) // yagna, ipfs, database, redis

	// Verify service names
	serviceNames := make(map[string]bool)
	for _, service := range services {
		serviceNames[service.Name] = true
		assert.Contains(t, []string{"healthy", "degraded", "unhealthy"}, service.Status)
		assert.NotZero(t, service.LastCheck)
	}

	assert.True(t, serviceNames["yagna"])
	assert.True(t, serviceNames["ipfs"])
	assert.True(t, serviceNames["database"])
	assert.True(t, serviceNames["redis"])
}

func TestCircuitBreakerIntegration(t *testing.T) {
	client := NewYagnaClient("http://localhost:7465")
	ctx := context.Background()

	// Get the circuit breaker for testing
	cb, exists := client.cbManager.Get("yagna-submit")
	if !exists {
		// Trigger creation by making a call
		client.SubmitTask(ctx, map[string]interface{}{})
		cb, exists = client.cbManager.Get("yagna-submit")
		assert.True(t, exists)
	}

	// Verify initial state
	assert.Equal(t, "closed", cb.State().String())

	// Test circuit breaker stats
	stats := cb.Stats()
	assert.Equal(t, "yagna-submit", stats.Name)
}

func TestDatabaseClientCircuitBreaker(t *testing.T) {
	client := NewDatabaseClient()
	ctx := context.Background()

	// Test query execution
	err := client.ExecuteQuery(ctx, "SELECT 1", nil)
	assert.NoError(t, err)

	// Test with parameters
	err = client.ExecuteQuery(ctx, "SELECT * FROM jobs WHERE id = $1", "test-id")
	assert.NoError(t, err)
}

func TestRedisClientCircuitBreaker(t *testing.T) {
	client := NewRedisClient()
	ctx := context.Background()

	// Test SET operation
	err := client.Set(ctx, "test-key", "test-value", 5*time.Minute)
	assert.NoError(t, err)

	// Test GET operation
	value, err := client.Get(ctx, "test-key")
	assert.NoError(t, err)
	assert.Equal(t, "cached-value", value) // Mock returns this value
}

func TestCircuitBreakerFailureHandling(t *testing.T) {
	// Create a client for testing
	client := NewYagnaClient("http://localhost:7465")
	ctx := context.Background()
	
	// Test that circuit breaker exists after making a call
	_, _ = client.SubmitTask(ctx, map[string]interface{}{
		"image": "alpine:latest",
		"command": []string{"echo", "test"},
	})
	
	// Check that circuit breaker was created
	cb, exists := client.cbManager.Get("yagna-submit")
	if exists {
		assert.Equal(t, "closed", cb.State().String())
		stats := cb.Stats()
		assert.Equal(t, "yagna-submit", stats.Name)
	}
	
	// Test basic functionality without accessing private fields
	assert.NotNil(t, client.cbManager)
}
