package hybrid

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
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

func New(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://project-beacon-production.up.railway.app"
	}
	// Determine HTTP timeout: default 120s, overridable via env
	timeoutSec := 120
	if v := os.Getenv("HYBRID_ROUTER_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeoutSec = n
		}
	} else if v := os.Getenv("HYBRID_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeoutSec = n
		}
	}
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

func (c *Client) RunInference(ctx context.Context, req InferenceRequest) (*InferenceResponse, error) {
	// Try common endpoint variants to be resilient to router path changes
	paths := []string{"/inference", "/api/v1/inference", "/api/inference", "/v1/inference"}
	var lastErr error
	for _, p := range paths {
		out, err := c.postInference(ctx, c.baseURL+p, req)
		if err == nil {
			return out, nil
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

