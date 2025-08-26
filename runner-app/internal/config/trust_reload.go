package config

import (
    "context"
    "os"
    "time"
)

// ReloadTrustedKeys forces a reload of the trusted keys from the provided path.
// If path is empty, TRUSTED_KEYS_FILE env var is used.
func ReloadTrustedKeys(path string) error {
    if path != "" {
        _ = os.Setenv("TRUSTED_KEYS_FILE", path)
    }
    ResetTrustedKeysCache()
    _, err := GetTrustedKeys()
    return err
}

// StartTrustedKeysReloader periodically reloads the trusted keys file.
// Intended to be started by main with application context.
func StartTrustedKeysReloader(ctx context.Context, path string, interval time.Duration) {
    if interval <= 0 {
        interval = time.Minute
    }
    // initial load
    _ = ReloadTrustedKeys(path)
    ticker := time.NewTicker(interval)
    go func() {
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                _ = ReloadTrustedKeys(path)
            }
        }
    }()
}
