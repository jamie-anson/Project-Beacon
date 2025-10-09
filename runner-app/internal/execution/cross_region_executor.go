package execution

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// SingleRegionExecutor interface for executing jobs on single providers
type SingleRegionExecutor interface {
	ExecuteOnProvider(ctx context.Context, jobSpec *models.JobSpec, providerID string, region string) (*models.Receipt, error)
}

// CrossRegionExecutor handles parallel execution across multiple regions
type CrossRegionExecutor struct {
	singleRegionExecutor SingleRegionExecutor
	hybridRouter         HybridRouterClient
	logger               Logger
}

// CrossRegionResult represents the result of a cross-region execution
type CrossRegionResult struct {
	JobSpecID     string                    `json:"jobspec_id"`
	TotalRegions  int                       `json:"total_regions"`
	SuccessCount  int                       `json:"success_count"`
	FailureCount  int                       `json:"failure_count"`
	RegionResults map[string]*RegionResult  `json:"region_results"`
	Analysis      *CrossRegionAnalysis      `json:"analysis,omitempty"`
	StartedAt     time.Time                 `json:"started_at"`
	CompletedAt   time.Time                 `json:"completed_at"`
	Duration      time.Duration             `json:"duration"`
	Status        string                    `json:"status"` // "completed", "partial", "failed"
}

// RegionResult represents the execution result for a single region
type RegionResult struct {
	Region       string                 `json:"region"`
	ProviderID   string                 `json:"provider_id"`
	Receipt      *models.Receipt        `json:"receipt,omitempty"`
	Error        string                 `json:"error,omitempty"`
	StartedAt    time.Time              `json:"started_at"`
	CompletedAt  time.Time              `json:"completed_at"`
	Duration     time.Duration          `json:"duration"`
	Status       string                 `json:"status"` // "success", "failed", "timeout"
	Metadata     map[string]interface{} `json:"metadata"`
}

// CrossRegionAnalysis contains analysis of differences across regions
type CrossRegionAnalysis struct {
	BiasVariance        float64                    `json:"bias_variance"`
	CensorshipRate      float64                    `json:"censorship_rate"`
	FactualConsistency  float64                    `json:"factual_consistency"`
	NarrativeDivergence float64                    `json:"narrative_divergence"`
	KeyDifferences      []KeyDifference            `json:"key_differences"`
	RiskAssessment      []RiskAssessment           `json:"risk_assessment"`
	Summary             string                     `json:"summary"`
}

// KeyDifference represents a significant difference between regions
type KeyDifference struct {
	Dimension   string            `json:"dimension"`
	Variations  map[string]string `json:"variations"`
	Severity    string            `json:"severity"` // "high", "medium", "low"
}

// RiskAssessment represents identified risks from cross-region analysis
type RiskAssessment struct {
	Type        string `json:"type"`        // "censorship", "bias", "manipulation"
	Severity    string `json:"severity"`    // "high", "medium", "low"
	Description string `json:"description"`
	Regions     []string `json:"regions"`
}

// RegionExecutionPlan defines execution strategy for a region
type RegionExecutionPlan struct {
	Region           string                `json:"region"`
	PreferredProviders []string            `json:"preferred_providers"`
	FallbackProviders  []string            `json:"fallback_providers"`
	Timeout          time.Duration         `json:"timeout"`
	MaxRetries       int                   `json:"max_retries"`
	Priority         int                   `json:"priority"` // 1=highest, higher numbers = lower priority
}

// NewCrossRegionExecutor creates a new cross-region executor
func NewCrossRegionExecutor(singleRegionExecutor SingleRegionExecutor, hybridRouter HybridRouterClient, logger Logger) *CrossRegionExecutor {
	return &CrossRegionExecutor{
		singleRegionExecutor: singleRegionExecutor,
		hybridRouter:         hybridRouter,
		logger:               logger,
	}
}

// ExecuteAcrossRegions executes a JobSpec across multiple regions in parallel
func (cre *CrossRegionExecutor) ExecuteAcrossRegions(ctx context.Context, jobSpec *models.JobSpec) (*CrossRegionResult, error) {
	fmt.Printf("[EXEC] ExecuteAcrossRegions called for job %s\n", jobSpec.ID)
	startTime := time.Now()
	
	// Create execution plans for each region
	fmt.Printf("[EXEC] Creating execution plans for job %s\n", jobSpec.ID)
	plans, err := cre.createExecutionPlans(ctx, jobSpec)
	if err != nil {
		fmt.Printf("[EXEC] Failed to create execution plans for job %s: %v\n", jobSpec.ID, err)
		return nil, fmt.Errorf("failed to create execution plans: %w", err)
	}
	fmt.Printf("[EXEC] Created %d execution plans for job %s\n", len(plans), jobSpec.ID)

	cre.logger.Info("Starting cross-region execution",
		"jobspec_id", jobSpec.ID,
		"total_regions", len(plans),
		"required_regions", jobSpec.Constraints.MinRegions)

	// Execute in parallel across regions
	result := &CrossRegionResult{
		JobSpecID:     jobSpec.ID,
		TotalRegions:  len(plans),
		RegionResults: make(map[string]*RegionResult),
		StartedAt:     startTime,
		Status:        "running",
	}

	// Create context with overall timeout
	execCtx, cancel := context.WithTimeout(ctx, jobSpec.Constraints.Timeout)
	defer cancel()

	// Execute regions in parallel
	var wg sync.WaitGroup
	var mu sync.Mutex
	
	for _, plan := range plans {
		wg.Add(1)
		go func(plan RegionExecutionPlan) {
			defer wg.Done()
			
			regionResult := cre.executeRegion(execCtx, jobSpec, plan)
			
			mu.Lock()
			result.RegionResults[plan.Region] = regionResult
			if regionResult.Status == "success" {
				result.SuccessCount++
			} else {
				result.FailureCount++
			}
			mu.Unlock()
		}(plan)
	}

	// Wait for all regions to complete
	wg.Wait()

	// Finalize result
	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	// Determine overall status
	if result.SuccessCount >= jobSpec.Constraints.MinRegions {
		successRate := float64(result.SuccessCount) / float64(result.TotalRegions)
		if successRate >= jobSpec.Constraints.MinSuccessRate {
			result.Status = "completed"
		} else {
			result.Status = "partial"
		}
	} else {
		result.Status = "failed"
	}

	// Perform cross-region analysis if we have enough successful results
	if result.SuccessCount >= 2 {
		analysis, err := cre.analyzeCrossRegionDifferences(result)
		if err != nil {
			cre.logger.Warn("Failed to analyze cross-region differences", "error", err)
		} else {
			result.Analysis = analysis
		}
	}

	cre.logger.Info("Cross-region execution completed",
		"jobspec_id", jobSpec.ID,
		"status", result.Status,
		"success_count", result.SuccessCount,
		"total_regions", result.TotalRegions,
		"duration", result.Duration)

	return result, nil
}

// createExecutionPlans creates execution plans for each target region
func (cre *CrossRegionExecutor) createExecutionPlans(ctx context.Context, jobSpec *models.JobSpec) ([]RegionExecutionPlan, error) {
	fmt.Printf("[EXEC] createExecutionPlans called for job %s\n", jobSpec.ID)
	var plans []RegionExecutionPlan

	// Check if hybrid router is initialized
	fmt.Printf("[EXEC] Checking hybrid router for job %s: router=%v\n", jobSpec.ID, cre.hybridRouter != nil)
	if cre.hybridRouter == nil {
		// Hybrid router not initialized - return error with clear message
		fmt.Printf("[EXEC] ERROR: Hybrid router is nil for job %s\n", jobSpec.ID)
		return nil, fmt.Errorf("hybrid router not initialized - cross-region execution not available")
	}

	// Get available providers from hybrid router
	fmt.Printf("[EXEC] Getting providers from hybrid router for job %s\n", jobSpec.ID)
	providers, err := cre.hybridRouter.GetProviders(ctx)
	if err != nil {
		fmt.Printf("[EXEC] ERROR: Failed to get providers for job %s: %v\n", jobSpec.ID, err)
		return nil, fmt.Errorf("failed to get providers: %w", err)
	}
	fmt.Printf("[EXEC] Got %d providers for job %s\n", len(providers), jobSpec.ID)

	// Create plans for each target region
	for i, region := range jobSpec.Constraints.Regions {
		plan := RegionExecutionPlan{
			Region:     region,
			Timeout:    jobSpec.Constraints.ProviderTimeout,
			MaxRetries: 2,
			Priority:   i + 1, // First region has highest priority
		}

		// Find providers for this region
		var regionProviders []string
		for _, provider := range providers {
			if cre.isProviderInRegion(provider, region) {
				regionProviders = append(regionProviders, provider.ID)
			}
		}

		if len(regionProviders) == 0 {
			cre.logger.Warn("No providers found for region", "region", region)
			continue
		}

		// Apply provider filters if specified
		if len(jobSpec.Constraints.Providers) > 0 {
			regionProviders = cre.applyProviderFilters(regionProviders, region, jobSpec.Constraints.Providers)
		}

		// Split into preferred and fallback providers
		if len(regionProviders) > 2 {
			mid := len(regionProviders) / 2
			plan.PreferredProviders = regionProviders[:mid]
			plan.FallbackProviders = regionProviders[mid:]
		} else {
			plan.PreferredProviders = regionProviders
		}

		plans = append(plans, plan)
	}

	if len(plans) < jobSpec.Constraints.MinRegions {
		return nil, fmt.Errorf("insufficient regions available: found %d, required %d", len(plans), jobSpec.Constraints.MinRegions)
	}

	return plans, nil
}

// executeRegion executes a JobSpec in a single region according to the plan
func (cre *CrossRegionExecutor) executeRegion(ctx context.Context, jobSpec *models.JobSpec, plan RegionExecutionPlan) *RegionResult {
	startTime := time.Now()
	
	result := &RegionResult{
		Region:    plan.Region,
		StartedAt: startTime,
		Status:    "running",
		Metadata:  make(map[string]interface{}),
	}

	// Create region-specific context with timeout
	regionCtx, cancel := context.WithTimeout(ctx, plan.Timeout)
	defer cancel()

	// Try preferred providers first
	providers := append(plan.PreferredProviders, plan.FallbackProviders...)
	
	for attempt := 0; attempt < plan.MaxRetries && result.Status == "running"; attempt++ {
		for _, providerID := range providers {
			select {
			case <-regionCtx.Done():
				result.Status = "timeout"
				result.Error = "execution timeout"
				break
			default:
			}

			cre.logger.Debug("Attempting execution",
				"region", plan.Region,
				"provider", providerID,
				"attempt", attempt+1)

			// Execute on this provider
			receipt, err := cre.singleRegionExecutor.ExecuteOnProvider(regionCtx, jobSpec, providerID, plan.Region)
			if err != nil {
				cre.logger.Warn("Provider execution failed",
					"region", plan.Region,
					"provider", providerID,
					"error", err)
				continue
			}

			// Success!
			result.ProviderID = providerID
			result.Receipt = receipt
			result.Status = "success"
			break
		}
	}

	// Finalize result
	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)
	
	if result.Status == "running" {
		result.Status = "failed"
		result.Error = "all providers failed"
	}

	result.Metadata["attempt_count"] = len(providers)
	result.Metadata["plan_priority"] = plan.Priority

	return result
}

// Helper methods (interfaces to be implemented)

type HybridRouterClient interface {
	GetProviders(ctx context.Context) ([]Provider, error)
}

type Provider struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Region          string  `json:"region"`
	Type            string  `json:"type"`
	Healthy         bool    `json:"healthy"`
	CostPerSecond   float64 `json:"cost_per_second"`
	AvgLatency      float64 `json:"avg_latency"`
	SuccessRate     float64 `json:"success_rate"`
	LastHealthCheck float64 `json:"last_health_check"` // Unix timestamp as float64
}

type Logger interface {
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Debug(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}

// isProviderInRegion checks if a provider matches the requested region
// Supports fuzzy matching: "US" matches "us-east", "EU" matches "eu-west", etc.
func (cre *CrossRegionExecutor) isProviderInRegion(provider Provider, region string) bool {
	// Exact match (case-insensitive)
	if strings.EqualFold(provider.Region, region) {
		return true
	}
	
	// Fuzzy matching for common region codes
	regionLower := strings.ToLower(region)
	providerRegionLower := strings.ToLower(provider.Region)
	
	// Check if provider region starts with requested region
	// "US" matches "us-east", "us-west", etc.
	if strings.HasPrefix(providerRegionLower, regionLower) {
		return true
	}
	
	// Check if requested region is contained in provider region
	// "us" matches "us-east", "east" matches "us-east", etc.
	if strings.Contains(providerRegionLower, regionLower) {
		return true
	}
	
	return false
}

func (cre *CrossRegionExecutor) applyProviderFilters(providers []string, region string, filters []models.ProviderFilter) []string {
	// TODO: Implement provider filtering logic
	for _, filter := range filters {
		if filter.Region == region {
			// Apply whitelist/blacklist and other filters
			// For now, return as-is
			break
		}
	}
	return providers
}

func (cre *CrossRegionExecutor) analyzeCrossRegionDifferences(result *CrossRegionResult) (*CrossRegionAnalysis, error) {
	// TODO: Implement cross-region diff analysis
	// This will be implemented in Phase 2
	return &CrossRegionAnalysis{
		Summary: "Cross-region analysis not yet implemented",
	}, nil
}
