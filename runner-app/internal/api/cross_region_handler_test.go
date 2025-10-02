package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTransformCrossRegionData(t *testing.T) {
	executions := []map[string]interface{}{
		{
			"id":          int64(1),
			"job_id":      "test-job",
			"region":      "us-east",
			"status":      "completed",
			"provider_id": "modal-us-001",
			"model_id":    "llama3.2-1b",
			"output": map[string]interface{}{
				"response": "Test response",
			},
			"started_at": time.Now(),
			"created_at": time.Now(),
		},
	}

	result := transformCrossRegionData(executions, "test-job", "llama3.2-1b")

	assert.NotNil(t, result)
	assert.Contains(t, result, "cross_region_execution")
	assert.Contains(t, result, "region_results")
	assert.Contains(t, result, "analysis")
	assert.Contains(t, result, "summary")

	// Verify cross_region_execution
	crossRegion := result["cross_region_execution"].(map[string]interface{})
	assert.Equal(t, "test-job", crossRegion["job_id"])
	assert.Equal(t, "llama3.2-1b", crossRegion["model_id"])

	// Verify region results
	regionResults := result["region_results"].([]map[string]interface{})
	assert.Equal(t, 1, len(regionResults))
	assert.Equal(t, "us-east", regionResults[0]["region"])
	assert.Equal(t, "US", regionResults[0]["region_code"])
	assert.Equal(t, "United States", regionResults[0]["region_name"])
	assert.Equal(t, "🇺🇸", regionResults[0]["flag"])
}

func TestTransformCrossRegionData_PartialFailures(t *testing.T) {
	executions := []map[string]interface{}{
		{
			"id":          int64(1),
			"region":      "us-east",
			"status":      "completed",
			"provider_id": "modal-us-001",
			"output": map[string]interface{}{
				"response": "Success",
			},
			"started_at": time.Now(),
			"created_at": time.Now(),
		},
		{
			"id":          int64(2),
			"region":      "eu-west",
			"status":      "failed",
			"provider_id": "modal-eu-001",
			"started_at":  time.Now(),
			"created_at":  time.Now(),
		},
	}

	result := transformCrossRegionData(executions, "test-job", "llama3.2-1b")

	summary := result["summary"].(map[string]interface{})
	assert.Equal(t, 2, summary["total_regions"])
	assert.Equal(t, 1, summary["completed_regions"])
	assert.Equal(t, 1, summary["failed_regions"])

	// Verify status is "partial"
	crossRegion := result["cross_region_execution"].(map[string]interface{})
	assert.Equal(t, "partial", crossRegion["status"])
}

func TestCalculateVariance(t *testing.T) {
	tests := []struct {
		name     string
		scores   []float64
		expected float64
	}{
		{
			name:     "empty scores",
			scores:   []float64{},
			expected: 0,
		},
		{
			name:     "single score",
			scores:   []float64{0.5},
			expected: 0,
		},
		{
			name:     "identical scores",
			scores:   []float64{0.5, 0.5, 0.5},
			expected: 0,
		},
		{
			name:     "varied scores",
			scores:   []float64{0.1, 0.2, 0.3},
			expected: 0.01, // Variance of [0.1, 0.2, 0.3] is ~0.0067, rounded to 0.01
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateVariance(tt.scores)
			assert.InDelta(t, tt.expected, result, 0.01)
		})
	}
}

func TestRegionMetadata(t *testing.T) {
	tests := []struct {
		region       string
		expectedCode string
		expectedName string
		expectedFlag string
	}{
		{"us-east", "US", "United States", "🇺🇸"},
		{"eu-west", "EU", "Europe", "🇪🇺"},
		{"asia-pacific", "ASIA", "Asia Pacific", "🌏"},
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			meta, ok := regionMetadata[tt.region]
			assert.True(t, ok)
			assert.Equal(t, tt.expectedCode, meta.Code)
			assert.Equal(t, tt.expectedName, meta.Name)
			assert.Equal(t, tt.expectedFlag, meta.Flag)
		})
	}
}
