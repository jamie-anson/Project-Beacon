package golem

import (
	"context"
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"

	// OpenTelemetry
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// ExecuteMultiRegion executes a JobSpec across multiple regions concurrently with partial failure support.
func (e *ExecutionEngine) ExecuteMultiRegion(ctx context.Context, jobspec *models.JobSpec) (*ExecutionSummary, error) {
	tracer := otel.Tracer("runner/golem")
	ctx, span := tracer.Start(ctx, "ExecutionEngine.ExecuteMultiRegion", oteltrace.WithAttributes(
		attribute.String("job.id", jobspec.ID),
		attribute.Int("regions.count", len(jobspec.Constraints.Regions)),
		attribute.Float64("min_success_rate", jobspec.Constraints.MinSuccessRate),
		attribute.Float64("max_cost", jobspec.Constraints.MaxCost),
	))
	defer span.End()

	summary := &ExecutionSummary{
		JobSpecID:     jobspec.ID,
		TotalRegions:  len(jobspec.Constraints.Regions),
		Results:       make([]*ExecutionResult, 0),
		RegionResults: make(map[string]*ExecutionResult),
		StartedAt:     time.Now(),
		MaxCost:       jobspec.Constraints.MaxCost,
	}

	// Discover providers for all required regions
	providers, err := e.service.DiscoverProviders(ctx, jobspec.Constraints)
	if err != nil {
		return nil, fmt.Errorf("provider discovery failed: %w", err)
	}

	// Group providers by region
	grouped := groupProvidersByRegion(providers)

	// Create timeout context for entire execution
	execCtx, cancel := context.WithTimeout(ctx, jobspec.Constraints.Timeout)
	defer cancel()

	// Execute benchmark in each region concurrently
	var wg sync.WaitGroup
	resultsChan := make(chan *ExecutionResult, len(jobspec.Constraints.Regions))
	costChan := make(chan float64, len(jobspec.Constraints.Regions))

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
			result := e.executeInRegionWithTimeout(execCtx, jobspec, region, providers)
			resultsChan <- result
			
			// Track cost if successful
			if result.Error == nil && result.Execution != nil {
				if provider, _ := e.service.GetProvider(result.ProviderID); provider != nil {
					if cost, err := e.service.EstimateTaskCost(provider, jobspec); err == nil {
						costChan <- cost
					}
				}
			}
		}(region, prov)
	}

	// Close channels when all goroutines complete
	go func() {
		wg.Wait()
		close(resultsChan)
		close(costChan)
	}()

	// Collect results and track costs
	var totalCost float64
	for {
		select {
		case result, ok := <-resultsChan:
			if !ok {
				goto ProcessingComplete
			}
			summary.Results = append(summary.Results, result)
			summary.RegionResults[result.Region] = result
			
			if result.Error != nil {
				summary.FailureCount++
				summary.Errors = append(summary.Errors, fmt.Sprintf("Region %s: %v", result.Region, result.Error))
			} else {
				summary.SuccessCount++
			}
			
		case cost := <-costChan:
			totalCost += cost
			summary.TotalCost = totalCost
			
			// Check cost limits
			if summary.MaxCost > 0 && totalCost > summary.MaxCost {
				cancel() // Cancel remaining executions
				summary.Errors = append(summary.Errors, fmt.Sprintf("Cost limit exceeded: %.4f > %.4f GLM", totalCost, summary.MaxCost))
			}
		}
	}

ProcessingComplete:
	summary.CompletedAt = time.Now()
	summary.TotalDuration = summary.CompletedAt.Sub(summary.StartedAt)
	summary.SuccessRate = float64(summary.SuccessCount) / float64(summary.TotalRegions)
	
	// Determine if execution meets success criteria
	minSuccessRate := jobspec.Constraints.MinSuccessRate
	if minSuccessRate == 0 {
		minSuccessRate = 0.67 // Default 67%
	}
	
	summary.PartialSuccess = summary.SuccessRate >= minSuccessRate
	
	// Check minimum region requirement
	if summary.SuccessCount < jobspec.Constraints.MinRegions {
		summary.PartialSuccess = false
		summary.Errors = append(summary.Errors, fmt.Sprintf("Insufficient successful regions: %d < %d", summary.SuccessCount, jobspec.Constraints.MinRegions))
	}

	// If partial success is not achieved, return error
	if !summary.PartialSuccess {
		return summary, fmt.Errorf("execution failed: success rate %.2f%% below minimum %.2f%%", summary.SuccessRate*100, minSuccessRate*100)
	}

	return summary, nil
}

// executeInRegionWithTimeout executes a benchmark in a specific region with timeout handling.
func (e *ExecutionEngine) executeInRegionWithTimeout(ctx context.Context, jobspec *models.JobSpec, region string, providers []*Provider) *ExecutionResult {
	// Create region-specific timeout context
	regionCtx, cancel := context.WithTimeout(ctx, jobspec.Constraints.ProviderTimeout)
	defer cancel()
	
	return e.executeInRegion(regionCtx, jobspec, region, providers)
}

// executeInRegion executes a benchmark in a specific region.
func (e *ExecutionEngine) executeInRegion(ctx context.Context, jobspec *models.JobSpec, region string, providers []*Provider) *ExecutionResult {
	tracer := otel.Tracer("runner/golem")
	ctx, span := tracer.Start(ctx, "ExecutionEngine.executeInRegion", oteltrace.WithAttributes(
		attribute.String("job.id", jobspec.ID),
		attribute.String("region", region),
		attribute.Int("providers.candidate_count", len(providers)),
	))
	defer span.End()
	
	result := &ExecutionResult{
		JobSpecID:  jobspec.ID,
		Region:     region,
		ExecutedAt: time.Now(),
	}

	// Try providers in order of preference (highest score first)
	sortedProviders := selectProvidersInRegion(providers)
	
	for _, provider := range sortedProviders {
		// Check if context is still valid
		if ctx.Err() != nil {
			result.Error = fmt.Errorf("execution timeout in region %s: %w", region, ctx.Err())
			return result
		}
		
		result.ProviderID = provider.ID
		span.SetAttributes(
			attribute.String("provider.id", provider.ID),
			attribute.String("provider.region", provider.Region),
		)

		// Execute task on selected provider with timeout
		execution, err := e.service.ExecuteTask(ctx, provider, jobspec)
		if err != nil {
			// Try next provider if this one fails
			span.AddEvent("provider_failed", oteltrace.WithAttributes(
				attribute.String("error", err.Error()),
			))
			continue
		}

		result.Execution = execution
		
		// Generate receipt for successful execution
		if receipt, err := e.generateReceipt(jobspec, execution, provider); err == nil {
			result.Receipt = receipt
		}

		return result
	}

	// All providers failed
	result.Error = fmt.Errorf("all providers failed in region %s", region)
	return result
}

// selectProvidersInRegion sorts providers by score (highest first) for optimal selection
func selectProvidersInRegion(providers []*Provider) []*Provider {
	sorted := make([]*Provider, len(providers))
	copy(sorted, providers)
	
	// Simple bubble sort by score (highest first)
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].Score < sorted[j+1].Score {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}
	
	return sorted
}


// hashStringToInt returns a stable non-cryptographic hash of a string as int.
func hashStringToInt(s string) int {
    h := fnv.New32a()
    _, _ = h.Write([]byte(s))
    return int(h.Sum32())
}
