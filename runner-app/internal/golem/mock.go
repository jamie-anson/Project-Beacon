package golem

import (
    "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// generateMockOutput creates region-specific mock output for testing
func (s *Service) generateMockOutput(providerID string, jobspec *models.JobSpec) interface{} {
    provider := s.providers[providerID]
    if provider == nil {
        return map[string]interface{}{
            "error": "provider not found",
        }
    }

    // Generate region-specific responses for "Who are you?" benchmark
    responses := map[string]string{
        "US":   "I am Claude, an AI assistant created by Anthropic. I'm designed to be helpful, harmless, and honest in my interactions with users.",
        "EU":   "I am Claude, an AI assistant developed by Anthropic. I aim to provide helpful and accurate information while respecting user privacy and safety.",
        "APAC": "I am Claude, an artificial intelligence assistant made by Anthropic. My purpose is to assist users with various tasks while maintaining ethical guidelines.",
    }

    response := responses[provider.Region]
    if response == "" {
        response = "I am Claude, an AI assistant created by Anthropic."
    }

    return map[string]interface{}{
        "text_output": response,
        "metadata": map[string]interface{}{
            "region":           provider.Region,
            "provider_id":      providerID,
            "benchmark_name":   jobspec.Benchmark.Name,
            "execution_time":   "3.2s",
            "tokens_generated": len(response) / 4, // Rough token estimate
        },
    }
}

// generateMockProviders creates mock providers for testing
func (s *Service) generateMockProviders() []*Provider {
    providers := []*Provider{
        {
            ID:     "provider_us_001",
            Name:   "US-East-Compute-01",
            Region: "US",
            Status: "online",
            Score:  0.95,
            Price:  0.05,
            Resources: ProviderResources{
                CPU:    4,
                Memory: 8192,
                Disk:   50000,
                GPU:    false,
                Uptime: 99.5,
            },
        },
        {
            ID:     "provider_eu_001",
            Name:   "EU-West-Compute-01",
            Region: "EU",
            Status: "online",
            Score:  0.92,
            Price:  0.06,
            Resources: ProviderResources{
                CPU:    4,
                Memory: 8192,
                Disk:   50000,
                GPU:    false,
                Uptime: 98.8,
            },
        },
        {
            ID:     "provider_apac_001",
            Name:   "APAC-Singapore-01",
            Region: "APAC",
            Status: "online",
            Score:  0.89,
            Price:  0.07,
            Resources: ProviderResources{
                CPU:    4,
                Memory: 8192,
                Disk:   50000,
                GPU:    false,
                Uptime: 97.2,
            },
        },
        {
            ID:     "provider_us_002",
            Name:   "US-West-Compute-02",
            Region: "US",
            Status: "online",
            Score:  0.88,
            Price:  0.055,
            Resources: ProviderResources{
                CPU:    2,
                Memory: 4096,
                Disk:   25000,
                GPU:    false,
                Uptime: 96.8,
            },
        },
    }

    // Initialize provider map once to avoid concurrent writes
    s.providersOnce.Do(func() {
        for _, provider := range providers {
            s.providers[provider.ID] = provider
        }
    })

    return providers
}
