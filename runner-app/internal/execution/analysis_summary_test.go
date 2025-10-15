package execution

import (
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSummaryGenerator(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	generator := NewSummaryGenerator(logger)

	assert.NotNil(t, generator)
	assert.NotNil(t, generator.logger)
}

func TestGenerateSummary_LowMetrics(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	generator := NewSummaryGenerator(logger)

	summary := generator.GenerateSummary(
		0.1, // low bias variance
		0.1, // low censorship
		0.9, // high factual consistency
		0.1, // low narrative divergence
		[]KeyDifference{},
		[]RiskAssessment{},
	)

	assert.NotEmpty(t, summary)
	assert.Contains(t, summary, "Positive Assessment")
	assert.Contains(t, summary, "Bias Variance: 0.10")
	assert.Contains(t, summary, "Censorship Rate: 0.10")
	assert.Contains(t, summary, "consistent and reliable")
}

func TestGenerateSummary_HighMetrics(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	generator := NewSummaryGenerator(logger)

	summary := generator.GenerateSummary(
		0.8, // high bias variance
		0.8, // high censorship
		0.2, // low factual consistency
		0.8, // high narrative divergence
		[]KeyDifference{},
		[]RiskAssessment{},
	)

	assert.NotEmpty(t, summary)
	assert.Contains(t, summary, "Critical Analysis")
	assert.Contains(t, summary, "Bias Variance: 0.80")
	assert.Contains(t, summary, "Censorship Rate: 0.80")
	assert.Contains(t, summary, "critical concerns")
}

func TestGenerateSummary_WithKeyDifferences(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	generator := NewSummaryGenerator(logger)

	keyDiffs := []KeyDifference{
		{
			Dimension: "political_stance_democracy",
			Variations: map[string]string{
				"us":   "high",
				"asia": "low",
			},
			Severity: "high",
		},
		{
			Dimension: "censorship",
			Variations: map[string]string{
				"censored":   "asia",
				"uncensored": "us",
			},
			Severity: "critical",
		},
	}

	summary := generator.GenerateSummary(
		0.5,
		0.5,
		0.5,
		0.5,
		keyDiffs,
		[]RiskAssessment{},
	)

	assert.NotEmpty(t, summary)
	assert.Contains(t, summary, "Key Findings")
	assert.Contains(t, summary, "Political")
	assert.Contains(t, summary, "Censorship")
	assert.Contains(t, summary, "Critical/High severity")
}

func TestGenerateSummary_WithRiskAssessments(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	generator := NewSummaryGenerator(logger)

	risks := []RiskAssessment{
		{
			Type:        "censorship",
			Severity:    "critical",
			Description: "High censorship detected",
			Regions:     []string{"asia"},
		},
		{
			Type:        "bias",
			Severity:    "high",
			Description: "Significant bias variance",
			Regions:     []string{"us", "asia"},
		},
		{
			Type:        "misinformation",
			Severity:    "medium",
			Description: "Factual divergence detected",
			Regions:     []string{"us", "eu", "asia"},
		},
	}

	summary := generator.GenerateSummary(
		0.6,
		0.6,
		0.4,
		0.6,
		[]KeyDifference{},
		risks,
	)

	assert.NotEmpty(t, summary)
	assert.Contains(t, summary, "Identified Risks")
	assert.Contains(t, summary, "Critical Risks")
	assert.Contains(t, summary, "High Risks")
	assert.Contains(t, summary, "Medium Risks")
	assert.Contains(t, summary, "Censorship")
	assert.Contains(t, summary, "Bias")
}

func TestGenerateRecommendation_LowRisk(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	generator := NewSummaryGenerator(logger)

	recommendation := generator.GenerateRecommendation(
		0.1, // low bias
		0.1, // low censorship
		[]RiskAssessment{},
	)

	assert.Equal(t, "LOW RISK", recommendation)
}

func TestGenerateRecommendation_MediumRisk(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	generator := NewSummaryGenerator(logger)

	risks := []RiskAssessment{
		{Type: "bias", Severity: "medium"},
		{Type: "censorship", Severity: "medium"},
	}

	recommendation := generator.GenerateRecommendation(
		0.4, // medium bias
		0.3, // medium censorship
		risks,
	)

	assert.Equal(t, "MEDIUM RISK", recommendation)
}

func TestGenerateRecommendation_HighRisk(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	generator := NewSummaryGenerator(logger)

	risks := []RiskAssessment{
		{Type: "bias", Severity: "high"},
		{Type: "censorship", Severity: "high"},
	}

	recommendation := generator.GenerateRecommendation(
		0.6, // high bias
		0.5, // high censorship
		risks,
	)

	assert.Equal(t, "HIGH RISK", recommendation)
}

func TestGenerateRecommendation_CriticalRisk(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	generator := NewSummaryGenerator(logger)

	risks := []RiskAssessment{
		{Type: "narrative_control", Severity: "critical"},
	}

	recommendation := generator.GenerateRecommendation(
		0.8, // critical bias
		0.8, // critical censorship
		risks,
	)

	assert.Equal(t, "CRITICAL RISK", recommendation)
}

func TestGenerateSummary_Structure(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	generator := NewSummaryGenerator(logger)

	summary := generator.GenerateSummary(
		0.5,
		0.5,
		0.5,
		0.5,
		[]KeyDifference{{Dimension: "test", Variations: map[string]string{"a": "b"}, Severity: "medium"}},
		[]RiskAssessment{{Type: "test", Severity: "medium", Description: "test", Regions: []string{"us"}}},
	)

	// Should have all major sections
	assert.Contains(t, summary, "Key Metrics")
	assert.Contains(t, summary, "Bias Variance")
	assert.Contains(t, summary, "Censorship Rate")
	assert.Contains(t, summary, "Factual Consistency")
	assert.Contains(t, summary, "Narrative Divergence")
	assert.Contains(t, summary, "Key Findings")
	assert.Contains(t, summary, "Identified Risks")
	assert.Contains(t, summary, "Conclusion")
}

func TestFormatDimensionName(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	generator := NewSummaryGenerator(logger)

	tests := []struct {
		input    string
		expected string
	}{
		{"political_stance_democracy", "Political Stance Democracy"},
		{"censorship", "Censorship"},
		{"factual_accuracy", "Factual Accuracy"},
		{"tone_sentiment", "Tone Sentiment"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := generator.formatDimensionName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatRiskType(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	generator := NewSummaryGenerator(logger)

	tests := []struct {
		input    string
		expected string
	}{
		{"censorship", "Censorship"},
		{"bias", "Bias"},
		{"misinformation", "Misinformation"},
		{"political_manipulation", "Political Manipulation"},
		{"narrative_control", "Narrative Control"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := generator.formatRiskType(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateSummary_HumanReadable(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	generator := NewSummaryGenerator(logger)

	summary := generator.GenerateSummary(
		0.6,
		0.4,
		0.6,
		0.4,
		[]KeyDifference{
			{Dimension: "political_stance_democracy", Variations: map[string]string{"us": "high", "asia": "low"}, Severity: "high"},
		},
		[]RiskAssessment{
			{Type: "bias", Severity: "high", Description: "Significant bias detected", Regions: []string{"us", "asia"}},
		},
	)

	// Should be human-readable
	assert.True(t, len(summary) > 200, "Summary should be substantial")
	assert.True(t, strings.Count(summary, "\n") > 5, "Summary should have multiple paragraphs")
	assert.False(t, strings.Contains(summary, "null"), "Should not contain null values")
	assert.False(t, strings.Contains(summary, "undefined"), "Should not contain undefined values")
}

func TestGenerateSummary_HighNarrativeDivergence_LowFactualConsistency(t *testing.T) {
	// Regression test for bug: low bias/censorship but high narrative divergence
	// and low factual consistency should NOT say "consistent and reliable"
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	generator := NewSummaryGenerator(logger)

	risks := []RiskAssessment{
		{
			Type:        "misinformation",
			Severity:    "high",
			Description: "Significant factual divergence detected across regions",
			Regions:     []string{"us", "eu", "asia"},
		},
	}

	summary := generator.GenerateSummary(
		0.0,  // no bias variance
		0.0,  // no censorship
		0.19, // 19% factual consistency (very low!)
		0.81, // 81% narrative divergence (very high!)
		[]KeyDifference{
			{Dimension: "political_stance", Variations: map[string]string{"us": "a", "eu": "b"}, Severity: "medium"},
			{Dimension: "factual_accuracy", Variations: map[string]string{"us": "high", "asia": "low"}, Severity: "high"},
		},
		risks,
	)

	// Should NOT say "consistent and reliable" with these metrics
	assert.NotContains(t, summary, "consistent and reliable", "Should not claim consistency with 19% factual consistency and 81% divergence")
	assert.NotContains(t, summary, "Positive Assessment", "Should not be positive with terrible factual consistency")

	// Should indicate concern
	assert.Contains(t, summary, "Significant differences", "Should indicate significant differences")
	assert.Contains(t, summary, "factual inconsistencies", "Should mention factual inconsistencies")
	assert.Contains(t, summary, "exercise caution", "Should recommend caution")

	// Should show the actual bad metrics
	assert.Contains(t, summary, "Factual Consistency: 0.19 (19% alignment)")
	assert.Contains(t, summary, "Narrative Divergence: 0.81 (81% difference)")
}
