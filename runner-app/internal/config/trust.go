package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

// TrustedKey represents a public key entry in the allowlist.
// JSON format (stored on disk):
// [
//   {"kid":"main-2025-q3","public_key":"<base64-ed25519>","status":"active","not_before":"2025-08-01T00:00:00Z","not_after":"2026-08-01T00:00:00Z"}
// ]
// Only JSON is supported (no external deps). YAML can be added later if needed.

type TrustedKey struct {
	KID       string `json:"kid"`
	PublicKey string `json:"public_key"`
	Status    string `json:"status"` // active | revoked
	NotBefore string `json:"not_before,omitempty"`
	NotAfter  string `json:"not_after,omitempty"`
}

type trustedRegistry struct {
	byPubKey map[string]TrustedKey
	byKID    map[string]TrustedKey
}

var (
	trustedOnce sync.Once
	trustedReg  *trustedRegistry
	trustedErr  error
)

// GetTrustedKeys returns a cached trusted keys registry loaded from TRUSTED_KEYS_FILE.
// If TRUSTED_KEYS_FILE is empty or file missing, returns an empty registry and nil error.
func GetTrustedKeys() (*trustedRegistry, error) {
	trustedOnce.Do(func() {
		trustedReg = &trustedRegistry{byPubKey: map[string]TrustedKey{}, byKID: map[string]TrustedKey{}}
		path := os.Getenv("TRUSTED_KEYS_FILE")
		if path == "" {
			return
		}
		b, err := os.ReadFile(path)
		if err != nil {
			// If file not found, treat as empty set
			if errors.Is(err, os.ErrNotExist) {
				return
			}
			trustedErr = fmt.Errorf("trusted keys: read file: %w", err)
			return
		}
		var entries []TrustedKey
		if err := json.Unmarshal(b, &entries); err != nil {
			trustedErr = fmt.Errorf("trusted keys: parse json: %w", err)
			return
		}
		for _, e := range entries {
			if e.PublicKey != "" {
				trustedReg.byPubKey[e.PublicKey] = e
			}
			if e.KID != "" {
				trustedReg.byKID[e.KID] = e
			}
		}
	})
	return trustedReg, trustedErr
}

// ResetTrustedKeysCache clears the cached registry and forces a reload on next call.
// Intended for use in tests only.
func ResetTrustedKeysCache() {
	trustedReg = nil
	trustedErr = nil
	trustedOnce = sync.Once{}
}

// EvaluateKeyTrust returns a status string and an optional reason.
// Possible statuses: "trusted", "revoked", "expired", "not_yet_valid", "unknown".
func EvaluateKeyTrust(entry *TrustedKey, now time.Time) (string, string) {
	if entry == nil {
		return "unknown", "no match in allowlist"
	}
	if entry.Status == "revoked" {
		return "revoked", "explicitly revoked"
	}
	if entry.NotBefore != "" {
		if t, err := time.Parse(time.RFC3339, entry.NotBefore); err == nil {
			if now.Before(t) {
				return "not_yet_valid", "before not_before"
			}
		}
	}
	if entry.NotAfter != "" {
		if t, err := time.Parse(time.RFC3339, entry.NotAfter); err == nil {
			if now.After(t) {
				return "expired", "after not_after"
			}
		}
	}
	return "trusted", "active"
}

// Lookup by public key string (base64-encoded)
func (r *trustedRegistry) ByPublicKey(pub string) *TrustedKey {
	if r == nil { return nil }
	if v, ok := r.byPubKey[pub]; ok { return &v }
	return nil
}

// Lookup by key id
func (r *trustedRegistry) ByKID(kid string) *TrustedKey {
	if r == nil { return nil }
	if v, ok := r.byKID[kid]; ok { return &v }
	return nil
}
