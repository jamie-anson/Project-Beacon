package integration

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/config"
	"github.com/jamie-anson/project-beacon-runner/internal/service"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobServiceIntegration(t *testing.T) {
	// Skip if no database URL is provided
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("Skipping integration test: DATABASE_URL not set")
	}

	// Test that we can create a JobsService
	jobsService := service.NewJobsService(nil) // Pass nil DB for now
	require.NotNil(t, jobsService, "JobsService should be created")

	// Test JobSpec creation and JSON marshaling
	jobSpec := createValidJobSpec()
	
	// Test JSON marshaling for service usage
	jobSpecJSON, err := json.Marshal(jobSpec)
	require.NoError(t, err, "Should be able to marshal JobSpec to JSON")
	
	// Test that we can unmarshal it back
	var unmarshaledSpec models.JobSpec
	err = json.Unmarshal(jobSpecJSON, &unmarshaledSpec)
	require.NoError(t, err, "Should be able to unmarshal JobSpec from JSON")
	
	// Verify the unmarshaled spec matches
	assert.Equal(t, jobSpec.ID, unmarshaledSpec.ID)
	assert.Equal(t, jobSpec.Version, unmarshaledSpec.Version)
	assert.Equal(t, jobSpec.Benchmark.Name, unmarshaledSpec.Benchmark.Name)
}

func TestConfigIntegration(t *testing.T) {
	// Test config loading and validation
	cfg := config.Load()
	require.NotNil(t, cfg, "Config should be loaded")
	
	// Test config validation
	err := cfg.Validate()
	if cfg.DatabaseURL != "" && cfg.RedisURL != "" {
		// If URLs are set, validation should pass
		assert.NoError(t, err, "Config validation should pass when URLs are set")
	} else {
		// If URLs are not set, we expect validation to fail or pass depending on implementation
		t.Logf("Config validation result: %v", err)
	}
	
	// Test that required fields have reasonable defaults
	assert.NotEmpty(t, cfg.HTTPPort, "HTTP port should be set")
	assert.Greater(t, cfg.DBTimeout, time.Duration(0), "DB timeout should be positive")
}

func TestJobSpecWorkflow(t *testing.T) {
	// Test the complete JobSpec workflow: creation -> validation -> JSON -> receipt
	
	// 1. Create a valid JobSpec
	jobSpec := createValidJobSpec()
	
	// 2. Validate it
	err := jobSpec.Validate()
	require.NoError(t, err, "JobSpec should be valid")
	
	// 3. Marshal to JSON (as would be done for storage/queue)
	jobSpecJSON, err := json.Marshal(jobSpec)
	require.NoError(t, err, "Should marshal to JSON")
	require.NotEmpty(t, jobSpecJSON, "JSON should not be empty")
	
	// 4. Create a receipt for this job
	executionDetails := models.ExecutionDetails{
		TaskID:      "task-001",
		ProviderID:  "provider-001", 
		Region:      "us-east-1",
		StartedAt:   time.Now().Add(-5 * time.Minute),
		CompletedAt: time.Now(),
		Duration:    5 * time.Minute,
		Status:      "completed",
	}
	
	output := models.ExecutionOutput{
		Data: map[string]interface{}{
			"result": "Hello, World!",
		},
		Hash: "output-hash-456",
		Metadata: map[string]interface{}{
			"execution_time": "5m",
		},
	}
	
	provenance := models.ProvenanceInfo{
		BenchmarkHash: "benchmark-hash-123",
		ProviderInfo: map[string]interface{}{
			"provider_id": "provider-001",
			"region":      "us-east-1",
		},
		ExecutionEnv: map[string]interface{}{
			"container_runtime": "docker",
		},
	}
	
	receipt := models.NewReceipt(jobSpec.ID, executionDetails, output, provenance)
	require.NotNil(t, receipt, "Receipt should be created")
	
	// 5. Verify receipt structure
	assert.Equal(t, jobSpec.ID, receipt.JobSpecID)
	assert.Equal(t, "v0.1.0", receipt.SchemaVersion)
	assert.Equal(t, "completed", receipt.ExecutionDetails.Status)
	
	// 6. Test receipt JSON serialization
	receiptJSON, err := json.Marshal(receipt)
	require.NoError(t, err, "Receipt should marshal to JSON")
	
	var unmarshaledReceipt models.Receipt
	err = json.Unmarshal(receiptJSON, &unmarshaledReceipt)
	require.NoError(t, err, "Receipt should unmarshal from JSON")
	
	assert.Equal(t, receipt.ID, unmarshaledReceipt.ID)
	assert.Equal(t, receipt.JobSpecID, unmarshaledReceipt.JobSpecID)
}

func TestCrossRegionDiffWorkflow(t *testing.T) {
	// Test cross-region diff creation and processing
	
	jobSpec := createValidJobSpec()
	
	// Create a cross-region diff
	diff := &models.CrossRegionDiff{
		ID:              "diff-001",
		JobSpecID:       jobSpec.ID,
		RegionA:         "us-east-1",
		RegionB:         "eu-west-1",
		SimilarityScore: 0.92,
		DiffData: models.DiffData{
			TextDiffs: []models.TextDiff{
				{
					Type:    "changed",
					LineNum: 1,
					Content: "Hello, World! (US)",
					Context: "output line 1",
				},
				{
					Type:    "changed", 
					LineNum: 1,
					Content: "Hello, World! (EU)",
					Context: "output line 1",
				},
			},
			StructDiffs: []models.StructuralDiff{
				{
					Path:     "$.result",
					Type:     "changed",
					OldValue: "Hello, World! (US)",
					NewValue: "Hello, World! (EU)",
				},
			},
			Summary: "Minor regional differences in output",
		},
		Classification: "minor",
		Metadata: map[string]interface{}{
			"analysis_version": "1.0",
			"threshold":        0.9,
		},
		CreatedAt: time.Now(),
	}
	
	// Test JSON serialization
	diffJSON, err := json.Marshal(diff)
	require.NoError(t, err, "Diff should marshal to JSON")
	
	var unmarshaledDiff models.CrossRegionDiff
	err = json.Unmarshal(diffJSON, &unmarshaledDiff)
	require.NoError(t, err, "Diff should unmarshal from JSON")
	
	// Verify structure
	assert.Equal(t, diff.ID, unmarshaledDiff.ID)
	assert.Equal(t, diff.JobSpecID, unmarshaledDiff.JobSpecID)
	assert.Equal(t, diff.SimilarityScore, unmarshaledDiff.SimilarityScore)
	assert.Equal(t, diff.Classification, unmarshaledDiff.Classification)
	assert.Len(t, unmarshaledDiff.DiffData.TextDiffs, 2)
	assert.Len(t, unmarshaledDiff.DiffData.StructDiffs, 1)
}

func TestErrorHandling(t *testing.T) {
	// Test various error conditions
	
	// Test invalid JobSpec validation
	invalidSpecs := []*models.JobSpec{
		{}, // Empty spec
		{ID: "test"}, // Missing version
		{ID: "test", Version: "1.0"}, // Missing benchmark
		{
			ID: "test", 
			Version: "1.0",
			Benchmark: models.BenchmarkSpec{}, // Empty benchmark
		},
	}
	
	for i, spec := range invalidSpecs {
		t.Run(fmt.Sprintf("invalid_spec_%d", i), func(t *testing.T) {
			err := spec.Validate()
			assert.Error(t, err, "Invalid spec should fail validation")
		})
	}
}

// Helper function to create a valid JobSpec for testing
func createValidJobSpec() *models.JobSpec {
	return &models.JobSpec{
		ID:      "integration-test-job-001",
		Version: "1.0",
		Benchmark: models.BenchmarkSpec{
			Name:        "Integration Test Benchmark",
			Description: "A benchmark for integration testing",
			Container: models.ContainerSpec{
				Image: "alpine:latest",
				Tag:   "latest",
				Command: []string{"echo", "Hello, World!"},
				Environment: map[string]string{
					"TEST_ENV": "integration",
				},
				Resources: models.ResourceSpec{
					CPU:    "100m",
					Memory: "128Mi",
				},
			},
			Input: models.InputSpec{
				Type: "prompt",
				Data: map[string]interface{}{
					"prompt": "Say hello to the world",
				},
				Hash: "input-hash-123",
			},
			Scoring: models.ScoringSpec{
				Method: "similarity",
				Parameters: map[string]interface{}{
					"threshold": 0.8,
					"algorithm": "cosine",
				},
			},
			Metadata: map[string]string{
				"category": "text-generation",
				"language": "en",
			},
		},
		Constraints: models.ExecutionConstraints{
			Regions:    []string{"us-east-1", "eu-west-1", "ap-southeast-1"},
			MinRegions: 3,
			Timeout:    10 * time.Minute,
			Providers: []models.ProviderFilter{
				{
					Region:   "us-east-1",
					MinScore: 0.8,
					MaxPrice: 0.1,
				},
			},
		},
		Metadata: map[string]interface{}{
			"test_type":    "integration",
			"created_by":   "integration_test",
			"description":  "Test job for integration testing",
		},
		CreatedAt: time.Now(),
	}
}
