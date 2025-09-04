package e2e

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

// Env helpers
func baseURL(t *testing.T) string {
	v := os.Getenv("BASE_URL")
	if v == "" {
		t.Skip("BASE_URL not set; skip E2E")
	}
	return v
}

func adminToken() string { return os.Getenv("ADMIN_TOKEN") }

// Test 1: Health check
func Test_Health(t *testing.T) {
	url := baseURL(t) + "/health"
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("health request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("health expected 200, got %d, body=%s", resp.StatusCode, string(b))
	}
}

// Test 2: Admin config visible (if ADMIN_TOKEN provided)
func Test_Config(t *testing.T) {
	if adminToken() == "" {
		t.Skip("ADMIN_TOKEN not set; skip config check")
	}
	req, _ := http.NewRequest("GET", baseURL(t)+"/admin/config", nil)
	req.Header.Set("X-Admin-Token", adminToken())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("config request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("config expected 200, got %d, body=%s", resp.StatusCode, string(b))
	}
}

// Helper to POST a signed jobspec (JSON)
func postJobspec(t *testing.T, body []byte) *http.Response {
	req, _ := http.NewRequest("POST", baseURL(t)+"/api/v1/jobs", bytes.NewReader(body))
	req.Header.Set("content-type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("submit failed: %v", err)
	}
	return resp
}

// Test 3: Submit signed job + replay (requires E2E_SIGNED_JOB or E2E_SIGNED_JOB_FILE)
func Test_JobSubmit_And_Replay(t *testing.T) {
	payload := os.Getenv("E2E_SIGNED_JOB")
	file := os.Getenv("E2E_SIGNED_JOB_FILE")
	var signed []byte
	var err error
	if payload != "" {
		// allow plain JSON or base64-encoded
		if json.Valid([]byte(payload)) {
			signed = []byte(payload)
		} else {
			signed, err = base64.StdEncoding.DecodeString(payload)
			if err != nil {
				t.Fatalf("E2E_SIGNED_JOB not valid JSON or base64: %v", err)
			}
		}
	} else if file != "" {
		signed, err = os.ReadFile(file)
		if err != nil { t.Fatalf("read E2E_SIGNED_JOB_FILE: %v", err) }
	} else {
		t.Skip("E2E_SIGNED_JOB(_FILE) not set; skip submit/replay test")
	}

	// 1st submit -> 202
	resp := postJobspec(t, signed)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("submit expected 202, got %d, body=%s", resp.StatusCode, string(b))
	}

	// 2nd submit -> 400 replay_detected
	resp2 := postJobspec(t, signed)
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp2.Body)
		t.Fatalf("replay expected 400, got %d, body=%s", resp2.StatusCode, string(b))
	}
}

// Test 4: Poll latest receipt for a job id (optional)
func Test_PollLatestReceipt(t *testing.T) {
	jobID := os.Getenv("JOB_ID")
	if jobID == "" {
		t.Skip("JOB_ID not set; skip receipt polling")
	}
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL(t) + "/api/v1/jobs/" + jobID + "?include=latest")
		if err == nil {
			if resp.StatusCode == http.StatusOK {
				b, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				// Basic smoke assertion
				if bytes.Contains(b, []byte("\"executions\"")) || bytes.Contains(b, []byte("\"status\"")) {
					return
				}
			}
			resp.Body.Close()
		}
		time.Sleep(3 * time.Second)
	}
	t.Fatalf("no receipt observed for %s within timeout", jobID)
}
