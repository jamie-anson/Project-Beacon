package config

import (
	"encoding/json"
	"encoding/base64"
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
		// Validate and build registry
		reg, vErr := validateAndBuildRegistry(entries)
		if vErr != nil {
			trustedErr = vErr
			return
		}
		trustedReg = reg
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

// validateAndBuildRegistry validates entries and constructs the registry.
func validateAndBuildRegistry(entries []TrustedKey) (*trustedRegistry, error) {
    reg := &trustedRegistry{byPubKey: map[string]TrustedKey{}, byKID: map[string]TrustedKey{}}
    seenKID := make(map[string]struct{})
    seenPK := make(map[string]struct{})

    for i, e := range entries {
        // Required fields
        if e.KID == "" {
            return nil, fmt.Errorf("trusted keys: entry %d missing kid", i)
        }
        if e.PublicKey == "" {
            return nil, fmt.Errorf("trusted keys: entry %d missing public_key (kid=%s)", i, e.KID)
        }
        // Status enum (optional defaults to active if empty elsewhere)
        if e.Status != "" && e.Status != "active" && e.Status != "revoked" {
            return nil, fmt.Errorf("trusted keys: entry %d invalid status %q (kid=%s)", i, e.Status, e.KID)
        }
        // Timestamp parse/ordering checks
        var nb, na time.Time
        var err error
        if e.NotBefore != "" {
            nb, err = time.Parse(time.RFC3339, e.NotBefore)
            if err != nil {
                return nil, fmt.Errorf("trusted keys: entry %d invalid not_before %q (kid=%s): %w", i, e.NotBefore, e.KID, err)
            }
        }
        if e.NotAfter != "" {
            na, err = time.Parse(time.RFC3339, e.NotAfter)
            if err != nil {
                return nil, fmt.Errorf("trusted keys: entry %d invalid not_after %q (kid=%s): %w", i, e.NotAfter, e.KID, err)
            }
        }
        if !nb.IsZero() && !na.IsZero() && nb.After(na) {
            return nil, fmt.Errorf("trusted keys: entry %d not_before after not_after (kid=%s)", i, e.KID)
        }
        // Base64 sanity check for public key
        if _, err := base64.StdEncoding.DecodeString(e.PublicKey); err != nil {
            if _, err2 := base64.RawStdEncoding.DecodeString(e.PublicKey); err2 != nil {
                return nil, fmt.Errorf("trusted keys: entry %d invalid base64 public_key (kid=%s): %v", i, e.KID, err)
            }
        }
        // Duplicate checks
        if _, ok := seenKID[e.KID]; ok {
            return nil, fmt.Errorf("trusted keys: duplicate kid %q", e.KID)
        }
        if _, ok := seenPK[e.PublicKey]; ok {
            return nil, fmt.Errorf("trusted keys: duplicate public_key for kid %q", e.KID)
        }
        seenKID[e.KID] = struct{}{}
        seenPK[e.PublicKey] = struct{}{}

        // Add to registry
        reg.byKID[e.KID] = e
        reg.byPubKey[e.PublicKey] = e
    }
    return reg, nil
}
