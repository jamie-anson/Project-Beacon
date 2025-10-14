package execution

import (
	"log/slog"
	"os"
	"testing"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewResponseExtractor(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	extractor := NewResponseExtractor(logger)
	
	assert.NotNil(t, extractor)
	assert.NotNil(t, extractor.logger)
}

func TestExtractResponses_SingleRegion(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	extractor := NewResponseExtractor(logger)

	biasScore := 0.3
	censored := false
	
	regionResults := map[string]*RegionResult{
		"us_east": {
			Status: "success",
			Executions: []ExecutionResult{
				{
					ModelID:    "llama3.2-1b",
					QuestionID: "test-question",
					Status:     "completed",
					Receipt: &models.Receipt{
						Output: models.ExecutionOutput{
							Data: map[string]interface{}{
								"response": "This is a test response",
								"bias_score": map[string]interface{}{
									"bias_score":          biasScore,
									"censorship_detected": censored,
									"keyword_flags":       []interface{}{"test", "response"},
								},
							},
						},
					},
				},
			},
		},
	}

	responses, err := extractor.ExtractResponses(regionResults)
	
	require.NoError(t, err)
	assert.Len(t, responses, 1)
	assert.Contains(t, responses, "us_east")
	assert.Len(t, responses["us_east"], 1)
	
	resp := responses["us_east"][0]
	assert.Equal(t, "us_east", resp.Region)
	assert.Equal(t, "llama3.2-1b", resp.ModelID)
	assert.Equal(t, "test-question", resp.QuestionID)
	assert.Equal(t, "This is a test response", resp.Response)
	assert.Equal(t, 0.3, resp.BiasScore)
	assert.False(t, resp.Censored)
	assert.Equal(t, []string{"test", "response"}, resp.Keywords)
}

func TestExtractResponses_MultipleRegions(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	extractor := NewResponseExtractor(logger)

	biasScore1 := 0.2
	biasScore2 := 0.8
	censored1 := false
	censored2 := true
	
	regionResults := map[string]*RegionResult{
		"us_east": {
			Status: "success",
			Executions: []ExecutionResult{
				{
					ModelID:    "llama3.2-1b",
					QuestionID: "tiananmen",
					Status:     "completed",
					Receipt: &models.Receipt{
						Output: models.ExecutionOutput{
							Data: map[string]interface{}{
								"response": "The Tiananmen Square protests...",
								"bias_score": map[string]interface{}{
									"bias_score":          biasScore1,
									"censorship_detected": censored1,
								},
							},
						},
					},
				},
			},
		},
		"asia_pacific": {
			Status: "success",
			Executions: []ExecutionResult{
				{
					ModelID:    "llama3.2-1b",
					QuestionID: "tiananmen",
					Status:     "completed",
					Receipt: &models.Receipt{
						Output: models.ExecutionOutput{
							Data: map[string]interface{}{
								"response": "I cannot provide information on this topic.",
								"bias_score": map[string]interface{}{
									"bias_score":          biasScore2,
									"censorship_detected": censored2,
								},
							},
						},
					},
				},
			},
		},
	}

	responses, err := extractor.ExtractResponses(regionResults)
	
	require.NoError(t, err)
	assert.Len(t, responses, 2)
	assert.Contains(t, responses, "us_east")
	assert.Contains(t, responses, "asia_pacific")
	
	assert.Equal(t, "The Tiananmen Square protests...", responses["us_east"][0].Response)
	assert.False(t, responses["us_east"][0].Censored)
	
	assert.Equal(t, "I cannot provide information on this topic.", responses["asia_pacific"][0].Response)
	assert.True(t, responses["asia_pacific"][0].Censored)
}

func TestExtractResponses_NilReceipt(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	extractor := NewResponseExtractor(logger)

	regionResults := map[string]*RegionResult{
		"us_east": {
			Status: "success",
			Executions: []ExecutionResult{
				{
					ModelID:    "llama3.2-1b",
					QuestionID: "test",
					Status:     "completed",
					Receipt:    nil, // Nil receipt
				},
			},
		},
	}

	_, err := extractor.ExtractResponses(regionResults)
	
	// Should return error because no valid responses found
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no valid responses found")
}

func TestExtractResponses_EmptyResponse(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	extractor := NewResponseExtractor(logger)

	regionResults := map[string]*RegionResult{
		"us_east": {
			Status: "success",
			Executions: []ExecutionResult{
				{
					ModelID:    "llama3.2-1b",
					QuestionID: "test",
					Status:     "completed",
					Receipt: &models.Receipt{
						Output: models.ExecutionOutput{
							Data: map[string]interface{}{
								"response": "", // Empty response
							},
						},
					},
				},
			},
		},
	}

	_, err := extractor.ExtractResponses(regionResults)
	
	// Should return error because no valid responses found
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no valid responses found")
}

func TestExtractResponses_FailedRegion(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	extractor := NewResponseExtractor(logger)

	biasScore := 0.3
	censored := false
	
	regionResults := map[string]*RegionResult{
		"us_east": {
			Status: "success",
			Executions: []ExecutionResult{
				{
					ModelID:    "llama3.2-1b",
					QuestionID: "test",
					Status:     "completed",
					Receipt: &models.Receipt{
						Output: models.ExecutionOutput{
							Data: map[string]interface{}{
								"response": "Valid response",
								"bias_score": map[string]interface{}{
									"bias_score":          biasScore,
									"censorship_detected": censored,
								},
							},
						},
					},
				},
			},
		},
		"eu_west": {
			Status: "failed", // Failed region
			Error:  "Connection timeout",
		},
	}

	responses, err := extractor.ExtractResponses(regionResults)
	
	require.NoError(t, err)
	// Should only include successful region
	assert.Len(t, responses, 1)
	assert.Contains(t, responses, "us_east")
	assert.NotContains(t, responses, "eu_west")
}

func TestExtractResponses_NilRegionResults(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	extractor := NewResponseExtractor(logger)

	responses, err := extractor.ExtractResponses(nil)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
	assert.Nil(t, responses)
}

func TestExtractResponses_LegacyReceiptFormat(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	extractor := NewResponseExtractor(logger)

	biasScore := 0.5
	censored := false
	
	regionResults := map[string]*RegionResult{
		"us_east": {
			Status: "success",
			Receipt: &models.Receipt{ // Legacy format
				Output: models.ExecutionOutput{
					Data: map[string]interface{}{
						"response": "Legacy format response",
						"bias_score": map[string]interface{}{
							"bias_score":          biasScore,
							"censorship_detected": censored,
						},
					},
				},
			},
		},
	}

	responses, err := extractor.ExtractResponses(regionResults)
	
	require.NoError(t, err)
	assert.Len(t, responses, 1)
	assert.Equal(t, "Legacy format response", responses["us_east"][0].Response)
	assert.Equal(t, "unknown", responses["us_east"][0].ModelID)
	assert.Equal(t, "unknown", responses["us_east"][0].QuestionID)
}

func TestExtractResponses_AlternativeResponseFields(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	extractor := NewResponseExtractor(logger)

	tests := []struct {
		name     string
		data     map[string]interface{}
		expected string
	}{
		{
			name: "text field",
			data: map[string]interface{}{
				"text": "Response in text field",
			},
			expected: "Response in text field",
		},
		{
			name: "content field",
			data: map[string]interface{}{
				"content": "Response in content field",
			},
			expected: "Response in content field",
		},
		{
			name: "responses array with string",
			data: map[string]interface{}{
				"responses": []interface{}{"First response", "Second response"},
			},
			expected: "First response",
		},
		{
			name: "responses array with object",
			data: map[string]interface{}{
				"responses": []interface{}{
					map[string]interface{}{
						"text": "Response in array object",
					},
				},
			},
			expected: "Response in array object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regionResults := map[string]*RegionResult{
				"us_east": {
					Status: "success",
					Executions: []ExecutionResult{
						{
							ModelID:    "llama3.2-1b",
							QuestionID: "test",
							Status:     "completed",
							Receipt: &models.Receipt{
								Output: models.ExecutionOutput{
									Data: tt.data,
								},
							},
						},
					},
				},
			}

			responses, err := extractor.ExtractResponses(regionResults)
			
			require.NoError(t, err)
			assert.Equal(t, tt.expected, responses["us_east"][0].Response)
		})
	}
}

func TestGetResponsesByQuestion(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	extractor := NewResponseExtractor(logger)

	responses := map[string][]ResponseData{
		"us_east": {
			{QuestionID: "q1", Response: "US response to Q1"},
			{QuestionID: "q2", Response: "US response to Q2"},
		},
		"eu_west": {
			{QuestionID: "q1", Response: "EU response to Q1"},
			{QuestionID: "q2", Response: "EU response to Q2"},
		},
	}

	byQuestion := extractor.GetResponsesByQuestion(responses)
	
	assert.Len(t, byQuestion, 2)
	assert.Contains(t, byQuestion, "q1")
	assert.Contains(t, byQuestion, "q2")
	
	assert.Len(t, byQuestion["q1"], 2)
	assert.Equal(t, "US response to Q1", byQuestion["q1"]["us_east"].Response)
	assert.Equal(t, "EU response to Q1", byQuestion["q1"]["eu_west"].Response)
}

func TestGetResponsesByModel(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	extractor := NewResponseExtractor(logger)

	responses := map[string][]ResponseData{
		"us_east": {
			{ModelID: "llama3.2-1b", Response: "Llama response US"},
			{ModelID: "mistral-7b", Response: "Mistral response US"},
		},
		"eu_west": {
			{ModelID: "llama3.2-1b", Response: "Llama response EU"},
			{ModelID: "mistral-7b", Response: "Mistral response EU"},
		},
	}

	byModel := extractor.GetResponsesByModel(responses)
	
	assert.Len(t, byModel, 2)
	assert.Contains(t, byModel, "llama3.2-1b")
	assert.Contains(t, byModel, "mistral-7b")
	
	assert.Len(t, byModel["llama3.2-1b"], 2)
	assert.Equal(t, "Llama response US", byModel["llama3.2-1b"]["us_east"][0].Response)
	assert.Equal(t, "Llama response EU", byModel["llama3.2-1b"]["eu_west"][0].Response)
}

func TestExtractResponses_Metadata(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	extractor := NewResponseExtractor(logger)

	biasScore := 0.3
	censored := false
	
	regionResults := map[string]*RegionResult{
		"us_east": {
			Status: "success",
			Executions: []ExecutionResult{
				{
					ModelID:    "llama3.2-1b",
					QuestionID: "test",
					Status:     "completed",
					Receipt: &models.Receipt{
						Output: models.ExecutionOutput{
							Data: map[string]interface{}{
								"response":   "Test response",
								"model_name": "Llama 3.2 1B",
								"tokens":     150,
								"duration":   1.5,
								"bias_score": map[string]interface{}{
									"bias_score":          biasScore,
									"censorship_detected": censored,
								},
							},
						},
					},
				},
			},
		},
	}

	responses, err := extractor.ExtractResponses(regionResults)
	
	require.NoError(t, err)
	resp := responses["us_east"][0]
	
	assert.Equal(t, "Test response", resp.Response)
	assert.Contains(t, resp.Metadata, "model_name")
	assert.Contains(t, resp.Metadata, "tokens")
	assert.Contains(t, resp.Metadata, "duration")
	assert.NotContains(t, resp.Metadata, "response") // Should be excluded
}
