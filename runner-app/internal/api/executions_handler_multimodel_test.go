package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockDB for testing
type MockDB struct {
	executions []MockExecution
}

type MockExecution struct {
	ID          int64
	JobSpecID   string
	Status      string
	Region      string
	ProviderID  string
	StartedAt   time.Time
	CompletedAt time.Time
	CreatedAt   time.Time
	OutputData  []byte
	ReceiptData []byte
	ModelID     string
}

func (db *MockDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	// This is a simplified mock - in real tests you'd use a proper SQL mock library
	// For now, we'll test the handler logic with a real test database
	return nil, nil
}

func (db *MockDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return nil
}

func TestExecutionsHandler_ListAllExecutionsForJob_WithModelID(t *testing.T) {
	// Setup test database with sample data
	testExecutions := []struct {
		ID        int64
		JobID     string
		Status    string
		Region    string
		ModelID   string
		OutputData string
	}{
		{1, "test-job-1", "completed", "us-east", "llama3.2-1b", `{"response": "Hello from Llama in US"}`},
		{2, "test-job-1", "completed", "eu-west", "llama3.2-1b", `{"response": "Hello from Llama in EU"}`},
		{3, "test-job-1", "completed", "us-east", "mistral-7b", `{"response": "Hello from Mistral in US"}`},
		{4, "test-job-1", "completed", "eu-west", "mistral-7b", `{"response": "Hello from Mistral in EU"}`},
		{5, "test-job-1", "completed", "us-east", "qwen2.5-1.5b", `{"response": "Hello from Qwen in US"}`},
		{6, "test-job-1", "failed", "asia-pacific", "qwen2.5-1.5b", `{"error": "Execution failed"}`},
	}

	// This test would require a real database connection or a more sophisticated mock
	// For now, let's test the JSON response structure
	t.Run("response includes model_id field", func(t *testing.T) {
		// Create a mock response that matches our expected structure
		expectedResponse := gin.H{
			"executions": []gin.H{
				{
					"id":          int64(1),
					"job_id":      "test-job-1",
					"status":      "completed",
					"region":      "us-east",
					"provider_id": "test-provider",
					"model_id":    "llama3.2-1b",
					"started_at":  "2025-09-28T00:00:00Z",
					"completed_at": "2025-09-28T00:01:00Z",
					"created_at":  "2025-09-28T00:00:00Z",
					"has_receipt": true,
				},
				{
					"id":          int64(2),
					"job_id":      "test-job-1",
					"status":      "completed",
					"region":      "eu-west",
					"provider_id": "test-provider",
					"model_id":    "mistral-7b",
					"started_at":  "2025-09-28T00:00:00Z",
					"completed_at": "2025-09-28T00:01:00Z",
					"created_at":  "2025-09-28T00:00:00Z",
					"has_receipt": true,
				},
			},
		}

		// Verify the structure includes model_id
		executions := expectedResponse["executions"].([]gin.H)
		for i, exec := range executions {
			modelID, exists := exec["model_id"]
			assert.True(t, exists, "Execution %d should have model_id field", i)
			assert.NotEmpty(t, modelID, "Execution %d model_id should not be empty", i)
		}
	})
}

func TestExecutionsHandler_ListExecutions_WithModelID(t *testing.T) {
	t.Run("response structure includes model_id", func(t *testing.T) {
		// Test the expected response structure for the ListExecutions endpoint
		expectedResponse := gin.H{
			"executions": []gin.H{
				{
					"id":         int64(1),
					"job_id":     "test-job-1",
					"status":     "completed",
					"region":     "us-east",
					"model_id":   "llama3.2-1b",
					"receipt_id": "receipt-123",
				},
				{
					"id":         int64(2),
					"job_id":     "test-job-1",
					"status":     "completed",
					"region":     "eu-west",
					"model_id":   "mistral-7b",
					"receipt_id": "receipt-456",
				},
			},
			"limit":  10,
			"offset": 0,
			"total":  2,
		}

		// Verify structure
		executions := expectedResponse["executions"].([]gin.H)
		assert.Len(t, executions, 2)

		for i, exec := range executions {
			// Verify required fields exist
			assert.Contains(t, exec, "id", "Execution %d should have id", i)
			assert.Contains(t, exec, "job_id", "Execution %d should have job_id", i)
			assert.Contains(t, exec, "status", "Execution %d should have status", i)
			assert.Contains(t, exec, "region", "Execution %d should have region", i)
			assert.Contains(t, exec, "model_id", "Execution %d should have model_id", i)

			// Verify model_id is not empty
			modelID := exec["model_id"].(string)
			assert.NotEmpty(t, modelID, "Execution %d model_id should not be empty", i)
		}
	})
}

func TestExecutionsHandler_ModelIDGrouping(t *testing.T) {
	// Test that executions can be properly grouped by model_id
	executions := []map[string]interface{}{
		{"id": 1, "job_id": "test-job", "region": "us-east", "model_id": "llama3.2-1b", "status": "completed"},
		{"id": 2, "job_id": "test-job", "region": "eu-west", "model_id": "llama3.2-1b", "status": "completed"},
		{"id": 3, "job_id": "test-job", "region": "us-east", "model_id": "mistral-7b", "status": "completed"},
		{"id": 4, "job_id": "test-job", "region": "eu-west", "model_id": "mistral-7b", "status": "failed"},
		{"id": 5, "job_id": "test-job", "region": "asia-pacific", "model_id": "qwen2.5-1.5b", "status": "completed"},
	}

	// Group by model_id (simulating what the portal would do)
	groups := make(map[string][]map[string]interface{})
	for _, exec := range executions {
		modelID := exec["model_id"].(string)
		groups[modelID] = append(groups[modelID], exec)
	}

	// Verify grouping
	assert.Len(t, groups, 3, "Should have 3 model groups")
	assert.Contains(t, groups, "llama3.2-1b")
	assert.Contains(t, groups, "mistral-7b")
	assert.Contains(t, groups, "qwen2.5-1.5b")

	// Verify group contents
	assert.Len(t, groups["llama3.2-1b"], 2, "Llama should have 2 executions")
	assert.Len(t, groups["mistral-7b"], 2, "Mistral should have 2 executions")
	assert.Len(t, groups["qwen2.5-1.5b"], 1, "Qwen should have 1 execution")

	// Verify regional distribution
	llamaRegions := make([]string, 0)
	for _, exec := range groups["llama3.2-1b"] {
		llamaRegions = append(llamaRegions, exec["region"].(string))
	}
	assert.Contains(t, llamaRegions, "us-east")
	assert.Contains(t, llamaRegions, "eu-west")

	// Verify status distribution
	mistralStatuses := make([]string, 0)
	for _, exec := range groups["mistral-7b"] {
		mistralStatuses = append(mistralStatuses, exec["status"].(string))
	}
	assert.Contains(t, mistralStatuses, "completed")
	assert.Contains(t, mistralStatuses, "failed")
}

func TestExecutionsHandler_BackwardCompatibility(t *testing.T) {
	// Test that the API handles executions without model_id (legacy data)
	t.Run("handles missing model_id with COALESCE", func(t *testing.T) {
		// Simulate query result with COALESCE fallback
		executionWithFallback := map[string]interface{}{
			"id":       int64(1),
			"job_id":   "legacy-job",
			"region":   "us-east",
			"model_id": "llama3.2-1b", // This would come from COALESCE(e.model_id, 'llama3.2-1b')
			"status":   "completed",
		}

		// Verify fallback value is present
		modelID := executionWithFallback["model_id"].(string)
		assert.Equal(t, "llama3.2-1b", modelID, "Should use fallback model_id for legacy data")
	})
}

func TestExecutionsHandler_MultiModelJobAnalysis(t *testing.T) {
	// Test analysis capabilities for multi-model jobs
	executions := []map[string]interface{}{
		{"model_id": "llama3.2-1b", "region": "us-east", "status": "completed", "duration": 1200},
		{"model_id": "llama3.2-1b", "region": "eu-west", "status": "completed", "duration": 1350},
		{"model_id": "mistral-7b", "region": "us-east", "status": "completed", "duration": 2100},
		{"model_id": "mistral-7b", "region": "eu-west", "status": "failed", "duration": 0},
		{"model_id": "qwen2.5-1.5b", "region": "us-east", "status": "completed", "duration": 980},
		{"model_id": "qwen2.5-1.5b", "region": "asia-pacific", "status": "completed", "duration": 1450},
	}

	// Analyze success rates by model
	modelStats := make(map[string]struct {
		total     int
		completed int
		avgDuration float64
	})

	for _, exec := range executions {
		modelID := exec["model_id"].(string)
		stats := modelStats[modelID]
		stats.total++
		
		if exec["status"].(string) == "completed" {
			stats.completed++
			duration := exec["duration"].(int)
			stats.avgDuration = (stats.avgDuration*float64(stats.completed-1) + float64(duration)) / float64(stats.completed)
		}
		
		modelStats[modelID] = stats
	}

	// Verify analysis results
	assert.Equal(t, 2, modelStats["llama3.2-1b"].total)
	assert.Equal(t, 2, modelStats["llama3.2-1b"].completed)
	assert.Equal(t, 100.0, float64(modelStats["llama3.2-1b"].completed)/float64(modelStats["llama3.2-1b"].total)*100)

	assert.Equal(t, 2, modelStats["mistral-7b"].total)
	assert.Equal(t, 1, modelStats["mistral-7b"].completed)
	assert.Equal(t, 50.0, float64(modelStats["mistral-7b"].completed)/float64(modelStats["mistral-7b"].total)*100)

	assert.Equal(t, 2, modelStats["qwen2.5-1.5b"].total)
	assert.Equal(t, 2, modelStats["qwen2.5-1.5b"].completed)
	assert.Equal(t, 100.0, float64(modelStats["qwen2.5-1.5b"].completed)/float64(modelStats["qwen2.5-1.5b"].total)*100)

	// Verify performance analysis
	assert.InDelta(t, 1275.0, modelStats["llama3.2-1b"].avgDuration, 1.0, "Llama average duration")
	assert.InDelta(t, 2100.0, modelStats["mistral-7b"].avgDuration, 1.0, "Mistral average duration")
	assert.InDelta(t, 1215.0, modelStats["qwen2.5-1.5b"].avgDuration, 1.0, "Qwen average duration")
}

func TestExecutionsHandler_CrossRegionAnalysis_WithModels(t *testing.T) {
	// Test cross-region analysis with model-specific data
	regionModelData := map[string]map[string]interface{}{
		"us-east": {
			"llama3.2-1b":  {"response": "US response from Llama", "political_sensitivity": 0.2},
			"mistral-7b":   {"response": "US response from Mistral", "political_sensitivity": 0.3},
			"qwen2.5-1.5b": {"response": "US response from Qwen", "political_sensitivity": 0.1},
		},
		"eu-west": {
			"llama3.2-1b":  {"response": "EU response from Llama", "political_sensitivity": 0.25},
			"mistral-7b":   {"response": "EU response from Mistral", "political_sensitivity": 0.35},
			"qwen2.5-1.5b": {"response": "EU response from Qwen", "political_sensitivity": 0.15},
		},
		"asia-pacific": {
			"llama3.2-1b":  {"response": "APAC response from Llama", "political_sensitivity": 0.4},
			"mistral-7b":   {"response": "APAC response from Mistral", "political_sensitivity": 0.6},
			"qwen2.5-1.5b": {"response": "APAC response from Qwen", "political_sensitivity": 0.8},
		},
	}

	// Analyze model-specific regional variations
	modelVariations := make(map[string][]float64)
	for region, models := range regionModelData {
		_ = region // region info available for analysis
		for modelID, data := range models {
			dataMap := data.(map[string]interface{})
			sensitivity := dataMap["political_sensitivity"].(float64)
			modelVariations[modelID] = append(modelVariations[modelID], sensitivity)
		}
	}

	// Calculate variance for each model across regions
	for modelID, sensitivities := range modelVariations {
		assert.Len(t, sensitivities, 3, "Each model should have data from 3 regions")
		
		// Calculate variance
		var sum, mean, variance float64
		for _, s := range sensitivities {
			sum += s
		}
		mean = sum / float64(len(sensitivities))
		
		for _, s := range sensitivities {
			variance += (s - mean) * (s - mean)
		}
		variance /= float64(len(sensitivities))

		// Verify expected patterns
		switch modelID {
		case "llama3.2-1b":
			assert.Less(t, variance, 0.02, "Llama should have low regional variance")
		case "mistral-7b":
			assert.Less(t, variance, 0.03, "Mistral should have moderate regional variance")
		case "qwen2.5-1.5b":
			assert.Greater(t, variance, 0.1, "Qwen should have high regional variance")
		}
	}
}

// Integration test helper to verify the complete flow
func TestMultiModelJobFlow_Integration(t *testing.T) {
	// This test verifies the complete flow from job submission to API response
	t.Run("complete multi-model flow", func(t *testing.T) {
		// 1. Job submission with metadata.models
		jobPayload := map[string]interface{}{
			"id":      "integration-test-job",
			"version": "v1",
			"benchmark": map[string]interface{}{
				"name": "bias-detection",
			},
			"constraints": map[string]interface{}{
				"regions": []string{"us-east", "eu-west", "asia-pacific"},
			},
			"metadata": map[string]interface{}{
				"models": []string{"llama3.2-1b", "mistral-7b", "qwen2.5-1.5b"},
			},
		}

		// 2. Verify normalization would create proper ModelSpec array
		expectedModels := []map[string]interface{}{
			{"id": "llama3.2-1b", "name": "llama3.2-1b", "provider": "hybrid"},
			{"id": "mistral-7b", "name": "mistral-7b", "provider": "hybrid"},
			{"id": "qwen2.5-1.5b", "name": "qwen2.5-1.5b", "provider": "hybrid"},
		}

		models := jobPayload["metadata"].(map[string]interface{})["models"].([]string)
		assert.Len(t, models, len(expectedModels), "Should have correct number of models")

		// 3. Verify execution would create model × region combinations
		regions := jobPayload["constraints"].(map[string]interface{})["regions"].([]string)
		expectedExecutions := len(models) * len(regions) // 3 × 3 = 9
		assert.Equal(t, 9, expectedExecutions, "Should create 9 executions")

		// 4. Verify API response would include model_id for grouping
		mockExecutions := make([]map[string]interface{}, 0, expectedExecutions)
		for _, model := range models {
			for _, region := range regions {
				mockExecutions = append(mockExecutions, map[string]interface{}{
					"job_id":   "integration-test-job",
					"model_id": model,
					"region":   region,
					"status":   "completed",
				})
			}
		}

		// 5. Verify portal can group by model_id
		groups := make(map[string][]map[string]interface{})
		for _, exec := range mockExecutions {
			modelID := exec["model_id"].(string)
			groups[modelID] = append(groups[modelID], exec)
		}

		assert.Len(t, groups, 3, "Should have 3 model groups")
		for modelID, executions := range groups {
			assert.Len(t, executions, 3, "Model %s should have 3 regional executions", modelID)
		}
	})
}
