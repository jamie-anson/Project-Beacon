package golem

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/circuitbreaker"
	apperrors "github.com/jamie-anson/project-beacon-runner/internal/errors"
)

// YagnaClient provides circuit breaker protected access to Yagna daemon
type YagnaClient struct {
	baseURL   string
	apiKey    string
	client    *http.Client
	cbManager *circuitbreaker.Manager
}

// TaskSpec represents a Golem task specification
type TaskSpec struct {
	Name        string            `json:"name"`
	Image       string            `json:"image"`
	Commands    []string          `json:"commands"`
	Resources   TaskResources     `json:"resources"`
	Constraints TaskConstraints   `json:"constraints"`
	Timeout     time.Duration     `json:"timeout"`
	Env         map[string]string `json:"env,omitempty"`
}

// TaskResources defines computational requirements
type TaskResources struct {
	CPU    float64 `json:"cpu"`    // CPU cores
	Memory int64   `json:"memory"` // Memory in MB
	Disk   int64   `json:"disk"`   // Disk space in MB
}

// TaskConstraints defines provider selection criteria
type TaskConstraints struct {
	Regions        []string `json:"regions,omitempty"`
	MinReputation  float64  `json:"min_reputation,omitempty"`
	MaxPrice       float64  `json:"max_price,omitempty"`
	RequiredProps  []string `json:"required_props,omitempty"`
	ForbiddenProps []string `json:"forbidden_props,omitempty"`
}

// Task represents a submitted Golem task
type Task struct {
	ID          string            `json:"id"`
	Status      TaskStatus        `json:"status"`
	Spec        TaskSpec          `json:"spec"`
	ProviderID  string            `json:"provider_id,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	StartedAt   *time.Time        `json:"started_at,omitempty"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
	Results     *TaskResults      `json:"results,omitempty"`
	Error       string            `json:"error,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// TaskStatus represents the current state of a task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// TaskResults contains the output of a completed task
type TaskResults struct {
	ExitCode int               `json:"exit_code"`
	Stdout   string            `json:"stdout"`
	Stderr   string            `json:"stderr"`
	Files    map[string][]byte `json:"files,omitempty"`
	Duration time.Duration     `json:"duration"`
	Cost     TaskCost          `json:"cost"`
}

// TaskCost represents the cost breakdown for task execution
type TaskCost struct {
	Total    float64 `json:"total"`
	CPU      float64 `json:"cpu"`
	Duration float64 `json:"duration"`
	Network  float64 `json:"network"`
	Storage  float64 `json:"storage"`
}

// Provider represents a Golem network provider
type Provider struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Region     string                 `json:"region"`
	Status     string                 `json:"status"`     // online, offline, busy
	Score      float64                `json:"score"`      // reputation score 0-1 (alias for Reputation)
	Reputation float64                `json:"reputation"` // reputation score 0-1
	Price      float64                `json:"price"`      // GLM per hour (alias for Pricing.CPUPerHour)
	Pricing    ProviderPricing        `json:"pricing"`
	Resources  ProviderResources      `json:"resources"`
	Properties map[string]interface{} `json:"properties"`
	Metadata   map[string]interface{} `json:"metadata"`
	Online     bool                   `json:"online"`
	LastSeen   time.Time              `json:"last_seen"`
}

// ProviderPricing represents provider cost structure
type ProviderPricing struct {
	CPUPerHour     float64 `json:"cpu_per_hour"`
	MemoryPerHour  float64 `json:"memory_per_hour"`
	StoragePerHour float64 `json:"storage_per_hour"`
	StartupCost    float64 `json:"startup_cost"`
}

// NewYagnaClient creates a new Yagna client with circuit breaker protection
func NewYagnaClient(baseURL, apiKey string) *YagnaClient {
	return &YagnaClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		cbManager: circuitbreaker.NewManager(),
	}
}

// SubmitTask submits a new task to the Golem network
func (c *YagnaClient) SubmitTask(ctx context.Context, spec TaskSpec) (*Task, error) {
	cbConfig := circuitbreaker.Config{
		Name:             "yagna-submit-task",
		MaxFailures:      5,
		Timeout:          15 * time.Second,
		MaxRequests:      3,
		SuccessThreshold: 2,
		IsFailure: func(err error) bool {
			// Don't count validation errors as circuit breaker failures
			return err != nil && !apperrors.IsType(err, apperrors.ValidationError)
		},
	}

	cb := c.cbManager.GetOrCreate("yagna-submit-task", cbConfig)

	var task *Task
	err := cb.Execute(ctx, func(ctx context.Context) error {
		reqBody, err := json.Marshal(spec)
		if err != nil {
			return apperrors.Wrap(err, apperrors.ValidationError, "failed to marshal task spec")
		}

		req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/tasks", bytes.NewBuffer(reqBody))
		if err != nil {
			return apperrors.Wrap(err, apperrors.InternalError, "failed to create request")
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.apiKey)

		resp, err := c.client.Do(req)
		if err != nil {
			return apperrors.Wrap(err, apperrors.ExternalServiceError, "yagna request failed")
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			return apperrors.Newf(apperrors.ExternalServiceError, "yagna returned status %d: %s", resp.StatusCode, string(body))
		}

		if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
			return apperrors.Wrap(err, apperrors.InternalError, "failed to decode response")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return task, nil
}

// GetTask retrieves task status and results
func (c *YagnaClient) GetTask(ctx context.Context, taskID string) (*Task, error) {
	cbConfig := circuitbreaker.DefaultConfig("yagna-get-task")
	cb := c.cbManager.GetOrCreate("yagna-get-task", cbConfig)

	var task *Task
	err := cb.Execute(ctx, func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/tasks/"+taskID, nil)
		if err != nil {
			return apperrors.Wrap(err, apperrors.InternalError, "failed to create request")
		}

		req.Header.Set("Authorization", "Bearer "+c.apiKey)

		resp, err := c.client.Do(req)
		if err != nil {
			return apperrors.Wrap(err, apperrors.ExternalServiceError, "yagna request failed")
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			return apperrors.Newf(apperrors.NotFoundError, "task %s not found", taskID)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return apperrors.Newf(apperrors.ExternalServiceError, "yagna returned status %d: %s", resp.StatusCode, string(body))
		}

		if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
			return apperrors.Wrap(err, apperrors.InternalError, "failed to decode response")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return task, nil
}

// CancelTask cancels a running task
func (c *YagnaClient) CancelTask(ctx context.Context, taskID string) error {
	cbConfig := circuitbreaker.DefaultConfig("yagna-cancel-task")
	cb := c.cbManager.GetOrCreate("yagna-cancel-task", cbConfig)

	return cb.Execute(ctx, func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, "DELETE", c.baseURL+"/tasks/"+taskID, nil)
		if err != nil {
			return apperrors.Wrap(err, apperrors.InternalError, "failed to create request")
		}

		req.Header.Set("Authorization", "Bearer "+c.apiKey)

		resp, err := c.client.Do(req)
		if err != nil {
			return apperrors.Wrap(err, apperrors.ExternalServiceError, "yagna request failed")
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			return apperrors.Newf(apperrors.NotFoundError, "task %s not found", taskID)
		}

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			body, _ := io.ReadAll(resp.Body)
			return apperrors.Newf(apperrors.ExternalServiceError, "yagna returned status %d: %s", resp.StatusCode, string(body))
		}

		return nil
	})
}

// ListProviders retrieves available providers with optional filtering
func (c *YagnaClient) ListProviders(ctx context.Context, region string) ([]*Provider, error) {
	cbConfig := circuitbreaker.DefaultConfig("yagna-list-providers")
	cb := c.cbManager.GetOrCreate("yagna-list-providers", cbConfig)

	var providers []*Provider
	err := cb.Execute(ctx, func(ctx context.Context) error {
		url := c.baseURL + "/providers"
		if region != "" {
			url += "?region=" + region
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return apperrors.Wrap(err, apperrors.InternalError, "failed to create request")
		}

		req.Header.Set("Authorization", "Bearer "+c.apiKey)

		resp, err := c.client.Do(req)
		if err != nil {
			return apperrors.Wrap(err, apperrors.ExternalServiceError, "yagna request failed")
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return apperrors.Newf(apperrors.ExternalServiceError, "yagna returned status %d: %s", resp.StatusCode, string(body))
		}

		if err := json.NewDecoder(resp.Body).Decode(&providers); err != nil {
			return apperrors.Wrap(err, apperrors.InternalError, "failed to decode response")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return providers, nil
}

// GetStats returns Yagna client statistics including circuit breaker status
func (c *YagnaClient) GetStats() map[string]circuitbreaker.Stats {
	stats := make(map[string]circuitbreaker.Stats)
	
	operations := []string{
		"yagna-submit-task",
		"yagna-get-task", 
		"yagna-cancel-task",
		"yagna-list-providers",
	}
	
	for _, op := range operations {
		if cb, exists := c.cbManager.Get(op); exists {
			stats[op] = cb.Stats()
		}
	}
	
	return stats
}

// Health checks the health of the Yagna daemon
func (c *YagnaClient) Health(ctx context.Context) error {
	cbConfig := circuitbreaker.Config{
		Name:        "yagna-health",
		MaxFailures: 3,
		Timeout:     5 * time.Second,
	}
	
	cb := c.cbManager.GetOrCreate("yagna-health", cbConfig)
	
	return cb.Execute(ctx, func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
		if err != nil {
			return err
		}
		
		resp, err := c.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("yagna health check failed with status %d", resp.StatusCode)
		}
		
		return nil
	})
}
