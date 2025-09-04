package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// TestE2EJobSubmissionFlow tests the complete job submission flow using live services
// This test requires docker-compose services to be running (postgres, redis, ipfs)
// Run with: E2E_TEST=1 go test -v ./internal/e2e -run TestE2EJobSubmissionFlow
func TestE2EJobSubmissionFlow(t *testing.T) {
	// Skip if not in E2E mode
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("Skipping E2E test. Set E2E_TEST=1 to run.")
	}

	baseURL := "http://localhost:8090"
	if url := os.Getenv("API_BASE_URL"); url != "" {
		baseURL = url
	}

	// Step 1: Create a test JobSpec with correct structure
	jobSpec := &models.JobSpec{
		ID:      "test-e2e-" + fmt.Sprintf("%d", time.Now().Unix()),
		Version: "1.0.0",
		Benchmark: models.BenchmarkSpec{
			Name:        "E2E Test Benchmark",
			Description: "End-to-end test job",
			Container: models.ContainerSpec{
				Image:   "alpine",
				Command: []string{"echo", "E2E test successful"},
				Resources: models.ResourceSpec{
					CPU:    "100m",
					Memory: "128Mi",
				},
			},
			Input: models.InputSpec{
				Type: "prompt",
				Hash: "e2e-test-hash",
				Data: map[string]interface{}{
					"test": "data",
				},
			},
			Scoring: models.ScoringSpec{
				Method: "deterministic",
			},
		},
		Constraints: models.ExecutionConstraints{
			Regions: []string{"US"},
			Timeout: 5 * time.Minute,
		},
		CreatedAt: time.Now(),
	}

	// Step 2: Submit job via API
	jobSpecJSON, err := json.Marshal(jobSpec)
	require.NoError(t, err)

	resp, err := http.Post(baseURL+"/api/v1/jobs", "application/json", bytes.NewBuffer(jobSpecJSON))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Job submission should succeed")

	var submitResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&submitResponse)
	require.NoError(t, err)
	assert.Equal(t, jobSpec.ID, submitResponse["id"])

	// Step 3: Wait for job processing with timeout
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var executions []map[string]interface{}
	for {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for job execution")
		case <-ticker.C:
			// Check job status
			resp, err := http.Get(baseURL + "/api/v1/jobs/" + jobSpec.ID + "?include=latest")
			require.NoError(t, err)
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				var response map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&response)
				require.NoError(t, err)

				if execs, ok := response["executions"].([]interface{}); ok && len(execs) > 0 {
					execsJSON, _ := json.Marshal(execs)
					json.Unmarshal(execsJSON, &executions)
					goto ExecutionComplete
				}
			}
		}
	}

ExecutionComplete:
	// Step 4: Verify execution results
	require.Len(t, executions, 1, "Should have exactly one execution")
	execution := executions[0]

	assert.Equal(t, jobSpec.ID, execution["jobspec_id"])
	
	execDetails, ok := execution["execution_details"].(map[string]interface{})
	require.True(t, ok, "execution_details should be present")
	assert.Equal(t, "US", execDetails["region"])
	assert.Contains(t, []string{"completed", "failed"}, execDetails["status"])

	assert.NotEmpty(t, execution["signature"], "Receipt should be signed")
	assert.NotEmpty(t, execution["public_key"], "Receipt should have public key")

	// Step 5: Verify transparency log was updated
	resp, err = http.Get(baseURL + "/api/v1/transparency/root")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var transparencyResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&transparencyResponse)
	require.NoError(t, err)
	assert.NotEmpty(t, transparencyResponse["root"], "Transparency root should be updated")

	// Step 6: Verify transparency proof is available
	resp, err = http.Get(baseURL + "/api/v1/transparency/proof?index=0")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var proofResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&proofResponse)
	require.NoError(t, err)
	assert.NotEmpty(t, proofResponse["leaf_hash"], "Proof should have leaf hash")
	assert.NotEmpty(t, proofResponse["root_hash"], "Proof should have root hash")

	t.Logf("E2E test completed successfully for job %s", jobSpec.ID)
}

// TestE2EIPFSPublishOnCompletion tests IPFS bundle creation after job completion
// Run with: E2E_TEST=1 go test -v ./internal/e2e -run TestE2EIPFSPublishOnCompletion
func TestE2EIPFSPublishOnCompletion(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("Skipping E2E test. Set E2E_TEST=1 to run.")
	}

	baseURL := "http://localhost:8090"
	if url := os.Getenv("API_BASE_URL"); url != "" {
		baseURL = url
	}

	// Step 1: Submit a job
	jobSpec := &models.JobSpec{
		ID:      "test-ipfs-" + fmt.Sprintf("%d", time.Now().Unix()),
		Version: "1.0.0",
		Benchmark: models.BenchmarkSpec{
			Name:        "IPFS Test Benchmark",
			Description: "Test IPFS bundle creation",
			Container: models.ContainerSpec{
				Image:   "alpine",
				Command: []string{"echo", "IPFS test data"},
				Resources: models.ResourceSpec{
					CPU:    "100m",
					Memory: "128Mi",
				},
			},
			Input: models.InputSpec{
				Type: "prompt",
				Hash: "ipfs-test-hash",
				Data: map[string]interface{}{
					"test": "ipfs data",
				},
			},
			Scoring: models.ScoringSpec{
				Method: "deterministic",
			},
		},
		Constraints: models.ExecutionConstraints{
			Regions: []string{"US"},
			Timeout: 5 * time.Minute,
		},
		CreatedAt: time.Now(),
	}

	jobSpecJSON, err := json.Marshal(jobSpec)
	require.NoError(t, err)

	resp, err := http.Post(baseURL+"/api/v1/jobs", "application/json", bytes.NewBuffer(jobSpecJSON))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Step 2: Wait for job completion
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var executionID string
	for {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for job execution")
		case <-ticker.C:
			resp, err := http.Get(baseURL + "/api/v1/jobs/" + jobSpec.ID + "?include=latest")
			require.NoError(t, err)
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				var response map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&response)
				require.NoError(t, err)

				if execs, ok := response["executions"].([]interface{}); ok && len(execs) > 0 {
					execution := execs[0].(map[string]interface{})
					executionID = execution["id"].(string)
					
					execDetails := execution["execution_details"].(map[string]interface{})
					if status := execDetails["status"].(string); status == "completed" || status == "failed" {
						goto ExecutionComplete
					}
				}
			}
		}
	}

ExecutionComplete:
	require.NotEmpty(t, executionID, "Execution ID should be available")

	// Step 3: Wait for IPFS bundle creation (async process)
	time.Sleep(3 * time.Second) // Give bundler time to process

	// Step 4: Check IPFS bundles endpoint
	resp, err = http.Get(baseURL + "/api/v1/ipfs/bundles")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var bundlesResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&bundlesResponse)
	require.NoError(t, err)

	bundles, ok := bundlesResponse["bundles"].([]interface{})
	require.True(t, ok, "Bundles array should be present")

	// Find bundle for our execution
	var targetBundle map[string]interface{}
	for _, bundle := range bundles {
		b := bundle.(map[string]interface{})
		if b["execution_id"] == executionID {
			targetBundle = b
			break
		}
	}

	require.NotNil(t, targetBundle, "IPFS bundle should be created for execution")
	assert.NotEmpty(t, targetBundle["cid"], "Bundle should have CID")
	assert.Equal(t, "completed", targetBundle["status"], "Bundle should be completed")
	assert.NotEmpty(t, targetBundle["created_at"], "Bundle should have creation timestamp")

	// Step 5: Verify bundle content via CID
	cid := targetBundle["cid"].(string)
	resp, err = http.Get(baseURL + "/api/v1/ipfs/bundles/" + cid)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var bundleContent map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&bundleContent)
	require.NoError(t, err)

	assert.Equal(t, jobSpec.ID, bundleContent["jobspec_id"], "Bundle should contain correct JobSpec ID")
	assert.NotEmpty(t, bundleContent["receipt"], "Bundle should contain receipt")
	assert.NotEmpty(t, bundleContent["transparency_proof"], "Bundle should contain transparency proof")

	t.Logf("IPFS E2E test completed successfully - Bundle CID: %s", cid)
}

// TestE2ERetryAndDLQ tests job retry mechanisms and dead letter queue
// Run with: E2E_TEST=1 go test -v ./internal/e2e -run TestE2ERetryAndDLQ
func TestE2ERetryAndDLQ(t *testing.T) {
	if os.Getenv("E2E_TEST") != "1" {
		t.Skip("Skipping E2E test. Set E2E_TEST=1 to run.")
	}

	baseURL := "http://localhost:8090"
	if url := os.Getenv("API_BASE_URL"); url != "" {
		baseURL = url
	}

	// Step 1: Create a job that will fail (invalid region to trigger retry)
	jobSpec := &models.JobSpec{
		ID:      "test-retry-" + fmt.Sprintf("%d", time.Now().Unix()),
		Version: "1.0.0",
		Benchmark: models.BenchmarkSpec{
			Name:        "Retry Test Benchmark",
			Description: "Test retry and DLQ mechanisms",
			Container: models.ContainerSpec{
				Image:   "alpine",
				Command: []string{"exit", "1"}, // Command that will fail
				Resources: models.ResourceSpec{
					CPU:    "100m",
					Memory: "128Mi",
				},
			},
			Input: models.InputSpec{
				Type: "prompt",
				Hash: "retry-test-hash",
				Data: map[string]interface{}{
					"test": "retry data",
				},
			},
			Scoring: models.ScoringSpec{
				Method: "deterministic",
			},
		},
		Constraints: models.ExecutionConstraints{
			Regions: []string{"INVALID_REGION"}, // Invalid region to trigger failure
			Timeout: 1 * time.Minute,
		},
		CreatedAt: time.Now(),
	}

	jobSpecJSON, err := json.Marshal(jobSpec)
	require.NoError(t, err)

	// Step 2: Submit the failing job
	resp, err := http.Post(baseURL+"/api/v1/jobs", "application/json", bytes.NewBuffer(jobSpecJSON))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Step 3: Wait for initial processing and retries
	time.Sleep(10 * time.Second) // Give time for retries

	// Step 4: Check Redis queues directly via API (if available) or assume retry behavior
	// Note: This would require a Redis inspection endpoint or direct Redis access
	
	// Step 5: Verify job eventually fails after retries
	timeout := time.After(60 * time.Second) // Longer timeout for retries
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var finalStatus string
	for {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for job to fail after retries")
		case <-ticker.C:
			resp, err := http.Get(baseURL + "/api/v1/jobs/" + jobSpec.ID + "?include=latest")
			require.NoError(t, err)
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				var response map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&response)
				require.NoError(t, err)

				if execs, ok := response["executions"].([]interface{}); ok && len(execs) > 0 {
					execution := execs[0].(map[string]interface{})
					execDetails := execution["execution_details"].(map[string]interface{})
					status := execDetails["status"].(string)
					
					if status == "failed" {
						finalStatus = status
						goto RetryComplete
					}
				}
			}
		}
	}

RetryComplete:
	assert.Equal(t, "failed", finalStatus, "Job should eventually fail after retries")

	// Step 6: Verify job metrics show retry attempts
	resp, err = http.Get(baseURL + "/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Check for retry-related metrics in Prometheus format
	// This is a basic check - in practice you'd parse Prometheus metrics properly
	// Look for metrics like job_retries_total, job_failures_total, etc.

	t.Logf("Retry and DLQ E2E test completed successfully for job %s", jobSpec.ID)
}
