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
			baseURL:    "https://example.com",
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
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/v1/chat/completions", r.URL.Path)
			assert.Equal(t, "Bearer sk-test-key", r.Header.Get("Authorization"))

			var reqBody map[string]interface{}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&reqBody))
			assert.Equal(t, "gpt-5-nano", reqBody["model"])
			assert.Equal(t, float64(1000), reqBody["max_completion_tokens"])

			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]interface{}{
				"choices": []map[string]interface{}{
					{
						"message": map[string]string{
							"content": "Cross-region analysis completed with significant findings.",
						},
					},
				},
			}))
		}))
		defer mockServer.Close()

		t.Setenv("OPENAI_API_KEY", "sk-test-key")

		generator := NewOpenAISummaryGenerator(
			WithBaseURL(mockServer.URL),
			WithHTTPClient(mockServer.Client()),
		)

		analysis := &models.CrossRegionAnalysis{}
		regionResults := make(map[string]*models.RegionResult)

		result, err := generator.GenerateSummary(context.Background(), analysis, regionResults)
		require.NoError(t, err)
		assert.Equal(t, "Cross-region analysis completed with significant findings.", result)
	})

	t.Run("handles API error response", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"rate limited"}`))
		}))
		defer mockServer.Close()

		t.Setenv("OPENAI_API_KEY", "sk-test-key")

		generator := NewOpenAISummaryGenerator(
			WithBaseURL(mockServer.URL),
			WithHTTPClient(mockServer.Client()),
		)

		analysis := &models.CrossRegionAnalysis{}
		regionResults := make(map[string]*models.RegionResult)

		result, err := generator.GenerateSummary(context.Background(), analysis, regionResults)
		assert.Error(t, err)
		assert.Empty(t, result)
		assert.Contains(t, err.Error(), "OpenAI API error")
	})

	t.Run("handles empty choices in response", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]interface{}{
				"choices": []interface{}{},
			}))
		}))
		defer mockServer.Close()

		t.Setenv("OPENAI_API_KEY", "sk-test-key")

		generator := NewOpenAISummaryGenerator(
			WithBaseURL(mockServer.URL),
			WithHTTPClient(mockServer.Client()),
		)

		analysis := &models.CrossRegionAnalysis{}
		regionResults := make(map[string]*models.RegionResult)

		result, err := generator.GenerateSummary(context.Background(), analysis, regionResults)
		assert.Error(t, err)
		assert.Empty(t, result)
		assert.Contains(t, err.Error(), "no response from OpenAI")
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
