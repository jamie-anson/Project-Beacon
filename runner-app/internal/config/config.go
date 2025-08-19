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
//
// Logging config is handled in internal/logging.

type Config struct {
	HTTPPort            string
	DatabaseURL         string
	RedisURL            string
	DBTimeout           time.Duration
	RedisTimeout        time.Duration
	WorkerFetchTimeout  time.Duration
	OutboxTick          time.Duration
	MaxAttempts         int
	UseMigrations       bool

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
	return &Config{
		HTTPPort:           httpPort,
		DatabaseURL:        getString("DATABASE_URL", "postgres://postgres:password@localhost:5433/beacon_runner?sslmode=disable"),
		RedisURL:           getString("REDIS_URL", "redis://localhost:6379"),
		DBTimeout:          time.Duration(getInt("DB_TIMEOUT_MS", 4000)) * time.Millisecond,
		RedisTimeout:       time.Duration(getInt("REDIS_TIMEOUT_MS", 2000)) * time.Millisecond,
		WorkerFetchTimeout: time.Duration(getInt("WORKER_FETCH_TIMEOUT_MS", 5000)) * time.Millisecond,
		OutboxTick:         time.Duration(getInt("OUTBOX_TICK_MS", 2000)) * time.Millisecond,
		MaxAttempts:        getInt("MAX_ATTEMPTS", 5),
		UseMigrations:      getBool("USE_MIGRATIONS", true),

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
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.RedisURL == "" {
		return fmt.Errorf("REDIS_URL is required")
	}
	return nil
}
