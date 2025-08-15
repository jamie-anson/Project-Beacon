package golem

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// NewYagnaRESTClient constructs a YagnaRESTClient with sane defaults reusing Service's HTTP settings.
func NewYagnaRESTClient(baseURL, appKey string, httpClient *http.Client) *YagnaRESTClient {
	c := &YagnaRESTClient{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		AppKey:     appKey,
		HTTPClient: httpClient,
		Timeout:    15 * time.Second,
	}
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{Timeout: c.Timeout}
	}
	return c
}

// withTimeout ensures a deadline is set for outgoing calls when caller hasn't set one.
func (c *YagnaRESTClient) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	to := c.Timeout
	if to <= 0 {
		to = 15 * time.Second
	}
	return context.WithTimeout(ctx, to)
}

// doReq performs an HTTP request with simple retry/backoff on transient errors.
func (c *YagnaRESTClient) doReq(ctx context.Context, method, url string, body []byte, contentType string) (status int, respBody []byte, err error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()

	maxAttempts := 3
	backoff := 200 * time.Millisecond

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, reqErr := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
		if reqErr != nil {
			err = reqErr
			break
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		if c.AppKey != "" {
			req.Header.Set("Authorization", "Bearer "+c.AppKey)
		}

		resp, doErr := c.HTTPClient.Do(req)
		if doErr != nil {
			// network/timeout error: retry
			if attempt < maxAttempts {
				select {
				case <-time.After(backoff):
					backoff *= 2
					continue
				case <-ctx.Done():
					return 0, nil, ctx.Err()
				}
			}
			return 0, nil, doErr
		}

		b, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
		_ = resp.Body.Close()

		status = resp.StatusCode
		respBody = b

		if status >= 200 && status < 300 {
			return status, respBody, nil
		}

		// Retry on 429 or 5xx
		if (status == 429 || (status >= 500 && status <= 599)) && attempt < maxAttempts {
			select {
			case <-time.After(backoff):
				backoff *= 2
				continue
			case <-ctx.Done():
				return status, respBody, ctx.Err()
			}
		}
		// Non-retryable or maxed out
		return status, respBody, fmt.Errorf("http %s %s -> %d: %s", method, url, status, strings.TrimSpace(string(respBody)))
	}
	return status, respBody, err
}

// EnsureDiscover lazy-discovers Market and Activity base prefixes (e.g. /market-api/v1, /ya-market/v1).
func (c *YagnaRESTClient) EnsureDiscover(ctx context.Context) {
    // Only discover once per client instance.
    if c.MarketBase != "" && c.ActivityBase != "" {
        return
    }
    marketCandidates := []string{"/market-api/v1", "/ya-market/v1", "/market/v1", "/market"}
    activityCandidates := []string{"/activity-api/v1", "/ya-activity/v1", "/activity", "/activities"}

    // Accept non-404 (e.g. 200/401/405) as a sign the base exists.
    test := func(path string) bool {
        status, _, err := c.doReq(ctx, http.MethodOptions, c.BaseURL+path, nil, "")
        if err != nil {
            return false
        }
        return status != http.StatusNotFound
    }

    for _, b := range marketCandidates {
        // Try base and a common subpath (scan) to avoid false positives
        if test(b) || test(b+"/scan") {
            c.MarketBase = strings.TrimRight(b, "/")
            break
        }
    }
    for _, b := range activityCandidates {
        if test(b) || test(b+"/") {
            c.ActivityBase = strings.TrimRight(b, "/")
            break
        }
    }
}

func (c *YagnaRESTClient) Probe(ctx context.Context) (string, map[string]any, error) {
	// Prefer discovering usable bases over relying on /version.
	c.EnsureDiscover(ctx)
	var info map[string]any
	if c.MarketBase != "" {
		// Try multiple GET candidates. Some Yagna setups 404 on base but expose '/scan'.
		candidates := []string{
			c.MarketBase + "/scan",
			c.MarketBase + "/",
			c.MarketBase,
		}
		for _, p := range candidates {
			status, b, _ := c.doReq(ctx, http.MethodGet, c.BaseURL+p, nil, "")
			switch {
			case status >= 200 && status < 300:
				_ = json.Unmarshal(b, &info)
				return p, info, nil
			case status == http.StatusUnauthorized || status == http.StatusMethodNotAllowed:
				// 401/405 imply the resource exists (auth/method needed)
				return p, nil, nil
			case status == http.StatusNotFound:
				continue
			}
		}
		// Discovery via OPTIONS may have succeeded; avoid false negative.
		return c.MarketBase + "/", nil, nil
	}
	// Fallback legacy probes
	probePaths := []string{"/version", "/"}
	var lastStatus int
	var lastBody string
	for _, p := range probePaths {
		status, b, err := c.doReq(ctx, http.MethodGet, c.BaseURL+p, nil, "")
		if err != nil {
			continue
		}
		lastStatus = status
		lastBody = string(b)
		if status >= 200 && status < 300 {
			_ = json.Unmarshal(b, &info)
			return p, info, nil
		}
	}
	return "", nil, fmt.Errorf("yagna probe non-2xx: %d: %s", lastStatus, lastBody)
}

func (c *YagnaRESTClient) CreateDemand(ctx context.Context, spec DemandSpec) (string, error) {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(spec); err != nil {
		return "", fmt.Errorf("encode demand: %w", err)
	}
	// Discover market base and try candidates.
	c.EnsureDiscover(ctx)
	endpoints := []string{}
	if c.MarketBase != "" {
		endpoints = append(endpoints,
			c.MarketBase+"/demands",
		)
	}
	// Legacy fallbacks
	endpoints = append(endpoints, "/market/demands", "/demands")
	for _, ep := range endpoints {
		url := c.BaseURL + ep
		status, b, err := c.doReq(ctx, http.MethodPost, url, buf.Bytes(), "application/json")
		if err != nil {
			continue
		}
		if status >= 200 && status < 300 {
			var m map[string]any
			if json.Unmarshal(b, &m) == nil {
				if v, ok := m["id"].(string); ok {
					return v, nil
				}
			}
			return string(b), nil // fallback
		}
		return "", fmt.Errorf("createDemand: %s -> %d: %s", url, status, strings.TrimSpace(string(b)))
	}
	return "", fmt.Errorf("createDemand: no known endpoints succeeded")
}

func (c *YagnaRESTClient) NegotiateAgreement(ctx context.Context, demandID string) (string, error) {
	c.EnsureDiscover(ctx)
	endpoints := []string{}
	if c.MarketBase != "" {
		endpoints = append(endpoints,
			fmt.Sprintf("%s/demands/%s/agree", c.MarketBase, demandID),
		)
	}
	// Legacy fallbacks
	endpoints = append(endpoints,
		fmt.Sprintf("/market/demands/%s/agree", demandID),
		fmt.Sprintf("/demands/%s/agree", demandID),
	)
	for _, ep := range endpoints {
		url := c.BaseURL + ep
		status, b, err := c.doReq(ctx, http.MethodPost, url, nil, "")
		if err != nil {
			continue
		}
		if status >= 200 && status < 300 {
			var m map[string]any
			if json.Unmarshal(b, &m) == nil {
				if v, ok := m["id"].(string); ok {
					return v, nil
				}
			}
			return string(b), nil // fallback
		}
		return "", fmt.Errorf("negotiateAgreement: %s -> %d: %s", url, status, strings.TrimSpace(string(b)))
	}
	return "", fmt.Errorf("negotiateAgreement: no known endpoints succeeded")
}

func (c *YagnaRESTClient) CreateActivity(ctx context.Context, agreementID string, jobspec *models.JobSpec) (string, error) {
	payload := map[string]any{
		"agreement_id": agreementID,
		"image":        jobspec.Benchmark.Container.Image,
		"resources": map[string]any{
			"cpu":    jobspec.Benchmark.Container.Resources.CPU,
			"memory": jobspec.Benchmark.Container.Resources.Memory,
			"gpu":    jobspec.Benchmark.Container.Resources.GPU,
		},
	}
	buf := new(bytes.Buffer)
	_ = json.NewEncoder(buf).Encode(payload)
	c.EnsureDiscover(ctx)
	endpoints := []string{}
	if c.ActivityBase != "" {
		endpoints = append(endpoints,
			c.ActivityBase+"/activity",
			c.ActivityBase+"/activities",
		)
	}
	// Legacy fallbacks
	endpoints = append(endpoints, "/activity", "/activities")
	for _, ep := range endpoints {
		url := c.BaseURL + ep
		status, b, err := c.doReq(ctx, http.MethodPost, url, buf.Bytes(), "application/json")
		if err != nil {
			continue
		}
		if status >= 200 && status < 300 {
			var m map[string]any
			if json.Unmarshal(b, &m) == nil {
				if v, ok := m["id"].(string); ok {
					return v, nil
				}
			}
			return string(b), nil
		}
		return "", fmt.Errorf("createActivity: %s -> %d: %s", url, status, strings.TrimSpace(string(b)))
	}
	return "", fmt.Errorf("createActivity: no known endpoints succeeded")
}

func (c *YagnaRESTClient) Exec(ctx context.Context, activityID string, jobspec *models.JobSpec) (stdout string, stderr string, exitCode int, err error) {
	c.EnsureDiscover(ctx)
	endpoints := []string{}
	if c.ActivityBase != "" {
		endpoints = append(endpoints,
			fmt.Sprintf("%s%s/activity/%s/exec", c.BaseURL, c.ActivityBase, activityID),
			fmt.Sprintf("%s%s/activities/%s/exec", c.BaseURL, c.ActivityBase, activityID),
		)
	}
	// Legacy fallbacks
	endpoints = append(endpoints,
		fmt.Sprintf("%s/activity/%s/exec", c.BaseURL, activityID),
		fmt.Sprintf("%s/activities/%s/exec", c.BaseURL, activityID),
	)
	payload := map[string]any{
		"command": jobspec.Benchmark.Container.Command,
		"env":     jobspec.Benchmark.Container.Environment,
		"timeout": int64(jobspec.Constraints.Timeout / time.Second),
	}
	for _, endpoint := range endpoints {
		buf := new(bytes.Buffer)
		_ = json.NewEncoder(buf).Encode(payload)
		status, b, err := c.doReq(ctx, http.MethodPost, endpoint, buf.Bytes(), "application/json")
		if err != nil {
			continue
		}
		if status >= 200 && status < 300 {
			var m map[string]any
			if json.Unmarshal(b, &m) == nil {
				if v, ok := m["stdout"].(string); ok {
					stdout = v
				}
				if v, ok := m["stderr"].(string); ok {
					stderr = v
				}
				if v, ok := m["exit_code"].(float64); ok {
					exitCode = int(v)
				} else {
					exitCode = 0
				}
				return stdout, stderr, exitCode, nil
			}
			return string(b), "", 0, nil
		}
		return "", "", 1, fmt.Errorf("exec: %s -> %d: %s", endpoint, status, strings.TrimSpace(string(b)))
	}
	return "", "", 1, fmt.Errorf("exec: no known exec endpoints succeeded; activity_id=%s", activityID)
}

func (c *YagnaRESTClient) StopActivity(ctx context.Context, activityID string) error {
	_ = ctx
	_ = activityID
	return nil
}
