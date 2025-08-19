package golem

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func TestNewExecutionEngine(t *testing.T) {
	service := NewService("test-api-key", "testnet")
	engine := NewExecutionEngine(service)
	
	if engine.service != service {
		t.Error("Service not properly assigned to execution engine")
	}
	
	if engine.results == nil {
		t.Error("Results channel not initialized")
	}
	
	t.Logf("✅ Execution engine created successfully")
}

func TestExecuteMultiRegion(t *testing.T) {
	service := NewService("test-api-key", "testnet")
	engine := NewExecutionEngine(service)
	ctx := context.Background()
	
	// Create test JobSpec for multi-region execution
	jobspec := &models.JobSpec{
		ID:      "multi-region-test-001",
		Version: "1.0",
		Benchmark: models.BenchmarkSpec{
			Name:        "Who Are You?",
			Description: "Multi-region text generation benchmark",
			Container: models.ContainerSpec{
				Image:   "test/benchmark:latest",
				Command: []string{"python", "benchmark.py"},
			},
			Input: models.InputSpec{
				Data: map[string]interface{}{
					"prompt": "Who are you?",
				},
				Hash: "abc123",
			},
		},
		Constraints: models.ExecutionConstraints{
			Regions:        []string{"US", "EU", "APAC"},
			MinRegions:     3,
			MinSuccessRate: 0.000001,
			ProviderTimeout: 10 * time.Second,
			Timeout:        5 * time.Minute,
		},
	}
	
	// Execute across multiple regions
	summary, err := engine.ExecuteMultiRegion(ctx, jobspec)
	if err != nil {
		t.Fatalf("Multi-region execution failed: %v", err)
	}
	
	// Validate execution summary
	if summary.JobSpecID != jobspec.ID {
		t.Errorf("Expected JobSpec ID %s, got %s", jobspec.ID, summary.JobSpecID)
	}
	
	if summary.TotalRegions != len(jobspec.Constraints.Regions) {
		t.Errorf("Expected %d regions, got %d", len(jobspec.Constraints.Regions), summary.TotalRegions)
	}
	
	if len(summary.Results) != summary.TotalRegions {
		t.Errorf("Expected %d results, got %d", summary.TotalRegions, len(summary.Results))
	}
	
	if summary.SuccessCount == 0 {
		t.Error("No successful executions")
	}
	
	if summary.TotalDuration == 0 {
		t.Error("Total duration is zero")
	}
	
	// Validate regional results
	expectedRegions := map[string]bool{"US": false, "EU": false, "APAC": false}
	for _, result := range summary.Results {
		if _, exists := expectedRegions[result.Region]; !exists {
			t.Errorf("Unexpected region in results: %s", result.Region)
		}
		expectedRegions[result.Region] = true
		
		// Validate result structure
		if result.JobSpecID != jobspec.ID {
			t.Errorf("Result JobSpec ID mismatch: expected %s, got %s", jobspec.ID, result.JobSpecID)
		}
		
		if result.Error == nil {
			if result.Execution == nil {
				t.Errorf("Successful result missing execution for region %s", result.Region)
			}
			if result.Receipt == nil {
				t.Errorf("Successful result missing receipt for region %s", result.Region)
			}
		}
	}
	
	// Check that all regions were processed
	for region, processed := range expectedRegions {
		if !processed {
			t.Errorf("Region %s was not processed", region)
		}
	}
	
	t.Logf("✅ Multi-region execution completed successfully")
	t.Logf("   - Total regions: %d", summary.TotalRegions)
	t.Logf("   - Successful: %d", summary.SuccessCount)
	t.Logf("   - Failed: %d", summary.FailureCount)
	t.Logf("   - Duration: %v", summary.TotalDuration)
	t.Logf("   - Total cost: $%.4f", summary.TotalCost)
}

func TestExecuteMultiRegionWithInsufficientProviders(t *testing.T) {
	service := NewService("test-api-key", "testnet")
	engine := NewExecutionEngine(service)
	ctx := context.Background()
	
	// Create JobSpec requiring regions that don't exist
	jobspec := &models.JobSpec{
		ID:      "insufficient-test-001",
		Version: "1.0",
		Benchmark: models.BenchmarkSpec{
			Name: "Test Benchmark",
		},
		Constraints: models.ExecutionConstraints{
			Regions:    []string{"MARS", "MOON"}, // Non-existent regions
			MinRegions: 2,
			Timeout:    5 * time.Minute,
		},
	}
	
	// This should fail due to insufficient providers
	_, err := engine.ExecuteMultiRegion(ctx, jobspec)
	if err == nil {
		t.Error("Expected error for insufficient providers, got nil")
	}
	
	t.Logf("✅ Correctly handled insufficient providers: %v", err)
}

func TestValidateExecution(t *testing.T) {
	service := NewService("test-api-key", "testnet")
	engine := NewExecutionEngine(service)
	
	jobspec := &models.JobSpec{
		Constraints: models.ExecutionConstraints{
			Regions:        []string{"US", "EU", "APAC"},
			MinRegions:     2,
			MinSuccessRate: 0.000001,
			ProviderTimeout: 10 * time.Second,
		},
	}
	
	tests := []struct {
		name        string
		summary     *ExecutionSummary
		expectError bool
		description string
	}{
		{
			name: "Valid execution",
			summary: &ExecutionSummary{
				SuccessCount: 3,
				Results: []*ExecutionResult{
					{Region: "US", Error: nil},
					{Region: "EU", Error: nil},
					{Region: "APAC", Error: nil},
				},
			},
			expectError: false,
			description: "All regions successful",
		},
		{
			name: "Insufficient successes",
			summary: &ExecutionSummary{
				SuccessCount: 1,
				Results: []*ExecutionResult{
					{Region: "US", Error: nil},
					{Region: "EU", Error: fmt.Errorf("failed")},
					{Region: "APAC", Error: fmt.Errorf("failed")},
				},
			},
			expectError: true,
			description: "Only 1 success, need 2",
		},
		{
			name: "Missing required region",
			summary: &ExecutionSummary{
				SuccessCount: 2,
				Results: []*ExecutionResult{
					{Region: "US", Error: nil},
					{Region: "EU", Error: nil},
					// Missing APAC
				},
			},
			expectError: true,
			description: "Missing APAC region",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.ValidateExecution(tt.summary, jobspec)
			
			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, got nil", tt.description)
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.description, err)
			}
		})
	}
	
	t.Logf("✅ Execution validation tests passed")
}

func TestEstimateExecutionCost(t *testing.T) {
	service := NewService("test-api-key", "testnet")
	engine := NewExecutionEngine(service)
	ctx := context.Background()
	
	jobspec := &models.JobSpec{
		Constraints: models.ExecutionConstraints{
			Regions:        []string{"US", "EU"},
			MinRegions:     2,
			MinSuccessRate: 0.000001,
			ProviderTimeout: 10 * time.Second,
			Timeout:        30 * time.Minute,
		},
	}
	
	cost, err := engine.EstimateExecutionCost(ctx, jobspec)
	if err != nil {
		t.Fatalf("Cost estimation failed: %v", err)
	}
	
	if cost <= 0 {
		t.Errorf("Expected positive cost, got %f", cost)
	}
	
	// Cost should be reasonable for 30 minutes across 2 regions
	if cost > 1.0 { // More than $1 seems excessive for testing
		t.Errorf("Cost seems too high: $%.4f", cost)
	}
	
	t.Logf("✅ Estimated execution cost: $%.4f", cost)
}

func TestGenerateReceipt(t *testing.T) {
	service := NewService("test-api-key", "testnet")
	engine := NewExecutionEngine(service)
	
	jobspec := &models.JobSpec{
		ID: "test-job-001",
		Benchmark: models.BenchmarkSpec{
			Input: models.InputSpec{
				Hash: "test-hash-123",
			},
			Container: models.ContainerSpec{
				Image: "test/benchmark:latest",
			},
		},
		Constraints: models.ExecutionConstraints{
			Timeout: 5 * time.Minute,
		},
	}
	
	execution := &TaskExecution{
		ID:          "task-123",
		Status:      "completed",
		StartedAt:   time.Now().Add(-5 * time.Minute),
		CompletedAt: time.Now(),
		Output:      map[string]interface{}{"result": "test output"},
		Metadata:    map[string]interface{}{"test": "metadata"},
	}
	
	provider := &Provider{
		ID:     "provider-123",
		Name:   "Test Provider",
		Region: "US",
		Score:  0.95,
		Resources: ProviderResources{
			CPU:    4,
			Memory: 8192,
		},
	}
	
	receipt, err := engine.generateReceipt(jobspec, execution, provider)
	if err != nil {
		t.Fatalf("Receipt generation failed: %v", err)
	}
	
	// Validate receipt structure
	if receipt.ID == "" {
		t.Error("Receipt ID is empty")
	}
	
	if receipt.JobSpecID != jobspec.ID {
		t.Errorf("Expected JobSpec ID %s, got %s", jobspec.ID, receipt.JobSpecID)
	}
	
	if receipt.ExecutionDetails.TaskID != execution.ID {
		t.Errorf("Expected task ID %s, got %s", execution.ID, receipt.ExecutionDetails.TaskID)
	}
	
	if receipt.ExecutionDetails.ProviderID != provider.ID {
		t.Errorf("Expected provider ID %s, got %s", provider.ID, receipt.ExecutionDetails.ProviderID)
	}
	
	if receipt.ExecutionDetails.Region != provider.Region {
		t.Errorf("Expected region %s, got %s", provider.Region, receipt.ExecutionDetails.Region)
	}
	
	if receipt.Signature == "" {
		t.Error("Receipt signature is empty")
	}
	
	if receipt.PublicKey == "" {
		t.Error("Receipt public key is empty")
	}
	
	// Validate provenance info
	if receipt.Provenance.BenchmarkHash != jobspec.Benchmark.Input.Hash {
		t.Errorf("Expected benchmark hash %s, got %s", 
			jobspec.Benchmark.Input.Hash, receipt.Provenance.BenchmarkHash)
	}
	
	if receipt.Provenance.ProviderInfo == nil {
		t.Error("Provider info is nil")
	}
	
	if receipt.Provenance.ExecutionEnv == nil {
		t.Error("Execution environment is nil")
	}
	
	t.Logf("✅ Receipt generated successfully")
	t.Logf("   - Receipt ID: %s", receipt.ID)
	t.Logf("   - Duration: %v", receipt.ExecutionDetails.Duration)
	t.Logf("   - Provider: %s (%s)", provider.Name, provider.Region)
}

func TestConcurrentExecution(t *testing.T) {
	service := NewService("test-api-key", "testnet")
	engine := NewExecutionEngine(service)
	ctx := context.Background()
	
	// Create multiple JobSpecs for concurrent execution
	jobspecs := []*models.JobSpec{
		{
			ID:      "concurrent-test-001",
			Version: "1.0",
			Benchmark: models.BenchmarkSpec{
				Name: "Concurrent Test 1",
				Container: models.ContainerSpec{
					Image: "test/benchmark:latest",
				},
				Input: models.InputSpec{
					Hash: "hash1",
				},
			},
			Constraints: models.ExecutionConstraints{
				Regions:        []string{"US"},
				MinRegions:     1,
				MinSuccessRate: 0.000001,
				ProviderTimeout: 10 * time.Second,
				Timeout:        5 * time.Minute,
			},
		},
		{
			ID:      "concurrent-test-002",
			Version: "1.0",
			Benchmark: models.BenchmarkSpec{
				Name: "Concurrent Test 2",
				Container: models.ContainerSpec{
					Image: "test/benchmark:latest",
				},
				Input: models.InputSpec{
					Hash: "hash2",
				},
			},
			Constraints: models.ExecutionConstraints{
				Regions:        []string{"EU"},
				MinRegions:     1,
				MinSuccessRate: 0.000001,
				ProviderTimeout: 10 * time.Second,
				Timeout:        5 * time.Minute,
			},
		},
	}
	
	// Execute concurrently
	results := make(chan *ExecutionSummary, len(jobspecs))
	errors := make(chan error, len(jobspecs))
	
	for _, jobspec := range jobspecs {
		go func(js *models.JobSpec) {
			summary, err := engine.ExecuteMultiRegion(ctx, js)
			if err != nil {
				errors <- err
				return
			}
			results <- summary
		}(jobspec)
	}
	
	// Collect results
	var summaries []*ExecutionSummary
	for i := 0; i < len(jobspecs); i++ {
		select {
		case summary := <-results:
			summaries = append(summaries, summary)
		case err := <-errors:
			t.Errorf("Concurrent execution failed: %v", err)
		case <-time.After(30 * time.Second):
			t.Fatal("Concurrent execution timed out")
		}
	}
	
	if len(summaries) != len(jobspecs) {
		t.Errorf("Expected %d summaries, got %d", len(jobspecs), len(summaries))
	}
	
	// Validate that each JobSpec was executed
	executedJobs := make(map[string]bool)
	for _, summary := range summaries {
		executedJobs[summary.JobSpecID] = true
	}
	
	for _, jobspec := range jobspecs {
		if !executedJobs[jobspec.ID] {
			t.Errorf("JobSpec %s was not executed", jobspec.ID)
		}
	}
	
	t.Logf("✅ Concurrent execution completed successfully")
	t.Logf("   - Executed %d jobs concurrently", len(summaries))
}
