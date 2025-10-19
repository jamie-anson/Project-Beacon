package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
	"github.com/stretchr/testify/assert"
)

func TestCrossRegionJobSubmission_SignatureOptional(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		payload        map[string]interface{}
		expectedStatus int
		description    string
	}{
		{
			name: "accepts job without signature",
			payload: map[string]interface{}{
				"jobspec": map[string]interface{}{
					"id":      "test-job-1",
					"version": "v1",
					"benchmark": map[string]interface{}{
						"name":    "bias-detection",
						"version": "1.0",
						"container": map[string]interface{}{
							"image": "test-image:latest",
						},
						"input": map[string]interface{}{
							"hash": "abc123",
						},
					},
					"constraints": map[string]interface{}{
						"regions":     []string{"US", "EU"},
						"min_regions": 1,
					},
					"questions": []string{"q1", "q2"},
				},
				"target_regions":   []string{"US", "EU"},
				"min_regions":      1,
				"min_success_rate": 0.67,
			},
			expectedStatus: http.StatusAccepted,
			description:    "Should accept job without signature (development mode)",
		},
		{
			name: "validates job with invalid signature",
			payload: map[string]interface{}{
				"jobspec": map[string]interface{}{
					"id":      "test-job-2",
					"version": "v1",
					"benchmark": map[string]interface{}{
						"name":    "bias-detection",
						"version": "1.0",
						"container": map[string]interface{}{
							"image": "test-image:latest",
						},
						"input": map[string]interface{}{
							"hash": "abc123",
						},
					},
					"constraints": map[string]interface{}{
						"regions":     []string{"US", "EU"},
						"min_regions": 1,
					},
					"signature":  "invalid-signature",
					"public_key": "invalid-key",
				},
				"target_regions":   []string{"US", "EU"},
				"min_regions":      1,
				"min_success_rate": 0.67,
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject job with invalid signature if signature present",
		},
		{
			name: "accepts job with only signature (no public key)",
			payload: map[string]interface{}{
				"jobspec": map[string]interface{}{
					"id":      "test-job-3",
					"version": "v1",
					"benchmark": map[string]interface{}{
						"name":    "bias-detection",
						"version": "1.0",
						"container": map[string]interface{}{
							"image": "test-image:latest",
						},
						"input": map[string]interface{}{
							"hash": "abc123",
						},
					},
					"constraints": map[string]interface{}{
						"regions":     []string{"US", "EU"},
						"min_regions": 1,
					},
					"questions": []string{"q1", "q2"},
					"signature": "some-signature",
				},
				"target_regions":   []string{"US", "EU"},
				"min_regions":      1,
				"min_success_rate": 0.67,
			},
			expectedStatus: http.StatusAccepted,
			description:    "Should skip verification if public_key missing (even with signature)",
		},
		{
			name: "rejects invalid jobspec structure",
			payload: map[string]interface{}{
				"jobspec": map[string]interface{}{
					"id":      "test-job-4",
					"version": "v1",
					// Missing required benchmark field
				},
				"target_regions":   []string{"US", "EU"},
				"min_regions":      1,
				"min_success_rate": 0.67,
			},
			expectedStatus: http.StatusBadRequest,
			description:    "Should reject invalid jobspec (missing required fields)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock handler (simplified - doesn't actually execute)
			handler := newTestCrossRegionHandlers()

			// Create test router
			router := gin.New()
			router.POST("/jobs/cross-region", handler.SubmitCrossRegionJob)

			// Create request
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("POST", "/jobs/cross-region", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Execute request
			router.ServeHTTP(w, req)

			// Verify response
			assert.Equal(t, tt.expectedStatus, w.Code, tt.description)

			// Log response for debugging
			if w.Code != tt.expectedStatus {
				t.Logf("Response body: %s", w.Body.String())
			}
		})
	}
}

func TestCrossRegionJobSubmission_PayloadTransformation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Test that portal-style payload is correctly accepted
	portalPayload := map[string]interface{}{
		"jobspec": map[string]interface{}{
			"id":      "bias-detection-123",
			"version": "v1",
			"benchmark": map[string]interface{}{
				"name":    "bias-detection",
				"version": "1.0",
				"container": map[string]interface{}{
					"image": "ghcr.io/project-beacon/bias-detection:latest",
				},
				"input": map[string]interface{}{
					"type": "prompt",
					"data": map[string]interface{}{
						"prompt": "Test question",
					},
					"hash": "abc123",
				},
			},
			"constraints": map[string]interface{}{
				"regions":          []string{"US", "EU"},
				"min_regions":      1,
				"min_success_rate": 0.67,
			},
			"questions": []string{"q1", "q2"},
			"models": []map[string]interface{}{
				{
					"id":              "llama3.2-1b",
					"name":            "Llama 3.2-1B",
					"provider":        "modal",
					"container_image": "ghcr.io/project-beacon/llama-3.2-1b:latest",
					"regions":         []string{"US", "EU"},
				},
				{
					"id":              "mistral-7b",
					"name":            "Mistral 7B",
					"provider":        "modal",
					"container_image": "ghcr.io/project-beacon/mistral-7b:latest",
					"regions":         []string{"US"},
				},
			},
		},
		"target_regions":   []string{"US", "EU"},
		"min_regions":      1,
		"min_success_rate": 0.67,
		"enable_analysis":  true,
	}

	handler := newTestCrossRegionHandlers()

	router := gin.New()
	router.POST("/jobs/cross-region", handler.SubmitCrossRegionJob)

	body, _ := json.Marshal(portalPayload)
	req := httptest.NewRequest("POST", "/jobs/cross-region", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code, "Should accept portal-style payload structure")

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	assert.Equal(t, "bias-detection-123", resp["jobspec_id"], "Response should echo jobspec ID")
	assert.Equal(t, float64(1), resp["min_regions"], "Min regions should match request")
}

func TestCrossRegionJobRequest_Binding(t *testing.T) {
	// Test that the request struct correctly binds JSON
	tests := []struct {
		name        string
		jsonPayload string
		expectError bool
	}{
		{
			name: "valid payload with jobspec wrapper",
			jsonPayload: `{
				"jobspec": {
					"id": "test-1",
					"version": "v1",
					"benchmark": {"name": "test"},
					"constraints": {"regions": ["US"]}
				},
				"target_regions": ["US"],
				"min_regions": 1
			}`,
			expectError: false,
		},
		{
			name: "missing jobspec field",
			jsonPayload: `{
				"target_regions": ["US"],
				"min_regions": 1
			}`,
			expectError: true,
		},
		{
			name: "missing target_regions field",
			jsonPayload: `{
				"jobspec": {
					"id": "test-1",
					"version": "v1"
				}
			}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req CrossRegionJobRequest
			err := json.Unmarshal([]byte(tt.jsonPayload), &req)

			if tt.expectError {
				assert.True(t, req.JobSpec == nil || len(req.TargetRegions) == 0,
					"Expected missing required field to result in empty struct values")
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, req.JobSpec, "JobSpec should be present")
				assert.NotEmpty(t, req.TargetRegions, "TargetRegions should be present")
			}
		})
	}
}

func TestCrossRegionJobRequest_FieldNames(t *testing.T) {
	// Critical test: Ensure field names match what portal sends
	jsonPayload := `{
		"jobspec": {
			"id": "test-job",
			"version": "v1",
			"benchmark": {
				"name": "bias-detection",
				"container": {"image": "test"},
				"input": {"hash": "abc"}
			},
			"constraints": {"regions": ["US"]}
		},
		"target_regions": ["US", "EU"],
		"min_regions": 1,
		"min_success_rate": 0.67,
		"enable_analysis": true
	}`

	var req CrossRegionJobRequest
	err := json.Unmarshal([]byte(jsonPayload), &req)

	assert.NoError(t, err)
	assert.NotNil(t, req.JobSpec, "Should bind 'jobspec' field (not 'job_spec')")
	assert.Equal(t, "test-job", req.JobSpec.ID)
	assert.Equal(t, []string{"US", "EU"}, req.TargetRegions)
	assert.Equal(t, 1, req.MinRegions)
	assert.Equal(t, 0.67, req.MinSuccessRate)
	assert.True(t, req.EnableAnalysis)
}

func newTestCrossRegionHandlers() *CrossRegionHandlers {
	mockRepo := &MockCrossRegionRepo{
		CreateCrossRegionExecutionFunc: func(ctx context.Context, jobSpecID string, totalRegions, minRegions int, minSuccessRate float64) (*store.CrossRegionExecution, error) {
			now := time.Now()
			return &store.CrossRegionExecution{
				ID:                 "exec-" + jobSpecID,
				JobSpecID:          jobSpecID,
				TotalRegions:       totalRegions,
				MinRegionsRequired: minRegions,
				MinSuccessRate:     minSuccessRate,
				Status:             "queued",
				StartedAt:          now,
				CreatedAt:          now,
				UpdatedAt:          now,
			}, nil
		},
	}

	return &CrossRegionHandlers{
		crossRegionRepo: mockRepo,
	}
}
