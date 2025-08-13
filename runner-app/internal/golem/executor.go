package golem

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// ExecutionEngine orchestrates multi-region benchmark execution
type ExecutionEngine struct {
	service *Service
	results chan *ExecutionResult
	mu      sync.RWMutex
}

// ExecutionResult represents the result of a benchmark execution
type ExecutionResult struct {
	JobSpecID    string           `json:"jobspec_id"`
	Region       string           `json:"region"`
	ProviderID   string           `json:"provider_id"`
	Execution    *TaskExecution   `json:"execution"`
	Receipt      *models.Receipt  `json:"receipt"`
	Error        error            `json:"error,omitempty"`
	ExecutedAt   time.Time        `json:"executed_at"`
}

// ExecutionSummary provides an overview of multi-region execution
type ExecutionSummary struct {
	JobSpecID      string                      `json:"jobspec_id"`
	TotalRegions   int                         `json:"total_regions"`
	SuccessCount   int                         `json:"success_count"`
	FailureCount   int                         `json:"failure_count"`
	Results        []*ExecutionResult          `json:"results"`
	RegionResults  map[string]*ExecutionResult `json:"region_results"`
	StartedAt      time.Time                   `json:"started_at"`
	CompletedAt    time.Time                   `json:"completed_at"`
	TotalDuration  time.Duration               `json:"total_duration"`
	TotalCost      float64                     `json:"total_cost"`
}

// NewExecutionEngine creates a new execution engine
func NewExecutionEngine(service *Service) *ExecutionEngine {
	return &ExecutionEngine{
		service: service,
		results: make(chan *ExecutionResult, 100),
	}
}

// ExecuteMultiRegion executes a JobSpec across multiple regions
func (e *ExecutionEngine) ExecuteMultiRegion(ctx context.Context, jobspec *models.JobSpec) (*ExecutionSummary, error) {
	summary := &ExecutionSummary{
		JobSpecID:     jobspec.ID,
		TotalRegions:  len(jobspec.Constraints.Regions),
		Results:       make([]*ExecutionResult, 0),
		RegionResults: make(map[string]*ExecutionResult),
		StartedAt:     time.Now(),
	}

	// Discover providers for all required regions
	providers, err := e.service.DiscoverProviders(ctx, jobspec.Constraints)
	if err != nil {
		return nil, fmt.Errorf("provider discovery failed: %w", err)
	}

	// Group providers by region
	regionProviders := make(map[string][]*Provider)
	for _, provider := range providers {
		regionProviders[provider.Region] = append(regionProviders[provider.Region], provider)
	}

	// Execute benchmark in each region concurrently
	var wg sync.WaitGroup
	resultsChan := make(chan *ExecutionResult, len(jobspec.Constraints.Regions))

	for _, region := range jobspec.Constraints.Regions {
		providers, exists := regionProviders[region]
		if !exists || len(providers) == 0 {
			// Create error result for missing region
			result := &ExecutionResult{
				JobSpecID:  jobspec.ID,
				Region:     region,
				Error:      fmt.Errorf("no providers available in region %s", region),
				ExecutedAt: time.Now(),
			}
			resultsChan <- result
			continue
		}

		wg.Add(1)
		go func(region string, providers []*Provider) {
			defer wg.Done()
			result := e.executeInRegion(ctx, jobspec, region, providers)
			resultsChan <- result
		}(region, providers)
	}

	// Wait for all executions to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	for result := range resultsChan {
		summary.Results = append(summary.Results, result)
		summary.RegionResults[result.Region] = result

		if result.Error != nil {
			summary.FailureCount++
		} else {
			summary.SuccessCount++
			
			// Calculate cost if execution was successful
			if result.Execution != nil {
				provider, _ := e.service.GetProvider(result.ProviderID)
				if provider != nil {
					cost, _ := e.service.EstimateTaskCost(provider, jobspec)
					summary.TotalCost += cost
				}
			}
		}
	}

	summary.CompletedAt = time.Now()
	summary.TotalDuration = summary.CompletedAt.Sub(summary.StartedAt)

	return summary, nil
}

// executeInRegion executes a benchmark in a specific region
func (e *ExecutionEngine) executeInRegion(ctx context.Context, jobspec *models.JobSpec, region string, providers []*Provider) *ExecutionResult {
	result := &ExecutionResult{
		JobSpecID:  jobspec.ID,
		Region:     region,
		ExecutedAt: time.Now(),
	}

	// Select best provider in region (highest score)
	var bestProvider *Provider
	for _, provider := range providers {
		if bestProvider == nil || provider.Score > bestProvider.Score {
			bestProvider = provider
		}
	}

	if bestProvider == nil {
		result.Error = fmt.Errorf("no suitable provider found in region %s", region)
		return result
	}

	result.ProviderID = bestProvider.ID

	// Execute task on selected provider
	execution, err := e.service.ExecuteTask(ctx, bestProvider, jobspec)
	if err != nil {
		result.Error = fmt.Errorf("execution failed on provider %s: %w", bestProvider.ID, err)
		return result
	}

	result.Execution = execution

	// Generate receipt for successful execution
	receipt, err := e.generateReceipt(jobspec, execution, bestProvider)
	if err != nil {
		result.Error = fmt.Errorf("receipt generation failed: %w", err)
		return result
	}

	result.Receipt = receipt

	return result
}

// generateReceipt creates a cryptographically signed receipt for the execution
func (e *ExecutionEngine) generateReceipt(jobspec *models.JobSpec, execution *TaskExecution, provider *Provider) (*models.Receipt, error) {
	receipt := &models.Receipt{
		ID:        fmt.Sprintf("receipt_%s_%s", execution.ID, provider.Region),
		JobSpecID: jobspec.ID,
		ExecutionDetails: models.ExecutionDetails{
			TaskID:      execution.ID,
			ProviderID:  provider.ID,
			Region:      provider.Region,
			StartedAt:   execution.StartedAt,
			CompletedAt: execution.CompletedAt,
			Duration:    execution.CompletedAt.Sub(execution.StartedAt),
			Status:      execution.Status,
		},
		Output: models.ExecutionOutput{
			Data:     execution.Output,
			Hash:     "", // TODO: Implement output hashing
			Metadata: execution.Metadata,
		},
		Provenance: models.ProvenanceInfo{
			BenchmarkHash: jobspec.Benchmark.Input.Hash,
			ProviderInfo: map[string]interface{}{
				"id":       provider.ID,
				"name":     provider.Name,
				"region":   provider.Region,
				"score":    provider.Score,
				"resources": provider.Resources,
			},
			ExecutionEnv: map[string]interface{}{
				"container_image": jobspec.Benchmark.Container.Image,
				"timeout":        jobspec.Constraints.Timeout.String(),
				"network":        e.service.network,
			},
		},
		CreatedAt: time.Now(),
	}

	// Sign the receipt (placeholder - would use actual signing key)
	// For now, we'll skip signing and just set placeholder values
	receipt.Signature = "mock_signature_placeholder"
	receipt.PublicKey = "mock_public_key_placeholder"

	return receipt, nil
}

// GetExecutionStatus returns the current status of an execution
func (e *ExecutionEngine) GetExecutionStatus(jobspecID string) (*ExecutionSummary, error) {
	// In a real implementation, this would query the database
	// For now, return a placeholder
	return &ExecutionSummary{
		JobSpecID:    jobspecID,
		TotalRegions: 0,
		SuccessCount: 0,
		FailureCount: 0,
		Results:      []*ExecutionResult{},
	}, nil
}

// CancelExecution cancels a running execution
func (e *ExecutionEngine) CancelExecution(ctx context.Context, jobspecID string) error {
	// In a real implementation, this would cancel running tasks
	// For now, return success
	return nil
}

// ValidateExecution validates that an execution meets the JobSpec requirements
func (e *ExecutionEngine) ValidateExecution(summary *ExecutionSummary, jobspec *models.JobSpec) error {
	// Check minimum regions requirement
	if summary.SuccessCount < jobspec.Constraints.MinRegions {
		return fmt.Errorf("insufficient successful executions: got %d, need %d", 
			summary.SuccessCount, jobspec.Constraints.MinRegions)
	}

	// Check that all required regions have results
	requiredRegions := make(map[string]bool)
	for _, region := range jobspec.Constraints.Regions {
		requiredRegions[region] = false
	}

	for _, result := range summary.Results {
		if result.Error == nil {
			requiredRegions[result.Region] = true
		}
	}

	for region, satisfied := range requiredRegions {
		if !satisfied {
			return fmt.Errorf("no successful execution in required region: %s", region)
		}
	}

	return nil
}

// EstimateExecutionCost estimates the total cost for multi-region execution
func (e *ExecutionEngine) EstimateExecutionCost(ctx context.Context, jobspec *models.JobSpec) (float64, error) {
	providers, err := e.service.DiscoverProviders(ctx, jobspec.Constraints)
	if err != nil {
		return 0, fmt.Errorf("provider discovery failed: %w", err)
	}

	// Group providers by region and select best in each
	regionProviders := make(map[string]*Provider)
	for _, provider := range providers {
		existing, exists := regionProviders[provider.Region]
		if !exists || provider.Score > existing.Score {
			regionProviders[provider.Region] = provider
		}
	}

	var totalCost float64
	for _, provider := range regionProviders {
		cost, err := e.service.EstimateTaskCost(provider, jobspec)
		if err != nil {
			continue // Skip providers with cost estimation errors
		}
		totalCost += cost
	}

	return totalCost, nil
}
