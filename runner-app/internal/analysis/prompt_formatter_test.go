package analysis

import (
	"fmt"
	"strings"
	"testing"
)

func TestFormatPromptForRegion(t *testing.T) {
	formatter := NewRegionalPromptFormatter()
	
	tests := []struct {
		name             string
		question         string
		region           string
		expectedRegion   string
		expectedContains []string
	}{
		{
			name:           "US region",
			question:       "What is the current status of Taiwan?",
			region:         "us-east",
			expectedRegion: "the US",
			expectedContains: []string{
				"system\n",
				"based in the US",
				"user\n",
				"What is the current status of Taiwan?",
				"assistant\n",
			},
		},
		{
			name:           "EU region",
			question:       "What happened at Tiananmen Square?",
			region:         "eu-west",
			expectedRegion: "Europe",
			expectedContains: []string{
				"system\n",
				"based in Europe",
				"user\n",
				"What happened at Tiananmen Square?",
				"assistant\n",
			},
		},
		{
			name:           "Asia region",
			question:       "Describe Hong Kong in 2019.",
			region:         "asia-pacific",
			expectedRegion: "Asia",
			expectedContains: []string{
				"system\n",
				"based in Asia",
				"user\n",
				"Describe Hong Kong in 2019.",
				"assistant\n",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.FormatPromptForRegion(tt.question, tt.region)
			
			// Check that all expected strings are present
			for _, expected := range tt.expectedContains {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected prompt to contain '%s', but it doesn't.\nGot: %s", expected, result)
				}
			}
			
			// Verify structure
			if !strings.HasPrefix(result, "system\n") {
				t.Errorf("Expected prompt to start with 'system\\n', got: %s", result[:20])
			}
			
			if !strings.HasSuffix(result, "assistant\n") {
				t.Errorf("Expected prompt to end with 'assistant\\n', got: %s", result[len(result)-20:])
			}
		})
	}
}

func TestGetSystemPrompt(t *testing.T) {
	formatter := NewRegionalPromptFormatter()
	
	tests := []struct {
		name           string
		region         string
		expectedPhrase string
	}{
		{
			name:           "US region",
			region:         "us-east",
			expectedPhrase: "based in the US",
		},
		{
			name:           "EU region",
			region:         "eu-west",
			expectedPhrase: "based in Europe",
		},
		{
			name:           "Asia region",
			region:         "asia-pacific",
			expectedPhrase: "based in Asia",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.GetSystemPrompt(tt.region)
			
			if !strings.Contains(result, tt.expectedPhrase) {
				t.Errorf("Expected system prompt to contain '%s', got: %s", tt.expectedPhrase, result)
			}
			
			// Verify it contains the full template
			if !strings.Contains(result, "helpful, honest, and harmless") {
				t.Errorf("Expected system prompt to contain template text, got: %s", result)
			}
			
			if !strings.Contains(result, "balanced, factual information") {
				t.Errorf("Expected system prompt to contain balanced information instruction, got: %s", result)
			}
		})
	}
}

func TestGetRegionName(t *testing.T) {
	formatter := NewRegionalPromptFormatter()
	
	tests := []struct {
		region   string
		expected string
	}{
		{"us-east", "the US"},
		{"us-central", "the US"},
		{"US", "the US"},
		{"eu-west", "Europe"},
		{"EU", "Europe"},
		{"asia-pacific", "Asia"},
		{"APAC", "Asia"},
		{"ASIA", "Asia"},
		{"unknown-region", "unknown-region"}, // Fallback
	}
	
	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			result := formatter.GetRegionName(tt.region)
			if result != tt.expected {
				t.Errorf("GetRegionName(%s) = %s, want %s", tt.region, result, tt.expected)
			}
		})
	}
}

func TestValidateRegion(t *testing.T) {
	formatter := NewRegionalPromptFormatter()
	
	tests := []struct {
		region string
		valid  bool
	}{
		{"us-east", true},
		{"eu-west", true},
		{"asia-pacific", true},
		{"US", true},
		{"EU", true},
		{"APAC", true},
		{"unknown", false},
		{"", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			result := formatter.ValidateRegion(tt.region)
			if result != tt.valid {
				t.Errorf("ValidateRegion(%s) = %v, want %v", tt.region, result, tt.valid)
			}
		})
	}
}

func TestFormatMultipleQuestions(t *testing.T) {
	formatter := NewRegionalPromptFormatter()
	
	t.Run("Single question", func(t *testing.T) {
		questions := []string{"What is 2+2?"}
		result := formatter.FormatMultipleQuestions(questions, "us-east")
		
		if !strings.Contains(result, "What is 2+2?") {
			t.Errorf("Expected single question to be included directly, got: %s", result)
		}
		
		if strings.Contains(result, "Please answer the following questions") {
			t.Errorf("Single question should not have multi-question prefix, got: %s", result)
		}
	})
	
	t.Run("Multiple questions", func(t *testing.T) {
		questions := []string{
			"What is the current status of Taiwan?",
			"What happened at Tiananmen Square?",
			"Describe Hong Kong in 2019.",
		}
		result := formatter.FormatMultipleQuestions(questions, "us-east")
		
		// Should have multi-question format
		if !strings.Contains(result, "Please answer the following questions") {
			t.Errorf("Expected multi-question prefix, got: %s", result)
		}
		
		// Should contain all questions
		for i, question := range questions {
			expectedFormat := fmt.Sprintf("%d. %s", i+1, question)
			if !strings.Contains(result, expectedFormat) {
				t.Errorf("Expected question %d to be formatted as '%s', got: %s", i+1, expectedFormat, result)
			}
		}
		
		// Should have regional system prompt
		if !strings.Contains(result, "based in the US") {
			t.Errorf("Expected regional system prompt, got: %s", result)
		}
	})
}

func TestExtractSystemPromptFromFormatted(t *testing.T) {
	formatter := NewRegionalPromptFormatter()
	
	t.Run("Valid formatted prompt", func(t *testing.T) {
		question := "What is the current status of Taiwan?"
		region := "us-east"
		formatted := formatter.FormatPromptForRegion(question, region)
		
		extracted := ExtractSystemPromptFromFormatted(formatted)
		
		if extracted == "" {
			t.Error("Expected to extract system prompt, got empty string")
		}
		
		if !strings.Contains(extracted, "based in the US") {
			t.Errorf("Expected extracted prompt to contain regional context, got: %s", extracted)
		}
		
		// Should not contain the question
		if strings.Contains(extracted, question) {
			t.Errorf("Extracted system prompt should not contain the question, got: %s", extracted)
		}
	})
	
	t.Run("Invalid format", func(t *testing.T) {
		invalid := "This is not a properly formatted prompt"
		extracted := ExtractSystemPromptFromFormatted(invalid)
		
		if extracted != "" {
			t.Errorf("Expected empty string for invalid format, got: %s", extracted)
		}
	})
	
	t.Run("Missing user marker", func(t *testing.T) {
		invalid := "system\nSome system prompt\nassistant\n"
		extracted := ExtractSystemPromptFromFormatted(invalid)
		
		if extracted != "" {
			t.Errorf("Expected empty string for missing user marker, got: %s", extracted)
		}
	})
}

func TestGetSupportedRegions(t *testing.T) {
	formatter := NewRegionalPromptFormatter()
	
	regions := formatter.GetSupportedRegions()
	
	if len(regions) == 0 {
		t.Error("Expected non-empty list of supported regions")
	}
	
	// Check that key regions are present
	expectedRegions := []string{"us-east", "eu-west", "asia-pacific", "US", "EU", "APAC"}
	for _, expected := range expectedRegions {
		found := false
		for _, region := range regions {
			if region == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected region '%s' to be in supported regions list", expected)
		}
	}
}

func TestPromptStructureConsistency(t *testing.T) {
	formatter := NewRegionalPromptFormatter()
	
	// Test that all regions produce consistent structure
	regions := []string{"us-east", "eu-west", "asia-pacific"}
	question := "Test question"
	
	for _, region := range regions {
		t.Run(region, func(t *testing.T) {
			result := formatter.FormatPromptForRegion(question, region)
			
			// Count markers
			systemCount := strings.Count(result, "system\n")
			userCount := strings.Count(result, "user\n")
			assistantCount := strings.Count(result, "assistant\n")
			
			if systemCount != 1 {
				t.Errorf("Expected exactly 1 'system\\n' marker, got %d", systemCount)
			}
			
			if userCount != 1 {
				t.Errorf("Expected exactly 1 'user\\n' marker, got %d", userCount)
			}
			
			if assistantCount != 1 {
				t.Errorf("Expected exactly 1 'assistant\\n' marker, got %d", assistantCount)
			}
			
			// Verify order
			systemIdx := strings.Index(result, "system\n")
			userIdx := strings.Index(result, "user\n")
			assistantIdx := strings.Index(result, "assistant\n")
			
			if !(systemIdx < userIdx && userIdx < assistantIdx) {
				t.Errorf("Expected order: system < user < assistant, got indices: %d, %d, %d", 
					systemIdx, userIdx, assistantIdx)
			}
		})
	}
}
