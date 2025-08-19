package integration

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobSpecStructure(t *testing.T) {
	// Test that we can create a valid JobSpec with the correct structure
	jobSpec := &models.JobSpec{
		ID:      "test-job-001",
		Version: "1.0",
		Benchmark: models.BenchmarkSpec{
			Name:        "Test Benchmark",
			Description: "A test benchmark for integration testing",
			Container: models.ContainerSpec{
				Image: "alpine:latest",
				Tag:   "latest",
				Command: []string{"echo", "hello world"},
				Resources: models.ResourceSpec{
					CPU:    "100m",
					Memory: "128Mi",
				},
			},
			Input: models.InputSpec{
				Type: "prompt",
				Data: map[string]interface{}{
					"prompt": "Say hello",
				},
				Hash: "abc123",
			},
			Scoring: models.ScoringSpec{
				Method: "similarity",
				Parameters: map[string]interface{}{
					"threshold": 0.8,
				},
			},
		},
		Constraints: models.ExecutionConstraints{
			Regions:    []string{"us-east-1", "eu-west-1"},
			MinRegions: 2,
			Timeout:    5 * time.Minute,
		},
		Metadata: map[string]interface{}{
			"test": "integration",
		},
		CreatedAt: time.Now(),
	}

	// Test validation
	err := jobSpec.Validate()
	require.NoError(t, err, "JobSpec should be valid")

	// Test JSON serialization
	jsonData, err := json.Marshal(jobSpec)
	require.NoError(t, err, "Should be able to marshal JobSpec to JSON")

	// Test JSON deserialization
	var deserializedSpec models.JobSpec
	err = json.Unmarshal(jsonData, &deserializedSpec)
	require.NoError(t, err, "Should be able to unmarshal JobSpec from JSON")

	// Verify key fields
	assert.Equal(t, jobSpec.ID, deserializedSpec.ID)
	assert.Equal(t, jobSpec.Version, deserializedSpec.Version)
	assert.Equal(t, jobSpec.Benchmark.Name, deserializedSpec.Benchmark.Name)
	assert.Equal(t, jobSpec.Benchmark.Container.Image, deserializedSpec.Benchmark.Container.Image)
	assert.Equal(t, jobSpec.Constraints.Regions, deserializedSpec.Constraints.Regions)
}

func TestJobSpecValidation(t *testing.T) {
	tests := []struct {
		name      string
		jobSpec   *models.JobSpec
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid jobspec",
			jobSpec: &models.JobSpec{
				ID:      "valid-job",
				Version: "1.0",
				Benchmark: models.BenchmarkSpec{
					Name: "Valid Benchmark",
					Container: models.ContainerSpec{
						Image: "alpine:latest",
					},
					Input: models.InputSpec{
						Hash: "abc123",
					},
				},
				Constraints: models.ExecutionConstraints{
					Regions: []string{"us-east-1"},
				},
			},
			expectErr: false,
		},
		{
			name: "missing ID",
			jobSpec: &models.JobSpec{
				Version: "1.0",
				Benchmark: models.BenchmarkSpec{
					Name: "Test",
					Container: models.ContainerSpec{
						Image: "alpine:latest",
					},
					Input: models.InputSpec{
						Hash: "abc123",
					},
				},
				Constraints: models.ExecutionConstraints{
					Regions: []string{"us-east-1"},
				},
			},
			expectErr: true,
			errMsg:    "jobspec ID is required",
		},
		{
			name: "missing version",
			jobSpec: &models.JobSpec{
				ID: "test-job",
				Benchmark: models.BenchmarkSpec{
					Name: "Test",
					Container: models.ContainerSpec{
						Image: "alpine:latest",
					},
					Input: models.InputSpec{
						Hash: "abc123",
					},
				},
				Constraints: models.ExecutionConstraints{
					Regions: []string{"us-east-1"},
				},
			},
			expectErr: true,
			errMsg:    "jobspec version is required",
		},
		{
			name: "missing benchmark name",
			jobSpec: &models.JobSpec{
				ID:      "test-job",
				Version: "1.0",
				Benchmark: models.BenchmarkSpec{
					Container: models.ContainerSpec{
						Image: "alpine:latest",
					},
					Input: models.InputSpec{
						Hash: "abc123",
					},
				},
				Constraints: models.ExecutionConstraints{
					Regions: []string{"us-east-1"},
				},
			},
			expectErr: true,
			errMsg:    "benchmark name is required",
		},
		{
			name: "missing container image",
			jobSpec: &models.JobSpec{
				ID:      "test-job",
				Version: "1.0",
				Benchmark: models.BenchmarkSpec{
					Name: "Test",
					Container: models.ContainerSpec{},
					Input: models.InputSpec{
						Hash: "abc123",
					},
				},
				Constraints: models.ExecutionConstraints{
					Regions: []string{"us-east-1"},
				},
			},
			expectErr: true,
			errMsg:    "container image is required",
		},
		{
			name: "missing regions",
			jobSpec: &models.JobSpec{
				ID:      "test-job",
				Version: "1.0",
				Benchmark: models.BenchmarkSpec{
					Name: "Test",
					Container: models.ContainerSpec{
						Image: "alpine:latest",
					},
					Input: models.InputSpec{
						Hash: "abc123",
					},
				},
				Constraints: models.ExecutionConstraints{
					Regions: []string{},
				},
			},
			expectErr: true,
			errMsg:    "at least one region constraint is required",
		},
		{
			name: "missing input hash",
			jobSpec: &models.JobSpec{
				ID:      "test-job",
				Version: "1.0",
				Benchmark: models.BenchmarkSpec{
					Name: "Test",
					Container: models.ContainerSpec{
						Image: "alpine:latest",
					},
					Input: models.InputSpec{},
				},
				Constraints: models.ExecutionConstraints{
					Regions: []string{"us-east-1"},
				},
			},
			expectErr: true,
			errMsg:    "input hash is required for integrity verification",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.jobSpec.Validate()
			if tt.expectErr {
				require.Error(t, err, "Expected validation error")
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err, "Expected no validation error")
			}
		})
	}
}

func TestReceiptStructure(t *testing.T) {
	// Test Receipt creation and structure
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
		Hash: "def456",
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

	receipt := models.NewReceipt("job-001", executionDetails, output, provenance)

	// Verify receipt structure
	assert.Equal(t, "v0.1.0", receipt.SchemaVersion)
	assert.Equal(t, "job-001", receipt.JobSpecID)
	assert.Equal(t, executionDetails.TaskID, receipt.ExecutionDetails.TaskID)
	assert.Equal(t, output.Hash, receipt.Output.Hash)
	assert.Equal(t, provenance.BenchmarkHash, receipt.Provenance.BenchmarkHash)
	assert.NotEmpty(t, receipt.ID)
	assert.NotZero(t, receipt.CreatedAt)

	// Test JSON serialization
	jsonData, err := json.Marshal(receipt)
	require.NoError(t, err, "Should be able to marshal Receipt to JSON")

	// Test JSON deserialization
	var deserializedReceipt models.Receipt
	err = json.Unmarshal(jsonData, &deserializedReceipt)
	require.NoError(t, err, "Should be able to unmarshal Receipt from JSON")

	assert.Equal(t, receipt.ID, deserializedReceipt.ID)
	assert.Equal(t, receipt.JobSpecID, deserializedReceipt.JobSpecID)
	assert.Equal(t, receipt.ExecutionDetails.Status, deserializedReceipt.ExecutionDetails.Status)
}

func TestCrossRegionDiffStructure(t *testing.T) {
	// Test CrossRegionDiff structure
	diff := &models.CrossRegionDiff{
		ID:              "diff-001",
		JobSpecID:       "job-001",
		RegionA:         "us-east-1",
		RegionB:         "eu-west-1",
		SimilarityScore: 0.95,
		DiffData: models.DiffData{
			TextDiffs: []models.TextDiff{
				{
					Type:    "changed",
					LineNum: 1,
					Content: "Hello, World!",
					Context: "output line 1",
				},
			},
			Summary: "Minor differences in output formatting",
		},
		Classification: "minor",
		Metadata: map[string]interface{}{
			"analysis_version": "1.0",
		},
		CreatedAt: time.Now(),
	}

	// Test JSON serialization
	jsonData, err := json.Marshal(diff)
	require.NoError(t, err, "Should be able to marshal CrossRegionDiff to JSON")

	// Test JSON deserialization
	var deserializedDiff models.CrossRegionDiff
	err = json.Unmarshal(jsonData, &deserializedDiff)
	require.NoError(t, err, "Should be able to unmarshal CrossRegionDiff from JSON")

	assert.Equal(t, diff.ID, deserializedDiff.ID)
	assert.Equal(t, diff.JobSpecID, deserializedDiff.JobSpecID)
	assert.Equal(t, diff.SimilarityScore, deserializedDiff.SimilarityScore)
	assert.Equal(t, diff.Classification, deserializedDiff.Classification)
	assert.Len(t, deserializedDiff.DiffData.TextDiffs, 1)
}
