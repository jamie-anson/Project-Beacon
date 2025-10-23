package hybrid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

// Client is a minimal HTTP client for the Hybrid Router
// Example base: https://project-beacon-production.up.railway.app
// Endpoints used: POST /inference, GET /health (optionally by caller)
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// trace context key and helpers
type traceIDKey struct{}

// WithTraceID returns a context carrying the provided trace ID
func WithTraceID(ctx context.Context, traceID string) context.Context {
	if traceID == "" {
		return ctx
	}
	return context.WithValue(ctx, traceIDKey{}, traceID)
}

// getTraceID extracts a trace ID from context, if present
func getTraceID(ctx context.Context) string {
	if v := ctx.Value(traceIDKey{}); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func New(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://project-beacon-production.up.railway.app"
	}
	// Determine HTTP timeout: default 300s (5 min) for Modal cold starts, overridable via env
	timeoutSec := 300
	envVarUsed := "default"
	envVarValue := ""

	if v := os.Getenv("HYBRID_ROUTER_TIMEOUT"); v != "" {
		envVarValue = v
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeoutSec = n
			envVarUsed = "HYBRID_ROUTER_TIMEOUT"
		} else {
			envVarUsed = "HYBRID_ROUTER_TIMEOUT (invalid)"
		}
	} else if v := os.Getenv("HYBRID_TIMEOUT"); v != "" {
		envVarValue = v
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeoutSec = n
			envVarUsed = "HYBRID_TIMEOUT"
		} else {
			envVarUsed = "HYBRID_TIMEOUT (invalid)"
		}
	}

	// LOG THE ACTUAL TIMEOUT VALUE BEING USED
	fmt.Printf("[HYBRID_CLIENT_INIT] timeout=%ds source=%s env_value=%q url=%s\n",
		timeoutSec, envVarUsed, envVarValue, baseURL)

	return &Client{
		baseURL:    trimRightSlash(baseURL),
		httpClient: &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
	}
}

func trimRightSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}

// InferenceRequest matches the Hybrid Router schema
type InferenceRequest struct {
	Model            string  `json:"model"`
	Prompt           string  `json:"prompt"`
	Temperature      float64 `json:"temperature"`
	MaxTokens        int     `json:"max_tokens"`
	RegionPreference string  `json:"region_preference,omitempty"`
	CostPriority     bool    `json:"cost_priority"`
}

// InferenceResponse is a subset of the router response
type InferenceResponse struct {
	Success      bool                   `json:"success"`
	Response     string                 `json:"response"`
	Error        string                 `json:"error"`
	ProviderUsed string                 `json:"provider_used"`
	InferenceSec float64                `json:"inference_time"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// Provider represents a provider from the hybrid router
type Provider struct {
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	Region          string  `json:"region"`
	Healthy         bool    `json:"healthy"`
	CostPerSecond   float64 `json:"cost_per_second"`
	AvgLatency      float64 `json:"avg_latency"`
	SuccessRate     float64 `json:"success_rate"`
	LastHealthCheck float64 `json:"last_health_check"` // Changed to float64 to match Railway response
}

// ProvidersResponse is the response from /providers endpoint
type ProvidersResponse struct {
	Providers []Provider `json:"providers"`
}

func (c *Client) RunInference(ctx context.Context, req InferenceRequest) (*InferenceResponse, error) {
	// Try common endpoint variants to be resilient to router path changes
	paths := []string{"/inference", "/api/v1/inference", "/api/inference", "/v1/inference"}
	var lastErr error
	for _, p := range paths {
		out, err := c.postInference(ctx, c.baseURL+p, req)
		if err == nil {
			return out, nil
		}
		// For non-404 HTTP errors (e.g., 429/500), return immediately to preserve status
		if he, ok := IsHybridError(err); ok {
			if he.Type == ErrorTypeHTTP && he.StatusCode != http.StatusNotFound {
				return nil, err
			}
		}

		// If it's a router error (not HTTP error), return the response with the error
		if IsRouterError(err) {
			return out, err
		}

		// If 404, try next path; otherwise, keep the error and continue
		if IsNotFound(err) {
			lastErr = fmt.Errorf("router http 404 on %s: %w", p, err)
			continue
		}
		// Non-404 errors: remember but still try other paths in case of proxy rewrites
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("router inference failed: unknown error")
	}
	return nil, lastErr
}

func (c *Client) postInference(ctx context.Context, url string, req InferenceRequest) (*InferenceResponse, error) {
	b, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, NewNetworkError("failed to create HTTP request", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
    // Propagate distributed trace id if present in context
    if tid := getTraceID(ctx); tid != "" {
        httpReq.Header.Set("X-Trace-Id", tid)
    }

	res, err := c.httpClient.Do(httpReq)
	if err != nil {
		// Check if it's a timeout error
		if ctx.Err() == context.DeadlineExceeded {
			return nil, NewTimeoutError("HTTP request timeout", err)
		}
		return nil, NewNetworkError("HTTP request failed", err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		// Include a snippet of the response body for diagnostics
		body, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return nil, NewHTTPError(res.StatusCode, string(body), url)
	}

	var out InferenceResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, NewJSONError("failed to decode response", err)
	}

	if !out.Success {
		// Propagate router error for higher-level logging/persistence
		msg := out.Error
		if msg == "" {
			msg = "unsuccessful"
		}
		return &out, NewRouterError(msg)
	}
	return &out, nil
}

// GetProviders retrieves the list of available providers from the hybrid router
func (c *Client) GetProviders(ctx context.Context) ([]Provider, error) {
	reqURL := c.baseURL + "/providers"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, NewNetworkError("failed to create HTTP request", err)
	}
	httpReq.Header.Set("Accept", "application/json")

	res, err := c.httpClient.Do(httpReq)
	if err != nil {
		// Detailed error logging for diagnosis
		fmt.Printf("[HYBRID] GetProviders HTTP request failed:\n")
		fmt.Printf("  URL: %s\n", reqURL)
		fmt.Printf("  Error: %v\n", err)
		fmt.Printf("  Error Type: %T\n", err)
		fmt.Printf("  Context Error: %v\n", ctx.Err())
		
		// Check if it's a url.Error with more details
		if urlErr, ok := err.(*url.Error); ok {
			fmt.Printf("  URL Error Op: %s\n", urlErr.Op)
			fmt.Printf("  URL Error URL: %s\n", urlErr.URL)
			fmt.Printf("  URL Error Unwrapped: %v (type: %T)\n", urlErr.Err, urlErr.Err)
		}
		
		if ctx.Err() == context.DeadlineExceeded {
			return nil, NewTimeoutError("HTTP request timeout", err)
		}
		return nil, NewNetworkError("HTTP request failed", err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return nil, NewHTTPError(res.StatusCode, string(body), reqURL)
	}

	var response ProvidersResponse
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, NewJSONError("failed to decode providers response", err)
	}

	return response.Providers, nil
}
