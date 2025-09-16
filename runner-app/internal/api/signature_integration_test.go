package api

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func TestJobSubmission_SignatureIntegration(t *testing.T) {
	// Generate test key pair
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	publicKeyB64 := base64.StdEncoding.EncodeToString(publicKey)

	tests := []struct {
		name           string
		setupJobSpec   func() *models.JobSpec
		expectedStatus int
		expectSuccess  bool
		description    string
	}{
		{
			name: "ValidSignedJobSubmission",
			setupJobSpec: func() *models.JobSpec {
				js := createTestJobSpec()
				js.PublicKey = publicKeyB64
				if err := js.Sign(privateKey); err != nil {
					t.Fatalf("failed to sign job spec: %v", err)
				}
				return js
			},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
			description:    "Valid signed job should be accepted",
		},
		{
			name: "UnsignedJobSubmission",
			setupJobSpec: func() *models.JobSpec {
				js := createTestJobSpec()
				// No signature or public key
				return js
			},
			expectedStatus: http.StatusOK,
			expectSuccess:  true,
			description:    "Unsigned job should be accepted (optional signatures)",
		},
		{
			name: "InvalidSignatureJobSubmission",
			setupJobSpec: func() *models.JobSpec {
				js := createTestJobSpec()
				js.PublicKey = publicKeyB64
				js.Signature = "invalid_signature_data"
				return js
			},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
			description:    "Job with invalid signature should be rejected",
		},
		{
			name: "TamperedSignedJobSubmission",
			setupJobSpec: func() *models.JobSpec {
				js := createTestJobSpec()
				js.PublicKey = publicKeyB64
				
				// Sign the job spec
				if err := js.Sign(privateKey); err != nil {
					t.Fatalf("failed to sign job spec: %v", err)
				}
				
				// Tamper with the content after signing
				js.Benchmark.Name = "tampered_benchmark_name"
				return js
			},
			expectedStatus: http.StatusBadRequest,
			expectSuccess:  false,
			description:    "Job tampered after signing should be rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test router
			router := setupTestRouter()
			
			// Prepare job spec
			jobSpec := tt.setupJobSpec()
			jsonData, err := json.Marshal(jobSpec)
			if err != nil {
				t.Fatalf("failed to marshal job spec: %v", err)
			}

			// Create request
			req, err := http.NewRequest("POST", "/jobs", bytes.NewBuffer(jsonData))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Record response
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d. Response: %s", 
					tt.expectedStatus, w.Code, w.Body.String())
			}

			// Parse response
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}

			// Check success field
			if success, ok := response["success"].(bool); ok {
				if success != tt.expectSuccess {
					t.Errorf("expected success=%v, got success=%v. %s", 
						tt.expectSuccess, success, tt.description)
				}
			} else if tt.expectSuccess {
				// If we expect success but there's no success field, that's an error
				t.Errorf("expected success response but got: %v", response)
			}

			// For successful submissions, check that we got a job ID
			if tt.expectSuccess {
				if jobID, ok := response["id"].(string); !ok || jobID == "" {
					t.Errorf("expected job ID in successful response, got: %v", response)
				}
			}

			// For failed submissions, check that we got an error message
			if !tt.expectSuccess {
				if errorMsg, ok := response["error"].(string); !ok || errorMsg == "" {
					t.Errorf("expected error message in failed response, got: %v", response)
				}
			}
		})
	}
}

func TestJobRetrieval_SignaturePreservation(t *testing.T) {
	// Generate test key pair
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	publicKeyB64 := base64.StdEncoding.EncodeToString(publicKey)

	// Create and sign a job spec
	jobSpec := createTestJobSpec()
	jobSpec.PublicKey = publicKeyB64
	if err := jobSpec.Sign(privateKey); err != nil {
		t.Fatalf("failed to sign job spec: %v", err)
	}

	// Submit the job
	router := setupTestRouter()
	jsonData, err := json.Marshal(jobSpec)
	if err != nil {
		t.Fatalf("failed to marshal job spec: %v", err)
	}

	req, err := http.NewRequest("POST", "/jobs", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("failed to submit job: %d - %s", w.Code, w.Body.String())
	}

	// Parse submission response to get job ID
	var submitResponse map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &submitResponse); err != nil {
		t.Fatalf("failed to parse submit response: %v", err)
	}

	jobID, ok := submitResponse["id"].(string)
	if !ok || jobID == "" {
		t.Fatalf("no job ID in submit response: %v", submitResponse)
	}

	// Retrieve the job
	req, err = http.NewRequest("GET", "/jobs/"+jobID, nil)
	if err != nil {
		t.Fatalf("failed to create get request: %v", err)
	}

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("failed to retrieve job: %d - %s", w.Code, w.Body.String())
	}

	// Parse retrieval response
	var getResponse map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &getResponse); err != nil {
		t.Fatalf("failed to parse get response: %v", err)
	}

	// Check that signature and public key are preserved
	retrievedJobSpec, ok := getResponse["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("no job data in get response: %v", getResponse)
	}

	retrievedSignature, ok := retrievedJobSpec["signature"].(string)
	if !ok || retrievedSignature != jobSpec.Signature {
		t.Errorf("signature not preserved. Expected: %s, Got: %s", 
			jobSpec.Signature, retrievedSignature)
	}

	retrievedPublicKey, ok := retrievedJobSpec["public_key"].(string)
	if !ok || retrievedPublicKey != jobSpec.PublicKey {
		t.Errorf("public key not preserved. Expected: %s, Got: %s", 
			jobSpec.PublicKey, retrievedPublicKey)
	}
}

// Helper function to create a test router with minimal setup
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	
	// Create a simple in-memory store for testing
	store := &mockJobStore{
		jobs: make(map[string]*models.JobSpec),
	}
	
	router := gin.New()
	
	// Add basic job endpoints
	router.POST("/jobs", func(c *gin.Context) {
		var jobSpec models.JobSpec
		if err := c.ShouldBindJSON(&jobSpec); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON", "success": false})
			return
		}

		// Validate and verify the job spec
		validator := models.NewJobSpecValidator()
		if err := validator.ValidateAndVerify(&jobSpec); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "success": false})
			return
		}

		// Store the job
		if err := store.StoreJob(&jobSpec); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store job", "success": false})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"id":      jobSpec.ID,
			"status":  "enqueued",
		})
	})

	router.GET("/jobs/:id", func(c *gin.Context) {
		jobID := c.Param("id")
		
		job, err := store.GetJob(jobID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    job,
		})
	})

	return router
}

// Mock job store for testing
type mockJobStore struct {
	jobs map[string]*models.JobSpec
}

func (s *mockJobStore) StoreJob(job *models.JobSpec) error {
	if job.ID == "" {
		job.ID = "test-job-" + time.Now().Format("20060102150405")
	}
	s.jobs[job.ID] = job
	return nil
}

func (s *mockJobStore) GetJob(id string) (*models.JobSpec, error) {
	if job, exists := s.jobs[id]; exists {
		return job, nil
	}
	return nil, nil
}

// Helper function to create a test job spec
func createTestJobSpec() *models.JobSpec {
	return &models.JobSpec{
		Version: "1.0",
		Benchmark: models.BenchmarkSpec{
			Name:        "signature-test-benchmark",
			Description: "Test benchmark for signature integration testing",
			Container: models.ContainerSpec{
				Image: "test:latest",
				Resources: models.ResourceSpec{
					CPU:    "100m",
					Memory: "128Mi",
				},
			},
			Input: models.InputSpec{
				Type: "prompt",
				Data: map[string]interface{}{"text": "integration test"},
				Hash: "integration-test-hash-12345",
			},
			Scoring: models.ScoringSpec{
				Method:     "similarity",
				Parameters: map[string]interface{}{"threshold": 0.8},
			},
			Metadata: map[string]interface{}{"test_type": "integration"},
		},
		Constraints: models.ExecutionConstraints{
			Regions:         []string{"US"},
			MinRegions:      1,
			MinSuccessRate:  0.67,
			Timeout:         5 * time.Minute,
			ProviderTimeout: 1 * time.Minute,
		},
		Metadata:  map[string]interface{}{"created_by": "integration_test"},
		CreatedAt: time.Now(),
	}
}
