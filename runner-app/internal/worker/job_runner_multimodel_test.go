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

func (m *MockExecRepo) InsertExecutionWithModelAndQuestion(ctx context.Context, jobID string, providerID string, region string, status string, startedAt time.Time, completedAt time.Time, outputJSON []byte, receiptJSON []byte, modelID string, questionID string) (int64, error) {
	args := m.Called(ctx, jobID, providerID, region, status, startedAt, completedAt, outputJSON, receiptJSON, modelID, questionID)

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
		name                 string
		spec                 *models.JobSpec
		expectedExecutions   int
		expectedModelRegions map[string][]string
		description          string
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

// TestExecuteMultiModelJob_SequentialQuestions tests sequential question batching
func TestExecuteMultiModelJob_SequentialQuestions(t *testing.T) {
	t.Run("sequential question execution with multiple models", func(t *testing.T) {
		// Setup mocks
		mockExecutor := &MockExecutor{}
		mockExecRepo := &MockExecRepo{}

		spec := &models.JobSpec{
			ID:        "test-job-sequential",
			Questions: []string{"Q1", "Q2", "Q3"},
			Models: []models.ModelSpec{
				{ID: "llama3.2-1b", Name: "Llama 3.2-1B", Provider: "hybrid", Regions: []string{"us-east", "eu-west"}},
				{ID: "qwen2.5-1.5b", Name: "Qwen 2.5-1.5B", Provider: "hybrid", Regions: []string{"us-east", "eu-west"}},
			},
			Metadata: make(map[string]interface{}),
		}

		// Setup executor expectations - should be called 12 times (3 questions × 2 models × 2 regions)
		mockExecutor.On("Execute", mock.Anything, mock.Anything, mock.Anything).
			Return("test-provider", "completed", []byte(`{"response":"test"}`), []byte(`{"id":"test-receipt"}`), nil)

		// Setup repo expectations
		mockExecRepo.On("InsertExecutionWithModel", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(int64(1), nil)

		// Create JobRunner
		runner := &JobRunner{
			ExecRepo:      mockExecRepo,
			maxConcurrent: 10,
		}

		// Execute
		ctx := context.Background()
		results, err := runner.executeMultiModelJob(ctx, spec.ID, spec, mockExecutor)

		// Verify
		require.NoError(t, err)
		assert.Equal(t, 12, len(results), "Expected 12 executions (3 questions × 2 models × 2 regions)")

		// Verify all results have question_id set
		questionCounts := make(map[string]int)
		for _, result := range results {
			assert.NotEmpty(t, result.QuestionID, "QuestionID should be populated")
			assert.Contains(t, []string{"Q1", "Q2", "Q3"}, result.QuestionID, "QuestionID should be one of the test questions")
			questionCounts[result.QuestionID]++
		}

		// Verify each question has 4 executions (2 models × 2 regions)
		for _, question := range spec.Questions {
			assert.Equal(t, 4, questionCounts[question], "Each question should have 4 executions")
		}

		// Verify all model IDs are set
		for _, result := range results {
			assert.NotEmpty(t, result.ModelID, "ModelID should be populated")
			assert.Contains(t, []string{"llama3.2-1b", "qwen2.5-1.5b"}, result.ModelID)
		}

		// Verify executions were called correct number of times
		mockExecutor.AssertNumberOfCalls(t, "Execute", 12)
		mockExecRepo.AssertNumberOfCalls(t, "InsertExecutionWithModel", 12)
	})
}

// TestExecuteMultiModelJob_QuestionBatchTiming tests that questions execute sequentially
func TestExecuteMultiModelJob_QuestionBatchTiming(t *testing.T) {
	t.Run("questions execute in order", func(t *testing.T) {
		mockExecutor := &MockExecutor{}
		mockExecRepo := &MockExecRepo{}

		var executionOrder []string
		var orderMu sync.Mutex

		spec := &models.JobSpec{
			ID:        "test-job-timing",
			Questions: []string{"Q1", "Q2"},
			Models: []models.ModelSpec{
				{ID: "llama3.2-1b", Regions: []string{"us-east"}},
			},
			Metadata: make(map[string]interface{}),
		}

		// Track execution order
		mockExecutor.On("Execute", mock.Anything, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				spec := args.Get(1).(*models.JobSpec)
				if len(spec.Questions) > 0 {
					orderMu.Lock()
					executionOrder = append(executionOrder, spec.Questions[0])
					orderMu.Unlock()
				}
			}).
			Return("test-provider", "completed", []byte(`{}`), []byte(`{}`), nil)

		mockExecRepo.On("InsertExecutionWithModel", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(int64(1), nil)

		runner := &JobRunner{
			ExecRepo:      mockExecRepo,
			maxConcurrent: 10,
		}

		ctx := context.Background()
		_, err := runner.executeMultiModelJob(ctx, spec.ID, spec, mockExecutor)

		require.NoError(t, err)

		// Verify Q1 executions come before Q2 executions
		orderMu.Lock()
		defer orderMu.Unlock()

		assert.Equal(t, 2, len(executionOrder), "Should have 2 executions")
		if len(executionOrder) == 2 {
			// All Q1 should complete before Q2 starts
			assert.Equal(t, "Q1", executionOrder[0], "First execution should be Q1")
			assert.Equal(t, "Q2", executionOrder[1], "Second execution should be Q2")
		}
	})
}

// TestExecuteMultiModelJob_BoundedConcurrencyPerQuestion tests concurrency limits
func TestExecuteMultiModelJob_BoundedConcurrencyPerQuestion(t *testing.T) {
	t.Run("respects semaphore limit per question", func(t *testing.T) {
		mockExecutor := &MockExecutor{}
		mockExecRepo := &MockExecRepo{}

		var maxConcurrent int
		var currentConcurrent int
		var concurrencyMu sync.Mutex

		spec := &models.JobSpec{
			ID:        "test-job-concurrency",
			Questions: []string{"Q1"},
			Models: []models.ModelSpec{
				{ID: "model1", Regions: []string{"r1", "r2", "r3"}},
				{ID: "model2", Regions: []string{"r1", "r2", "r3"}},
				{ID: "model3", Regions: []string{"r1", "r2", "r3"}},
			},
			Metadata: make(map[string]interface{}),
		}

		// Track concurrent executions
		mockExecutor.On("Execute", mock.Anything, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				concurrencyMu.Lock()
				currentConcurrent++
				if currentConcurrent > maxConcurrent {
					maxConcurrent = currentConcurrent
				}
				concurrencyMu.Unlock()

				time.Sleep(10 * time.Millisecond) // Simulate work

				concurrencyMu.Lock()
				currentConcurrent--
				concurrencyMu.Unlock()
			}).
			Return("test-provider", "completed", []byte(`{}`), []byte(`{}`), nil)

		mockExecRepo.On("InsertExecutionWithModel", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(int64(1), nil)

		runner := &JobRunner{
			ExecRepo:      mockExecRepo,
			maxConcurrent: 5, // Limit to 5 concurrent
		}

		ctx := context.Background()
		_, err := runner.executeMultiModelJob(ctx, spec.ID, spec, mockExecutor)

		require.NoError(t, err)

		// Verify max concurrent never exceeded semaphore limit
		concurrencyMu.Lock()
		defer concurrencyMu.Unlock()
		assert.LessOrEqual(t, maxConcurrent, 5, "Should not exceed semaphore limit of 5")
	})
}

// TestExecuteMultiModelJob_ContextCancellation tests graceful cancellation
func TestExecuteMultiModelJob_ContextCancellation(t *testing.T) {
	t.Run("handles context cancellation gracefully", func(t *testing.T) {
		mockExecutor := &MockExecutor{}
		mockExecRepo := &MockExecRepo{}

		spec := &models.JobSpec{
			ID:        "test-job-cancel",
			Questions: []string{"Q1", "Q2", "Q3"},
			Models: []models.ModelSpec{
				{ID: "llama3.2-1b", Regions: []string{"us-east"}},
			},
			Metadata: make(map[string]interface{}),
		}

		// Create cancellable context
		ctx, cancel := context.WithCancel(context.Background())

		var executionCount int
		var countMu sync.Mutex

		mockExecutor.On("Execute", mock.Anything, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				countMu.Lock()
				executionCount++
				count := executionCount
				countMu.Unlock()

				// Cancel after first execution
				if count == 1 {
					time.Sleep(50 * time.Millisecond)
					cancel()
				}
			}).
			Return("test-provider", "completed", []byte(`{}`), []byte(`{}`), nil)

		mockExecRepo.On("InsertExecutionWithModel", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(int64(1), nil)

		runner := &JobRunner{
			ExecRepo:      mockExecRepo,
			maxConcurrent: 10,
		}

		results, err := runner.executeMultiModelJob(ctx, spec.ID, spec, mockExecutor)

		// Should complete without error (cancellation is graceful)
		require.NoError(t, err)

		// Some results should have "cancelled" status
		cancelledCount := 0
		for _, result := range results {
			if result.Status == "cancelled" {
				cancelledCount++
			}
		}

		// At least some executions should be cancelled
		assert.Greater(t, cancelledCount, 0, "Some executions should be cancelled")
	})
}

// TestExecutionResult_QuestionIDPopulated tests QuestionID field
func TestExecutionResult_QuestionIDPopulated(t *testing.T) {
	t.Run("ExecutionResult has QuestionID field", func(t *testing.T) {
		result := ExecutionResult{
			Region:      "us-east",
			ModelID:     "llama3.2-1b",
			QuestionID:  "test-question",
			Status:      "completed",
			ProviderID:  "test-provider",
			StartedAt:   time.Now(),
			CompletedAt: time.Now(),
		}

		assert.Equal(t, "test-question", result.QuestionID)
		assert.Equal(t, "llama3.2-1b", result.ModelID)
		assert.Equal(t, "us-east", result.Region)
	})
}
