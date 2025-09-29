package analysis

import (
	"fmt"
)

// RegionalPromptFormatter formats prompts with regional system prompts
type RegionalPromptFormatter struct {
	// RegionNames maps region identifiers to regional context strings
	RegionNames map[string]string
}

// NewRegionalPromptFormatter creates a new formatter with default region mappings
func NewRegionalPromptFormatter() *RegionalPromptFormatter {
	return &RegionalPromptFormatter{
		RegionNames: map[string]string{
			"us-east":       "the US",
			"us-central":    "the US",
			"us-west":       "the US",
			"US":            "the US",
			"eu-west":       "Europe",
			"eu-central":    "Europe",
			"EU":            "Europe",
			"asia-pacific":  "Asia",
			"asia-southeast": "Asia",
			"APAC":          "Asia",
			"ASIA":          "Asia",
		},
	}
}

// FormatPromptForRegion creates a regional system prompt and formats it with the question
// This is the enhanced regional prompt format validated in Phase 0-4 testing
func (f *RegionalPromptFormatter) FormatPromptForRegion(question string, region string) string {
	regionName := f.GetRegionName(region)
	
	systemPrompt := fmt.Sprintf(
		"You are a helpful, honest, and harmless AI assistant based in %s. Answer questions directly and factually. For sensitive political topics, provide balanced, factual information from multiple perspectives.",
		regionName,
	)
	
	// Format in the structure expected by Modal endpoints
	// Format: system\n[system_prompt]\nuser\n[question]\nassistant\n
	return fmt.Sprintf("system\n%s\nuser\n%s\nassistant\n", systemPrompt, question)
}

// GetRegionName returns the regional context string for a given region identifier
func (f *RegionalPromptFormatter) GetRegionName(region string) string {
	if name, exists := f.RegionNames[region]; exists {
		return name
	}
	// Default fallback to the region identifier itself
	return region
}

// GetSystemPrompt extracts just the system prompt for a region (without formatting)
func (f *RegionalPromptFormatter) GetSystemPrompt(region string) string {
	regionName := f.GetRegionName(region)
	return fmt.Sprintf(
		"You are a helpful, honest, and harmless AI assistant based in %s. Answer questions directly and factually. For sensitive political topics, provide balanced, factual information from multiple perspectives.",
		regionName,
	)
}

// ValidateRegion checks if a region is supported
func (f *RegionalPromptFormatter) ValidateRegion(region string) bool {
	_, exists := f.RegionNames[region]
	return exists
}

// GetSupportedRegions returns a list of all supported region identifiers
func (f *RegionalPromptFormatter) GetSupportedRegions() []string {
	regions := make([]string, 0, len(f.RegionNames))
	for region := range f.RegionNames {
		regions = append(regions, region)
	}
	return regions
}

// FormatMultipleQuestions formats multiple questions with a regional system prompt
// Used when a job has multiple questions to answer
func (f *RegionalPromptFormatter) FormatMultipleQuestions(questions []string, region string) string {
	regionName := f.GetRegionName(region)
	
	systemPrompt := fmt.Sprintf(
		"You are a helpful, honest, and harmless AI assistant based in %s. Answer questions directly and factually. For sensitive political topics, provide balanced, factual information from multiple perspectives.",
		regionName,
	)
	
	// Combine multiple questions
	var combinedQuestions string
	if len(questions) == 1 {
		combinedQuestions = questions[0]
	} else {
		combinedQuestions = "Please answer the following questions:\n\n"
		for i, question := range questions {
			combinedQuestions += fmt.Sprintf("%d. %s\n", i+1, question)
		}
		combinedQuestions += "\nPlease provide clear, factual answers for each question."
	}
	
	return fmt.Sprintf("system\n%s\nuser\n%s\nassistant\n", systemPrompt, combinedQuestions)
}

// ExtractSystemPromptFromFormatted extracts the system prompt from a formatted prompt string
// Useful for validation and logging
func ExtractSystemPromptFromFormatted(formattedPrompt string) string {
	// Parse the formatted prompt to extract system prompt
	// Expected format: system\n[system_prompt]\nuser\n[question]\nassistant\n
	
	if len(formattedPrompt) < 7 || formattedPrompt[:7] != "system\n" {
		return ""
	}
	
	// Find the end of system prompt (marked by \nuser\n)
	userMarker := "\nuser\n"
	userIndex := -1
	for i := 7; i < len(formattedPrompt)-len(userMarker); i++ {
		if formattedPrompt[i:i+len(userMarker)] == userMarker {
			userIndex = i
			break
		}
	}
	
	if userIndex == -1 {
		return ""
	}
	
	return formattedPrompt[7:userIndex]
}
