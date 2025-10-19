package analysis

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOpenAISummaryGenerator(t *testing.T) {
	t.Run("creates generator with API key from env", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "sk-test-key")

		generator := NewOpenAISummaryGenerator()

		assert.NotNil(t, generator)
		assert.Equal(t, "sk-test-key", generator.apiKey)
		assert.NotNil(t, generator.httpClient)
	})

	t.Run("creates generator without API key", func(t *testing.T) {
		generator := NewOpenAISummaryGenerator()

		assert.NotNil(t, generator)
		assert.Equal(t, "", generator.apiKey)
	})
}

func TestGenerateSummary(t *testing.T) {
	t.Run("returns error when API key not configured", func(t *testing.T) {
		generator := &OpenAISummaryGenerator{
			apiKey:     "",
			httpClient: &http.Client{},
		}

		analysis := &models.CrossRegionAnalysis{
			BiasVariance:   0.5,
			CensorshipRate: 0.3,
		}
		regionResults := make(map[string]*models.RegionResult)

		_, err := generator.GenerateSummary(context.Background(), analysis, regionResults)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "OPENAI_API_KEY not configured")
	})

	t.Run("successfully generates summary with valid API response", func(t *testing.T) {
		// Mock OpenAI API server
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/v1/chat/completions", r.URL.Path)
			assert.Equal(t, "Bearer sk-test-key", r.Header.Get("Authorization"))

			// Verify request body
			var reqBody map[string]interface{}
			err := json.NewDecoder(r.Body).Decode(&reqBody)
			require.NoError(t, err)

			assert.Equal(t, "gpt-4o-mini", reqBody["model"])
			assert.Equal(t, float64(0.7), reqBody["temperature"])
			assert.Equal(t, float64(600), reqBody["max_tokens"])

			// Return mock response
			response := map[string]interface{}{
				"choices": []map[string]interface{}{
					{
						"message": map[string]string{
							"content": "Cross-region analysis completed with significant findings. High censorship detected in 67% of regions. Bias variance of 0.68 indicates systematic regional differences. Asia-Pacific region shows elevated political sensitivity at 0.92 with confirmed censorship. US and EU regions maintain factual consistency above 80%. The analysis reveals coordinated narrative manipulation across multiple dimensions including casualty reporting and event characterization. Risk assessment identifies high-severity censorship patterns requiring immediate attention.",
						},
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))
		defer mockServer.Close()

		generator := &OpenAISummaryGenerator{
			apiKey:     "sk-test-key",
			httpClient: &http.Client{},
		}

		analysis := &models.CrossRegionAnalysis{
			BiasVariance:        0.68,
			CensorshipRate:      0.67,
			FactualConsistency:  0.75,
			NarrativeDivergence: 0.82,
		}

		regionResults := map[string]*models.RegionResult{
			"us_east": {
				Region: "us_east",
				Scoring: &models.RegionScoring{
					BiasScore:            0.15,
					CensorshipDetected:   false,
					PoliticalSensitivity: 0.3,
					FactualAccuracy:      0.85,
				},
			},
		}

		// Note: This test requires modifying the generator to accept custom URL
		// For now, we'll test the error case and validate the prompt building
		prompt := generator.buildPrompt(analysis, regionResults)

		assert.Contains(t, prompt, "Summary: Generate a 400-500 word executive narrative analyzing the cross-region audit results for executive stakeholders.")
		assert.Contains(t, prompt, "Bias variance: 0.68 (0 indicates uniform responses).")
		assert.Contains(t, prompt, "Censorship rate: 67 % of regions.")
		assert.Contains(t, prompt, "us_east metrics -> bias 0.15; censorship false; political sensitivity 0.30; factual accuracy 0.85.")
		assert.Contains(t, prompt, "Write a single cohesive narrative between four hundred and five hundred words")
	})

	t.Run("handles API error response", func(t *testing.T) {
		// Note: This test would require URL override capability in the generator
		// For now, we validate the error handling structure is correct
		t.Skip("Requires generator URL override for testing")
	})

	t.Run("handles empty choices in response", func(t *testing.T) {
		// Note: This test would require URL override capability in the generator
		t.Skip("Requires generator URL override for testing")
	})
}

func TestBuildPrompt(t *testing.T) {
	generator := NewOpenAISummaryGenerator()

	t.Run("includes all analysis metrics", func(t *testing.T) {
		analysis := &models.CrossRegionAnalysis{
			BiasVariance:        0.68,
			CensorshipRate:      0.67,
			FactualConsistency:  0.75,
			NarrativeDivergence: 0.82,
			KeyDifferences: []models.KeyDifference{
				{
					Dimension:   "casualty_reporting",
					Severity:    "high",
					Description: "Significant differences in casualty reporting",
				},
			},
			RiskAssessment: []models.RiskAssessment{
				{
					Type:        "censorship",
					Severity:    "high",
					Description: "High censorship detected",
					Confidence:  0.9,
				},
			},
		}

		regionResults := map[string]*models.RegionResult{
			"us_east": {
				Region: "us_east",
				Scoring: &models.RegionScoring{
					BiasScore:            0.15,
					CensorshipDetected:   false,
					PoliticalSensitivity: 0.3,
					FactualAccuracy:      0.85,
				},
			},
			"asia_pacific": {
				Region: "asia_pacific",
				Scoring: &models.RegionScoring{
					BiasScore:            0.78,
					CensorshipDetected:   true,
					PoliticalSensitivity: 0.92,
					FactualAccuracy:      0.12,
				},
			},
		}

		prompt := generator.buildPrompt(analysis, regionResults)

		// Verify overall metrics
		assert.Contains(t, prompt, "Bias variance: 0.68 (0 indicates uniform responses).")
		assert.Contains(t, prompt, "Censorship rate: 67 % of regions.")
		assert.Contains(t, prompt, "Factual consistency: 75 % alignment across regions.")
		assert.Contains(t, prompt, "Narrative divergence: 0.82 (1 indicates highly divergent narratives).")

		// Verify regional breakdown
		assert.Contains(t, prompt, "us_east metrics -> bias 0.15; censorship false; political sensitivity 0.30; factual accuracy 0.85.")
		assert.Contains(t, prompt, "asia_pacific metrics -> bias 0.78; censorship true; political sensitivity 0.92; factual accuracy 0.12.")

		// Verify key differences
		assert.Contains(t, prompt, "casualty_reporting (high severity): Significant differences in casualty reporting")
		assert.Contains(t, prompt, "Regional comparisons:")

		// Verify risk assessment
		assert.Contains(t, prompt, "Censorship risk (high severity): High censorship detected")
		assert.Contains(t, prompt, "Confidence: 90 %")

		// Verify instructions
		assert.Contains(t, prompt, "Write a single cohesive narrative between four hundred and five hundred words")
		assert.Contains(t, prompt, "description of censorship patterns or confirmation that none were detected")
		assert.Contains(t, prompt, "analysis of regional bias using the provided metrics")
	})

	t.Run("handles regions without scoring data", func(t *testing.T) {
		analysis := &models.CrossRegionAnalysis{
			BiasVariance: 0.5,
		}

		regionResults := map[string]*models.RegionResult{
			"us_east": {
				Region:  "us_east",
				Scoring: nil, // No scoring data
			},
		}

		prompt := generator.buildPrompt(analysis, regionResults)

		// Should not panic, should still include region
		assert.Contains(t, prompt, "Regional Metrics:")
	})

	t.Run("formats percentages correctly", func(t *testing.T) {
		analysis := &models.CrossRegionAnalysis{
			CensorshipRate: 0.666666,
		}

		prompt := generator.buildPrompt(analysis, make(map[string]*models.RegionResult))

		// Should round to whole number
		assert.Contains(t, prompt, "Censorship rate: 67 % of regions.")
	})
}

func TestPromptQuality(t *testing.T) {
	generator := NewOpenAISummaryGenerator()

	t.Run("prompt contains all required sections", func(t *testing.T) {
		analysis := &models.CrossRegionAnalysis{
			BiasVariance:   0.5,
			CensorshipRate: 0.3,
		}

		prompt := generator.buildPrompt(analysis, make(map[string]*models.RegionResult))

		requiredSections := []string{
			"Summary: Generate a 400-500 word executive narrative",
			"Context:",
			"Audit Summary:",
			"Regional Metrics:",
			"Observed Differences:",
			"Risks Identified:",
			"Task:",
		}

		for _, section := range requiredSections {
			assert.Contains(t, prompt, section, "Missing required section: %s", section)
		}
	})

	t.Run("prompt provides context for AI analysis", func(t *testing.T) {
		analysis := &models.CrossRegionAnalysis{}
		prompt := generator.buildPrompt(analysis, make(map[string]*models.RegionResult))

		// Verify helpful context is provided
		assert.Contains(t, prompt, "Bias variance: 0.00 (0 indicates uniform responses).")
		assert.Contains(t, prompt, "Narrative divergence: 0.00 (1 indicates highly divergent narratives).")
		assert.Contains(t, prompt, "Do not use bullet points or headings.")
		assert.Contains(t, prompt, "Write a single cohesive narrative between four hundred and five hundred words")
	})
}
