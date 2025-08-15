package queue

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps a Redis client for simple queue operations
type Client struct {
	redis *redis.Client
}

// Ping checks connectivity to Redis
func (c *Client) Ping(ctx context.Context) error {
    if c == nil || c.redis == nil {
        return nil
    }
    return c.redis.Ping(ctx).Err()
}

// Close closes the underlying Redis client
func (c *Client) Close() error {
    if c == nil || c.redis == nil {
        return nil
    }
    return c.redis.Close()
}

// NewFromEnv initializes a Redis client using REDIS_URL env var or defaults
func NewFromEnv() (*Client, error) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		url = "redis://localhost:6379"
	}
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	cl := redis.NewClient(opt)
	return &Client{redis: cl}, nil
}

// MustNewFromEnv panics on error; suited for worker startup
func MustNewFromEnv() *Client {
	c, err := NewFromEnv()
	if err != nil {
		log.Fatalf("failed to init redis: %v", err)
	}
	return c
}

// Enqueue pushes a payload onto a list (acts as a FIFO queue)
func (c *Client) Enqueue(ctx context.Context, queue string, payload []byte) error {
	return c.redis.RPush(ctx, queue, payload).Err()
}

// StartWorker starts a blocking consumer loop on a queue using BRPOP
// handler should return nil on success; non-nil errors are logged and message is dropped for now.
func (c *Client) StartWorker(ctx context.Context, queue string, handler func([]byte) error) {
	log.Printf("queue worker started for '%s'", queue)
	for {
		select {
		case <-ctx.Done():
			log.Printf("queue worker stopping for '%s'", queue)
			return
		default:
		}

		// Block for up to 5 seconds waiting for an item
		res, err := c.redis.BRPop(ctx, 5*time.Second, queue).Result()
		if err == redis.Nil {
			continue // timeout, loop again
		}
		if err != nil {
			// transient error
			log.Printf("queue BRPOP error: %v", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if len(res) != 2 {
			continue
		}
		payload := []byte(res[1])
		if err := handler(payload); err != nil {
			log.Printf("queue handler error: %v", err)
		}
	}
}
