package golem

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// ExecuteMultiRegion executes a JobSpec across multiple regions concurrently.
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
	grouped := groupProvidersByRegion(providers)

	// Execute benchmark in each region concurrently
	var wg sync.WaitGroup
	resultsChan := make(chan *ExecutionResult, len(jobspec.Constraints.Regions))

	for _, region := range jobspec.Constraints.Regions {
		prov, exists := grouped[region]
		if !exists || len(prov) == 0 {
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
			// Per-region timeout derived from jobspec constraints
			regionCtx := ctx
			var cancel context.CancelFunc
			if jobspec.Constraints.Timeout > 0 {
				regionCtx, cancel = context.WithTimeout(ctx, jobspec.Constraints.Timeout)
			} else {
				regionCtx, cancel = context.WithCancel(ctx)
			}
			defer cancel()

			select {
			case <-regionCtx.Done():
				return
			default:
			}
			result := e.executeInRegion(regionCtx, jobspec, region, providers)
			resultsChan <- result
		}(region, prov)
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

// executeInRegion executes a benchmark in a specific region.
func (e *ExecutionEngine) executeInRegion(ctx context.Context, jobspec *models.JobSpec, region string, providers []*Provider) *ExecutionResult {
	result := &ExecutionResult{
		JobSpecID:  jobspec.ID,
		Region:     region,
		ExecutedAt: time.Now(),
	}

	// Select best provider in region (highest score)
	bestProvider := selectBestProviderInRegion(providers)
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
