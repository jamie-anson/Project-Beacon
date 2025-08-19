package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_LoadFromEnv_Success(t *testing.T) {
	// Set environment variables
	os.Setenv("HTTP_PORT", "8080")
	os.Setenv("DATABASE_URL", "postgres://test:test@localhost/testdb")
	os.Setenv("REDIS_URL", "redis://localhost:6379")
	os.Setenv("GOLEM_NETWORK", "testnet")
	os.Setenv("IPFS_URL", "http://localhost:5001")
	defer cleanupEnv()

	config := Load()
	err := config.Validate()
	require.NoError(t, err)

	assert.Equal(t, ":8080", config.HTTPPort)
	assert.Equal(t, "postgres://test:test@localhost/testdb", config.DatabaseURL)
	assert.Equal(t, "redis://localhost:6379", config.RedisURL)
	assert.Equal(t, "testnet", config.GolemNetwork)
	assert.Equal(t, "http://localhost:5001", config.IPFSURL)
}

func TestConfig_LoadFromEnv_Defaults(t *testing.T) {
	// Clear environment variables to test defaults
	cleanupEnv()

	config := Load()
	err := config.Validate()
	require.NoError(t, err)

	assert.Equal(t, ":8090", config.HTTPPort)
	assert.Equal(t, "testnet", config.GolemNetwork)
	assert.Equal(t, 4000*time.Millisecond, config.DBTimeout)
	assert.Equal(t, 2000*time.Millisecond, config.RedisTimeout)
	assert.Equal(t, 5, config.MaxAttempts)
}

func TestConfig_LoadFromEnv_RequiredFields(t *testing.T) {
    // Load() provides defaults; required-field errors should be validated on explicit configs.
    // This test ensures Validate() flags missing fields when not provided.
    t.Run("missing database url", func(t *testing.T) {
        cfg := &Config{RedisURL: "redis://localhost:6379"}
        err := cfg.Validate()
        require.Error(t, err)
        assert.Contains(t, err.Error(), "DATABASE_URL is required")
    })

    t.Run("missing redis url", func(t *testing.T) {
        cfg := &Config{DatabaseURL: "postgres://localhost/test"}
        err := cfg.Validate()
        require.Error(t, err)
        assert.Contains(t, err.Error(), "REDIS_URL is required")
    })
}

func TestConfig_Validation_Success(t *testing.T) {
	config := &Config{
		DatabaseURL: "postgres://localhost/test",
		RedisURL:    "redis://localhost:6379",
	}

	err := config.Validate()
	require.NoError(t, err)
}

func TestConfig_Validation_MissingDatabase(t *testing.T) {
	config := &Config{
		RedisURL: "redis://localhost:6379",
	}

	err := config.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_URL is required")
}

func TestConfig_Validation_MissingRedis(t *testing.T) {
	config := &Config{
		DatabaseURL: "postgres://localhost/test",
	}

	err := config.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "REDIS_URL is required")
}

func TestConfig_HTTPPortFormatting(t *testing.T) {
	// Test port formatting with colon prefix
	os.Setenv("HTTP_PORT", "8080")
	defer cleanupEnv()

	config := Load()
	assert.Equal(t, ":8080", config.HTTPPort)

	// Test port already has colon
	os.Setenv("HTTP_PORT", ":9000")
	config = Load()
	assert.Equal(t, ":9000", config.HTTPPort)
}

func TestConfig_TimeoutDefaults(t *testing.T) {
	cleanupEnv()
	
	config := Load()
	
	// Test default timeout values
	assert.Equal(t, 4000*time.Millisecond, config.DBTimeout)
	assert.Equal(t, 2000*time.Millisecond, config.RedisTimeout)
	assert.Equal(t, 5000*time.Millisecond, config.WorkerFetchTimeout)
	assert.Equal(t, 2000*time.Millisecond, config.OutboxTick)
}


// Helper functions

func cleanupEnv() {
	envVars := []string{
		"HTTP_PORT", "DATABASE_URL", "REDIS_URL", "GOLEM_NETWORK",
		"IPFS_URL", "REQUEST_TIMEOUT", "RATE_LIMIT_RPM", "ENABLE_METRICS",
		"ENV", "LOG_LEVEL",
	}
	for _, env := range envVars {
		os.Unsetenv(env)
	}
}

func createTempConfigFile(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "config-test-*.yaml")
	require.NoError(t, err)
	
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	
	err = tmpFile.Close()
	require.NoError(t, err)
	
	return tmpFile.Name()
}
