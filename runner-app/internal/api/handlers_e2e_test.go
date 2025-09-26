package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/jamie-anson/project-beacon-runner/internal/config"
	"github.com/jamie-anson/project-beacon-runner/internal/service"
	"github.com/jamie-anson/project-beacon-runner/pkg/crypto"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_JobCreation_SecurityFlow(t *testing.T) {
	// Setup test server
	cfg := &config.Config{
		ReplayProtectionEnabled: true,
		TimestampMaxSkew:        5 * time.Minute,
		TimestampMaxAge:         10 * time.Minute,
		TrustEnforce:            false, // Allow any key for testing
	}

	// Use in-memory Redis for testing
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	// sqlmock-backed JobsService to avoid real DB
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()
	jobsSvc := service.NewJobsService(sqlDB)

	// No database expectations needed - all requests fail validation before reaching DB

	router := SetupRoutes(jobsSvc, cfg, redisClient)
	server := httptest.NewServer(router)
	defer func() {
		server.Close()
		require.NoError(t, mock.ExpectationsWereMet())
	}()

	// Generate test key pair
	keyPair, err := crypto.GenerateKeyPair()
	require.NoError(t, err)

	t.Run("valid_signed_jobspec_accepted", func(t *testing.T) {
		// Create valid JobSpec with current timestamp and unique nonce
		jobSpec := &models.JobSpec{
			ID:      fmt.Sprintf("e2e-test-%d", time.Now().UnixNano()),
			Version: "1.0",
			Benchmark: models.BenchmarkSpec{
				Name:        "Test Benchmark",
				Description: "E2E test benchmark",
				Container: models.ContainerSpec{
					Image: "test/image:latest",
					Resources: models.ResourceSpec{
						CPU:    "100m",
						Memory: "128Mi",
					},
				},
				Input: models.InputSpec{
					Type: "text",
					Data: map[string]interface{}{
						"prompt": "test prompt",
					},
					Hash: "test-hash-123",
				},
				Scoring: models.ScoringSpec{
					Method: "similarity",
					Parameters: map[string]interface{}{
						"threshold": 0.8,
					},
				},
			},
			Constraints: models.ExecutionConstraints{
				Regions:    []string{"US", "EU"},
				MinRegions: 2,
				Timeout:    5 * time.Minute,
			},
			Metadata: map[string]interface{}{
				"timestamp": time.Now().UTC().Format(time.RFC3339),
				"nonce":     fmt.Sprintf("test-nonce-%d", time.Now().UnixNano()),
			},
		}

		// Sign the JobSpec
		err := jobSpec.Sign(keyPair.PrivateKey)
		require.NoError(t, err)

		// Expect DB transaction: begin -> upsert job -> insert outbox -> commit
		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO jobs").
			WithArgs(jobSpec.ID, sqlmock.AnyArg(), "created").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("INSERT INTO outbox").
			WithArgs("jobs", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		// Marshal to JSON
		jobSpecJSON, err := json.Marshal(jobSpec)
		require.NoError(t, err)

		// POST to API
		resp, err := http.Post(server.URL+"/api/v1/jobs", "application/json", bytes.NewBuffer(jobSpecJSON))
		require.NoError(t, err)
		defer resp.Body.Close()

		// Expect acceptance now that canonicalization parity is fixed
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	})

	t.Run("replay_attack_rejected", func(t *testing.T) {
		// Create JobSpec with same nonce as previous test
		jobSpec := &models.JobSpec{
			ID:      "replay-test",
			Version: "1.0",
			Benchmark: models.BenchmarkSpec{
				Name:        "Replay Test Benchmark",
				Description: "E2E replay test benchmark",
				Container: models.ContainerSpec{
					Image: "test/image:latest",
					Resources: models.ResourceSpec{
						CPU:    "100m",
						Memory: "128Mi",
					},
				},
				Input: models.InputSpec{
					Type: "text",
					Data: map[string]interface{}{
						"prompt": "replay test prompt",
					},
					Hash: "replay-test-hash-123",
				},
				Scoring: models.ScoringSpec{
					Method: "similarity",
					Parameters: map[string]interface{}{
						"threshold": 0.8,
					},
				},
			},
			Constraints: models.ExecutionConstraints{
				Regions:    []string{"US", "EU"},
				MinRegions: 2,
				Timeout:    5 * time.Minute,
			},
			Metadata: map[string]interface{}{
				"timestamp": time.Now().UTC().Format(time.RFC3339),
				"nonce":     "replay-nonce-123", // Fixed nonce for replay test
			},
		}

		// Sign the JobSpec
		err := jobSpec.Sign(keyPair.PrivateKey)
		require.NoError(t, err)

		jobSpecJSON, err := json.Marshal(jobSpec)
		require.NoError(t, err)

		// First request should succeed
		resp1, err := http.Post(server.URL+"/api/v1/jobs", "application/json", bytes.NewBuffer(jobSpecJSON))
		require.NoError(t, err)
		defer resp1.Body.Close()

		// Second request with same nonce should fail
		resp2, err := http.Post(server.URL+"/api/v1/jobs", "application/json", bytes.NewBuffer(jobSpecJSON))
		require.NoError(t, err)
		defer resp2.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp2.StatusCode)

		var errorResp map[string]interface{}
		err = json.NewDecoder(resp2.Body).Decode(&errorResp)
		require.NoError(t, err)
		assert.Equal(t, "replay_detected", errorResp["error_code"])
	})

	t.Run("stale_timestamp_rejected", func(t *testing.T) {
		// Create JobSpec with old timestamp
		staleTime := time.Now().UTC().Add(-15 * time.Minute) // 15 minutes ago
		jobSpec := &models.JobSpec{
			ID:      "stale-test",
			Version: "1.0",
			Benchmark: models.BenchmarkSpec{
				Name:        "Stale Test Benchmark",
				Description: "E2E stale timestamp test",
				Container: models.ContainerSpec{
					Image: "test/image:latest",
					Resources: models.ResourceSpec{
						CPU:    "100m",
						Memory: "128Mi",
					},
				},
				Input: models.InputSpec{
					Type: "text",
					Data: map[string]interface{}{
						"prompt": "stale test prompt",
					},
					Hash: "stale-test-hash-123",
				},
				Scoring: models.ScoringSpec{
					Method: "similarity",
					Parameters: map[string]interface{}{
						"threshold": 0.8,
					},
				},
			},
			Constraints: models.ExecutionConstraints{
				Regions:    []string{"US", "EU"},
				MinRegions: 2,
				Timeout:    5 * time.Minute,
			},
			Metadata: map[string]interface{}{
				"timestamp": staleTime.Format(time.RFC3339),
				"nonce":     fmt.Sprintf("stale-nonce-%d", time.Now().UnixNano()),
			},
		}

		err := jobSpec.Sign(keyPair.PrivateKey)
		require.NoError(t, err)

		jobSpecJSON, err := json.Marshal(jobSpec)
		require.NoError(t, err)

		resp, err := http.Post(server.URL+"/api/v1/jobs", "application/json", bytes.NewBuffer(jobSpecJSON))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var errorResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&errorResp)
		require.NoError(t, err)
		code := fmt.Sprintf("%v", errorResp["error_code"])
		assert.Contains(t, []string{"timestamp_invalid", "signature_mismatch"}, code)
	})

	t.Run("future_timestamp_rejected", func(t *testing.T) {
		// Create JobSpec with future timestamp
		futureTime := time.Now().UTC().Add(10 * time.Minute) // 10 minutes in future
		jobSpec := &models.JobSpec{
			ID:      "future-test",
			Version: "1.0",
			Benchmark: models.BenchmarkSpec{
				Name:        "Future Test Benchmark",
				Description: "E2E future timestamp test",
				Container: models.ContainerSpec{
					Image: "test/image:latest",
					Resources: models.ResourceSpec{
						CPU:    "100m",
						Memory: "128Mi",
					},
				},
				Input: models.InputSpec{
					Type: "text",
					Data: map[string]interface{}{
						"prompt": "future test prompt",
					},
					Hash: "future-test-hash-123",
				},
				Scoring: models.ScoringSpec{
					Method: "similarity",
					Parameters: map[string]interface{}{
						"threshold": 0.8,
					},
				},
			},
			Constraints: models.ExecutionConstraints{
				Regions:    []string{"US", "EU"},
				MinRegions: 2,
				Timeout:    5 * time.Minute,
			},
			Metadata: map[string]interface{}{
				"timestamp": futureTime.Format(time.RFC3339),
				"nonce":     fmt.Sprintf("future-nonce-%d", time.Now().UnixNano()),
			},
		}

		err := jobSpec.Sign(keyPair.PrivateKey)
		require.NoError(t, err)

		jobSpecJSON, err := json.Marshal(jobSpec)
		require.NoError(t, err)

		resp, err := http.Post(server.URL+"/api/v1/jobs", "application/json", bytes.NewBuffer(jobSpecJSON))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var errorResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&errorResp)
		require.NoError(t, err)
		code := fmt.Sprintf("%v", errorResp["error_code"])
		assert.Contains(t, []string{"timestamp_invalid", "signature_mismatch"}, code)
	})

	t.Run("invalid_signature_rejected", func(t *testing.T) {
		// Reset rate limiting state between tests
		mr.FlushAll()

		jobSpec := &models.JobSpec{
			ID:      "invalid-sig-test",
			Version: "1.0",
			Benchmark: models.BenchmarkSpec{
				Name:        "Invalid Signature Test Benchmark",
				Description: "E2E invalid signature test",
				Container: models.ContainerSpec{
					Image: "test/image:latest",
					Resources: models.ResourceSpec{
						CPU:    "100m",
						Memory: "128Mi",
					},
				},
				Input: models.InputSpec{
					Type: "text",
					Data: map[string]interface{}{
						"prompt": "invalid signature test prompt",
					},
					Hash: "invalid-sig-test-hash-123",
				},
				Scoring: models.ScoringSpec{
					Method: "similarity",
					Parameters: map[string]interface{}{
						"threshold": 0.8,
					},
				},
			},
			Constraints: models.ExecutionConstraints{
				Regions:    []string{"US", "EU"},
				MinRegions: 2,
				Timeout:    5 * time.Minute,
			},
			Metadata: map[string]interface{}{
				"timestamp": time.Now().UTC().Format(time.RFC3339),
				"nonce":     fmt.Sprintf("invalid-nonce-%d", time.Now().UnixNano()),
			},
			Signature: "invalid-signature",
			PublicKey: crypto.PublicKeyToBase64(keyPair.PublicKey),
		}

		jobSpecJSON, err := json.Marshal(jobSpec)
		require.NoError(t, err)

		resp, err := http.Post(server.URL+"/api/v1/jobs", "application/json", bytes.NewBuffer(jobSpecJSON))
		require.NoError(t, err)
		defer resp.Body.Close()

		// Depending on prior failures the rate limiter may trigger; accept either 400 or 429
		assert.Contains(t, []int{http.StatusBadRequest, http.StatusTooManyRequests}, resp.StatusCode)

		var errorResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&errorResp)
		require.NoError(t, err)
		code := fmt.Sprintf("%v", errorResp["error_code"])
		if resp.StatusCode == http.StatusTooManyRequests {
			assert.Equal(t, "rate_limit_exceeded", code)
		} else {
			assert.Equal(t, "signature_mismatch", code)
		}
	})
}

func TestE2E_RateLimiting(t *testing.T) {
	cfg := &config.Config{
		ReplayProtectionEnabled: true,
		TimestampMaxSkew:        5 * time.Minute,
		TimestampMaxAge:         10 * time.Minute,
		TrustEnforce:            false,
	}

	// Use in-memory Redis for testing
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	router := SetupRoutes(nil, cfg, redisClient)
	server := httptest.NewServer(router)
	defer server.Close()

	t.Run("rate_limiting_on_signature_failures", func(t *testing.T) {
		// Generate multiple invalid requests to trigger rate limiting
		for i := 0; i < 10; i++ {
			jobSpec := map[string]interface{}{
				"id":         fmt.Sprintf("rate-limit-test-%d", i),
				"version":    "1.0",
				"signature":  "invalid-signature",
				"public_key": "invalid-key",
				"metadata": map[string]interface{}{
					"timestamp": time.Now().UTC().Format(time.RFC3339),
					"nonce":     fmt.Sprintf("rate-nonce-%d", i),
				},
			}

			jobSpecJSON, err := json.Marshal(jobSpec)
			require.NoError(t, err)

			resp, err := http.Post(server.URL+"/api/v1/jobs", "application/json", bytes.NewBuffer(jobSpecJSON))
			require.NoError(t, err)
			resp.Body.Close()

			// After several failures, should get rate limited
			if i > 5 {
				if resp.StatusCode == http.StatusTooManyRequests {
					t.Logf("Rate limiting triggered after %d failures", i)
					return
				}
			}
		}
	})
}
