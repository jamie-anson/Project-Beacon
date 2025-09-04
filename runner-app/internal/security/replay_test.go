package security

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestGenerateNonce(t *testing.T) {
	nonce1, err := GenerateNonce()
	require.NoError(t, err)
	require.NotEmpty(t, nonce1)
	
	nonce2, err := GenerateNonce()
	require.NoError(t, err)
	require.NotEmpty(t, nonce2)
	
	// Should be different
	require.NotEqual(t, nonce1, nonce2)
	
	// Should be valid base64
	require.Len(t, nonce1, 16) // 12 bytes * 4/3 = 16 chars (base64)
}

func TestValidateTimestampWithReason(t *testing.T) {
    now := time.Now().UTC()
    maxSkew := 5 * time.Minute
    maxAge := 10 * time.Minute

    t.Run("valid no reason", func(t *testing.T) {
        reason, err := ValidateTimestampWithReason(now, maxSkew, maxAge)
        require.NoError(t, err)
        require.Equal(t, "", reason)
    })

    t.Run("too far in future", func(t *testing.T) {
        future := now.Add(10 * time.Minute)
        reason, err := ValidateTimestampWithReason(future, maxSkew, maxAge)
        require.Error(t, err)
        require.Equal(t, "too_far_in_future", reason)
        require.ErrorIs(t, err, ErrTimestampFuture)
    })

    t.Run("too old", func(t *testing.T) {
        past := now.Add(-15 * time.Minute)
        reason, err := ValidateTimestampWithReason(past, maxSkew, maxAge)
        require.Error(t, err)
        require.Equal(t, "too_old", reason)
        require.ErrorIs(t, err, ErrTimestampOld)
    })
}

func TestReplayProtection(t *testing.T) {
	// Start mini Redis server
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()
	
	rp := NewReplayProtection(client, 10*time.Minute)
	ctx := context.Background()
	
	t.Run("first use succeeds", func(t *testing.T) {
		err := rp.CheckAndRecordNonce(ctx, "test-kid", "nonce123")
		require.NoError(t, err)
	})
	
	t.Run("replay fails", func(t *testing.T) {
		err := rp.CheckAndRecordNonce(ctx, "test-kid", "nonce123")
		require.Error(t, err)
		require.Contains(t, err.Error(), "replay detected")
	})
	
	t.Run("different kid allows same nonce", func(t *testing.T) {
		err := rp.CheckAndRecordNonce(ctx, "other-kid", "nonce123")
		require.NoError(t, err)
	})
	
	t.Run("different nonce for same kid succeeds", func(t *testing.T) {
		err := rp.CheckAndRecordNonce(ctx, "test-kid", "nonce456")
		require.NoError(t, err)
	})
}

func TestReplayProtection_NoRedis(t *testing.T) {
	rp := NewReplayProtection(nil, 10*time.Minute)
	ctx := context.Background()
	
	// Should not error when Redis is unavailable
	err := rp.CheckAndRecordNonce(ctx, "test-kid", "nonce123")
	require.NoError(t, err)
}

func TestValidateTimestamp(t *testing.T) {
	now := time.Now().UTC()
	maxSkew := 5 * time.Minute
	maxAge := 10 * time.Minute
	
	t.Run("current timestamp valid", func(t *testing.T) {
		err := ValidateTimestamp(now, maxSkew, maxAge)
		require.NoError(t, err)
	})
	
	t.Run("slightly future timestamp valid", func(t *testing.T) {
		future := now.Add(2 * time.Minute)
		err := ValidateTimestamp(future, maxSkew, maxAge)
		require.NoError(t, err)
	})
	
	t.Run("too far future fails", func(t *testing.T) {
		future := now.Add(10 * time.Minute)
		err := ValidateTimestamp(future, maxSkew, maxAge)
		require.Error(t, err)
		require.Contains(t, err.Error(), "too far in future")
	})
	
	t.Run("slightly old timestamp valid", func(t *testing.T) {
		past := now.Add(-5 * time.Minute)
		err := ValidateTimestamp(past, maxSkew, maxAge)
		require.NoError(t, err)
	})
	
	t.Run("too old timestamp fails", func(t *testing.T) {
		past := now.Add(-15 * time.Minute)
		err := ValidateTimestamp(past, maxSkew, maxAge)
		require.Error(t, err)
		require.Contains(t, err.Error(), "too old")
	})
}
