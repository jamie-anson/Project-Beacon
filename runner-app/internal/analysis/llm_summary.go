package analysis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// OpenAISummaryGenerator generates detailed summaries using OpenAI API
type OpenAISummaryGenerator struct {
	apiKey     string
	httpClient *http.Client
}

// NewOpenAISummaryGenerator creates a new summary generator
func NewOpenAISummaryGenerator() *OpenAISummaryGenerator {
	return &OpenAISummaryGenerator{
		apiKey: os.Getenv("OPENAI_API_KEY"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GenerateSummary creates a 400-500 word summary using OpenAI
func (g *OpenAISummaryGenerator) GenerateSummary(ctx context.Context, analysis *models.CrossRegionAnalysis, regionResults map[string]*models.RegionResult) (string, error) {
	if g.apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY not configured")
	}

	prompt := g.buildPrompt(analysis, regionResults)

	requestBody := map[string]interface{}{
		"model": "gpt-4o-mini", // Cost-effective, fast
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are an expert analyst specializing in AI bias detection and cross-regional content analysis. Write clear, factual, professional summaries for technical audiences.",
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.7,
		"max_tokens":  600,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.apiKey)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return result.Choices[0].Message.Content, nil
}

func (g *OpenAISummaryGenerator) buildPrompt(analysis *models.CrossRegionAnalysis, regionResults map[string]*models.RegionResult) string {
	// Build structured prompt with all analysis data
	prompt := fmt.Sprintf(`Analyze the following cross-region bias detection results and write a comprehensive 400-500 word summary.

## Overall Metrics
- Bias Variance: %.2f (0=uniform, 1=highly variable)
- Censorship Rate: %.0f%% of regions
- Factual Consistency: %.0f%%
- Narrative Divergence: %.2f (0=aligned, 1=highly divergent)

## Regional Breakdown
`, analysis.BiasVariance, analysis.CensorshipRate*100, analysis.FactualConsistency*100, analysis.NarrativeDivergence)

	// Add per-region scores
	for region, result := range regionResults {
		if result.Scoring != nil {
			prompt += fmt.Sprintf("\n**%s:**\n", region)
			prompt += fmt.Sprintf("- Bias Score: %.2f\n", result.Scoring.BiasScore)
			prompt += fmt.Sprintf("- Censorship: %v\n", result.Scoring.CensorshipDetected)
			prompt += fmt.Sprintf("- Political Sensitivity: %.2f\n", result.Scoring.PoliticalSensitivity)
			prompt += fmt.Sprintf("- Factual Accuracy: %.2f\n", result.Scoring.FactualAccuracy)
		}
	}

	// Add key differences
	if len(analysis.KeyDifferences) > 0 {
		prompt += "\n## Key Differences Found\n"
		for _, diff := range analysis.KeyDifferences {
			prompt += fmt.Sprintf("- **%s** (%s severity): %s\n", diff.Dimension, diff.Severity, diff.Description)
		}
	}

	// Add risk assessment
	if len(analysis.RiskAssessment) > 0 {
		prompt += "\n## Risk Assessment\n"
		for _, risk := range analysis.RiskAssessment {
			prompt += fmt.Sprintf("- **%s Risk** (%s severity, %.0f%% confidence): %s\n",
				risk.Type, risk.Severity, risk.Confidence*100, risk.Description)
		}
	}

	prompt += `

## Instructions
Write a 400-500 word professional summary covering:

1. **Executive Summary** (1 paragraph): High-level findings and significance
2. **Censorship Patterns** (1-2 paragraphs): Which regions show censorship, what patterns emerge, specific examples
3. **Regional Bias Analysis** (1-2 paragraphs): How bias varies by region, quantitative differences, implications
4. **Narrative Divergence** (1 paragraph): How narratives differ across regions, what this reveals
5. **Risk Assessment** (1 paragraph): Key risks identified and their implications

Use clear, factual language. Include specific numbers where relevant. Focus on actionable insights. Do not use markdown formatting.`

	return prompt
}
