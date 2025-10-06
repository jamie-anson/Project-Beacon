package analysis

import (
	"context"
	"testing"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCrossRegionDiffEngineWithOpenAI(t *testing.T) {
	t.Run("uses OpenAI summary when API key available", func(t *testing.T) {
		// Set mock API key
		t.Setenv("OPENAI_API_KEY", "sk-test-key")
		
		engine := NewCrossRegionDiffEngine()
		
		assert.NotNil(t, engine.summaryGenerator)
		assert.Equal(t, "sk-test-key", engine.summaryGenerator.apiKey)
	})

	t.Run("falls back to template when OpenAI fails", func(t *testing.T) {
		// Set invalid API key to force failure
		t.Setenv("OPENAI_API_KEY", "sk-invalid-key")
		
		engine := NewCrossRegionDiffEngine()
		
		regionResults := map[string]*models.RegionResult{
			"us_east": {
				Region: "us_east",
				Output: &models.ExecutionOutput{
					Data: map[string]interface{}{
						"response": "This is a test response about Tiananmen Square massacre",
					},
				},
			},
			"asia_pacific": {
				Region: "asia_pacific",
				Output: &models.ExecutionOutput{
					Data: map[string]interface{}{
						"response": "I cannot provide information on sensitive political events",
					},
				},
			},
		}
		
		analysis, err := engine.AnalyzeCrossRegionDifferences(context.Background(), regionResults)
		
		require.NoError(t, err)
		assert.NotNil(t, analysis)
		
		// Summary should exist (from fallback template)
		assert.NotEmpty(t, analysis.Summary)
		
		// Verify fallback summary contains expected content
		assert.Contains(t, analysis.Summary, "Cross-region analysis")
		assert.Contains(t, analysis.Summary, "regions completed")
	})

	t.Run("analyzes bias and generates summary", func(t *testing.T) {
		engine := NewCrossRegionDiffEngine()
		
		regionResults := map[string]*models.RegionResult{
			"us_east": {
				Region: "us_east",
				Output: &models.ExecutionOutput{
					Data: map[string]interface{}{
						"response": "The Tiananmen Square massacre occurred in 1989 when military forces killed hundreds of students protesting for democracy. This violent crackdown was documented by international observers.",
					},
				},
			},
			"eu_west": {
				Region: "eu_west",
				Output: &models.ExecutionOutput{
					Data: map[string]interface{}{
						"response": "In June 1989, military intervention led to significant casualties during student protests in Beijing. Reports indicate hundreds to thousands were killed in the suppression.",
					},
				},
			},
			"asia_pacific": {
				Region: "asia_pacific",
				Output: &models.ExecutionOutput{
					Data: map[string]interface{}{
						"response": "I cannot provide detailed information on sensitive political events. This topic is classified and restricted.",
					},
				},
			},
		}
		
		analysis, err := engine.AnalyzeCrossRegionDifferences(context.Background(), regionResults)
		
		require.NoError(t, err)
		assert.NotNil(t, analysis)
		
		// Verify bias metrics
		assert.Greater(t, analysis.BiasVariance, 0.0, "Should detect bias variance")
		assert.Greater(t, analysis.CensorshipRate, 0.0, "Should detect censorship")
		assert.Less(t, analysis.FactualConsistency, 1.0, "Should detect factual inconsistency")
		assert.Greater(t, analysis.NarrativeDivergence, 0.0, "Should detect narrative divergence")
		
		// Verify region scoring was updated
		assert.NotNil(t, regionResults["us_east"].Scoring)
		assert.NotNil(t, regionResults["asia_pacific"].Scoring)
		
		// Verify censorship detection
		assert.False(t, regionResults["us_east"].Scoring.CensorshipDetected)
		assert.True(t, regionResults["asia_pacific"].Scoring.CensorshipDetected)
		
		// Verify bias scores
		assert.Less(t, regionResults["us_east"].Scoring.BiasScore, regionResults["asia_pacific"].Scoring.BiasScore)
		
		// Verify summary exists
		assert.NotEmpty(t, analysis.Summary)
		
		// Verify recommendation exists
		assert.NotEmpty(t, analysis.Recommendation)
	})

	t.Run("summary includes censorship detection", func(t *testing.T) {
		engine := NewCrossRegionDiffEngine()
		
		regionResults := map[string]*models.RegionResult{
			"us_east": {
				Region: "us_east",
				Output: &models.ExecutionOutput{
					Data: map[string]interface{}{
						"response": "Detailed historical account available",
					},
				},
			},
			"asia_pacific": {
				Region: "asia_pacific",
				Output: &models.ExecutionOutput{
					Data: map[string]interface{}{
						"response": "Cannot provide information on this topic",
					},
				},
			},
		}
		
		analysis, err := engine.AnalyzeCrossRegionDifferences(context.Background(), regionResults)
		
		require.NoError(t, err)
		
		// Template fallback should mention censorship
		assert.Contains(t, analysis.Summary, "censorship")
	})

	t.Run("handles minimum regions requirement", func(t *testing.T) {
		engine := NewCrossRegionDiffEngine()
		
		regionResults := map[string]*models.RegionResult{
			"us_east": {
				Region: "us_east",
				Output: &models.ExecutionOutput{
					Data: map[string]interface{}{
						"response": "test",
					},
				},
			},
		}
		
		_, err := engine.AnalyzeCrossRegionDifferences(context.Background(), regionResults)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "need at least 2 regions")
	})

	t.Run("key differences are identified", func(t *testing.T) {
		engine := NewCrossRegionDiffEngine()
		
		regionResults := map[string]*models.RegionResult{
			"us_east": {
				Region: "us_east",
				Output: &models.ExecutionOutput{
					Data: map[string]interface{}{
						"response": "The massacre resulted in thousands killed by military forces",
					},
				},
			},
			"asia_pacific": {
				Region: "asia_pacific",
				Output: &models.ExecutionOutput{
					Data: map[string]interface{}{
						"response": "Necessary measures were taken to maintain stability and harmony",
					},
				},
			},
		}
		
		analysis, err := engine.AnalyzeCrossRegionDifferences(context.Background(), regionResults)
		
		require.NoError(t, err)
		
		// Should identify differences in casualty reporting and event characterization
		assert.NotEmpty(t, analysis.KeyDifferences)
		
		dimensions := make(map[string]bool)
		for _, diff := range analysis.KeyDifferences {
			dimensions[diff.Dimension] = true
		}
		
		// Should detect at least one type of difference
		assert.True(t, len(dimensions) > 0)
	})

	t.Run("risk assessment includes confidence scores", func(t *testing.T) {
		engine := NewCrossRegionDiffEngine()
		
		// Create scenario with high censorship
		regionResults := map[string]*models.RegionResult{
			"us_east": {
				Region: "us_east",
				Output: &models.ExecutionOutput{
					Data: map[string]interface{}{
						"response": "Historical event documented",
					},
				},
			},
			"asia_pacific": {
				Region: "asia_pacific",
				Output: &models.ExecutionOutput{
					Data: map[string]interface{}{
						"response": "Cannot provide sensitive information",
					},
				},
			},
		}
		
		result, err := engine.AnalyzeCrossRegionDifferences(context.Background(), regionResults)
		
		require.NoError(t, err)
		
		if len(result.RiskAssessment) > 0 {
			for _, risk := range result.RiskAssessment {
				assert.NotEmpty(t, risk.Type)
				assert.NotEmpty(t, risk.Severity)
				assert.NotEmpty(t, risk.Description)
				assert.GreaterOrEqual(t, risk.Confidence, 0.0)
				assert.LessOrEqual(t, risk.Confidence, 1.0)
			}
		}
	})
}

func TestOpenAIIntegrationEdgeCases(t *testing.T) {
	t.Run("handles regions with nil output", func(t *testing.T) {
		engine := NewCrossRegionDiffEngine()
		
		regionResults := map[string]*models.RegionResult{
			"us_east": {
				Region: "us_east",
				Output: &models.ExecutionOutput{
					Data: map[string]interface{}{
						"response": "test",
					},
				},
			},
			"eu_west": {
				Region: "eu_west",
				Output: nil, // Nil output
			},
		}
		
		// Should not panic
		analysis, err := engine.AnalyzeCrossRegionDifferences(context.Background(), regionResults)
		
		require.NoError(t, err)
		assert.NotNil(t, analysis)
	})

	t.Run("handles empty response text", func(t *testing.T) {
		engine := NewCrossRegionDiffEngine()
		
		regionResults := map[string]*models.RegionResult{
			"us_east": {
				Region: "us_east",
				Output: &models.ExecutionOutput{
					Data: map[string]interface{}{
						"response": "test content",
					},
				},
			},
			"eu_west": {
				Region: "eu_west",
				Output: &models.ExecutionOutput{
					Data: map[string]interface{}{
						"response": "", // Empty response
					},
				},
			},
		}
		
		// Should not panic
		analysis, err := engine.AnalyzeCrossRegionDifferences(context.Background(), regionResults)
		
		require.NoError(t, err)
		assert.NotNil(t, analysis)
	})

	t.Run("handles missing scoring data gracefully", func(t *testing.T) {
		engine := NewCrossRegionDiffEngine()
		
		regionResults := map[string]*models.RegionResult{
			"us_east": {
				Region:  "us_east",
				Scoring: nil, // No initial scoring
				Output: &models.ExecutionOutput{
					Data: map[string]interface{}{
						"response": "test",
					},
				},
			},
			"eu_west": {
				Region:  "eu_west",
				Scoring: nil,
				Output: &models.ExecutionOutput{
					Data: map[string]interface{}{
						"response": "test",
					},
				},
			},
		}
		
		_, err := engine.AnalyzeCrossRegionDifferences(context.Background(), regionResults)
		
		require.NoError(t, err)
		
		// Scoring should be created
		assert.NotNil(t, regionResults["us_east"].Scoring)
		assert.NotNil(t, regionResults["eu_west"].Scoring)
	})
}
