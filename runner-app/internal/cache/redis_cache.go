package cache

import (
	"context"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache defines minimal cache contract
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

// RedisCache implements Cache using Redis
type RedisCache struct {
	rdb *redis.Client
	pfx string
}

func NewRedisCacheFromEnv(prefix string) (*RedisCache, error) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		url = "redis://localhost:6379"
	}
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	cl := redis.NewClient(opt)
	return &RedisCache{rdb: cl, pfx: prefix}, nil
}

func (c *RedisCache) key(k string) string { return c.pfx + k }

func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if c == nil || c.rdb == nil {
		return nil, false, nil
	}
	res, err := c.rdb.Get(ctx, c.key(key)).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return res, true, nil
}

func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Set(ctx, c.key(key), value, ttl).Err()
}
