package golem

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// NewYagnaRESTClient constructs a Yagna REST client implementation.
func NewYagnaRESTClient(baseURL, appKey string, httpClient *http.Client) YagnaClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	// Normalize base URL (trim trailing slash)
	b := strings.TrimRight(baseURL, "/")
	return &YagnaRESTClient{
		BaseURL:      b,
		AppKey:       appKey,
		HTTPClient:   httpClient,
		Timeout:      10 * time.Second,
		MarketBase:   "/market-api/v1",
		ActivityBase: "/activity-api/v1",
	}
}

// withTimeout returns a context with a bounded timeout derived from client settings.
func (c *YagnaRESTClient) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	to := c.Timeout
	if to <= 0 {
		to = 10 * time.Second
	}
	return context.WithTimeout(ctx, to)
}

// doReq performs an HTTP request with basic retry/backoff for transient failures.
func (c *YagnaRESTClient) doReq(ctx context.Context, req *http.Request) (*http.Response, []byte, error) {
	// Attach auth header if provided
	if c.AppKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.AppKey)
	}
	attempts := 3
	backoff := 200 * time.Millisecond
	for i := 0; i < attempts; i++ {
		ctx2, cancel := c.withTimeout(ctx)
		req = req.WithContext(ctx2)
		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			cancel()
			if i == attempts-1 {
				return nil, nil, err
			}
			time.Sleep(backoff)
			backoff *= 2
			continue
		}
		defer cancel()
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		// Retry on 429/5xx
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			if i == attempts-1 {
				return resp, body, fmt.Errorf("yagna http %d: %s", resp.StatusCode, string(body))
			}
			time.Sleep(backoff)
			backoff *= 2
			continue
		}
		return resp, body, readErr
	}
	return nil, nil, errors.New("unreachable retry loop")
}

// Probe best-effort checks connectivity. We keep this permissive to avoid blocking non-SDK flows.
func (c *YagnaRESTClient) Probe(ctx context.Context) (string, map[string]any, error) {
	// Try a lightweight call; ignore errors and return minimal success so SDK discovery can proceed in dev.
	req, _ := http.NewRequest(http.MethodGet, c.BaseURL+"/version", nil)
	resp, body, err := c.doReq(ctx, req)
	if err != nil || resp.StatusCode >= 400 {
		// Fall back to reporting base path with empty version map; caller can decide what to do.
		return "/", map[string]any{}, nil
	}
	var v map[string]any
	_ = json.Unmarshal(body, &v)
	return "/version", v, nil
}

// CreateDemand is a placeholder; real implementation can be added later.
func (c *YagnaRESTClient) CreateDemand(ctx context.Context, spec DemandSpec) (string, error) {
	return "", errors.New("CreateDemand not implemented in MVP client")
}

// NegotiateAgreement is a placeholder; real implementation can be added later.
func (c *YagnaRESTClient) NegotiateAgreement(ctx context.Context, demandID string) (string, error) {
	return "", errors.New("NegotiateAgreement not implemented in MVP client")
}

// CreateActivity is a placeholder; real implementation can be added later.
func (c *YagnaRESTClient) CreateActivity(ctx context.Context, agreementID string, jobspec *models.JobSpec) (string, error) {
	return "", errors.New("CreateActivity not implemented in MVP client")
}

// Exec is a placeholder; real implementation can be added later.
func (c *YagnaRESTClient) Exec(ctx context.Context, activityID string, jobspec *models.JobSpec) (string, string, int, error) {
	return "", "", -1, errors.New("Exec not implemented in MVP client")
}

// StopActivity is a placeholder; real implementation can be added later.
func (c *YagnaRESTClient) StopActivity(ctx context.Context, activityID string) error {
	return errors.New("StopActivity not implemented in MVP client")
}
