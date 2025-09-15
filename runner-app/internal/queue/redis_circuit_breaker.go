package queue

import (
	"context"
	"log"
	"net"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jamie-anson/project-beacon-runner/internal/circuitbreaker"
)

// RedisCircuitBreaker wraps Redis client with circuit breaker protection
type RedisCircuitBreaker struct {
	client  *redis.Client
	breaker *circuitbreaker.CircuitBreaker
}

// NewRedisCircuitBreaker creates a new Redis client with circuit breaker protection
func NewRedisCircuitBreaker(client *redis.Client, name string) *RedisCircuitBreaker {
	config := circuitbreaker.Config{
		Name:             name,
		MaxFailures:      3, // Open after 3 consecutive failures
		Timeout:          10 * time.Second, // Try again after 10 seconds
		MaxRequests:      2, // Allow 2 requests in half-open state
		SuccessThreshold: 2, // Need 2 successes to close
		IsFailure: func(err error) bool {
			if err == nil {
				return false
			}
			
			// Don't count context cancellation as failures
			if err == context.Canceled || err == context.DeadlineExceeded {
				return false
			}
			
			// Don't count Redis nil results as failures
			if err == redis.Nil {
				return false
			}
			
			// Count network errors and connection failures as failures
			if isNetworkError(err) {
				return true
			}
			
			// Count Redis connection errors as failures
			if strings.Contains(err.Error(), "connection refused") ||
				strings.Contains(err.Error(), "no route to host") ||
				strings.Contains(err.Error(), "timeout") ||
				strings.Contains(err.Error(), "broken pipe") {
				return true
			}
			
			return true
		},
	}
	
	return &RedisCircuitBreaker{
		client:  client,
		breaker: circuitbreaker.New(config),
	}
}

// isNetworkError checks if the error is a network-related error
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	
	// Check for net.Error interface
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout() || netErr.Temporary()
	}
	
	// Check for common network error types
	if _, ok := err.(*net.OpError); ok {
		return true
	}
	
	if _, ok := err.(*net.DNSError); ok {
		return true
	}
	
	return false
}

// LPush executes LPUSH with circuit breaker protection
func (rcb *RedisCircuitBreaker) LPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	var result *redis.IntCmd
	
	err := rcb.breaker.Execute(ctx, func(ctx context.Context) error {
		result = rcb.client.LPush(ctx, key, values...)
		return result.Err()
	})
	
	if err != nil {
		// Return a failed command if circuit breaker prevented execution
		result = redis.NewIntCmd(ctx, "lpush", key)
		result.SetErr(err)
	}
	
	return result
}

// BRPop executes BRPOP with circuit breaker protection
func (rcb *RedisCircuitBreaker) BRPop(ctx context.Context, timeout time.Duration, keys ...string) *redis.StringSliceCmd {
	var result *redis.StringSliceCmd
	
	err := rcb.breaker.Execute(ctx, func(ctx context.Context) error {
		result = rcb.client.BRPop(ctx, timeout, keys...)
		return result.Err()
	})
	
	if err != nil {
		result = redis.NewStringSliceCmd(ctx, "brpop")
		result.SetErr(err)
	}
	
	return result
}

// ZAdd executes ZADD with circuit breaker protection
func (rcb *RedisCircuitBreaker) ZAdd(ctx context.Context, key string, members ...*redis.Z) *redis.IntCmd {
	var result *redis.IntCmd
	
	err := rcb.breaker.Execute(ctx, func(ctx context.Context) error {
		result = rcb.client.ZAdd(ctx, key, members...)
		return result.Err()
	})
	
	if err != nil {
		result = redis.NewIntCmd(ctx, "zadd", key)
		result.SetErr(err)
	}
	
	return result
}

// ZRem executes ZREM with circuit breaker protection
func (rcb *RedisCircuitBreaker) ZRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	var result *redis.IntCmd
	
	err := rcb.breaker.Execute(ctx, func(ctx context.Context) error {
		result = rcb.client.ZRem(ctx, key, members...)
		return result.Err()
	})
	
	if err != nil {
		result = redis.NewIntCmd(ctx, "zrem", key)
		result.SetErr(err)
	}
	
	return result
}

// Del executes DEL with circuit breaker protection
func (rcb *RedisCircuitBreaker) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	var result *redis.IntCmd
	
	err := rcb.breaker.Execute(ctx, func(ctx context.Context) error {
		result = rcb.client.Del(ctx, keys...)
		return result.Err()
	})
	
	if err != nil {
		result = redis.NewIntCmd(ctx, "del")
		result.SetErr(err)
	}
	
	return result
}

// ZRangeByScore executes ZRANGEBYSCORE with circuit breaker protection
func (rcb *RedisCircuitBreaker) ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) *redis.StringSliceCmd {
	var result *redis.StringSliceCmd
	
	err := rcb.breaker.Execute(ctx, func(ctx context.Context) error {
		result = rcb.client.ZRangeByScore(ctx, key, opt)
		return result.Err()
	})
	
	if err != nil {
		result = redis.NewStringSliceCmd(ctx, "zrangebyscore", key)
		result.SetErr(err)
	}
	
	return result
}

// Ping executes PING with circuit breaker protection
func (rcb *RedisCircuitBreaker) Ping(ctx context.Context) *redis.StatusCmd {
	var result *redis.StatusCmd
	
	err := rcb.breaker.Execute(ctx, func(ctx context.Context) error {
		result = rcb.client.Ping(ctx)
		return result.Err()
	})
	
	if err != nil {
		result = redis.NewStatusCmd(ctx, "ping")
		result.SetErr(err)
	}
	
	return result
}

// Stats returns circuit breaker statistics
func (rcb *RedisCircuitBreaker) Stats() circuitbreaker.Stats {
	return rcb.breaker.Stats()
}

// State returns the current circuit breaker state
func (rcb *RedisCircuitBreaker) State() circuitbreaker.State {
	return rcb.breaker.State()
}

// LogStats logs current circuit breaker statistics
func (rcb *RedisCircuitBreaker) LogStats() {
	stats := rcb.Stats()
	log.Printf("Redis Circuit Breaker Stats: %s", stats.String())
}
