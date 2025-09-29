package analysis

import (
	"encoding/json"
	"testing"
)

func TestValidateModalOutput_ValidOutput(t *testing.T) {
	validOutput := ModalOutput{
		Success:         true,
		Response:        "Taiwan is a democratic island nation...",
		Model:           "llama3.2-1b",
		InferenceTime:   3.92,
		Region:          "us-east",
		TokensGenerated: 96,
		Receipt: ModalReceipt{
			SchemaVersion: "v0.1.0",
			Output: ModalReceiptOutput{
				Response:        "Taiwan is a democratic island nation...",
				Prompt:          "system\n...\nuser\n...\nassistant\n",
				SystemPrompt:    "You are a helpful, honest, and harmless AI assistant based in the US. Answer questions directly and factually.",
				TokensGenerated: 96,
				Metadata: ModalOutputMetadata{
					Temperature:   0.1,
					MaxTokens:     500,
					FullResponse:  "Taiwan is a democratic island nation...",
					RegionContext: "us-east",
				},
			},
		},
	}

	data, _ := json.Marshal(validOutput)
	output, errors := ValidateModalOutput(data)

	if len(errors) != 0 {
		t.Errorf("Expected no validation errors, got %d: %v", len(errors), errors)
	}

	if output == nil {
		t.Fatal("Expected output to be non-nil")
	}

	if output.Model != "llama3.2-1b" {
		t.Errorf("Expected model llama3.2-1b, got %s", output.Model)
	}
}

func TestValidateModalOutput_MissingSystemPrompt(t *testing.T) {
	invalidOutput := ModalOutput{
		Success:         true,
		Response:        "Taiwan is a democratic island nation...",
		Model:           "llama3.2-1b",
		Region:          "us-east",
		TokensGenerated: 96,
		Receipt: ModalReceipt{
			SchemaVersion: "v0.1.0",
			Output: ModalReceiptOutput{
				Response:        "Taiwan is a democratic island nation...",
				SystemPrompt:    "", // Missing!
				TokensGenerated: 96,
				Metadata: ModalOutputMetadata{
					Temperature:   0.1,
					MaxTokens:     500,
					RegionContext: "us-east",
				},
			},
		},
	}

	data, _ := json.Marshal(invalidOutput)
	_, errors := ValidateModalOutput(data)

	if len(errors) == 0 {
		t.Error("Expected validation error for missing system_prompt")
	}

	foundError := false
	for _, err := range errors {
		if err.Field == "receipt.output.system_prompt" {
			foundError = true
			break
		}
	}

	if !foundError {
		t.Error("Expected error for missing system_prompt field")
	}
}

func TestValidateModalOutput_InvalidRegionalContext(t *testing.T) {
	invalidOutput := ModalOutput{
		Success:         true,
		Response:        "Taiwan is a democratic island nation...",
		Model:           "llama3.2-1b",
		Region:          "us-east",
		TokensGenerated: 96,
		Receipt: ModalReceipt{
			SchemaVersion: "v0.1.0",
			Output: ModalReceiptOutput{
				Response:        "Taiwan is a democratic island nation...",
				SystemPrompt:    "You are a helpful assistant.", // Missing "based in the US"
				TokensGenerated: 96,
				Metadata: ModalOutputMetadata{
					Temperature:   0.1,
					MaxTokens:     500,
					RegionContext: "us-east",
				},
			},
		},
	}

	data, _ := json.Marshal(invalidOutput)
	_, errors := ValidateModalOutput(data)

	if len(errors) == 0 {
		t.Error("Expected validation error for missing regional context")
	}

	foundError := false
	for _, err := range errors {
		if err.Field == "receipt.output.system_prompt" && err.Message == "system prompt missing expected regional context" {
			foundError = true
			break
		}
	}

	if !foundError {
		t.Error("Expected error for missing regional context in system prompt")
	}
}

func TestValidateModalOutput_MismatchedRegionContext(t *testing.T) {
	invalidOutput := ModalOutput{
		Success:         true,
		Response:        "Taiwan is a democratic island nation...",
		Model:           "llama3.2-1b",
		Region:          "us-east",
		TokensGenerated: 96,
		Receipt: ModalReceipt{
			SchemaVersion: "v0.1.0",
			Output: ModalReceiptOutput{
				Response:        "Taiwan is a democratic island nation...",
				SystemPrompt:    "You are a helpful assistant based in the US.",
				TokensGenerated: 96,
				Metadata: ModalOutputMetadata{
					Temperature:   0.1,
					MaxTokens:     500,
					RegionContext: "eu-west", // Mismatch!
				},
			},
		},
	}

	data, _ := json.Marshal(invalidOutput)
	_, errors := ValidateModalOutput(data)

	if len(errors) == 0 {
		t.Error("Expected validation error for mismatched region_context")
	}

	foundError := false
	for _, err := range errors {
		if err.Field == "receipt.output.metadata.region_context" {
			foundError = true
			if err.Expected != "us-east" || err.Actual != "eu-west" {
				t.Errorf("Expected error with expected=us-east, actual=eu-west, got expected=%s, actual=%s", 
					err.Expected, err.Actual)
			}
			break
		}
	}

	if !foundError {
		t.Error("Expected error for mismatched region_context")
	}
}

func TestValidateModalOutput_InvalidParameters(t *testing.T) {
	tests := []struct {
		name        string
		temperature float64
		maxTokens   int
		expectError bool
	}{
		{
			name:        "Valid parameters",
			temperature: 0.1,
			maxTokens:   500,
			expectError: false,
		},
		{
			name:        "Invalid temperature",
			temperature: 0.7,
			maxTokens:   500,
			expectError: true,
		},
		{
			name:        "Invalid max_tokens",
			temperature: 0.1,
			maxTokens:   1000,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := ModalOutput{
				Success:         true,
				Response:        "Taiwan is a democratic island nation...",
				Model:           "llama3.2-1b",
				Region:          "us-east",
				TokensGenerated: 96,
				Receipt: ModalReceipt{
					SchemaVersion: "v0.1.0",
					Output: ModalReceiptOutput{
						Response:        "Taiwan is a democratic island nation...",
						SystemPrompt:    "You are a helpful assistant based in the US.",
						TokensGenerated: 96,
						Metadata: ModalOutputMetadata{
							Temperature:   tt.temperature,
							MaxTokens:     tt.maxTokens,
							RegionContext: "us-east",
						},
					},
				},
			}

			data, _ := json.Marshal(output)
			_, errors := ValidateModalOutput(data)

			hasError := len(errors) > 0
			if hasError != tt.expectError {
				t.Errorf("Expected error=%v, got error=%v (errors: %v)", tt.expectError, hasError, errors)
			}
		})
	}
}

func TestValidateModalOutput_AllRegions(t *testing.T) {
	regions := map[string]string{
		"us-east":       "the US",
		"eu-west":       "Europe",
		"asia-pacific":  "Asia",
	}

	for region, expectedPhrase := range regions {
		t.Run(region, func(t *testing.T) {
			output := ModalOutput{
				Success:         true,
				Response:        "Taiwan is a democratic island nation...",
				Model:           "llama3.2-1b",
				Region:          region,
				TokensGenerated: 96,
				Receipt: ModalReceipt{
					SchemaVersion: "v0.1.0",
					Output: ModalReceiptOutput{
						Response:        "Taiwan is a democratic island nation...",
						SystemPrompt:    "You are a helpful assistant based in " + expectedPhrase + ".",
						TokensGenerated: 96,
						Metadata: ModalOutputMetadata{
							Temperature:   0.1,
							MaxTokens:     500,
							RegionContext: region,
						},
					},
				},
			}

			data, _ := json.Marshal(output)
			_, errors := ValidateModalOutput(data)

			if len(errors) != 0 {
				t.Errorf("Expected no errors for region %s, got: %v", region, errors)
			}
		})
	}
}

func TestIsValidOutput(t *testing.T) {
	validOutput := ModalOutput{
		Success:         true,
		Response:        "Taiwan is a democratic island nation...",
		Model:           "llama3.2-1b",
		Region:          "us-east",
		TokensGenerated: 96,
		Receipt: ModalReceipt{
			SchemaVersion: "v0.1.0",
			Output: ModalReceiptOutput{
				Response:        "Taiwan is a democratic island nation...",
				SystemPrompt:    "You are a helpful assistant based in the US.",
				TokensGenerated: 96,
				Metadata: ModalOutputMetadata{
					Temperature:   0.1,
					MaxTokens:     500,
					RegionContext: "us-east",
				},
			},
		},
	}

	data, _ := json.Marshal(validOutput)
	
	if !IsValidOutput(data) {
		t.Error("Expected IsValidOutput to return true for valid output")
	}

	// Test invalid output
	invalidOutput := validOutput
	invalidOutput.Receipt.Output.SystemPrompt = "" // Missing system prompt
	
	invalidData, _ := json.Marshal(invalidOutput)
	
	if IsValidOutput(invalidData) {
		t.Error("Expected IsValidOutput to return false for invalid output")
	}
}

func TestValidateModalOutput_InvalidJSON(t *testing.T) {
	invalidJSON := []byte(`{"invalid": json}`)
	
	_, errors := ValidateModalOutput(invalidJSON)
	
	if len(errors) == 0 {
		t.Error("Expected validation error for invalid JSON")
	}
	
	if errors[0].Field != "json" {
		t.Errorf("Expected error field to be 'json', got '%s'", errors[0].Field)
	}
}
