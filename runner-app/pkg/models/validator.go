package models

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// JobSpecValidator provides validation utilities for JobSpec
type JobSpecValidator struct{}

// NewJobSpecValidator creates a new validator instance
func NewJobSpecValidator() *JobSpecValidator {
	return &JobSpecValidator{}
}

// ValidateAndVerify performs both structural validation and signature verification
func (v *JobSpecValidator) ValidateAndVerify(jobSpec *JobSpec) error {
	// Check if signature is required (this should be configurable, but for now enforce it)
	if jobSpec.Signature == "" || jobSpec.PublicKey == "" {
		return fmt.Errorf("signature is required")
	}
	
	// Verify signature first (before validation which generates ID)
	if err := jobSpec.VerifySignature(); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	// Then validate (this may generate ID if missing)
	if err := jobSpec.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}

// ValidateJobSpecID checks if the job ID follows the expected format
func (v *JobSpecValidator) ValidateJobSpecID(id string) error {
	if len(id) == 0 {
		return fmt.Errorf("job ID cannot be empty")
	}
	if len(id) > 255 {
		return fmt.Errorf("job ID too long: max 255 characters")
	}
	
	// Allow alphanumeric, hyphens, and underscores
	validID := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validID.MatchString(id) {
		return fmt.Errorf("job ID contains invalid characters: only alphanumeric, hyphens, and underscores allowed")
	}
	
	return nil
}

// ValidateContainerImage checks if the container image reference is valid
func (v *JobSpecValidator) ValidateContainerImage(image, tag string) error {
	if image == "" {
		return fmt.Errorf("container image cannot be empty")
	}
	
	// Basic image name validation
	validImage := regexp.MustCompile(`^[a-z0-9]+([._-][a-z0-9]+)*(/[a-z0-9]+([._-][a-z0-9]+)*)*$`)
	if !validImage.MatchString(image) {
		return fmt.Errorf("invalid container image format")
	}
	
	if tag != "" {
		validTag := regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)
		if !validTag.MatchString(tag) {
			return fmt.Errorf("invalid container tag format")
		}
	}
	
	return nil
}

// ValidateRegions checks if the specified regions are supported
func (v *JobSpecValidator) ValidateRegions(regions []string, minRegions int) error {
	supportedRegions := map[string]bool{
		"US":   true,
		"EU":   true,
		"APAC": true,
		"CA":   true,
		"SA":   true,
		"AF":   true,
		"ME":   true,
	}
	
	if len(regions) < minRegions {
		return fmt.Errorf("insufficient regions: need at least %d, got %d", minRegions, len(regions))
	}
	
	for _, region := range regions {
		if !supportedRegions[region] {
			return fmt.Errorf("unsupported region: %s", region)
		}
	}
	
	return nil
}

// ValidateTimeout checks if the timeout is within acceptable bounds
func (v *JobSpecValidator) ValidateTimeout(timeout time.Duration) error {
	minTimeout := 30 * time.Second
	maxTimeout := 60 * time.Minute
	
	if timeout < minTimeout {
		return fmt.Errorf("timeout too short: minimum %v", minTimeout)
	}
	if timeout > maxTimeout {
		return fmt.Errorf("timeout too long: maximum %v", maxTimeout)
	}
	
	return nil
}

// ValidateInputHash checks if the input hash is properly formatted
func (v *JobSpecValidator) ValidateInputHash(hash string) error {
	if hash == "" {
		return fmt.Errorf("input hash cannot be empty")
	}
	
	// Support SHA256 hashes with optional prefix
	hash = strings.TrimPrefix(hash, "sha256:")
	
	if len(hash) != 64 {
		return fmt.Errorf("invalid hash length: expected 64 characters for SHA256")
	}
	
	validHex := regexp.MustCompile(`^[a-fA-F0-9]+$`)
	if !validHex.MatchString(hash) {
		return fmt.Errorf("invalid hash format: must be hexadecimal")
	}
	
	return nil
}

// ComputeJobSpecHash computes a deterministic hash of the JobSpec (excluding signature fields)
func (v *JobSpecValidator) ComputeJobSpecHash(jobspec *JobSpec) (string, error) {
	// Create a copy without signature fields for hashing
	jobspecCopy := *jobspec
	jobspecCopy.Signature = ""
	jobspecCopy.PublicKey = ""
	
	// Serialize to canonical JSON
	jsonBytes, err := json.Marshal(jobspecCopy)
	if err != nil {
		return "", fmt.Errorf("failed to marshal jobspec for hashing: %w", err)
	}
	
	// Compute SHA256 hash
	hash := sha256.Sum256(jsonBytes)
	return hex.EncodeToString(hash[:]), nil
}

// SanitizeJobSpec removes or normalizes potentially problematic fields
func (v *JobSpecValidator) SanitizeJobSpec(jobspec *JobSpec) {
	// Normalize regions to uppercase
	for i, region := range jobspec.Constraints.Regions {
		jobspec.Constraints.Regions[i] = strings.ToUpper(region)
	}
	
	// Set default values if missing
	if jobspec.Constraints.MinRegions == 0 {
		jobspec.Constraints.MinRegions = len(jobspec.Constraints.Regions)
		if jobspec.Constraints.MinRegions == 0 {
			jobspec.Constraints.MinRegions = 3
		}
	}
	
	if jobspec.Constraints.Timeout == 0 {
		jobspec.Constraints.Timeout = 10 * time.Minute
	}
	
	// Ensure metadata is initialized
	if jobspec.Metadata == nil {
		jobspec.Metadata = make(map[string]interface{})
	}
	
	if jobspec.Benchmark.Metadata == nil {
		jobspec.Benchmark.Metadata = make(map[string]interface{})
	}
}

// ExtractJobSpecSummary creates a summary of the JobSpec for logging/display
func (v *JobSpecValidator) ExtractJobSpecSummary(jobspec *JobSpec) map[string]interface{} {
	summary := map[string]interface{}{
		"id":             jobspec.ID,
		"version":        jobspec.Version,
		"benchmark_name": jobspec.Benchmark.Name,
		"container":      jobspec.Benchmark.Container.Image + ":" + jobspec.Benchmark.Container.Tag,
		"regions":        jobspec.Constraints.Regions,
		"min_regions":    jobspec.Constraints.MinRegions,
		"timeout":        jobspec.Constraints.Timeout.String(),
		"created_at":     jobspec.CreatedAt.Format(time.RFC3339),
		"has_signature":  jobspec.Signature != "",
	}
	
	// Include questions if present (critical for bias-detection jobs)
	if len(jobspec.Questions) > 0 {
		summary["questions"] = jobspec.Questions
		summary["questions_count"] = len(jobspec.Questions)
	}
	
	return summary
}
