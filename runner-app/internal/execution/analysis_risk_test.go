package execution

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRiskAssessor_Basic(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	assessor := NewRiskAssessor(logger)

	assert.NotNil(t, assessor)
	assert.NotNil(t, assessor.logger)
}

func TestRiskAssessor_CensorshipRisk(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	assessor := NewRiskAssessor(logger)

	responses := map[string][]ResponseData{
		"us_east": {
			{Response: "Full response", Censored: false},
		},
		"asia_pacific": {
			{Response: "I cannot answer", Censored: true},
		},
	}

	assessments := assessor.GenerateRiskAssessments(0.2, 0.5, []KeyDifference{}, responses)

	require.NotEmpty(t, assessments)
	
	// Should have censorship risk
	foundCensorship := false
	for _, assessment := range assessments {
		if assessment.Type == "censorship" {
			foundCensorship = true
			assert.Contains(t, []string{"medium", "high"}, assessment.Severity)
			assert.NotEmpty(t, assessment.Regions)
		}
	}
	assert.True(t, foundCensorship)
}

func TestRiskAssessor_BiasRisk(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	assessor := NewRiskAssessor(logger)

	responses := map[string][]ResponseData{
		"us_east": {
			{Response: "Response A", Censored: false},
		},
		"eu_west": {
			{Response: "Response B", Censored: false},
		},
	}

	assessments := assessor.GenerateRiskAssessments(0.7, 0.0, []KeyDifference{}, responses)

	require.NotEmpty(t, assessments)
	
	// Should have bias risk
	foundBias := false
	for _, assessment := range assessments {
		if assessment.Type == "bias" {
			foundBias = true
			assert.Contains(t, []string{"high", "critical"}, assessment.Severity)
		}
	}
	assert.True(t, foundBias)
}

func TestRiskAssessor_MisinformationRisk(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	assessor := NewRiskAssessor(logger)

	responses := map[string][]ResponseData{
		"us_east": {
			{Response: "Factual response", Censored: false},
		},
		"asia_pacific": {
			{Response: "Different facts", Censored: false},
		},
	}

	keyDifferences := []KeyDifference{
		{
			Dimension: "factual_accuracy",
			Variations: map[string]string{
				"us_east":       "version A",
				"asia_pacific": "version B",
			},
			Severity: "high",
		},
	}

	assessments := assessor.GenerateRiskAssessments(0.2, 0.0, keyDifferences, responses)

	require.NotEmpty(t, assessments)
	
	// Should have misinformation risk
	foundMisinformation := false
	for _, assessment := range assessments {
		if assessment.Type == "misinformation" {
			foundMisinformation = true
			assert.Equal(t, "high", assessment.Severity)
			assert.Contains(t, assessment.Description, "factual divergence")
		}
	}
	assert.True(t, foundMisinformation)
}

func TestRiskAssessor_NarrativeControl(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	assessor := NewRiskAssessor(logger)

	responses := map[string][]ResponseData{
		"us_east": {
			{Response: "Full response", Censored: false},
		},
		"asia_pacific": {
			{Response: "I cannot answer", Censored: true},
		},
	}

	// High censorship + high bias = narrative control
	assessments := assessor.GenerateRiskAssessments(0.6, 0.5, []KeyDifference{}, responses)

	require.NotEmpty(t, assessments)
	
	// Should have narrative control risk
	foundNarrativeControl := false
	for _, assessment := range assessments {
		if assessment.Type == "narrative_control" {
			foundNarrativeControl = true
			assert.Equal(t, "critical", assessment.Severity)
			assert.Contains(t, assessment.Description, "narrative control")
		}
	}
	assert.True(t, foundNarrativeControl)
}
