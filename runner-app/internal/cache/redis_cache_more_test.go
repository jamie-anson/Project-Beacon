package cache

import (
	"os"
	"testing"
)

func TestNewRedisCacheFromEnv_InvalidURL(t *testing.T) {
	old := os.Getenv("REDIS_URL")
	t.Cleanup(func() { os.Setenv("REDIS_URL", old) })
	os.Setenv("REDIS_URL", "://bad")

	c, err := NewRedisCacheFromEnv("t:")
	if err == nil {
		t.Fatalf("expected error for invalid REDIS_URL, got cache: %#v", c)
	}
}
