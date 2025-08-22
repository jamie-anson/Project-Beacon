package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func withTempTrustedFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "trusted-keys-*.json")
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}

func TestGetTrustedKeys_EmptyWhenNoEnv(t *testing.T) {
	ResetTrustedKeysCache()
	os.Unsetenv("TRUSTED_KEYS_FILE")
	r, err := GetTrustedKeys()
	require.NoError(t, err)
	require.NotNil(t, r)
}

func TestGetTrustedKeys_LoadsAndEvaluates(t *testing.T) {
	ResetTrustedKeysCache()
	json := `[
	  {"kid":"k1","public_key":"PUB1","status":"active","not_before":"2025-08-01T00:00:00Z","not_after":"2025-12-01T00:00:00Z"},
	  {"kid":"k2","public_key":"PUB2","status":"revoked"}
	]`
	path := withTempTrustedFile(t, json)
	defer os.Remove(path)
	os.Setenv("TRUSTED_KEYS_FILE", path)
	// reset singletons by reloading process-wise is non-trivial; rely on first-call semantics in this test process
	r, err := GetTrustedKeys()
	require.NoError(t, err)
	require.NotNil(t, r)

	entry1 := r.ByPublicKey("PUB1")
	require.NotNil(t, entry1)
	status, _ := EvaluateKeyTrust(entry1, time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC))
	require.Equal(t, "trusted", status)
	status, _ = EvaluateKeyTrust(entry1, time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC))
	require.Equal(t, "not_yet_valid", status)
	status, _ = EvaluateKeyTrust(entry1, time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC))
	require.Equal(t, "expired", status)

	entry2 := r.ByPublicKey("PUB2")
	require.NotNil(t, entry2)
	status, _ = EvaluateKeyTrust(entry2, time.Now().UTC())
	require.Equal(t, "revoked", status)

	entry3 := r.ByPublicKey("UNKNOWN")
	require.Nil(t, entry3)
	status, _ = EvaluateKeyTrust(entry3, time.Now().UTC())
	require.Equal(t, "unknown", status)
}
