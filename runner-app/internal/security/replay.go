package security

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ReplayProtection provides nonce-based replay protection using Redis
type ReplayProtection struct {
	client *redis.Client
	maxAge time.Duration // TTL for nonces
}

// NewReplayProtection creates a new replay protection instance
func NewReplayProtection(client *redis.Client, maxAge time.Duration) *ReplayProtection {
	return &ReplayProtection{
		client: client,
		maxAge: maxAge,
	}
}

// GenerateNonce creates a cryptographically secure 96-bit nonce
func GenerateNonce() (string, error) {
	bytes := make([]byte, 12) // 96 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// CheckAndRecordNonce verifies a nonce hasn't been used and records it
// Returns error if nonce was already used (replay attack)
func (rp *ReplayProtection) CheckAndRecordNonce(ctx context.Context, kid, nonce string) error {
	if rp.client == nil {
		// If Redis unavailable, skip replay protection (log warning elsewhere)
		return nil
	}

	key := fmt.Sprintf("nonce:%s:%s", kid, nonce)
	
	// Use SET with NX (only if not exists) and EX (expiry)
	result := rp.client.SetNX(ctx, key, "1", rp.maxAge)
	if err := result.Err(); err != nil {
		return fmt.Errorf("redis nonce check: %w", err)
	}
	
	// If SetNX returned false, key already existed (replay)
	if !result.Val() {
		return fmt.Errorf("replay detected: nonce already used")
	}
	
	return nil
}

// ValidateTimestamp checks if timestamp is within acceptable skew
func ValidateTimestamp(ts time.Time, maxSkew time.Duration, maxAge time.Duration) error {
	now := time.Now().UTC()
	
	// Check if timestamp is too far in the future
	if ts.After(now.Add(maxSkew)) {
		return fmt.Errorf("timestamp too far in future: %v > %v", ts, now.Add(maxSkew))
	}
	
	// Check if timestamp is too old
	if ts.Before(now.Add(-maxAge)) {
		return fmt.Errorf("timestamp too old: %v < %v", ts, now.Add(-maxAge))
	}
	
	return nil
}

// Exported errors for structured classification
var (
	ErrTimestampFuture = errors.New("timestamp_too_far_in_future")
	ErrTimestampOld    = errors.New("timestamp_too_old")
)

// ValidateTimestampWithReason returns a short machine-readable reason and error.
// reason values: "too_far_in_future", "too_old", or "" if valid.
func ValidateTimestampWithReason(ts time.Time, maxSkew time.Duration, maxAge time.Duration) (string, error) {
	now := time.Now().UTC()
	if ts.After(now.Add(maxSkew)) {
		return "too_far_in_future", ErrTimestampFuture
	}
	if ts.Before(now.Add(-maxAge)) {
		return "too_old", ErrTimestampOld
	}
	return "", nil
}
