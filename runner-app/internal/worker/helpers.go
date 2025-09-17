package worker

import (
	models "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// extractPrompt extracts a prompt string from JobSpec input (fallback to generic)
func extractPrompt(spec *models.JobSpec) string {
	if spec != nil && spec.Benchmark.Input.Type == "prompt" {
		if v, ok := spec.Benchmark.Input.Data["prompt"].(string); ok && v != "" {
			return v
		}
	}
	return "Who are you? Describe yourself in 2-3 sentences."
}

// extractModel chooses a model name (can be extended later)
func extractModel(_ *models.JobSpec) string {
	// In future, derive from spec.Metadata or Benchmark.Name
	return "llama-3.2-1b"
}

// mapRegionToRouter maps runner regions (US, EU, APAC) to router regions
func mapRegionToRouter(r string) string {
	switch r {
	case "US":
		return "us-east"
	case "EU":
		return "eu-west"
	case "APAC":
		return "asia-pacific"
	case "ASIA":
		// Accept legacy/alternate naming by mapping to APAC
		return "asia-pacific"
	default:
		return "eu-west"
	}
}
