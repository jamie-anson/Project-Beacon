package cache

import (
	"context"
	"testing"
	"time"
)

// Ensure nil receiver is safely handled for Get and Set without Redis
func TestRedisCache_NilReceiverSafety(t *testing.T) {
	var c *RedisCache
	ctx := context.Background()

	// Get on nil should return (nil, false, nil)
	b, ok, err := c.Get(ctx, "k")
	if err != nil || ok || b != nil {
		t.Fatalf("nil Get unexpected: b=%v ok=%v err=%v", b, ok, err)
	}

	// Set on nil should be a no-op and not error
	if err := c.Set(ctx, "k", []byte("v"), time.Minute); err != nil {
		t.Fatalf("nil Set unexpected error: %v", err)
	}
}

// Ensure methods also handle struct with nil client
func TestRedisCache_NilClientSafety(t *testing.T) {
	c := &RedisCache{rdb: nil, pfx: "p:"}
	ctx := context.Background()

	b, ok, err := c.Get(ctx, "k")
	if err != nil || ok || b != nil {
		t.Fatalf("nil-client Get unexpected: b=%v ok=%v err=%v", b, ok, err)
	}
	if err := c.Set(ctx, "k", []byte("v"), time.Second); err != nil {
		t.Fatalf("nil-client Set unexpected error: %v", err)
	}
}
