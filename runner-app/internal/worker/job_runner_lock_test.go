package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRedisLockPreventsDoubleProcessing verifies that Redis lock prevents duplicate job processing
func TestRedisLockPreventsDoubleProcessing(t *testing.T) {
	// Skip if no Redis available
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping test")
	}
	defer client.Close()

	// Test job envelope
	jobID := fmt.Sprintf("test-lock-%d", time.Now().UnixNano())
	envelope := map[string]interface{}{
		"id":          jobID,
		"enqueued_at": time.Now().Format(time.RFC3339),
		"attempt":     0,
	}
	envelopeJSON, _ := json.Marshal(envelope)

	// Enqueue the same job twice
	testQueue := "test:lock:queue"
	err := client.LPush(ctx, testQueue, envelopeJSON).Err()
	require.NoError(t, err)
	err = client.LPush(ctx, testQueue, envelopeJSON).Err()
	require.NoError(t, err)

	// Track processing attempts
	var processCount int
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Simulate two workers trying to process the same job
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Dequeue job
			result, err := client.BRPop(ctx, 1*time.Second, testQueue).Result()
			if err != nil {
				return // Queue empty
			}

			var env map[string]interface{}
			json.Unmarshal([]byte(result[1]), &env)
			jobIDFromQueue := env["id"].(string)

			// Try to acquire lock (simulating JobRunner logic)
			lockKey := fmt.Sprintf("job:processing:%s", jobIDFromQueue)
			lockTTL := 15 * time.Minute

			acquired, err := client.SetNX(ctx, lockKey, "1", lockTTL).Result()
			if err != nil {
				t.Logf("Worker %d: Failed to acquire lock: %v", workerID, err)
				return
			}

			if !acquired {
				t.Logf("Worker %d: Lock already held, skipping job %s", workerID, jobIDFromQueue)
				return
			}

			t.Logf("Worker %d: ✅ Acquired lock for job %s", workerID, jobIDFromQueue)

			// Simulate processing
			time.Sleep(100 * time.Millisecond)

			mu.Lock()
			processCount++
			mu.Unlock()

			// Release lock
			client.Del(ctx, lockKey)
			t.Logf("Worker %d: Released lock for job %s", workerID, jobIDFromQueue)
		}(i)
	}

	wg.Wait()

	// Assert: Only one worker should have processed the job
	assert.Equal(t, 1, processCount, "Expected exactly 1 job processing, got %d", processCount)

	// Cleanup
	client.Del(ctx, testQueue)
}

// TestRedisLockRaceCondition tests lock behavior under high concurrency
func TestRedisLockRaceCondition(t *testing.T) {
	// Skip if no Redis available
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping test")
	}
	defer client.Close()

	jobID := fmt.Sprintf("test-race-%d", time.Now().UnixNano())
	lockKey := fmt.Sprintf("job:processing:%s", jobID)
	lockTTL := 15 * time.Minute

	// Track successful lock acquisitions
	var successCount int
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Simulate 10 workers trying to acquire lock simultaneously
	workerCount := 10
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Try to acquire lock
			acquired, err := client.SetNX(ctx, lockKey, fmt.Sprintf("worker-%d", workerID), lockTTL).Result()
			if err != nil {
				t.Logf("Worker %d: Error acquiring lock: %v", workerID, err)
				return
			}

			if acquired {
				mu.Lock()
				successCount++
				mu.Unlock()
				t.Logf("Worker %d: ✅ Acquired lock", workerID)
				
				// Hold lock briefly
				time.Sleep(50 * time.Millisecond)
			} else {
				t.Logf("Worker %d: ❌ Lock already held", workerID)
			}
		}(i)
	}

	wg.Wait()

	// Assert: Only ONE worker should have acquired the lock
	assert.Equal(t, 1, successCount, "Expected exactly 1 successful lock acquisition, got %d", successCount)

	// Cleanup
	client.Del(ctx, lockKey)
}

// TestRedisLockExpiration tests that locks expire correctly
func TestRedisLockExpiration(t *testing.T) {
	// Skip if no Redis available
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping test")
	}
	defer client.Close()

	jobID := fmt.Sprintf("test-expire-%d", time.Now().UnixNano())
	lockKey := fmt.Sprintf("job:processing:%s", jobID)
	lockTTL := 2 * time.Second // Short TTL for testing

	// Acquire lock
	acquired, err := client.SetNX(ctx, lockKey, "1", lockTTL).Result()
	require.NoError(t, err)
	require.True(t, acquired, "Should acquire lock initially")

	// Try to acquire again immediately - should fail
	acquired2, err := client.SetNX(ctx, lockKey, "1", lockTTL).Result()
	require.NoError(t, err)
	assert.False(t, acquired2, "Should not acquire lock while held")

	// Wait for lock to expire
	t.Logf("Waiting for lock to expire (TTL: %v)", lockTTL)
	time.Sleep(lockTTL + 500*time.Millisecond)

	// Try to acquire again after expiration - should succeed
	acquired3, err := client.SetNX(ctx, lockKey, "1", lockTTL).Result()
	require.NoError(t, err)
	assert.True(t, acquired3, "Should acquire lock after expiration")

	// Cleanup
	client.Del(ctx, lockKey)
}

// TestRedisLockVisibility logs lock acquisition timing for visibility
func TestRedisLockVisibility(t *testing.T) {
	// Skip if no Redis available
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping test")
	}
	defer client.Close()

	jobID := fmt.Sprintf("test-visibility-%d", time.Now().UnixNano())
	lockKey := fmt.Sprintf("job:processing:%s", jobID)
	lockTTL := 15 * time.Minute

	// Measure lock acquisition timing
	start := time.Now()
	acquired, err := client.SetNX(ctx, lockKey, "1", lockTTL).Result()
	duration := time.Since(start)

	require.NoError(t, err)
	require.True(t, acquired)

	t.Logf("🔍 VISIBILITY: Lock acquisition took %v", duration)
	t.Logf("🔍 VISIBILITY: Lock key: %s", lockKey)
	t.Logf("🔍 VISIBILITY: Lock TTL: %v", lockTTL)

	// Check lock exists
	exists, err := client.Exists(ctx, lockKey).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), exists, "Lock should exist")

	// Get TTL
	ttl, err := client.TTL(ctx, lockKey).Result()
	require.NoError(t, err)
	t.Logf("🔍 VISIBILITY: Remaining TTL: %v", ttl)

	// Cleanup
	delStart := time.Now()
	client.Del(ctx, lockKey)
	delDuration := time.Since(delStart)
	t.Logf("🔍 VISIBILITY: Lock deletion took %v", delDuration)
}
