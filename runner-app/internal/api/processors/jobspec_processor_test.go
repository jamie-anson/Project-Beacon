package processors

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func TestNormalizeModelsFromMetadata(t *testing.T) {
	processor := NewJobSpecProcessor()

	tests := []struct {
		name           string
		spec           *models.JobSpec
		expectedModels []models.ModelSpec
		description    string
	}{
		{
			name: "string array models",
			spec: &models.JobSpec{
				ID: "test-job-1",
				Metadata: map[string]interface{}{
					"models": []interface{}{"llama3.2-1b", "mistral-7b", "qwen2.5-1.5b"},
				},
				Constraints: models.ExecutionConstraints{
					Regions: []string{"us-east", "eu-west", "asia-pacific"},
				},
			},
			expectedModels: []models.ModelSpec{
				{ID: "llama3.2-1b", Name: "llama3.2-1b", Provider: "hybrid", Regions: []string{"us-east", "eu-west", "asia-pacific"}},
				{ID: "mistral-7b", Name: "mistral-7b", Provider: "hybrid", Regions: []string{"us-east", "eu-west", "asia-pacific"}},
				{ID: "qwen2.5-1.5b", Name: "qwen2.5-1.5b", Provider: "hybrid", Regions: []string{"us-east", "eu-west", "asia-pacific"}},
			},
			description: "Should normalize array of model ID strings",
		},
		{
			name: "object array models with names",
			spec: &models.JobSpec{
				ID: "test-job-2",
				Metadata: map[string]interface{}{
					"models": []interface{}{
						map[string]interface{}{"id": "llama3.2-1b", "name": "Llama 3.2-1B Instruct"},
						map[string]interface{}{"id": "mistral-7b", "name": "Mistral 7B Instruct"},
					},
				},
				Constraints: models.ExecutionConstraints{
					Regions: []string{"us-east", "eu-west"},
				},
			},
			expectedModels: []models.ModelSpec{
				{ID: "llama3.2-1b", Name: "Llama 3.2-1B Instruct", Provider: "hybrid", Regions: []string{"us-east", "eu-west"}},
				{ID: "mistral-7b", Name: "Mistral 7B Instruct", Provider: "hybrid", Regions: []string{"us-east", "eu-west"}},
			},
			description: "Should normalize array of model objects with names",
		},
		{
			name: "no models in metadata",
			spec: &models.JobSpec{
				ID: "test-job-4",
				Metadata: map[string]interface{}{
					"other_field": "value",
				},
				Constraints: models.ExecutionConstraints{
					Regions: []string{"us-east"},
				},
			},
			expectedModels: nil,
			description:    "Should not modify spec when no models in metadata",
		},
		{
			name: "models already populated",
			spec: &models.JobSpec{
				ID: "test-job-5",
				Models: []models.ModelSpec{
					{ID: "existing-model", Name: "Existing Model", Provider: "golem"},
				},
				Metadata: map[string]interface{}{
					"models": []interface{}{"llama3.2-1b", "mistral-7b"},
				},
				Constraints: models.ExecutionConstraints{
					Regions: []string{"us-east"},
				},
			},
			expectedModels: []models.ModelSpec{
				{ID: "existing-model", Name: "Existing Model", Provider: "golem"},
			},
			description: "Should not modify spec when models already populated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy to avoid modifying the original
			specCopy := *tt.spec
			if tt.spec.Models != nil {
				specCopy.Models = make([]models.ModelSpec, len(tt.spec.Models))
				copy(specCopy.Models, tt.spec.Models)
			}

			processor.NormalizeModelsFromMetadata(&specCopy)

			if tt.expectedModels == nil {
				assert.Nil(t, specCopy.Models, tt.description)
			} else {
				require.Equal(t, len(tt.expectedModels), len(specCopy.Models), tt.description)
				for i, expected := range tt.expectedModels {
					assert.Equal(t, expected.ID, specCopy.Models[i].ID, "Model ID mismatch at index %d", i)
					assert.Equal(t, expected.Name, specCopy.Models[i].Name, "Model Name mismatch at index %d", i)
					assert.Equal(t, expected.Provider, specCopy.Models[i].Provider, "Model Provider mismatch at index %d", i)
					assert.Equal(t, expected.Regions, specCopy.Models[i].Regions, "Model Regions mismatch at index %d", i)
				}
			}
		})
	}
}

func TestNormalizeModelsFromMetadata_EdgeCases(t *testing.T) {
	processor := NewJobSpecProcessor()

	t.Run("nil metadata", func(t *testing.T) {
		spec := &models.JobSpec{
			ID:       "test-job",
			Metadata: nil,
			Constraints: models.ExecutionConstraints{
				Regions: []string{"us-east"},
			},
		}

		processor.NormalizeModelsFromMetadata(spec)
		assert.Nil(t, spec.Models)
	})

	t.Run("empty models array", func(t *testing.T) {
		spec := &models.JobSpec{
			ID: "test-job",
			Metadata: map[string]interface{}{
				"models": []interface{}{},
			},
			Constraints: models.ExecutionConstraints{
				Regions: []string{"us-east"},
			},
		}

		processor.NormalizeModelsFromMetadata(spec)
		assert.Empty(t, spec.Models)
	})

	t.Run("object with empty id", func(t *testing.T) {
		spec := &models.JobSpec{
			ID: "test-job",
			Metadata: map[string]interface{}{
				"models": []interface{}{
					map[string]interface{}{"id": "", "name": "Empty ID Model"},
				},
			},
			Constraints: models.ExecutionConstraints{
				Regions: []string{"us-east"},
			},
		}

		processor.NormalizeModelsFromMetadata(spec)
		assert.Empty(t, spec.Models, "Should skip objects with empty ID")
	})
}
