package hybrid

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
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
	return &Client{
		baseURL:    trimRightSlash(baseURL),
		httpClient: &http.Client{Timeout: 30 * time.Second},
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
	b, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/inference", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	res, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("router http %d", res.StatusCode)
	}
	var out InferenceResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}
