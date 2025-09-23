package hybrid

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_RunInference_Success(t *testing.T) {
	// Create a test server that returns a successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/inference" {
			http.NotFound(w, r)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"success": true,
			"response": "Hello, world!",
			"provider_used": "test-provider",
			"inference_time": 0.5,
			"metadata": {"test": "value"}
		}`))
	}))
	defer server.Close()

	client := New(server.URL)
	req := InferenceRequest{
		Model:       "test-model",
		Prompt:      "Hello",
		Temperature: 0.7,
		MaxTokens:   100,
	}

	resp, err := client.RunInference(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !resp.Success {
		t.Error("Expected Success to be true")
	}

	if resp.Response != "Hello, world!" {
		t.Errorf("Expected Response to be 'Hello, world!', got %v", resp.Response)
	}

	if resp.ProviderUsed != "test-provider" {
		t.Errorf("Expected ProviderUsed to be 'test-provider', got %v", resp.ProviderUsed)
	}
}

func TestClient_RunInference_RouterError(t *testing.T) {
	// Create a test server that returns a router error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"success": false,
			"error": "Model not available",
			"response": "",
			"provider_used": ""
		}`))
	}))
	defer server.Close()

	client := New(server.URL)
	req := InferenceRequest{
		Model:  "unavailable-model",
		Prompt: "Hello",
	}

	resp, err := client.RunInference(context.Background(), req)
	
	// Should return response even on router error (our implementation does this)
	if resp == nil {
		t.Fatalf("Expected response to be non-nil even on router error, got error: %v", err)
	}

	if resp.Success {
		t.Error("Expected Success to be false")
	}

	// Check that we get a router error
	if err == nil {
		t.Fatal("Expected error for router failure")
	}
	
	if !IsRouterError(err) {
		t.Errorf("Expected router error, got %v", err)
	}
}

func TestClient_RunInference_HTTPError(t *testing.T) {
	// Create a test server that returns HTTP 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := New(server.URL)
	req := InferenceRequest{
		Model:  "test-model",
		Prompt: "Hello",
	}

	_, err := client.RunInference(context.Background(), req)
	
	if err == nil {
		t.Fatal("Expected error for HTTP 500")
	}

	// Check that we get an HTTP error
	if !IsHTTPStatus(err, 500) {
		t.Errorf("Expected HTTP 500 error, got %v", err)
	}

	hybridErr, ok := IsHybridError(err)
	if !ok {
		t.Fatal("Expected HybridError")
	}

	if hybridErr.StatusCode != 500 {
		t.Errorf("Expected StatusCode 500, got %v", hybridErr.StatusCode)
	}
}

func TestClient_RunInference_NotFound(t *testing.T) {
	// Create a test server that returns 404 for all paths
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	client := New(server.URL)
	req := InferenceRequest{
		Model:  "test-model",
		Prompt: "Hello",
	}

	_, err := client.RunInference(context.Background(), req)
	
	if err == nil {
		t.Fatal("Expected error for 404")
	}

	// Should try multiple paths and eventually fail with a wrapped NotFound error
	// The error message shows it tried multiple paths and got 404s
	if err.Error() == "" {
		t.Error("Expected non-empty error message")
	}
}

func TestClient_RunInference_PathFallback(t *testing.T) {
	// Create a test server that only responds to /api/v1/inference
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/inference" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"success": true,
				"response": "Success on v1 API",
				"provider_used": "test-provider"
			}`))
			return
		}
		
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	client := New(server.URL)
	req := InferenceRequest{
		Model:  "test-model",
		Prompt: "Hello",
	}

	resp, err := client.RunInference(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.Response != "Success on v1 API" {
		t.Errorf("Expected response from v1 API, got %v", resp.Response)
	}
}

func TestClient_RunInference_Timeout(t *testing.T) {
	// Create a test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	client := New(server.URL)
	client.httpClient.Timeout = 50 * time.Millisecond // Short timeout

	req := InferenceRequest{
		Model:  "test-model",
		Prompt: "Hello",
	}

	_, err := client.RunInference(context.Background(), req)
	
	if err == nil {
		t.Fatal("Expected timeout error")
	}

	// Should be a network error (timeout is handled as network error in our implementation)
	hybridErr, ok := IsHybridError(err)
	if !ok {
		t.Fatal("Expected HybridError")
	}

	if hybridErr.Type != ErrorTypeNetwork {
		t.Errorf("Expected network error type, got %v", hybridErr.Type)
	}
}

func TestClient_RunInference_InvalidJSON(t *testing.T) {
	// Create a test server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	client := New(server.URL)
	req := InferenceRequest{
		Model:  "test-model",
		Prompt: "Hello",
	}

	_, err := client.RunInference(context.Background(), req)
	
	if err == nil {
		t.Fatal("Expected JSON error")
	}

	// Check that we get a JSON error
	hybridErr, ok := IsHybridError(err)
	if !ok {
		t.Fatal("Expected HybridError")
	}

	if hybridErr.Type != ErrorTypeJSON {
		t.Errorf("Expected JSON error type, got %v", hybridErr.Type)
	}
}

func TestNew_DefaultURL(t *testing.T) {
	client := New("")
	
	expected := "https://project-beacon-production.up.railway.app"
	if client.baseURL != expected {
		t.Errorf("Expected default baseURL to be %v, got %v", expected, client.baseURL)
	}
}

func TestClient_RunInference_503(t *testing.T) {
	// Create a test server that returns HTTP 503 Service Unavailable
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/inference" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{
			"error": "Service Unavailable",
			"message": "Temporarily unable to process requests",
			"retry_after": 60
		}`))
	}))
	defer server.Close()

	client := New(server.URL)
	req := InferenceRequest{
		Model:       "test-model",
		Prompt:      "Hello",
		Temperature: 0.7,
		MaxTokens:   100,
	}

	_, err := client.RunInference(context.Background(), req)

	if err == nil {
		t.Fatal("Expected error for HTTP 503")
	}

	// Check that we get an HTTP 503 error
	if !IsHTTPStatus(err, 503) {
		t.Errorf("Expected HTTP 503 error, got %v", err)
	}

	hybridErr, ok := IsHybridError(err)
	if !ok {
		t.Fatal("Expected HybridError")
	}

	if hybridErr.StatusCode != 503 {
		t.Errorf("Expected StatusCode 503, got %v", hybridErr.StatusCode)
	}

	if hybridErr.Type != ErrorTypeHTTP {
		t.Errorf("Expected HTTP error type, got %v", hybridErr.Type)
	}
}

func TestClient_RunInference_429(t *testing.T) {
	// Create a test server that returns HTTP 429 Too Many Requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/inference" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Limit", "100")
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", "1640995200")
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{
			"error": "Too Many Requests",
			"message": "Rate limit exceeded",
			"retry_after": 60
		}`))
	}))
	defer server.Close()

	client := New(server.URL)
	req := InferenceRequest{
		Model:       "test-model",
		Prompt:      "Hello",
		Temperature: 0.7,
		MaxTokens:   100,
	}

	_, err := client.RunInference(context.Background(), req)

	if err == nil {
		t.Fatal("Expected error for HTTP 429")
	}

	// Check that we get an HTTP 429 error
	if !IsHTTPStatus(err, 429) {
		t.Errorf("Expected HTTP 429 error, got %v", err)
	}

	hybridErr, ok := IsHybridError(err)
	if !ok {
		t.Fatal("Expected HybridError")
	}

	if hybridErr.StatusCode != 429 {
		t.Errorf("Expected StatusCode 429, got %v", hybridErr.StatusCode)
	}

	if hybridErr.Type != ErrorTypeHTTP {
		t.Errorf("Expected HTTP error type, got %v", hybridErr.Type)
	}

	// Note: HybridError doesn't store rate limit headers in this implementation
	// The test verifies the basic 429 error handling without checking header preservation
}
