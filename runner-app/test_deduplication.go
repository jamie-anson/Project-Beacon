package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

func main() {
	ctx := context.Background()
	
	// Connect to local Redis
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer client.Close()
	
	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		fmt.Printf("❌ Failed to connect to Redis: %v\n", err)
		return
	}
	fmt.Println("✅ Connected to Redis")
	
	// Clean up any existing locks
	jobID := "test-job-123"
	lockKey := fmt.Sprintf("job:processing:%s", jobID)
	client.Del(ctx, lockKey)
	
	fmt.Println("\n🧪 Testing Deduplication Logic")
	fmt.Println("================================")
	
	// Simulate 5 workers trying to process the same job simultaneously
	numWorkers := 5
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex
	
	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			// Try to acquire lock
			lockTTL := 15 * time.Minute
			acquired, err := client.SetNX(ctx, lockKey, fmt.Sprintf("worker-%d", workerID), lockTTL).Result()
			
			if err != nil {
				fmt.Printf("❌ Worker %d: Error acquiring lock: %v\n", workerID, err)
				return
			}
			
			if !acquired {
				fmt.Printf("⏭️  Worker %d: Lock already held, skipping\n", workerID)
				return
			}
			
			// Lock acquired!
			fmt.Printf("🔒 Worker %d: Acquired lock, processing job...\n", workerID)
			mu.Lock()
			successCount++
			mu.Unlock()
			
			// Simulate job processing
			time.Sleep(100 * time.Millisecond)
			
			// Release lock
			if err := client.Del(ctx, lockKey).Err(); err != nil {
				fmt.Printf("⚠️  Worker %d: Failed to release lock: %v\n", workerID, err)
			} else {
				fmt.Printf("🔓 Worker %d: Released lock\n", workerID)
			}
		}(i)
		
		// Stagger worker starts slightly
		time.Sleep(10 * time.Millisecond)
	}
	
	wg.Wait()
	
	fmt.Println("\n📊 Results")
	fmt.Println("==========")
	fmt.Printf("Workers launched: %d\n", numWorkers)
	fmt.Printf("Jobs processed: %d\n", successCount)
	
	if successCount == 1 {
		fmt.Println("✅ SUCCESS: Only 1 worker processed the job (no duplicates)")
	} else {
		fmt.Printf("❌ FAILURE: %d workers processed the job (expected 1)\n", successCount)
	}
	
	// Test retry scenario
	fmt.Println("\n🔄 Testing Retry Scenario")
	fmt.Println("=========================")
	
	// Simulate job status check
	jobStatuses := []string{"completed", "failed", "cancelled", "processing"}
	
	for _, status := range jobStatuses {
		fmt.Printf("\nJob status: %s\n", status)
		
		switch status {
		case "completed":
			fmt.Println("  → Skip (already done)")
		case "failed", "cancelled":
			fmt.Println("  → Allow retry (no lock check)")
		case "processing":
			// Check if lock exists
			exists, _ := client.Exists(ctx, lockKey).Result()
			if exists > 0 {
				fmt.Println("  → Lock exists, skip (another worker processing)")
			} else {
				fmt.Println("  → No lock, acquire and process")
			}
		default:
			fmt.Println("  → Acquire lock and process")
		}
	}
	
	// Clean up
	client.Del(ctx, lockKey)
	fmt.Println("\n✅ Test complete!")
}
