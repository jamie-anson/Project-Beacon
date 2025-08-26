package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
	"strings"
)

// Config holds runtime configuration and timeouts
// Values are loaded from environment variables with sane defaults.
//
// Env vars:
//   HTTP_PORT (default :8090)
//   DATABASE_URL
//   REDIS_URL
//   DB_TIMEOUT_MS (default 4000)
//   REDIS_TIMEOUT_MS (default 2000)
//   WORKER_FETCH_TIMEOUT_MS (default 5000)
//   OUTBOX_TICK_MS (default 2000)
//   MAX_ATTEMPTS (default 5)
//   USE_MIGRATIONS (default true)
//   PORT_STRATEGY (default fallback)
//   PORT_RANGE (default 8090-8099)
//   RUNNER_HTTP_ADDR_FILE (default .runner-http.addr)
//   RUNNER_SIG_BYPASS (default false)
//
// Logging config is handled in internal/logging.

type Config struct {
	HTTPPort            string
	// Strategy for selecting the HTTP port: strict | fallback | ephemeral
	PortStrategy        string
	// When using fallback, range of ports to scan (inclusive)
	PortRangeStart      int
	PortRangeEnd        int
	// When using ephemeral, write the resolved addr to this file
	AddrFile            string
	// ResolvedAddr is the final bound address (set at runtime in main)
	ResolvedAddr        string
	DatabaseURL         string
	RedisURL            string
	DBTimeout           time.Duration
	RedisTimeout        time.Duration
	WorkerFetchTimeout  time.Duration
	OutboxTick          time.Duration
	MaxAttempts         int
	UseMigrations       bool

    // Signature trust enforcement
    TrustEnforce        bool
    TrustedKeysFile     string
    TrustedKeysReload   time.Duration
    // Development-only: bypass signature verification
    SigBypass           bool

    // Security settings
    TimestampMaxSkew    time.Duration
    TimestampMaxAge     time.Duration
    ReplayProtectionEnabled bool

	// Queue names
	JobsQueueName       string

	// External service URLs
	IPFSNodeURL         string
	IPFSURL             string
	IPFSGateway         string
	YagnaURL            string
	GolemAPIKey         string
	GolemNetwork        string
}

func Load() *Config {
	httpPort := getString("HTTP_PORT", ":8090")
	if httpPort != "" && !strings.HasPrefix(httpPort, ":") {
		httpPort = ":" + httpPort
	}

	// Port strategy and range
	portStrategy := getString("PORT_STRATEGY", "fallback")
	// Parse PORT_RANGE in the form "start-end"
	rangeStr := getString("PORT_RANGE", "8090-8099")
	rangeStart, rangeEnd := 8090, 8099
	if dash := strings.Index(rangeStr, "-"); dash > 0 {
		if a, err := strconv.Atoi(strings.TrimSpace(rangeStr[:dash])); err == nil {
			rangeStart = a
		}
		if b, err := strconv.Atoi(strings.TrimSpace(rangeStr[dash+1:])); err == nil {
			rangeEnd = b
		}
	}
	addrFile := getString("RUNNER_HTTP_ADDR_FILE", ".runner-http.addr")
	cfg := &Config{
		HTTPPort:           httpPort,
		PortStrategy:       portStrategy,
		PortRangeStart:     rangeStart,
		PortRangeEnd:       rangeEnd,
		AddrFile:           addrFile,
		DatabaseURL:        getString("DATABASE_URL", "postgres://postgres:password@localhost:5433/beacon_runner?sslmode=disable"),
		RedisURL:           getString("REDIS_URL", "redis://localhost:6379"),
		DBTimeout:          time.Duration(getInt("DB_TIMEOUT_MS", 4000)) * time.Millisecond,
		RedisTimeout:       time.Duration(getInt("REDIS_TIMEOUT_MS", 2000)) * time.Millisecond,
		WorkerFetchTimeout: time.Duration(getInt("WORKER_FETCH_TIMEOUT_MS", 5000)) * time.Millisecond,
		OutboxTick:         time.Duration(getInt("OUTBOX_TICK_MS", 2000)) * time.Millisecond,
		MaxAttempts:        getInt("MAX_ATTEMPTS", 5),
		UseMigrations:      getBool("USE_MIGRATIONS", true),
        TrustEnforce:       getBool("TRUST_ENFORCE", false),
        TrustedKeysFile:    getString("TRUSTED_KEYS_FILE", ""),
        TrustedKeysReload:  time.Duration(getInt("TRUSTED_KEYS_RELOAD_SECONDS", 0)) * time.Second,
        SigBypass:          getBool("RUNNER_SIG_BYPASS", false),
        TimestampMaxSkew:   time.Duration(getInt("TIMESTAMP_MAX_SKEW_MINUTES", 5)) * time.Minute,
        TimestampMaxAge:    time.Duration(getInt("TIMESTAMP_MAX_AGE_MINUTES", 10)) * time.Minute,
        ReplayProtectionEnabled: getBool("REPLAY_PROTECTION_ENABLED", true),

		// Queue names
		JobsQueueName:      getString("JOBS_QUEUE_NAME", "jobs"),
		
		// External services
		IPFSNodeURL:        getString("IPFS_NODE_URL", "http://localhost:5001"),
		IPFSURL:            getString("IPFS_URL", "http://localhost:5001"),
		IPFSGateway:        getString("IPFS_GATEWAY", "https://ipfs.io"),
		YagnaURL:           getString("YAGNA_URL", "http://localhost:7465"),
		GolemAPIKey:        getString("GOLEM_API_KEY", ""),
		GolemNetwork:       getString("GOLEM_NETWORK", "testnet"),
	}

    // In CI, hard-disable signature bypass regardless of env request
    if os.Getenv("CI") == "true" {
        cfg.SigBypass = false
    }

    return cfg
}

func getString(key, def string) string {
	if v := os.Getenv(key); v != "" { return v }
	return def
}

func getInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil { return n }
	}
	return def
}

func getBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil { return b }
	}
	return def
}

// Validate checks required configuration values
func (c *Config) Validate() error {
	// HTTP port like ":8090" with numeric component
	if c.HTTPPort == "" || c.HTTPPort[0] != ':' {
		return fmt.Errorf("HTTP_PORT must be in the form :<port>, got %q", c.HTTPPort)
	}
	if _, err := strconv.Atoi(c.HTTPPort[1:]); err != nil {
		return fmt.Errorf("HTTP_PORT must have numeric port: %v", err)
	}
	// Defaults for tests that construct Config manually
	if strings.TrimSpace(c.PortStrategy) == "" {
		c.PortStrategy = "fallback"
	}
	if c.PortRangeStart == 0 && c.PortRangeEnd == 0 {
		c.PortRangeStart, c.PortRangeEnd = 8090, 8099
	}
	// Port strategy
	switch strings.ToLower(strings.TrimSpace(c.PortStrategy)) {
	case "strict", "fallback", "ephemeral":
		// ok
	default:
		return fmt.Errorf("PORT_STRATEGY must be one of strict,fallback,ephemeral; got %q", c.PortStrategy)
	}
	// Range validation
	if c.PortRangeStart <= 0 || c.PortRangeStart > 65535 {
		return fmt.Errorf("PORT_RANGE start must be within 1-65535")
	}
	if c.PortRangeEnd <= 0 || c.PortRangeEnd > 65535 {
		return fmt.Errorf("PORT_RANGE end must be within 1-65535")
	}
	if c.PortRangeStart > c.PortRangeEnd {
		return fmt.Errorf("PORT_RANGE start must be <= end")
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.RedisURL == "" {
		return fmt.Errorf("REDIS_URL is required")
	}
	if c.DBTimeout <= 0 {
		return fmt.Errorf("DB_TIMEOUT_MS must be > 0")
	}
	if c.RedisTimeout <= 0 {
		return fmt.Errorf("REDIS_TIMEOUT_MS must be > 0")
	}
	if c.WorkerFetchTimeout <= 0 {
		return fmt.Errorf("WORKER_FETCH_TIMEOUT_MS must be > 0")
	}
	if c.OutboxTick <= 0 {
		return fmt.Errorf("OUTBOX_TICK_MS must be > 0")
	}
	if strings.TrimSpace(c.JobsQueueName) == "" {
		return fmt.Errorf("JOBS_QUEUE_NAME must be non-empty")
	}
	return nil
}
