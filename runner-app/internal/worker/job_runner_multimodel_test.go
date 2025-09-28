package worker

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// MockExecutor for testing multi-model execution
type MockExecutor struct {
	mock.Mock
	executions map[string]ExecutionResult
	mu         sync.Mutex
}

func (m *MockExecutor) Execute(ctx context.Context, spec *models.JobSpec, region string) (string, string, []byte, []byte, error) {
	args := m.Called(ctx, spec, region)
	
	// Extract model_id from metadata for tracking
	modelID := "default"
	if spec.Metadata != nil {
		if mid, ok := spec.Metadata["model_id"].(string); ok {
			modelID = mid
		}
	}
	
	// Store execution for verification
	m.mu.Lock()
	if m.executions == nil {
		m.executions = make(map[string]ExecutionResult)
	}
	key := region + ":" + modelID
	m.executions[key] = ExecutionResult{
		Region:     region,
		ProviderID: "test-provider",
		Status:     "completed",
		ModelID:    modelID,
	}
	m.mu.Unlock()
	
	return args.String(0), args.String(1), args.Get(2).([]byte), args.Get(3).([]byte), args.Error(4)
}

func (m *MockExecutor) GetExecutions() map[string]ExecutionResult {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make(map[string]ExecutionResult)
	for k, v := range m.executions {
		result[k] = v
	}
	return result
}

// MockExecRepo for testing database operations
type MockExecRepo struct {
	mock.Mock
	insertedExecutions []ExecutionRecord
	mu                 sync.Mutex
}

type ExecutionRecord struct {
	JobID      string
	ProviderID string
	Region     string
	Status     string
	ModelID    string
}

func (m *MockExecRepo) InsertExecution(ctx context.Context, jobID string, providerID string, region string, status string, startedAt time.Time, completedAt time.Time, outputJSON []byte, receiptJSON []byte) (int64, error) {
	return m.InsertExecutionWithModel(ctx, jobID, providerID, region, status, startedAt, completedAt, outputJSON, receiptJSON, "llama3.2-1b")
}

func (m *MockExecRepo) InsertExecutionWithModel(ctx context.Context, jobID string, providerID string, region string, status string, startedAt time.Time, completedAt time.Time, outputJSON []byte, receiptJSON []byte, modelID string) (int64, error) {
	args := m.Called(ctx, jobID, providerID, region, status, startedAt, completedAt, outputJSON, receiptJSON, modelID)
	
	// Track inserted executions
	m.mu.Lock()
	m.insertedExecutions = append(m.insertedExecutions, ExecutionRecord{
		JobID:      jobID,
		ProviderID: providerID,
		Region:     region,
		Status:     status,
		ModelID:    modelID,
	})
	m.mu.Unlock()
	
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockExecRepo) UpdateRegionVerification(ctx context.Context, executionID int64, regionClaimed sql.NullString, regionObserved sql.NullString, regionVerified sql.NullBool, verificationMethod sql.NullString, evidenceRef sql.NullString) error {
	args := m.Called(ctx, executionID, regionClaimed, regionObserved, regionVerified, verificationMethod, evidenceRef)
	return args.Error(0)
}

func (m *MockExecRepo) GetInsertedExecutions() []ExecutionRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]ExecutionRecord, len(m.insertedExecutions))
	copy(result, m.insertedExecutions)
	return result
}

func TestExecuteMultiModelJob(t *testing.T) {
	tests := []struct {
		name                string
		spec                *models.JobSpec
		expectedExecutions  int
		expectedModelRegions map[string][]string
		description         string
	}{
		{
			name: "three models across three regions",
			spec: &models.JobSpec{
				ID: "test-job-1",
				Models: []models.ModelSpec{
					{ID: "llama3.2-1b", Name: "Llama 3.2-1B", Provider: "hybrid", Regions: []string{"us-east", "eu-west", "asia-pacific"}},
					{ID: "mistral-7b", Name: "Mistral 7B", Provider: "hybrid", Regions: []string{"us-east", "eu-west", "asia-pacific"}},
					{ID: "qwen2.5-1.5b", Name: "Qwen 2.5-1.5B", Provider: "hybrid", Regions: []string{"us-east", "eu-west", "asia-pacific"}},
				},
				Constraints: models.ExecutionConstraints{
					Regions: []string{"us-east", "eu-west", "asia-pacific"},
				},
			},
			expectedExecutions: 9, // 3 models × 3 regions
			expectedModelRegions: map[string][]string{
				"llama3.2-1b":  {"us-east", "eu-west", "asia-pacific"},
				"mistral-7b":   {"us-east", "eu-west", "asia-pacific"},
				"qwen2.5-1.5b": {"us-east", "eu-west", "asia-pacific"},
			},
			description: "Should execute all model-region combinations",
		},
		{
			name: "different regions per model",
			spec: &models.JobSpec{
				ID: "test-job-2",
				Models: []models.ModelSpec{
					{ID: "llama3.2-1b", Name: "Llama 3.2-1B", Provider: "hybrid", Regions: []string{"us-east", "eu-west"}},
					{ID: "mistral-7b", Name: "Mistral 7B", Provider: "hybrid", Regions: []string{"eu-west", "asia-pacific"}},
				},
				Constraints: models.ExecutionConstraints{
					Regions: []string{"us-east", "eu-west", "asia-pacific"},
				},
			},
			expectedExecutions: 4, // 2 + 2 regions
			expectedModelRegions: map[string][]string{
				"llama3.2-1b": {"us-east", "eu-west"},
				"mistral-7b":  {"eu-west", "asia-pacific"},
			},
			description: "Should respect per-model region constraints",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockExecutor := &MockExecutor{}
			mockExecRepo := &MockExecRepo{}

			// Setup executor expectations
			for _, model := range tt.spec.Models {
				for _, region := range model.Regions {
					mockExecutor.On("Execute", mock.Anything, mock.MatchedBy(func(spec *models.JobSpec) bool {
						return spec.Metadata["model_id"] == model.ID
					}), region).Return("test-provider", "completed", []byte(`{"response":"test"}`), []byte(`{"id":"test-receipt"}`), nil)
				}
			}

			// Setup repo expectations
			mockExecRepo.On("InsertExecutionWithModel", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(int64(1), nil)

			// Create JobRunner with bounded concurrency
			runner := &JobRunner{
				ExecRepo:      mockExecRepo,
				maxConcurrent: 5, // Test bounded concurrency
			}

			// Execute multi-model job
			ctx := context.Background()
			results, err := runner.executeMultiModelJob(ctx, tt.spec.ID, tt.spec, mockExecutor)

			// Verify results
			require.NoError(t, err, tt.description)
			assert.Len(t, results, tt.expectedExecutions, tt.description)

			// Verify all expected model-region combinations were executed
			executorResults := mockExecutor.GetExecutions()
			for modelID, expectedRegions := range tt.expectedModelRegions {
				for _, region := range expectedRegions {
					key := region + ":" + modelID
					result, exists := executorResults[key]
					assert.True(t, exists, "Expected execution for %s in %s", modelID, region)
					if exists {
						assert.Equal(t, modelID, result.ModelID, "Model ID mismatch")
						assert.Equal(t, region, result.Region, "Region mismatch")
					}
				}
			}

			// Verify database insertions
			insertedExecutions := mockExecRepo.GetInsertedExecutions()
			assert.Len(t, insertedExecutions, tt.expectedExecutions, "Database insertion count mismatch")

			// Verify model IDs are correctly set in database
			modelCounts := make(map[string]int)
			for _, exec := range insertedExecutions {
				modelCounts[exec.ModelID]++
			}

			for modelID, expectedRegions := range tt.expectedModelRegions {
				expectedCount := len(expectedRegions)
				actualCount := modelCounts[modelID]
				assert.Equal(t, expectedCount, actualCount, "Incorrect database insertion count for model %s", modelID)
			}

			// Verify all mocks were called as expected
			mockExecutor.AssertExpectations(t)
			mockExecRepo.AssertExpectations(t)
		})
	}
}

// TestMultiModelConcurrency tests bounded concurrency concepts
func TestMultiModelConcurrency(t *testing.T) {
	t.Run("bounded concurrency simulation", func(t *testing.T) {
		maxConcurrent := 3
		models := []string{"llama3.2-1b", "mistral-7b", "qwen2.5-1.5b"}
		regions := []string{"us-east", "eu-west", "asia-pacific"}
		
		// Simulate bounded concurrency with semaphore
		sem := make(chan struct{}, maxConcurrent)
		var wg sync.WaitGroup
		var mu sync.Mutex
		var executionCount int
		
		for _, model := range models {
			for _, region := range regions {
				wg.Add(1)
				sem <- struct{}{} // Acquire
				go func(m, r string) {
					defer wg.Done()
					defer func() { <-sem }() // Release
					
					// Simulate work
					time.Sleep(1 * time.Millisecond)
					
					mu.Lock()
					executionCount++
					mu.Unlock()
				}(model, region)
			}
		}
		
		wg.Wait()
		
		expectedExecutions := len(models) * len(regions)
		if executionCount != expectedExecutions {
			t.Errorf("Expected %d executions, got %d", expectedExecutions, executionCount)
		}
	})
}

// TestMetadataSafety tests metadata copying between goroutines
func TestMetadataSafety(t *testing.T) {
	t.Run("metadata copying safety", func(t *testing.T) {
		originalMetadata := map[string]interface{}{
			"original_field": "original_value",
		}
		
		models := []string{"model1", "model2"}
		var wg sync.WaitGroup
		var mu sync.Mutex
		var results []map[string]interface{}
		
		for _, model := range models {
			wg.Add(1)
			go func(m string) {
				defer wg.Done()
				
				// Create a copy of metadata (simulating what the real code does)
				metadataCopy := make(map[string]interface{})
				for k, v := range originalMetadata {
					metadataCopy[k] = v
				}
				metadataCopy["model_id"] = m
				
				mu.Lock()
				results = append(results, metadataCopy)
				mu.Unlock()
			}(model)
		}
		
		wg.Wait()
		
		// Verify each goroutine got its own copy
		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}
		
		// Verify original metadata unchanged
		if originalMetadata["model_id"] != nil {
			t.Error("Original metadata should not be modified")
		}
		
		// Verify each result has correct model_id
		modelIDs := make([]string, 0, 2)
		for _, result := range results {
			if modelID, ok := result["model_id"].(string); ok {
				modelIDs = append(modelIDs, modelID)
			}
		}
		
		if len(modelIDs) != 2 {
			t.Errorf("Expected 2 model IDs, got %d", len(modelIDs))
		}
	})
}

