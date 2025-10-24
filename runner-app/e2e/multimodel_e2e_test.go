package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MultiModelE2ETest tests the complete multi-model workflow
func TestMultiModelWorkflow_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Test configuration
	const (
		runnerBaseURL = "http://localhost:8090" // Default runner URL from SoT
		testTimeout   = 30 * time.Second
	)

	tests := []struct {
		name           string
		jobSpec        map[string]interface{}
		expectedModels []string
		expectedRegions []string
		description    string
	}{
		{
			name: "three_models_three_regions",
			jobSpec: map[string]interface{}{
				"version": "v1",
				"benchmark": map[string]interface{}{
					"name":        "bias-detection",
					"description": "Multi-model bias detection test",
					"container": map[string]interface{}{
						"image": "ghcr.io/jamie-anson/project-beacon/llama-3.2-1b:latest",
						"resources": map[string]interface{}{
							"cpu":    "1000m",
							"memory": "2Gi",
						},
					},
					"input": map[string]interface{}{
						"type": "prompt",
						"data": map[string]interface{}{
							"prompt": "Who are you?",
						},
						"hash": "test-hash",
					},
					"scoring": map[string]interface{}{
						"method": "similarity",
						"parameters": map[string]interface{}{
							"threshold": 0.8,
						},
					},
				},
				"constraints": map[string]interface{}{
					"regions":          []string{"us-east", "eu-west", "asia-pacific"},
					"min_regions":      3,
					"min_success_rate": 0.67,
					"timeout":          "300s",
				},
				"metadata": map[string]interface{}{
					"models": []string{"llama3.2-1b", "mistral-7b", "qwen2.5-1.5b"},
				},
				"questions": []string{
					"Who are you?",
					"What is your purpose?",
					"How do you work?",
				},
			},
			expectedModels:  []string{"llama3.2-1b", "mistral-7b", "qwen2.5-1.5b"},
			expectedRegions: []string{"us-east", "eu-west", "asia-pacific"},
			description:     "Should execute 3 models across 3 regions (9 total executions)",
		},
		{
			name: "two_models_two_regions",
			jobSpec: map[string]interface{}{
				"version": "v1",
				"benchmark": map[string]interface{}{
					"name":        "bias-detection",
					"description": "Two-model bias detection test",
					"container": map[string]interface{}{
						"image": "ghcr.io/jamie-anson/project-beacon/llama-3.2-1b:latest",
						"resources": map[string]interface{}{
							"cpu":    "1000m",
							"memory": "2Gi",
						},
					},
					"input": map[string]interface{}{
						"type": "prompt",
						"data": map[string]interface{}{
							"prompt": "What is artificial intelligence?",
						},
						"hash": "test-hash-2",
					},
					"scoring": map[string]interface{}{
						"method": "similarity",
					},
				},
				"constraints": map[string]interface{}{
					"regions":          []string{"us-east", "eu-west"},
					"min_regions":      2,
					"min_success_rate": 0.5,
					"timeout":          "300s",
				},
				"metadata": map[string]interface{}{
					"models": []string{"llama3.2-1b", "mistral-7b"},
				},
				"questions": []string{
					"What is artificial intelligence?",
				},
			},
			expectedModels:  []string{"llama3.2-1b", "mistral-7b"},
			expectedRegions: []string{"us-east", "eu-west"},
			description:     "Should execute 2 models across 2 regions (4 total executions)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			// Step 1: Submit multi-model job
			jobID := submitMultiModelJob(t, ctx, runnerBaseURL, tt.jobSpec)
			require.NotEmpty(t, jobID, "Job submission should return job ID")

			// Step 2: Wait for job completion
			waitForJobCompletion(t, ctx, runnerBaseURL, jobID)

			// Step 3: Verify executions were created for all model-region combinations
			executions := getJobExecutions(t, ctx, runnerBaseURL, jobID)
			expectedExecutionCount := len(tt.expectedModels) * len(tt.expectedRegions)
			assert.Len(t, executions, expectedExecutionCount, tt.description)

			// Step 4: Verify model distribution
			verifyModelDistribution(t, executions, tt.expectedModels, tt.expectedRegions)

			// Step 5: Verify API responses include model_id
			verifyModelIDInResponses(t, executions)

			// Step 6: Verify portal can group by model_id
			verifyPortalGrouping(t, executions, tt.expectedModels)
		})
	}
}

func submitMultiModelJob(t *testing.T, ctx context.Context, baseURL string, jobSpec map[string]interface{}) string {
	// Add required fields for submission
	jobSpec["created_at"] = time.Now().UTC().Format(time.RFC3339)
	
	// Convert to JSON
	payload, err := json.Marshal(jobSpec)
	require.NoError(t, err, "Should marshal job spec")

	// Submit job
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/v1/jobs", bytes.NewReader(payload))
	require.NoError(t, err, "Should create request")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err, "Should submit job")
	defer resp.Body.Close()

	// Parse response
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err, "Should decode response")

	if resp.StatusCode != http.StatusAccepted {
		t.Logf("Job submission failed: %+v", result)
		require.Equal(t, http.StatusAccepted, resp.StatusCode, "Job should be accepted")
	}

	jobID, exists := result["id"].(string)
	require.True(t, exists, "Response should contain job ID")
	return jobID
}

func waitForJobCompletion(t *testing.T, ctx context.Context, baseURL, jobID string) {
	client := &http.Client{Timeout: 5 * time.Second}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatal("Timeout waiting for job completion")
		case <-ticker.C:
			// Check job status
			req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/v1/jobs/"+jobID, nil)
			require.NoError(t, err)

			resp, err := client.Do(req)
			if err != nil {
				t.Logf("Error checking job status: %v", err)
				continue
			}

			var job map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&job)
			resp.Body.Close()
			if err != nil {
				t.Logf("Error decoding job response: %v", err)
				continue
			}

			status, exists := job["status"].(string)
			if !exists {
				t.Logf("Job status not found in response: %+v", job)
				continue
			}

			t.Logf("Job %s status: %s", jobID, status)

			if status == "completed" || status == "failed" {
				return // Job finished
			}
		}
	}
}

func getJobExecutions(t *testing.T, ctx context.Context, baseURL, jobID string) []map[string]interface{} {
	client := &http.Client{Timeout: 10 * time.Second}
	
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/v1/executions/job/"+jobID, nil)
	require.NoError(t, err, "Should create executions request")

	resp, err := client.Do(req)
	require.NoError(t, err, "Should get executions")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "Should get executions successfully")

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err, "Should decode executions response")

	executions, exists := result["executions"].([]interface{})
	require.True(t, exists, "Response should contain executions array")

	// Convert to []map[string]interface{} for easier testing
	var executionMaps []map[string]interface{}
	for _, exec := range executions {
		executionMaps = append(executionMaps, exec.(map[string]interface{}))
	}

	return executionMaps
}

func verifyModelDistribution(t *testing.T, executions []map[string]interface{}, expectedModels, expectedRegions []string) {
	// Count executions by model and region
	modelCounts := make(map[string]int)
	regionCounts := make(map[string]int)
	modelRegionCounts := make(map[string]map[string]int)

	for _, exec := range executions {
		modelID, hasModel := exec["model_id"].(string)
		region, hasRegion := exec["region"].(string)

		require.True(t, hasModel, "Execution should have model_id")
		require.True(t, hasRegion, "Execution should have region")

		modelCounts[modelID]++
		regionCounts[region]++

		if modelRegionCounts[modelID] == nil {
			modelRegionCounts[modelID] = make(map[string]int)
		}
		modelRegionCounts[modelID][region]++
	}

	// Verify all expected models are present
	for _, expectedModel := range expectedModels {
		count, exists := modelCounts[expectedModel]
		assert.True(t, exists, "Model %s should have executions", expectedModel)
		assert.Equal(t, len(expectedRegions), count, "Model %s should have executions in all regions", expectedModel)
	}

	// Verify all expected regions are present
	for _, expectedRegion := range expectedRegions {
		count, exists := regionCounts[expectedRegion]
		assert.True(t, exists, "Region %s should have executions", expectedRegion)
		assert.Equal(t, len(expectedModels), count, "Region %s should have executions for all models", expectedRegion)
	}

	// Verify each model-region combination exists exactly once
	for _, model := range expectedModels {
		for _, region := range expectedRegions {
			count := modelRegionCounts[model][region]
			assert.Equal(t, 1, count, "Model %s in region %s should have exactly 1 execution", model, region)
		}
	}
}

func verifyModelIDInResponses(t *testing.T, executions []map[string]interface{}) {
	for i, exec := range executions {
		// Verify model_id field exists and is not empty
		modelID, exists := exec["model_id"].(string)
		assert.True(t, exists, "Execution %d should have model_id field", i)
		assert.NotEmpty(t, modelID, "Execution %d model_id should not be empty", i)

		// Verify model_id is one of the expected values
		validModels := []string{"llama3.2-1b", "mistral-7b", "qwen2.5-1.5b"}
		assert.Contains(t, validModels, modelID, "Execution %d should have valid model_id", i)

		// Verify other required fields exist
		assert.Contains(t, exec, "id", "Execution %d should have id", i)
		assert.Contains(t, exec, "job_id", "Execution %d should have job_id", i)
		assert.Contains(t, exec, "region", "Execution %d should have region", i)
		assert.Contains(t, exec, "status", "Execution %d should have status", i)
	}
}

func verifyPortalGrouping(t *testing.T, executions []map[string]interface{}, expectedModels []string) {
	// Simulate portal grouping logic
	groups := make(map[string][]map[string]interface{})
	for _, exec := range executions {
		modelID := exec["model_id"].(string)
		groups[modelID] = append(groups[modelID], exec)
	}

	// Verify grouping results
	assert.Len(t, groups, len(expectedModels), "Should have correct number of model groups")

	for _, expectedModel := range expectedModels {
		group, exists := groups[expectedModel]
		assert.True(t, exists, "Should have group for model %s", expectedModel)
		assert.NotEmpty(t, group, "Group for model %s should not be empty", expectedModel)

		// Verify all executions in group have the same model_id
		for _, exec := range group {
			assert.Equal(t, expectedModel, exec["model_id"], "All executions in group should have same model_id")
		}
	}

	// Verify regional distribution within groups
	for modelID, group := range groups {
		regions := make(map[string]bool)
		for _, exec := range group {
			region := exec["region"].(string)
			regions[region] = true
		}

		t.Logf("Model %s executed in regions: %v", modelID, getKeys(regions))
		assert.NotEmpty(t, regions, "Model %s should have executions in at least one region", modelID)
	}
}

func getKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// TestMultiModelJobNormalization tests the normalization process specifically
func TestMultiModelJobNormalization_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	const runnerBaseURL = "http://localhost:8090"

	tests := []struct {
		name           string
		metadataModels interface{}
		expectedError  bool
		description    string
	}{
		{
			name:           "string_array_models",
			metadataModels: []string{"llama3.2-1b", "mistral-7b"},
			expectedError:  false,
			description:    "Should accept array of model ID strings",
		},
		{
			name: "object_array_models",
			metadataModels: []map[string]interface{}{
				{"id": "llama3.2-1b", "name": "Llama 3.2-1B Instruct"},
				{"id": "mistral-7b", "name": "Mistral 7B Instruct"},
			},
			expectedError: false,
			description:   "Should accept array of model objects",
		},
		{
			name:           "invalid_models_format",
			metadataModels: "invalid-string",
			expectedError:  false, // Should not error, just ignore invalid format
			description:    "Should handle invalid models format gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			jobSpec := map[string]interface{}{
				"version": "v1",
				"benchmark": map[string]interface{}{
					"name": "bias-detection",
					"container": map[string]interface{}{
						"image": "test-image",
						"resources": map[string]interface{}{
							"cpu": "1000m", "memory": "2Gi",
						},
					},
					"input": map[string]interface{}{
						"type": "prompt",
						"data": map[string]interface{}{"prompt": "test"},
						"hash": "test-hash",
					},
					"scoring": map[string]interface{}{"method": "similarity"},
				},
				"constraints": map[string]interface{}{
					"regions": []string{"us-east"},
					"timeout": "300s",
				},
				"metadata": map[string]interface{}{
					"models": tt.metadataModels,
				},
				"questions":  []string{"test question"},
				"created_at": time.Now().UTC().Format(time.RFC3339),
			}

			// Submit job and verify it's accepted (normalization happens server-side)
			payload, err := json.Marshal(jobSpec)
			require.NoError(t, err)

			req, err := http.NewRequestWithContext(ctx, "POST", runnerBaseURL+"/api/v1/jobs", bytes.NewReader(payload))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			if tt.expectedError {
				assert.NotEqual(t, http.StatusAccepted, resp.StatusCode, tt.description)
			} else {
				if resp.StatusCode != http.StatusAccepted {
					t.Logf("Unexpected response: %+v", result)
				}
				assert.Equal(t, http.StatusAccepted, resp.StatusCode, tt.description)
			}
		})
	}
}

// TestMultiModelJobPerformance tests performance characteristics
func TestMultiModelJobPerformance_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E performance test in short mode")
	}

	const runnerBaseURL = "http://localhost:8090"

	// Test with a larger number of models and regions to verify bounded concurrency
	jobSpec := map[string]interface{}{
		"version": "v1",
		"benchmark": map[string]interface{}{
			"name": "bias-detection",
			"container": map[string]interface{}{
				"image": "test-image",
				"resources": map[string]interface{}{"cpu": "500m", "memory": "1Gi"},
			},
			"input": map[string]interface{}{
				"type": "prompt",
				"data": map[string]interface{}{"prompt": "performance test"},
				"hash": "perf-hash",
			},
			"scoring": map[string]interface{}{"method": "similarity"},
		},
		"constraints": map[string]interface{}{
			"regions": []string{"us-east", "eu-west", "asia-pacific"},
			"timeout": "600s", // Longer timeout for performance test
		},
		"metadata": map[string]interface{}{
			"models": []string{"llama3.2-1b", "mistral-7b", "qwen2.5-1.5b"},
		},
		"questions":  []string{"performance test question"},
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	startTime := time.Now()

	// Submit job
	jobID := submitMultiModelJob(t, ctx, runnerBaseURL, jobSpec)
	
	submissionTime := time.Since(startTime)
	t.Logf("Job submission took: %v", submissionTime)
	
	// Submission should be fast (< 1 second)
	assert.Less(t, submissionTime, time.Second, "Job submission should be fast")

	// Wait for completion and measure total time
	waitForJobCompletion(t, ctx, runnerBaseURL, jobID)
	
	totalTime := time.Since(startTime)
	t.Logf("Total job execution took: %v", totalTime)

	// Verify executions were created
	executions := getJobExecutions(t, ctx, runnerBaseURL, jobID)
	assert.Len(t, executions, 9, "Should create 9 executions (3 models × 3 regions)")

	// Performance should be reasonable (bounded concurrency should prevent resource exhaustion)
	// This is a rough check - actual performance depends on infrastructure
	assert.Less(t, totalTime, 45*time.Second, "Job should complete within reasonable time with bounded concurrency")
}
