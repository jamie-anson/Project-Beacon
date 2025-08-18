package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds runtime configuration and timeouts
// Values are loaded from environment variables with sane defaults.
//
// Env vars:
//   HTTP_PORT (default :8080)
//   DB_TIMEOUT_MS (default 4000)
//   REDIS_TIMEOUT_MS (default 2000)
//   WORKER_FETCH_TIMEOUT_MS (default 5000)
//   OUTBOX_TICK_MS (default 2000)
//   MAX_ATTEMPTS (default 5)
//
// Logging config is handled in internal/logging.

type Config struct {
	HTTPPort            string
	DatabaseURL         string
	DBTimeout           time.Duration
	RedisTimeout        time.Duration
	WorkerFetchTimeout  time.Duration
	OutboxTick          time.Duration
	MaxAttempts         int
}

func Load() *Config {
	return &Config{
		HTTPPort:           getString("HTTP_PORT", ":8090"),
		DatabaseURL:        getString("DATABASE_URL", "postgres://postgres:password@localhost:5433/beacon_runner?sslmode=disable"),
		DBTimeout:          time.Duration(getInt("DB_TIMEOUT_MS", 4000)) * time.Millisecond,
		RedisTimeout:       time.Duration(getInt("REDIS_TIMEOUT_MS", 2000)) * time.Millisecond,
		WorkerFetchTimeout: time.Duration(getInt("WORKER_FETCH_TIMEOUT_MS", 5000)) * time.Millisecond,
		OutboxTick:         time.Duration(getInt("OUTBOX_TICK_MS", 2000)) * time.Millisecond,
		MaxAttempts:        getInt("MAX_ATTEMPTS", 5),
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
