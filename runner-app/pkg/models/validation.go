package models

import (
	"fmt"
	"strings"
	"time"
)

// Validate validates the JobSpec and auto-generates ID if missing
func (js *JobSpec) Validate() error {
	// Auto-generate ID if missing (for job creation)
	if js.ID == "" && js.JobSpecID == "" {
		// Generate ID from benchmark name and timestamp
		timestamp := time.Now().Unix()
		if js.Benchmark.Name != "" {
			js.ID = fmt.Sprintf("%s-%d", js.Benchmark.Name, timestamp)
		} else {
			js.ID = fmt.Sprintf("job-%d", timestamp)
		}
	}
	// Normalize: if JobSpecID is provided but ID is empty, copy it over
	if js.ID == "" && js.JobSpecID != "" {
		js.ID = js.JobSpecID
	}
	// Set default version if missing
	if js.Version == "" {
		js.Version = "1.0"
	}
	
	// Handle portal compatibility: if benchmark.version exists but jobspec version doesn't, use it
	if js.Benchmark.Version != "" && js.Version == "1.0" {
		js.Version = js.Benchmark.Version
	}
	
	if js.Version == "" {
		return fmt.Errorf("jobspec version is required")
	}
	if js.Benchmark.Name == "" {
		return fmt.Errorf("benchmark name is required")
	}
	
	// Auto-populate missing benchmark fields for portal compatibility
	if js.Benchmark.Description == "" {
		js.Benchmark.Description = fmt.Sprintf("Benchmark: %s", js.Benchmark.Name)
	}
	if js.Benchmark.Scoring.Method == "" {
		js.Benchmark.Scoring.Method = "default"
		if js.Benchmark.Scoring.Parameters == nil {
			js.Benchmark.Scoring.Parameters = make(map[string]interface{})
		}
	}
	if js.Benchmark.Metadata == nil {
		js.Benchmark.Metadata = make(map[string]interface{})
	}
	
	if js.Benchmark.Container.Image == "" {
		return fmt.Errorf("container image is required")
	}
	if len(js.Constraints.Regions) == 0 {
		return fmt.Errorf("at least one region constraint is required")
	}
	if js.Constraints.MinRegions < 1 {
		js.Constraints.MinRegions = 1 // Default to 1 region for portal compatibility
	}
	if js.Constraints.MinSuccessRate == 0 {
		js.Constraints.MinSuccessRate = 0.67 // Default to 67% success rate
	}
	if js.Constraints.Timeout == 0 {
		js.Constraints.Timeout = 10 * time.Minute // Default timeout
	}
	if js.Constraints.ProviderTimeout == 0 {
		js.Constraints.ProviderTimeout = 5 * time.Minute // Default provider timeout (increased for cold starts)
	}
	if js.Benchmark.Input.Hash == "" {
		return fmt.Errorf("input hash is required for integrity verification")
	}
	// Enforce questions for bias-detection v1
	if strings.EqualFold(js.Version, "v1") {
		name := strings.ToLower(js.Benchmark.Name)
		if strings.Contains(name, "bias") {
			if len(js.Questions) == 0 {
				return fmt.Errorf("questions are required for bias-detection v1 jobspec")
			}
		}
	}
	
	return nil
}

func (js *JobSpec) validateBenchmark() error {
	if js.Benchmark.Name == "" {
		return fmt.Errorf("benchmark name is required")
	}

	// Validate container
	if js.Benchmark.Container.Image == "" {
		return fmt.Errorf("container image is required")
	}

	// Input hash is required for integrity verification
	if js.Benchmark.Input.Hash == "" {
		return fmt.Errorf("input hash is required for integrity verification")
	}

	return nil
}

func (js *JobSpec) validateConstraints() error {
	if len(js.Constraints.Regions) == 0 {
		return fmt.Errorf("at least one region is required")
	}

	if js.Constraints.MinRegions <= 0 {
		js.Constraints.MinRegions = 1 // Default to 1
	}

	if js.Constraints.MinRegions > len(js.Constraints.Regions) {
		return fmt.Errorf("min_regions (%d) cannot exceed available regions (%d)", 
			js.Constraints.MinRegions, len(js.Constraints.Regions))
	}

	if js.Constraints.Timeout <= 0 {
		js.Constraints.Timeout = 5 * time.Minute // Default timeout
	}

	// Validate region names
	validRegions := map[string]bool{
		"US": true, "EU": true, "APAC": true, "ASIA": true,
	}
	for _, region := range js.Constraints.Regions {
		if !validRegions[region] {
			return fmt.Errorf("invalid region: %s (valid regions: US, EU, APAC, ASIA)", region)
		}
	}

	return nil
}
