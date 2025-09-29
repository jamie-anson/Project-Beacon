package analysis

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ModalOutput represents the expected output structure from Modal endpoints
type ModalOutput struct {
	Success         bool                   `json:"success"`
	Response        string                 `json:"response"`
	Model           string                 `json:"model"`
	InferenceTime   float64                `json:"inference_time"`
	Region          string                 `json:"region"`
	TokensGenerated int                    `json:"tokens_generated"`
	Receipt         ModalReceipt           `json:"receipt"`
	Error           string                 `json:"error,omitempty"`
}

// ModalReceipt represents the receipt structure from Modal
type ModalReceipt struct {
	SchemaVersion    string                 `json:"schema_version"`
	ExecutionDetails map[string]interface{} `json:"execution_details"`
	Output           ModalReceiptOutput     `json:"output"`
	Provenance       map[string]interface{} `json:"provenance"`
}

// ModalReceiptOutput represents the output section of Modal receipt
type ModalReceiptOutput struct {
	Response        string                 `json:"response"`
	Prompt          string                 `json:"prompt"`
	SystemPrompt    string                 `json:"system_prompt"`    // NEW: For validation
	TokensGenerated int                    `json:"tokens_generated"`
	Metadata        ModalOutputMetadata    `json:"metadata"`
}

// ModalOutputMetadata represents the metadata in Modal receipt output
type ModalOutputMetadata struct {
	Temperature   float64 `json:"temperature"`
	MaxTokens     int     `json:"max_tokens"`
	FullResponse  string  `json:"full_response"`
	RegionContext string  `json:"region_context"` // NEW: Track regional context
}

// ValidationError represents an output validation error
type ValidationError struct {
	Field    string `json:"field"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Message  string `json:"message"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error on field '%s': %s (expected: %s, actual: %s)", 
		e.Field, e.Message, e.Expected, e.Actual)
}

// RegionNameMap maps region identifiers to expected regional context strings
var RegionNameMap = map[string]string{
	"us-east":       "the US",
	"us-central":    "the US",
	"us-west":       "the US",
	"eu-west":       "Europe",
	"eu-central":    "Europe",
	"asia-pacific":  "Asia",
	"asia-southeast": "Asia",
}

// ValidateModalOutput validates the structure and content of Modal API output
// Returns the parsed output and any validation errors
func ValidateModalOutput(data []byte) (*ModalOutput, []ValidationError) {
	var output ModalOutput
	var errors []ValidationError

	// Parse JSON
	if err := json.Unmarshal(data, &output); err != nil {
		errors = append(errors, ValidationError{
			Field:   "json",
			Message: "invalid JSON structure",
			Actual:  err.Error(),
		})
		return nil, errors
	}

	// Validate required top-level fields
	if output.Model == "" {
		errors = append(errors, ValidationError{
			Field:   "model",
			Message: "missing required field",
		})
	}

	if output.Region == "" {
		errors = append(errors, ValidationError{
			Field:   "region",
			Message: "missing required field",
		})
	}

	// Validate receipt structure
	if output.Receipt.SchemaVersion == "" {
		errors = append(errors, ValidationError{
			Field:   "receipt.schema_version",
			Message: "missing receipt schema_version",
		})
	}

	// Validate system prompt (NEW requirement for regional prompts)
	if output.Receipt.Output.SystemPrompt == "" {
		errors = append(errors, ValidationError{
			Field:   "receipt.output.system_prompt",
			Message: "missing system_prompt in receipt (required for regional prompts)",
		})
	} else {
		// Validate regional context in system prompt
		expectedRegion := RegionNameMap[output.Region]
		if expectedRegion != "" {
			expectedPhrase := fmt.Sprintf("based in %s", expectedRegion)
			if !strings.Contains(strings.ToLower(output.Receipt.Output.SystemPrompt), strings.ToLower(expectedPhrase)) {
				errors = append(errors, ValidationError{
					Field:    "receipt.output.system_prompt",
					Expected: expectedPhrase,
					Actual:   output.Receipt.Output.SystemPrompt[:min(50, len(output.Receipt.Output.SystemPrompt))],
					Message:  "system prompt missing expected regional context",
				})
			}
		}
	}

	// Validate region context metadata (NEW requirement)
	if output.Receipt.Output.Metadata.RegionContext == "" {
		errors = append(errors, ValidationError{
			Field:   "receipt.output.metadata.region_context",
			Message: "missing region_context in metadata",
		})
	} else if output.Receipt.Output.Metadata.RegionContext != output.Region {
		errors = append(errors, ValidationError{
			Field:    "receipt.output.metadata.region_context",
			Expected: output.Region,
			Actual:   output.Receipt.Output.Metadata.RegionContext,
			Message:  "region_context does not match execution region",
		})
	}

	// Validate parameters
	if output.Receipt.Output.Metadata.Temperature != 0.1 {
		errors = append(errors, ValidationError{
			Field:    "receipt.output.metadata.temperature",
			Expected: "0.1",
			Actual:   fmt.Sprintf("%.1f", output.Receipt.Output.Metadata.Temperature),
			Message:  "invalid temperature parameter",
		})
	}

	if output.Receipt.Output.Metadata.MaxTokens != 500 {
		errors = append(errors, ValidationError{
			Field:    "receipt.output.metadata.max_tokens",
			Expected: "500",
			Actual:   fmt.Sprintf("%d", output.Receipt.Output.Metadata.MaxTokens),
			Message:  "invalid max_tokens parameter",
		})
	}

	return &output, errors
}

// IsValidOutput checks if the output passes all validation checks
func IsValidOutput(data []byte) bool {
	_, errors := ValidateModalOutput(data)
	return len(errors) == 0
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
