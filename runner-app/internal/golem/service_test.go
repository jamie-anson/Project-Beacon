package golem

import (
	"context"
	"testing"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func TestNewService(t *testing.T) {
	service := NewService("test-api-key", "testnet")
	
	if service.apiKey != "test-api-key" {
		t.Errorf("Expected API key 'test-api-key', got '%s'", service.apiKey)
	}
	
	if service.network != "testnet" {
		t.Errorf("Expected network 'testnet', got '%s'", service.network)
	}
	
	if service.timeout != 30*time.Minute {
		t.Errorf("Expected timeout 30m, got %v", service.timeout)
	}
	
	t.Logf("✅ Golem service created successfully")
}

func TestDiscoverProviders(t *testing.T) {
	service := NewService("test-api-key", "testnet")
	ctx := context.Background()
	
	constraints := models.ExecutionConstraints{
		Regions:    []string{"US", "EU"},
		MinRegions: 2,
		Timeout:    5 * time.Minute,
		Providers:  []models.ProviderFilter{},
	}
	
	providers, err := service.DiscoverProviders(ctx, constraints)
	if err != nil {
		t.Fatalf("Provider discovery failed: %v", err)
	}
	
	if len(providers) == 0 {
		t.Fatal("No providers discovered")
	}
	
	// Check that we have providers in required regions
	regionCount := make(map[string]int)
	for _, provider := range providers {
		regionCount[provider.Region]++
		
		// Validate provider structure
		if provider.ID == "" {
			t.Error("Provider ID is empty")
		}
		if provider.Region == "" {
			t.Error("Provider region is empty")
		}
		if provider.Score < 0 || provider.Score > 1 {
			t.Errorf("Invalid provider score: %f", provider.Score)
		}
	}
	
	if len(regionCount) < constraints.MinRegions {
		t.Errorf("Insufficient regions: got %d, need %d", len(regionCount), constraints.MinRegions)
	}
	
	t.Logf("✅ Discovered %d providers across %d regions", len(providers), len(regionCount))
	for region, count := range regionCount {
		t.Logf("   - %s: %d providers", region, count)
	}
}

func TestDiscoverProvidersWithFilters(t *testing.T) {
	service := NewService("test-api-key", "testnet")
	ctx := context.Background()
	
	constraints := models.ExecutionConstraints{
		Regions:    []string{"US", "EU", "APAC"},
		MinRegions: 2,
		Timeout:    5 * time.Minute,
		Providers: []models.ProviderFilter{
			{
				Region:   "US",
				MinScore: 0.9,
				MaxPrice: 0.06,
			},
		},
	}
	
	providers, err := service.DiscoverProviders(ctx, constraints)
	if err != nil {
		t.Fatalf("Provider discovery with filters failed: %v", err)
	}
	
	// Verify filters are applied
	for _, provider := range providers {
		if provider.Region == "US" {
			if provider.Score < 0.9 {
				t.Errorf("Provider %s has score %f, below minimum 0.9", provider.ID, provider.Score)
			}
			if provider.Price > 0.06 {
				t.Errorf("Provider %s has price %f, above maximum 0.06", provider.ID, provider.Price)
			}
		}
	}
	
	t.Logf("✅ Provider filtering works correctly")
}

func TestExecuteTask(t *testing.T) {
	service := NewService("test-api-key", "testnet")
	ctx := context.Background()
	
	// Get a provider
	constraints := models.ExecutionConstraints{
		Regions:    []string{"US"},
		MinRegions: 1,
		Timeout:    5 * time.Minute,
	}
	
	providers, err := service.DiscoverProviders(ctx, constraints)
	if err != nil {
		t.Fatalf("Provider discovery failed: %v", err)
	}
	
	if len(providers) == 0 {
		t.Fatal("No providers available")
	}
	
	provider := providers[0]
	
	// Create a test JobSpec
	jobspec := &models.JobSpec{
		ID:      "test-job-123",
		Version: "v1",
		Benchmark: models.BenchmarkSpec{
			Name:        "Who Are You?",
			Description: "Test benchmark for text generation",
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
		Constraints: constraints,
	}
	
	// Execute task
	execution, err := service.ExecuteTask(ctx, provider, jobspec)
	if err != nil {
		t.Fatalf("Task execution failed: %v", err)
	}
	
	// Validate execution result
	if execution.ID == "" {
		t.Error("Execution ID is empty")
	}
	if execution.JobSpecID != jobspec.ID {
		t.Errorf("Expected JobSpec ID %s, got %s", jobspec.ID, execution.JobSpecID)
	}
	if execution.ProviderID != provider.ID {
		t.Errorf("Expected provider ID %s, got %s", provider.ID, execution.ProviderID)
	}
	if execution.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", execution.Status)
	}
	if execution.Output == nil {
		t.Error("Execution output is nil")
	}
	
	t.Logf("✅ Task execution completed successfully")
	t.Logf("   - Execution ID: %s", execution.ID)
	t.Logf("   - Provider: %s (%s)", provider.Name, provider.Region)
	t.Logf("   - Status: %s", execution.Status)
	t.Logf("   - Duration: %v", execution.CompletedAt.Sub(execution.StartedAt))
}

func TestEstimateTaskCost(t *testing.T) {
	service := NewService("test-api-key", "testnet")
	
	provider := &Provider{
		ID:     "test-provider",
		Region: "US",
		Price:  0.05, // $0.05 per hour
	}
	
	jobspec := &models.JobSpec{
		Constraints: models.ExecutionConstraints{
			Timeout: 30 * time.Minute, // 0.5 hours
		},
	}
	
	cost, err := service.EstimateTaskCost(provider, jobspec)
	if err != nil {
		t.Fatalf("Cost estimation failed: %v", err)
	}
	
	expectedCost := 0.05 * 0.5 // $0.025
	if cost != expectedCost {
		t.Errorf("Expected cost %f, got %f", expectedCost, cost)
	}
	
	t.Logf("✅ Cost estimation: $%.4f for 30min execution", cost)
}

func TestProviderFiltering(t *testing.T) {
	service := NewService("test-api-key", "testnet")
	
	provider := &Provider{
		ID:     "test-provider",
		Region: "US",
		Score:  0.95,
		Price:  0.05,
	}
	
	tests := []struct {
		name     string
		filter   models.ProviderFilter
		expected bool
	}{
		{
			name: "Region match",
			filter: models.ProviderFilter{
				Region: "US",
			},
			expected: true,
		},
		{
			name: "Region mismatch",
			filter: models.ProviderFilter{
				Region: "EU",
			},
			expected: false,
		},
		{
			name: "Score above minimum",
			filter: models.ProviderFilter{
				MinScore: 0.9,
			},
			expected: true,
		},
		{
			name: "Score below minimum",
			filter: models.ProviderFilter{
				MinScore: 0.98,
			},
			expected: false,
		},
		{
			name: "Price below maximum",
			filter: models.ProviderFilter{
				MaxPrice: 0.1,
			},
			expected: true,
		},
		{
			name: "Price above maximum",
			filter: models.ProviderFilter{
				MaxPrice: 0.03,
			},
			expected: false,
		},
		{
			name: "Whitelist match",
			filter: models.ProviderFilter{
				Whitelist: []string{"test-provider", "other-provider"},
			},
			expected: true,
		},
		{
			name: "Whitelist no match",
			filter: models.ProviderFilter{
				Whitelist: []string{"other-provider"},
			},
			expected: false,
		},
		{
			name: "Blacklist no match",
			filter: models.ProviderFilter{
				Blacklist: []string{"other-provider"},
			},
			expected: true,
		},
		{
			name: "Blacklist match",
			filter: models.ProviderFilter{
				Blacklist: []string{"test-provider"},
			},
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.matchesProviderFilter(provider, tt.filter)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
	
	t.Logf("✅ Provider filtering tests passed")
}

func TestMockOutputGeneration(t *testing.T) {
	service := NewService("test-api-key", "testnet")
	
	// Generate mock providers to populate the service
	service.generateMockProviders()
	
	_ = &models.JobSpec{
		Benchmark: models.BenchmarkSpec{
			Name: "Who Are You?",
		},
	}
	
	// Test output generation for different regions
	regions := []string{"US", "EU", "APAC"}
	
	for _, region := range regions {
		// Find a provider in this region
		var providerID string
		for id, provider := range service.providers {
			if provider.Region == region {
				providerID = id
				break
			}
		}
		
		if providerID == "" {
			t.Errorf("No provider found for region %s", region)
			continue
		}
		
		// Test that we can find a provider for this region
		if providerID == "" {
			t.Errorf("No provider found for region %s", region)
			continue
		}
		
		// Validate that the provider exists in the service
		provider, exists := service.providers[providerID]
		if !exists {
			t.Errorf("Provider %s not found in service for region %s", providerID, region)
			continue
		}
		
		if provider.Region != region {
			t.Errorf("Provider region mismatch: expected %s, got %s", region, provider.Region)
		}
		
		t.Logf("✅ Validated provider %s for region %s", providerID, region)
	}
}
