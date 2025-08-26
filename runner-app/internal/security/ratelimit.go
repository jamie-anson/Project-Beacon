package security

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter provides rate limiting for signature verification failures
type RateLimiter struct {
	client *redis.Client
}

// NewRateLimiter creates a new rate limiter instance
func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{
		client: client,
	}
}

// CheckSignatureFailureRate checks if IP/key has exceeded signature failure rate limits
// Returns error if rate limit exceeded
func (rl *RateLimiter) CheckSignatureFailureRate(ctx context.Context, clientIP, kid string) error {
	if rl.client == nil {
		// If Redis unavailable, skip rate limiting
		return nil
	}

	// Check both IP-based and KID-based rate limits
	if err := rl.checkLimit(ctx, "ip:"+clientIP, 10, time.Minute); err != nil {
		return fmt.Errorf("IP rate limit exceeded: %w", err)
	}
	
	if kid != "" {
		if err := rl.checkLimit(ctx, "kid:"+kid, 5, time.Minute); err != nil {
			return fmt.Errorf("key rate limit exceeded: %w", err)
		}
	}
	
	return nil
}

// RecordSignatureFailure records a signature verification failure for rate limiting
func (rl *RateLimiter) RecordSignatureFailure(ctx context.Context, clientIP, kid string) {
	if rl.client == nil {
		return
	}

	// Record failure for IP
	rl.increment(ctx, "ip:"+clientIP, time.Minute)
	
	// Record failure for KID if present
	if kid != "" {
		rl.increment(ctx, "kid:"+kid, time.Minute)
	}
}

// checkLimit checks if the current count exceeds the limit within the window
func (rl *RateLimiter) checkLimit(ctx context.Context, key string, limit int, window time.Duration) error {
	rateLimitKey := fmt.Sprintf("rate_limit:%s", key)
	
	count, err := rl.client.Get(ctx, rateLimitKey).Int()
	if err != nil && err != redis.Nil {
		// If Redis error, allow request (fail open)
		return nil
	}
	
	if count >= limit {
		return fmt.Errorf("rate limit exceeded: %d/%d requests in %v", count, limit, window)
	}
	
	return nil
}

// increment increments the counter for the given key with TTL
func (rl *RateLimiter) increment(ctx context.Context, key string, ttl time.Duration) {
	rateLimitKey := fmt.Sprintf("rate_limit:%s", key)
	
	pipe := rl.client.TxPipeline()
	pipe.Incr(ctx, rateLimitKey)
	pipe.Expire(ctx, rateLimitKey, ttl)
	_, _ = pipe.Exec(ctx) // Ignore errors (fail open)
}
