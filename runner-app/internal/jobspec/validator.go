package jobspec

import (
	"encoding/json"
	"fmt"
	"time"
	"os"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// Validator handles JobSpec validation and processing
type Validator struct{}

// NewValidator creates a new JobSpec validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateJobSpec performs comprehensive validation of a JobSpec
func (v *Validator) ValidateJobSpec(jobspecJSON []byte) (*models.JobSpec, error) {
	var jobspec models.JobSpec
	
	// Validate structure against JSON Schema first
	if err := ValidateJSONSchema(jobspecJSON); err != nil {
		return nil, err
	}

	// Parse JSON
	if err := json.Unmarshal(jobspecJSON, &jobspec); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Set creation time if not provided
	if jobspec.CreatedAt.IsZero() {
		jobspec.CreatedAt = time.Now()
	}

	// Validate structure and required fields
	if err := jobspec.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Verify Ed25519 signature (enabled by default). Allow dev override via env.
	if os.Getenv("VALIDATION_SKIP_SIGNATURE") != "true" {
		if err := jobspec.VerifySignature(); err != nil {
			return nil, fmt.Errorf("signature verification failed: %w", err)
		}
	}

	return &jobspec, nil
}

// CreateSampleJobSpec creates a sample JobSpec for testing
func (v *Validator) CreateSampleJobSpec() *models.JobSpec {
	return &models.JobSpec{
		ID:      "sample-benchmark-001",
		Version: "1.0",
		Benchmark: models.BenchmarkSpec{
			Name:        "Who Are You?",
			Description: "Text generation benchmark to detect regional AI model differences",
			Container: models.ContainerSpec{
				Image: "beacon/text-gen",
				Tag:   "latest",
				Resources: models.ResourceSpec{
					CPU:    "1000m",
					Memory: "512Mi",
				},
			},
			Input: models.InputSpec{
				Type: "prompt",
				Data: map[string]interface{}{
					"prompt": "Who are you? Describe yourself in 2-3 sentences.",
				},
				Hash: "sha256:abc123def456", // Placeholder hash
			},
			Scoring: models.ScoringSpec{
				Method: "similarity",
				Parameters: map[string]interface{}{
					"algorithm": "levenshtein",
					"threshold": 0.8,
				},
			},
		},
		Constraints: models.ExecutionConstraints{
			Regions:         []string{"US", "EU", "APAC"},
			MinRegions:      3,
			MinSuccessRate:  0.67,
			Timeout:         10 * time.Minute,
			ProviderTimeout: 2 * time.Minute,
		},
		Metadata: map[string]interface{}{
			"created_by": "project-beacon-test",
			"purpose":    "mvp-testing",
		},
		CreatedAt: time.Now(),
	}
}

// ValidateRegions checks if the specified regions are supported
func (v *Validator) ValidateRegions(regions []string) error {
	supportedRegions := map[string]bool{
		"US":   true,
		"EU":   true,
		"APAC": true,
		"CA":   true,
		"SA":   true,
		"AF":   true,
		"ME":   true,
	}

	for _, region := range regions {
		if !supportedRegions[region] {
			return fmt.Errorf("unsupported region: %s", region)
		}
	}

	return nil
}

// ValidateContainerSpec checks container specification validity
func (v *Validator) ValidateContainerSpec(container models.ContainerSpec) error {
	if container.Image == "" {
		return fmt.Errorf("container image is required")
	}
	
	if container.Tag == "" {
		container.Tag = "latest" // Default tag
	}

	// Validate resource specifications
	if container.Resources.CPU == "" {
		return fmt.Errorf("CPU resource specification is required")
	}
	
	if container.Resources.Memory == "" {
		return fmt.Errorf("memory resource specification is required")
	}

	return nil
}

// EstimateExecutionCost estimates the cost of running a JobSpec
func (v *Validator) EstimateExecutionCost(jobspec *models.JobSpec) (float64, error) {
	// Simple cost estimation based on regions and timeout
	baseCostPerRegion := 0.10 // $0.10 per region
	timeoutMinutes := float64(jobspec.Constraints.Timeout) / float64(time.Minute)
	timeCostMultiplier := timeoutMinutes / 10.0 // Base cost for 10 minutes

	totalCost := float64(len(jobspec.Constraints.Regions)) * baseCostPerRegion * timeCostMultiplier

	return totalCost, nil
}
