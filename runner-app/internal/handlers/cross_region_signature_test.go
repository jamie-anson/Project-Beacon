package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
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
					"questions": []string{"q1", "q2"}, // Required for bias-detection
				},
				"target_regions":   []string{"US", "EU"},
				"min_regions":      1,
				"min_success_rate": 0.67,
			},
			expectedStatus: http.StatusOK,
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
					"questions": []string{"q1", "q2"}, // Required for bias-detection
					"signature": "some-signature",
				},
				"target_regions":   []string{"US", "EU"},
				"min_regions":      1,
				"min_success_rate": 0.67,
			},
			expectedStatus: http.StatusOK,
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
			handler := &CrossRegionHandlers{
				crossRegionExecutor: nil, // Mock would go here
				crossRegionRepo:     nil, // Mock would go here
				diffEngine:          nil, // Mock would go here
			}

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
			"models":    []string{"llama3.2-1b", "mistral-7b"},
		},
		"target_regions":   []string{"US", "EU"},
		"min_regions":      1,
		"min_success_rate": 0.67,
		"enable_analysis":  true,
	}

	handler := &CrossRegionHandlers{
		crossRegionExecutor: nil,
		crossRegionRepo:     nil,
		diffEngine:          nil,
	}

	router := gin.New()
	router.POST("/jobs/cross-region", handler.SubmitCrossRegionJob)

	body, _ := json.Marshal(portalPayload)
	req := httptest.NewRequest("POST", "/jobs/cross-region", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should accept the payload (even though execution will fail due to nil mocks)
	// We're just testing that the payload structure is accepted
	assert.NotEqual(t, http.StatusBadRequest, w.Code, "Should accept portal-style payload structure")
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
				// Note: JSON unmarshaling won't fail for missing fields,
				// but Gin's ShouldBindJSON with binding:"required" will
				assert.Nil(t, req.JobSpec, "JobSpec should be nil for invalid payload")
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, req.JobSpec, "JobSpec should be present")
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
