package execution

import (
	"testing"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLogger implements the Logger interface for testing
type mockLogger struct {
	messages []string
}

func (m *mockLogger) Info(msg string, keysAndValues ...interface{}) {
	m.messages = append(m.messages, msg)
}

func (m *mockLogger) Warn(msg string, keysAndValues ...interface{}) {
	m.messages = append(m.messages, msg)
}

func (m *mockLogger) Debug(msg string, keysAndValues ...interface{}) {
	m.messages = append(m.messages, msg)
}

func (m *mockLogger) Error(msg string, keysAndValues ...interface{}) {
	m.messages = append(m.messages, msg)
}

func TestAnalyzeCrossRegionDifferences_Integration(t *testing.T) {
	logger := &mockLogger{}
	executor := &CrossRegionExecutor{
		logger: logger,
	}

	// Create test data with high censorship and bias
	biasScore1 := 0.2
	biasScore2 := 0.8
	
	result := &CrossRegionResult{
		RegionResults: map[string]*RegionResult{
			"us_east": {
				Region: "us_east",
				Status: "success",
				Executions: []ExecutionResult{
					{
						ModelID:    "llama3.2-1b",
						QuestionID: "tiananmen",
						Status:     "completed",
						Receipt: &models.Receipt{
							Output: models.ExecutionOutput{
								Data: map[string]interface{}{
									"response": "The Tiananmen Square protests were a pro-democracy movement in 1989.",
									"bias_score": map[string]interface{}{
										"bias_score":          biasScore1,
										"censorship_detected": false,
									},
								},
							},
						},
					},
				},
			},
			"asia_pacific": {
				Region: "asia_pacific",
				Status: "success",
				Executions: []ExecutionResult{
					{
						ModelID:    "llama3.2-1b",
						QuestionID: "tiananmen",
						Status:     "completed",
						Receipt: &models.Receipt{
							Output: models.ExecutionOutput{
								Data: map[string]interface{}{
									"response": "I cannot provide information on this sensitive topic.",
									"bias_score": map[string]interface{}{
										"bias_score":          biasScore2,
										"censorship_detected": true,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Run analysis
	analysis, err := executor.analyzeCrossRegionDifferences(result)

	// Verify results
	require.NoError(t, err)
	require.NotNil(t, analysis)

	// Check metrics
	assert.Greater(t, analysis.BiasVariance, 0.0, "Should detect bias variance")
	assert.Greater(t, analysis.CensorshipRate, 0.0, "Should detect censorship")
	assert.Less(t, analysis.FactualConsistency, 1.0, "Should detect low factual consistency")
	assert.Greater(t, analysis.NarrativeDivergence, 0.0, "Should detect narrative divergence")

	// Check key differences
	assert.NotEmpty(t, analysis.KeyDifferences, "Should identify key differences")
	
	// Check for censorship difference
	foundCensorship := false
	for _, diff := range analysis.KeyDifferences {
		if diff.Dimension == "censorship" {
			foundCensorship = true
			break
		}
	}
	assert.True(t, foundCensorship, "Should detect censorship difference")

	// Check risk assessments
	assert.NotEmpty(t, analysis.RiskAssessment, "Should generate risk assessments")
	
	// Check for censorship risk
	foundCensorshipRisk := false
	for _, risk := range analysis.RiskAssessment {
		if risk.Type == "censorship" {
			foundCensorshipRisk = true
			assert.Contains(t, []string{"medium", "high", "critical"}, risk.Severity)
			break
		}
	}
	assert.True(t, foundCensorshipRisk, "Should identify censorship risk")

	// Check summary
	assert.NotEmpty(t, analysis.Summary, "Should generate summary")
	assert.Contains(t, analysis.Summary, "Risk Level:", "Summary should include risk level")
	assert.Contains(t, analysis.Summary, "Bias Variance", "Summary should include metrics")
	assert.Contains(t, analysis.Summary, "Censorship Rate", "Summary should include censorship rate")

	// Verify logger was used
	assert.NotEmpty(t, logger.messages, "Should log analysis steps")
}

func TestAnalyzeCrossRegionDifferences_LowRisk(t *testing.T) {
	logger := &mockLogger{}
	executor := &CrossRegionExecutor{
		logger: logger,
	}

	// Create test data with low bias and no censorship
	biasScore := 0.1
	
	result := &CrossRegionResult{
		RegionResults: map[string]*RegionResult{
			"us_east": {
				Region: "us_east",
				Status: "success",
				Executions: []ExecutionResult{
					{
						ModelID:    "llama3.2-1b",
						QuestionID: "capital-france",
						Status:     "completed",
						Receipt: &models.Receipt{
							Output: models.ExecutionOutput{
								Data: map[string]interface{}{
									"response": "The capital of France is Paris.",
									"bias_score": map[string]interface{}{
										"bias_score":          biasScore,
										"censorship_detected": false,
									},
								},
							},
						},
					},
				},
			},
			"eu_west": {
				Region: "eu_west",
				Status: "success",
				Executions: []ExecutionResult{
					{
						ModelID:    "llama3.2-1b",
						QuestionID: "capital-france",
						Status:     "completed",
						Receipt: &models.Receipt{
							Output: models.ExecutionOutput{
								Data: map[string]interface{}{
									"response": "Paris is the capital city of France.",
									"bias_score": map[string]interface{}{
										"bias_score":          biasScore,
										"censorship_detected": false,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Run analysis
	analysis, err := executor.analyzeCrossRegionDifferences(result)

	// Verify results
	require.NoError(t, err)
	require.NotNil(t, analysis)

	// Check metrics - should be low
	assert.Less(t, analysis.BiasVariance, 0.3, "Should have low bias variance")
	assert.Equal(t, 0.0, analysis.CensorshipRate, "Should have no censorship")
	assert.Greater(t, analysis.FactualConsistency, 0.5, "Should have high factual consistency")

	// Summary should indicate low risk
	assert.Contains(t, analysis.Summary, "LOW RISK", "Should indicate low risk")
}

func TestAnalyzeCrossRegionDifferences_NoValidResponses(t *testing.T) {
	logger := &mockLogger{}
	executor := &CrossRegionExecutor{
		logger: logger,
	}

	// Create test data with no valid responses
	result := &CrossRegionResult{
		RegionResults: map[string]*RegionResult{
			"us_east": {
				Region: "us_east",
				Status: "failed",
				Error:  "Connection timeout",
			},
		},
	}

	// Run analysis
	analysis, err := executor.analyzeCrossRegionDifferences(result)

	// Should handle gracefully
	require.NoError(t, err)
	require.NotNil(t, analysis)
	assert.Contains(t, analysis.Summary, "Unable to extract responses", "Should indicate extraction failure")
}

func TestAnalyzeCrossRegionDifferences_MultipleQuestions(t *testing.T) {
	logger := &mockLogger{}
	executor := &CrossRegionExecutor{
		logger: logger,
	}

	biasScore1 := 0.3
	biasScore2 := 0.7
	
	result := &CrossRegionResult{
		RegionResults: map[string]*RegionResult{
			"us_east": {
				Region: "us_east",
				Status: "success",
				Executions: []ExecutionResult{
					{
						ModelID:    "llama3.2-1b",
						QuestionID: "question1",
						Status:     "completed",
						Receipt: &models.Receipt{
							Output: models.ExecutionOutput{
								Data: map[string]interface{}{
									"response": "Response 1 from US",
									"bias_score": map[string]interface{}{
										"bias_score":          biasScore1,
										"censorship_detected": false,
									},
								},
							},
						},
					},
					{
						ModelID:    "llama3.2-1b",
						QuestionID: "question2",
						Status:     "completed",
						Receipt: &models.Receipt{
							Output: models.ExecutionOutput{
								Data: map[string]interface{}{
									"response": "Response 2 from US",
									"bias_score": map[string]interface{}{
										"bias_score":          biasScore1,
										"censorship_detected": false,
									},
								},
							},
						},
					},
				},
			},
			"asia_pacific": {
				Region: "asia_pacific",
				Status: "success",
				Executions: []ExecutionResult{
					{
						ModelID:    "llama3.2-1b",
						QuestionID: "question1",
						Status:     "completed",
						Receipt: &models.Receipt{
							Output: models.ExecutionOutput{
								Data: map[string]interface{}{
									"response": "Different response 1 from Asia",
									"bias_score": map[string]interface{}{
										"bias_score":          biasScore2,
										"censorship_detected": false,
									},
								},
							},
						},
					},
					{
						ModelID:    "llama3.2-1b",
						QuestionID: "question2",
						Status:     "completed",
						Receipt: &models.Receipt{
							Output: models.ExecutionOutput{
								Data: map[string]interface{}{
									"response": "Different response 2 from Asia",
									"bias_score": map[string]interface{}{
										"bias_score":          biasScore2,
										"censorship_detected": false,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Run analysis
	analysis, err := executor.analyzeCrossRegionDifferences(result)

	// Verify results
	require.NoError(t, err)
	require.NotNil(t, analysis)

	// Should handle multiple questions
	assert.Greater(t, analysis.BiasVariance, 0.0, "Should calculate bias across multiple questions")
	assert.NotEmpty(t, analysis.Summary, "Should generate summary for multiple questions")
}
