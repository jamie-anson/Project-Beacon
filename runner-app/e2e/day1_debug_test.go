package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

// Test the complete job lifecycle: submit -> enqueue -> execute -> receipt
func Test_Day1_JobLifecycle_Debug(t *testing.T) {
	if baseURL(t) == "" {
		t.Skip("BASE_URL not set")
	}

	// Step 1: Create a minimal test job
	testJob := map[string]interface{}{
		"id":      "day1-debug-test",
		"version": "1.0",
		"benchmark": map[string]interface{}{
			"name":        "Day-1 Debug Test",
			"description": "Debug test for job lifecycle",
			"container": map[string]interface{}{
				"image":   "busybox",
				"tag":     "latest",
				"command": []string{"sh", "-c", "echo 'DEBUG TEST SUCCESS' && date"},
				"resources": map[string]interface{}{
					"cpu":    "500m",
					"memory": "256Mi",
				},
			},
			"input": map[string]interface{}{
				"type": "command",
				"data": map[string]interface{}{
					"message": "debug test",
				},
				"hash": "sha256:debug-test-hash",
			},
			"scoring": map[string]interface{}{
				"method":     "success_rate",
				"parameters": map[string]interface{}{},
			},
			"metadata": map[string]interface{}{},
		},
		"constraints": map[string]interface{}{
			"regions":     []string{"US"},
			"min_regions": 1,
			"timeout":     600000000000,
			"providers":   []interface{}{},
		},
		"metadata": map[string]interface{}{
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"nonce":     fmt.Sprintf("debug-test-nonce-%d", time.Now().Unix()),
		},
	}

	jobJSON, err := json.Marshal(testJob)
	if err != nil {
		t.Fatalf("Failed to marshal test job: %v", err)
	}

	// Step 2: Submit unsigned job (should fail with proper error)
	t.Log("Step 2: Testing unsigned job submission...")
	resp := postJobspec(t, jobJSON)
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	t.Logf("Unsigned job response: %d, body: %s", resp.StatusCode, string(body))
	
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 for unsigned job, got %d", resp.StatusCode)
	}

	// Step 3: Check existing jobs and their status
	t.Log("Step 3: Checking existing jobs...")
	resp2, err := http.Get(baseURL(t) + "/api/v1/jobs?limit=10")
	if err != nil {
		t.Fatalf("Failed to get jobs: %v", err)
	}
	defer resp2.Body.Close()
	
	jobsBody, _ := io.ReadAll(resp2.Body)
	t.Logf("Existing jobs: %s", string(jobsBody))

	// Parse jobs to check status
	var jobsResp map[string]interface{}
	if err := json.Unmarshal(jobsBody, &jobsResp); err == nil {
		if jobs, ok := jobsResp["jobs"].([]interface{}); ok {
			for _, job := range jobs {
				if jobMap, ok := job.(map[string]interface{}); ok {
					jobID := jobMap["id"]
					status := jobMap["status"]
					t.Logf("Job %s: status=%s", jobID, status)
				}
			}
		}
	}

	// Step 4: Check specific job details with executions
	t.Log("Step 4: Checking job execution details...")
	if jobsResp != nil {
		if jobs, ok := jobsResp["jobs"].([]interface{}); ok && len(jobs) > 0 {
			if firstJob, ok := jobs[0].(map[string]interface{}); ok {
				if jobID, ok := firstJob["id"].(string); ok {
					execURL := fmt.Sprintf("%s/api/v1/jobs/%s?include=executions", baseURL(t), jobID)
					resp3, err := http.Get(execURL)
					if err != nil {
						t.Logf("Failed to get job executions: %v", err)
					} else {
						defer resp3.Body.Close()
						execBody, _ := io.ReadAll(resp3.Body)
						t.Logf("Job %s executions: %s", jobID, string(execBody))
					}
				}
			}
		}
	}
}

// Test queue worker status and Redis connectivity
func Test_Day1_QueueWorker_Debug(t *testing.T) {
	if baseURL(t) == "" {
		t.Skip("BASE_URL not set")
	}

	// Check health with detailed service status
	t.Log("Checking detailed health status...")
	resp, err := http.Get(baseURL(t) + "/health")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	defer resp.Body.Close()
	
	healthBody, _ := io.ReadAll(resp.Body)
	t.Logf("Health status: %s", string(healthBody))

	// Check readiness
	resp2, err := http.Get(baseURL(t) + "/health/ready")
	if err != nil {
		t.Fatalf("Readiness check failed: %v", err)
	}
	defer resp2.Body.Close()
	
	readyBody, _ := io.ReadAll(resp2.Body)
	t.Logf("Readiness status: %s", string(readyBody))
}

// Test transparency system
func Test_Day1_Transparency_Debug(t *testing.T) {
	if baseURL(t) == "" {
		t.Skip("BASE_URL not set")
	}

	// Check transparency root
	t.Log("Checking transparency root...")
	resp, err := http.Get(baseURL(t) + "/api/v1/transparency/root")
	if err != nil {
		t.Fatalf("Transparency root failed: %v", err)
	}
	defer resp.Body.Close()
	
	rootBody, _ := io.ReadAll(resp.Body)
	t.Logf("Transparency root: %s", string(rootBody))

	// Check proof endpoint
	t.Log("Checking transparency proof...")
	resp2, err := http.Get(baseURL(t) + "/api/v1/transparency/proof?index=0")
	if err != nil {
		t.Logf("Transparency proof request failed: %v", err)
	} else {
		defer resp2.Body.Close()
		proofBody, _ := io.ReadAll(resp2.Body)
		t.Logf("Transparency proof: %s", string(proofBody))
	}
}

// Test admin endpoints (if admin token available)
func Test_Day1_Admin_Debug(t *testing.T) {
	if adminToken() == "" {
		t.Skip("ADMIN_TOKEN not set")
	}

	// Check admin config
	t.Log("Checking admin config...")
	req, _ := http.NewRequest("GET", baseURL(t)+"/admin/config", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Admin config failed: %v", err)
	}
	defer resp.Body.Close()
	
	configBody, _ := io.ReadAll(resp.Body)
	t.Logf("Admin config response: %d, body: %s", resp.StatusCode, string(configBody))

	// Check admin flags
	t.Log("Checking admin flags...")
	req2, _ := http.NewRequest("GET", baseURL(t)+"/admin/flags", nil)
	req2.Header.Set("Authorization", "Bearer "+adminToken())
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Logf("Admin flags request failed: %v", err)
	} else {
		defer resp2.Body.Close()
		flagsBody, _ := io.ReadAll(resp2.Body)
		t.Logf("Admin flags response: %d, body: %s", resp2.StatusCode, string(flagsBody))
	}

	// Check admin hints
	t.Log("Checking admin hints...")
	resp3, err := http.Get(baseURL(t) + "/admin/hints")
	if err != nil {
		t.Logf("Admin hints request failed: %v", err)
	} else {
		defer resp3.Body.Close()
		hintsBody, _ := io.ReadAll(resp3.Body)
		t.Logf("Admin hints: %s", string(hintsBody))
	}
}

// Test job status polling with timeout
func Test_Day1_JobStatus_Polling(t *testing.T) {
	if baseURL(t) == "" {
		t.Skip("BASE_URL not set")
	}

	// Get existing jobs first
	resp, err := http.Get(baseURL(t) + "/api/v1/jobs?limit=5")
	if err != nil {
		t.Fatalf("Failed to get jobs: %v", err)
	}
	defer resp.Body.Close()
	
	jobsBody, _ := io.ReadAll(resp.Body)
	var jobsResp map[string]interface{}
	if err := json.Unmarshal(jobsBody, &jobsResp); err != nil {
		t.Fatalf("Failed to parse jobs response: %v", err)
	}

	jobs, ok := jobsResp["jobs"].([]interface{})
	if !ok || len(jobs) == 0 {
		t.Skip("No jobs found to poll")
	}

	// Poll first job for 30 seconds
	firstJob := jobs[0].(map[string]interface{})
	jobID := firstJob["id"].(string)
	initialStatus := firstJob["status"].(string)
	
	t.Logf("Polling job %s (initial status: %s) for 30 seconds...", jobID, initialStatus)
	
	deadline := time.Now().Add(30 * time.Second)
	statusChanges := []string{initialStatus}
	
	for time.Now().Before(deadline) {
		jobURL := fmt.Sprintf("%s/api/v1/jobs/%s?include=executions", baseURL(t), jobID)
		resp, err := http.Get(jobURL)
		if err != nil {
			t.Logf("Failed to poll job: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}
		
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		
		var jobResp map[string]interface{}
		if err := json.Unmarshal(body, &jobResp); err == nil {
			if job, ok := jobResp["job"].(map[string]interface{}); ok {
				if statusVal, ok := job["status"]; ok && statusVal != nil {
					currentStatus := statusVal.(string)
					if currentStatus != statusChanges[len(statusChanges)-1] {
						statusChanges = append(statusChanges, currentStatus)
						t.Logf("Job %s status changed: %s", jobID, currentStatus)
					}
				}
			}
			
			// Check for executions
			if executions, ok := jobResp["executions"]; ok && executions != nil {
				t.Logf("Job %s has executions: %v", jobID, executions)
			}
		}
		
		time.Sleep(2 * time.Second)
	}
	
	t.Logf("Job %s status changes over 30s: %v", jobID, statusChanges)
}
