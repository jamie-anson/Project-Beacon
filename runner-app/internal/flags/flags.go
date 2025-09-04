package flags

import (
	"encoding/json"
	"os"
	"sync"
)

// Flags holds runtime-togglable feature flags.
// Defaults are conservative; can be overridden via env or admin endpoint.
type Flags struct {
	EnableCache      bool `json:"enable_cache"`
	EnableWebSockets bool `json:"enable_websockets"`
	ReadOnlyMode     bool `json:"read_only_mode"`
}

var (
	current Flags
	mu      sync.RWMutex
)

func init() {
	// Initialize from environment with sensible defaults
	current = Flags{
		EnableCache:      getBool("ENABLE_CACHE", true),
		EnableWebSockets: getBool("ENABLE_WEBSOCKETS", true),
		ReadOnlyMode:     getBool("READ_ONLY_MODE", false),
	}
}

func getBool(k string, def bool) bool {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	if v == "1" || v == "true" || v == "TRUE" || v == "True" {
		return true
	}
	return false
}

// Get returns a copy of the current flags snapshot.
func Get() Flags {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// Set replaces the current flags with the provided value.
func Set(f Flags) {
	mu.Lock()
	defer mu.Unlock()
	current = f
}

// UpdateFromJSON merges provided JSON bytes into current flags.
func UpdateFromJSON(b []byte) error {
	mu.Lock()
	defer mu.Unlock()
	var incoming map[string]any
	if err := json.Unmarshal(b, &incoming); err != nil {
		return err
	}
	// Merge known keys only
	if v, ok := incoming["enable_cache"].(bool); ok {
		current.EnableCache = v
	}
	if v, ok := incoming["enable_websockets"].(bool); ok {
		current.EnableWebSockets = v
	}
	if v, ok := incoming["read_only_mode"].(bool); ok {
		current.ReadOnlyMode = v
	}
	return nil
}
