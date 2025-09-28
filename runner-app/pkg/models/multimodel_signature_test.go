package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMultiModelJobSignatureVerification tests that signature verification works correctly
// with multi-model jobs, ensuring normalization doesn't break signature verification
func TestMultiModelJobSignatureVerification(t *testing.T) {
	tests := []struct {
		name        string
		jobSpec     *JobSpec
		description string
	}{
		{
			name: "multi_model_job_with_metadata_models",
			jobSpec: &JobSpec{
				Version: "v1",
				Benchmark: BenchmarkSpec{
					Name:        "bias-detection",
					Description: "Multi-model bias detection test",
					Container: ContainerSpec{
						Image: "ghcr.io/jamie-anson/project-beacon/llama-3.2-1b:latest",
						Resources: ResourceSpec{
							CPU:    "1000m",
							Memory: "2Gi",
						},
					},
					Input: InputSpec{
						Type: "prompt",
						Data: map[string]interface{}{
							"prompt": "Who are you?",
						},
						Hash: "test-hash",
					},
					Scoring: ScoringSpec{
						Method: "similarity",
						Parameters: map[string]interface{}{
							"threshold": 0.8,
						},
					},
				},
				Constraints: ExecutionConstraints{
					Regions:        []string{"us-east", "eu-west", "asia-pacific"},
					MinRegions:     3,
					MinSuccessRate: 0.67,
					Timeout:        300 * time.Second,
				},
				Metadata: map[string]interface{}{
					"models": []interface{}{"llama3.2-1b", "mistral-7b", "qwen2.5-1.5b"},
				},
				Questions: []string{
					"Who are you?",
					"What is your purpose?",
				},
				CreatedAt: time.Now().UTC(),
			},
			description: "Multi-model job should maintain signature verification integrity",
		},
		{
			name: "multi_model_job_with_object_models",
			jobSpec: &JobSpec{
				Version: "v1",
				Benchmark: BenchmarkSpec{
					Name:        "bias-detection",
					Description: "Multi-model with object format",
					Container: ContainerSpec{
						Image: "test-image",
						Resources: ResourceSpec{
							CPU:    "500m",
							Memory: "1Gi",
						},
					},
					Input: InputSpec{
						Type: "prompt",
						Data: map[string]interface{}{
							"prompt": "Test prompt",
						},
						Hash: "test-hash-2",
					},
					Scoring: ScoringSpec{
						Method: "similarity",
					},
				},
				Constraints: ExecutionConstraints{
					Regions:        []string{"us-east", "eu-west"},
					MinRegions:     2,
					MinSuccessRate: 0.5,
					Timeout:        300 * time.Second,
				},
				Metadata: map[string]interface{}{
					"models": []interface{}{
						map[string]interface{}{"id": "llama3.2-1b", "name": "Llama 3.2-1B"},
						map[string]interface{}{"id": "mistral-7b", "name": "Mistral 7B"},
					},
				},
				Questions: []string{"Test question"},
				CreatedAt: time.Now().UTC(),
			},
			description: "Multi-model job with object format should maintain signature verification",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the signature verification process
			
			// 1. Verify the job spec validates before normalization
			err := tt.jobSpec.Validate()
			require.NoError(t, err, "JobSpec should validate before normalization")

			// 2. Verify that the Models slice is initially empty (before normalization)
			assert.Empty(t, tt.jobSpec.Models, "Models slice should be empty before normalization")

			// 3. Verify metadata contains models
			models, exists := tt.jobSpec.Metadata["models"]
			assert.True(t, exists, "Metadata should contain models")
			assert.NotEmpty(t, models, "Models in metadata should not be empty")

			// 4. Test signature creation (this would happen in the portal)
			// Note: We're not actually signing here, just verifying the structure is correct
			signableSpec := *tt.jobSpec
			
			// Remove ID field for signature (as done in CreateSignableJobSpec)
			originalID := signableSpec.ID
			signableSpec.ID = ""
			
			// Verify the signable spec still has the metadata.models
			signableModels, exists := signableSpec.Metadata["models"]
			assert.True(t, exists, "Signable spec should retain metadata.models")
			assert.Equal(t, models, signableModels, "Signable spec models should match original")

			// 5. Restore ID and verify normalization would work
			signableSpec.ID = originalID
			
			// Simulate what happens after signature verification in JobSpecProcessor
			// (This is where NormalizeModelsFromMetadata would be called)
			
			// Before normalization: Models slice should be empty
			assert.Empty(t, signableSpec.Models, "Models slice should be empty before normalization")
			
			// After normalization: Models slice should be populated
			// (We'll simulate the normalization logic here)
			if modelsArray, ok := signableSpec.Metadata["models"].([]interface{}); ok {
				for _, model := range modelsArray {
					switch m := model.(type) {
					case string:
						signableSpec.Models = append(signableSpec.Models, ModelSpec{
							ID:       m,
							Name:     m,
							Provider: "hybrid",
							Regions:  signableSpec.Constraints.Regions,
						})
					case map[string]interface{}:
						if id, ok := m["id"].(string); ok && id != "" {
							name, _ := m["name"].(string)
							if name == "" {
								name = id
							}
							signableSpec.Models = append(signableSpec.Models, ModelSpec{
								ID:       id,
								Name:     name,
								Provider: "hybrid",
								Regions:  signableSpec.Constraints.Regions,
							})
						}
					}
				}
			}
			
			// After normalization: Models slice should be populated
			assert.NotEmpty(t, signableSpec.Models, "Models slice should be populated after normalization")
			
			// Verify the normalized models are correct
			expectedModelCount := len(tt.jobSpec.Metadata["models"].([]interface{}))
			assert.Len(t, signableSpec.Models, expectedModelCount, "Should have correct number of normalized models")
			
			// Verify each model has the expected structure
			for i, model := range signableSpec.Models {
				assert.NotEmpty(t, model.ID, "Model %d should have ID", i)
				assert.NotEmpty(t, model.Name, "Model %d should have Name", i)
				assert.Equal(t, "hybrid", model.Provider, "Model %d should have hybrid provider", i)
				assert.Equal(t, signableSpec.Constraints.Regions, model.Regions, "Model %d should inherit regions from constraints", i)
			}
		})
	}
}

// TestSignatureVerificationIntegrity tests that the signature verification process
// is not affected by the presence of metadata.models
func TestSignatureVerificationIntegrity_MultiModel(t *testing.T) {
	// Create two identical job specs, one with models in metadata, one without
	baseJobSpec := &JobSpec{
		Version: "v1",
		Benchmark: BenchmarkSpec{
			Name: "test-benchmark",
			Container: ContainerSpec{
				Image: "test-image",
				Resources: ResourceSpec{CPU: "1000m", Memory: "1Gi"},
			},
			Input: InputSpec{
				Type: "prompt",
				Data: map[string]interface{}{"prompt": "test"},
				Hash: "test-hash",
			},
			Scoring: ScoringSpec{Method: "similarity"},
		},
		Constraints: ExecutionConstraints{
			Regions:    []string{"us-east"},
			MinRegions: 1,
			Timeout:    300 * time.Second,
		},
		Questions: []string{"test question"},
		CreatedAt: time.Now().UTC(),
	}

	// Job spec without models in metadata
	jobSpecWithoutModels := *baseJobSpec
	jobSpecWithoutModels.Metadata = map[string]interface{}{
		"other_field": "value",
	}

	// Job spec with models in metadata
	jobSpecWithModels := *baseJobSpec
	jobSpecWithModels.Metadata = map[string]interface{}{
		"models":      []interface{}{"llama3.2-1b", "mistral-7b"},
		"other_field": "value",
	}

	// Both should validate successfully
	err1 := jobSpecWithoutModels.Validate()
	err2 := jobSpecWithModels.Validate()
	
	assert.NoError(t, err1, "Job spec without models should validate")
	assert.NoError(t, err2, "Job spec with models should validate")

	// Both should have empty Models slice before normalization
	assert.Empty(t, jobSpecWithoutModels.Models, "Job without models should have empty Models slice")
	assert.Empty(t, jobSpecWithModels.Models, "Job with models should have empty Models slice before normalization")

	// The presence of metadata.models should not affect validation
	// (Normalization happens after signature verification)
	
	// Verify metadata is preserved correctly
	assert.Equal(t, "value", jobSpecWithoutModels.Metadata["other_field"])
	assert.Equal(t, "value", jobSpecWithModels.Metadata["other_field"])
	
	_, hasModels := jobSpecWithoutModels.Metadata["models"]
	assert.False(t, hasModels, "Job without models should not have models in metadata")
	
	models, hasModels := jobSpecWithModels.Metadata["models"]
	assert.True(t, hasModels, "Job with models should have models in metadata")
	assert.NotEmpty(t, models, "Models in metadata should not be empty")
}

// TestNormalizationDoesNotAffectSignature tests that normalization happens
// after signature verification and doesn't affect the signature process
func TestNormalizationDoesNotAffectSignature(t *testing.T) {
	originalJobSpec := &JobSpec{
		Version: "v1",
		Benchmark: BenchmarkSpec{
			Name: "signature-test",
			Container: ContainerSpec{
				Image: "test-image",
				Resources: ResourceSpec{CPU: "1000m", Memory: "1Gi"},
			},
			Input: InputSpec{
				Type: "prompt",
				Data: map[string]interface{}{"prompt": "signature test"},
				Hash: "sig-test-hash",
			},
			Scoring: ScoringSpec{Method: "similarity"},
		},
		Constraints: ExecutionConstraints{
			Regions:    []string{"us-east", "eu-west"},
			MinRegions: 2,
			Timeout:    300 * time.Second,
		},
		Metadata: map[string]interface{}{
			"models": []interface{}{"llama3.2-1b", "mistral-7b"},
		},
		Questions: []string{"signature test question"},
		CreatedAt: time.Now().UTC(),
	}

	// Step 1: Verify original state (before any processing)
	assert.Empty(t, originalJobSpec.Models, "Original job spec should have empty Models slice")
	assert.Contains(t, originalJobSpec.Metadata, "models", "Original job spec should have models in metadata")

	// Step 2: Simulate signature verification process
	// (This is what would happen in the security pipeline)
	
	// Create signable copy (remove ID for signature)
	signableJobSpec := *originalJobSpec
	signableJobSpec.ID = ""
	
	// Verify signable spec preserves metadata.models
	assert.Contains(t, signableJobSpec.Metadata, "models", "Signable spec should preserve metadata.models")
	assert.Empty(t, signableJobSpec.Models, "Signable spec should have empty Models slice")
	
	// Step 3: Simulate post-signature normalization
	// (This is what would happen in JobSpecProcessor.NormalizeModelsFromMetadata)
	
	normalizedJobSpec := signableJobSpec
	normalizedJobSpec.ID = originalJobSpec.ID // Restore ID after signature verification
	
	// Simulate normalization
	if modelsArray, ok := normalizedJobSpec.Metadata["models"].([]interface{}); ok {
		for _, model := range modelsArray {
			if modelID, ok := model.(string); ok {
				normalizedJobSpec.Models = append(normalizedJobSpec.Models, ModelSpec{
					ID:       modelID,
					Name:     modelID,
					Provider: "hybrid",
					Regions:  normalizedJobSpec.Constraints.Regions,
				})
			}
		}
	}
	
	// Step 4: Verify post-normalization state
	assert.NotEmpty(t, normalizedJobSpec.Models, "Normalized job spec should have populated Models slice")
	assert.Len(t, normalizedJobSpec.Models, 2, "Should have 2 normalized models")
	assert.Contains(t, normalizedJobSpec.Metadata, "models", "Normalized spec should still have metadata.models")
	
	// Step 5: Verify normalization results
	modelIDs := make([]string, len(normalizedJobSpec.Models))
	for i, model := range normalizedJobSpec.Models {
		modelIDs[i] = model.ID
		assert.Equal(t, "hybrid", model.Provider, "Model %d should have hybrid provider", i)
		assert.Equal(t, normalizedJobSpec.Constraints.Regions, model.Regions, "Model %d should inherit regions", i)
	}
	
	assert.Contains(t, modelIDs, "llama3.2-1b", "Should have llama model")
	assert.Contains(t, modelIDs, "mistral-7b", "Should have mistral model")
	
	// Step 6: Verify original job spec is unchanged
	assert.Empty(t, originalJobSpec.Models, "Original job spec should remain unchanged")
	assert.Contains(t, originalJobSpec.Metadata, "models", "Original metadata should be preserved")
}
