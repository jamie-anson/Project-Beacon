package security

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestRateLimiter(t *testing.T) {
	// Start mini Redis server
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()
	
	rl := NewRateLimiter(client)
	ctx := context.Background()
	
	t.Run("initial requests allowed", func(t *testing.T) {
		err := rl.CheckSignatureFailureRate(ctx, "192.168.1.1", "test-kid")
		require.NoError(t, err)
	})
	
	t.Run("IP rate limit", func(t *testing.T) {
		ip := "192.168.1.2"
		
		// Record 9 failures (under limit of 10)
		for i := 0; i < 9; i++ {
			rl.RecordSignatureFailure(ctx, ip, "")
		}
		
		// Should still be allowed
		err := rl.CheckSignatureFailureRate(ctx, ip, "")
		require.NoError(t, err)
		
		// Record one more failure (hits limit)
		rl.RecordSignatureFailure(ctx, ip, "")
		
		// Should now be rate limited
		err = rl.CheckSignatureFailureRate(ctx, ip, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "IP rate limit exceeded")
	})
	
	t.Run("KID rate limit", func(t *testing.T) {
		kid := "test-kid-2"
		ip := "192.168.1.3"
		
		// Record 4 failures (under limit of 5)
		for i := 0; i < 4; i++ {
			rl.RecordSignatureFailure(ctx, ip, kid)
		}
		
		// Should still be allowed
		err := rl.CheckSignatureFailureRate(ctx, ip, kid)
		require.NoError(t, err)
		
		// Record one more failure (hits limit)
		rl.RecordSignatureFailure(ctx, ip, kid)
		
		// Should now be rate limited
		err = rl.CheckSignatureFailureRate(ctx, ip, kid)
		require.Error(t, err)
		require.Contains(t, err.Error(), "key rate limit exceeded")
	})
	
	t.Run("different IPs independent", func(t *testing.T) {
		// Rate limit one IP
		ip1 := "192.168.1.4"
		for i := 0; i < 10; i++ {
			rl.RecordSignatureFailure(ctx, ip1, "")
		}
		
		// Different IP should still work
		ip2 := "192.168.1.5"
		err := rl.CheckSignatureFailureRate(ctx, ip2, "")
		require.NoError(t, err)
	})
}

func TestRateLimiter_NoRedis(t *testing.T) {
	rl := NewRateLimiter(nil)
	ctx := context.Background()
	
	// Should not error when Redis is unavailable
	err := rl.CheckSignatureFailureRate(ctx, "192.168.1.1", "test-kid")
	require.NoError(t, err)
	
	// Recording failures should not panic
	rl.RecordSignatureFailure(ctx, "192.168.1.1", "test-kid")
}
