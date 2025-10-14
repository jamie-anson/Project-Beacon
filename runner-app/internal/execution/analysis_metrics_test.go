package execution

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMetricsCalculator(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	calculator := NewMetricsCalculator(logger)
	
	assert.NotNil(t, calculator)
	assert.NotNil(t, calculator.logger)
}

func TestCalculateBiasVariance_IdenticalResponses(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	calculator := NewMetricsCalculator(logger)

	responses := map[string][]ResponseData{
		"us_east": {
			{QuestionID: "q1", Response: "Paris is the capital of France", BiasScore: 0.1},
		},
		"eu_west": {
			{QuestionID: "q1", Response: "The capital of France is Paris", BiasScore: 0.1},
		},
	}

	variance := calculator.CalculateBiasVariance(responses)
	
	// Identical bias scores should result in 0 variance
	assert.Equal(t, 0.0, variance)
}

func TestCalculateBiasVariance_CompletelyDifferent(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	calculator := NewMetricsCalculator(logger)

	responses := map[string][]ResponseData{
		"us_east": {
			{QuestionID: "q1", Response: "Response A", BiasScore: 0.0},
		},
		"asia_pacific": {
			{QuestionID: "q1", Response: "Response B", BiasScore: 1.0},
		},
	}

	variance := calculator.CalculateBiasVariance(responses)
	
	// Maximum difference in bias scores should result in high variance
	assert.Greater(t, variance, 0.8)
	assert.LessOrEqual(t, variance, 1.0)
}

func TestCalculateBiasVariance_PartialDifferences(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	calculator := NewMetricsCalculator(logger)

	responses := map[string][]ResponseData{
		"us_east": {
			{QuestionID: "q1", Response: "Response A", BiasScore: 0.3},
		},
		"eu_west": {
			{QuestionID: "q1", Response: "Response B", BiasScore: 0.5},
		},
		"asia_pacific": {
			{QuestionID: "q1", Response: "Response C", BiasScore: 0.4},
		},
	}

	variance := calculator.CalculateBiasVariance(responses)
	
	// Moderate differences should result in moderate variance
	assert.Greater(t, variance, 0.0)
	assert.Less(t, variance, 0.7)
}

func TestCalculateBiasVariance_SingleRegion(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	calculator := NewMetricsCalculator(logger)

	responses := map[string][]ResponseData{
		"us_east": {
			{QuestionID: "q1", Response: "Response A", BiasScore: 0.5},
		},
	}

	variance := calculator.CalculateBiasVariance(responses)
	
	// Single region should have 0 variance
	assert.Equal(t, 0.0, variance)
}

func TestCalculateCensorshipRate_NoCensorship(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	calculator := NewMetricsCalculator(logger)

	responses := map[string][]ResponseData{
		"us_east": {
			{Response: "This is a normal response", Censored: false},
			{Response: "Another normal response", Censored: false},
		},
		"eu_west": {
			{Response: "Yet another normal response", Censored: false},
		},
	}

	rate := calculator.CalculateCensorshipRate(responses)
	
	assert.Equal(t, 0.0, rate)
}

func TestCalculateCensorshipRate_AllCensored(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	calculator := NewMetricsCalculator(logger)

	responses := map[string][]ResponseData{
		"asia_pacific": {
			{Response: "I cannot provide information on this topic", Censored: true},
			{Response: "I'm not able to discuss this", Censored: true},
		},
	}

	rate := calculator.CalculateCensorshipRate(responses)
	
	assert.Equal(t, 1.0, rate)
}

func TestCalculateCensorshipRate_PartialCensorship(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	calculator := NewMetricsCalculator(logger)

	responses := map[string][]ResponseData{
		"us_east": {
			{Response: "Normal response", Censored: false},
			{Response: "Another normal response", Censored: false},
		},
		"asia_pacific": {
			{Response: "I cannot answer this question", Censored: true},
			{Response: "I'm unable to provide information", Censored: true},
		},
	}

	rate := calculator.CalculateCensorshipRate(responses)
	
	// 2 out of 4 responses are censored = 0.5
	assert.Equal(t, 0.5, rate)
}

func TestDetectCensorshipInText(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	calculator := NewMetricsCalculator(logger)

	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{
			name:     "I cannot answer",
			text:     "I cannot answer this question",
			expected: true,
		},
		{
			name:     "I can't provide",
			text:     "I can't provide information on this topic",
			expected: true,
		},
		{
			name:     "I'm not able to",
			text:     "I'm not able to discuss this matter",
			expected: true,
		},
		{
			name:     "I decline to answer",
			text:     "I decline to answer this question",
			expected: true,
		},
		{
			name:     "Normal response",
			text:     "This is a normal response with information",
			expected: false,
		},
		{
			name:     "Contains 'can' but not censorship",
			text:     "I can help you with this question",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculator.detectCensorshipInText(tt.text)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateFactualConsistency_SameFacts(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	calculator := NewMetricsCalculator(logger)

	responses := map[string][]ResponseData{
		"us_east": {
			{QuestionID: "q1", Response: "Paris is the capital of France"},
		},
		"eu_west": {
			{QuestionID: "q1", Response: "The capital of France is Paris"},
		},
	}

	consistency := calculator.CalculateFactualConsistency(responses)
	
	// Similar responses should have high consistency
	assert.Greater(t, consistency, 0.3)
	assert.LessOrEqual(t, consistency, 1.0)
}

func TestCalculateFactualConsistency_ContradictoryFacts(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	calculator := NewMetricsCalculator(logger)

	responses := map[string][]ResponseData{
		"us_east": {
			{QuestionID: "q1", Response: "The Tiananmen Square protests occurred in 1989"},
		},
		"asia_pacific": {
			{QuestionID: "q1", Response: "I cannot provide information on this topic"},
		},
	}

	consistency := calculator.CalculateFactualConsistency(responses)
	
	// Completely different responses should have low consistency
	assert.Less(t, consistency, 0.3)
	assert.GreaterOrEqual(t, consistency, 0.0)
}

func TestCalculateFactualConsistency_SingleRegion(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	calculator := NewMetricsCalculator(logger)

	responses := map[string][]ResponseData{
		"us_east": {
			{QuestionID: "q1", Response: "Some response"},
		},
	}

	consistency := calculator.CalculateFactualConsistency(responses)
	
	// Single region is consistent with itself
	assert.Equal(t, 1.0, consistency)
}

func TestCalculateNarrativeDivergence_SameNarrative(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	calculator := NewMetricsCalculator(logger)

	responses := map[string][]ResponseData{
		"us_east": {
			{QuestionID: "q1", Response: "Paris is the capital of France"},
		},
		"eu_west": {
			{QuestionID: "q1", Response: "The capital of France is Paris"},
		},
	}

	divergence := calculator.CalculateNarrativeDivergence(responses)
	
	// Similar narratives should have low divergence
	assert.Less(t, divergence, 0.7)
	assert.GreaterOrEqual(t, divergence, 0.0)
}

func TestCalculateNarrativeDivergence_DifferentNarratives(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	calculator := NewMetricsCalculator(logger)

	responses := map[string][]ResponseData{
		"us_east": {
			{QuestionID: "q1", Response: "The Tiananmen Square protests were a pro-democracy movement"},
		},
		"asia_pacific": {
			{QuestionID: "q1", Response: "I cannot discuss this sensitive political topic"},
		},
	}

	divergence := calculator.CalculateNarrativeDivergence(responses)
	
	// Different narratives should have high divergence
	assert.Greater(t, divergence, 0.7)
	assert.LessOrEqual(t, divergence, 1.0)
}

func TestCalculateTextSimilarity(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	calculator := NewMetricsCalculator(logger)

	tests := []struct {
		name     string
		text1    string
		text2    string
		minSim   float64
		maxSim   float64
	}{
		{
			name:   "Identical texts",
			text1:  "The quick brown fox jumps over the lazy dog",
			text2:  "The quick brown fox jumps over the lazy dog",
			minSim: 0.9,
			maxSim: 1.0,
		},
		{
			name:   "Similar texts",
			text1:  "Paris is the capital of France",
			text2:  "The capital of France is Paris",
			minSim: 0.5,
			maxSim: 1.0,
		},
		{
			name:   "Completely different",
			text1:  "The weather is sunny today",
			text2:  "Mathematics involves numbers and equations",
			minSim: 0.0,
			maxSim: 0.2,
		},
		{
			name:   "Empty strings",
			text1:  "",
			text2:  "",
			minSim: 1.0,
			maxSim: 1.0,
		},
		{
			name:   "One empty",
			text1:  "Some text",
			text2:  "",
			minSim: 0.0,
			maxSim: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			similarity := calculator.calculateTextSimilarity(tt.text1, tt.text2)
			assert.GreaterOrEqual(t, similarity, tt.minSim)
			assert.LessOrEqual(t, similarity, tt.maxSim)
		})
	}
}

func TestTokenize(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	calculator := NewMetricsCalculator(logger)

	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "Simple sentence",
			text:     "The quick brown fox",
			expected: []string{"the", "quick", "brown", "fox"},
		},
		{
			name:     "With punctuation",
			text:     "Hello, world! How are you?",
			expected: []string{"hello", "world", "how", "are", "you"},
		},
		{
			name:     "Filters short words",
			text:     "I am a big dog",
			expected: []string{"big", "dog"},
		},
		{
			name:     "Empty string",
			text:     "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := calculator.tokenize(tt.text)
			assert.Equal(t, tt.expected, tokens)
		})
	}
}

func TestMean(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	calculator := NewMetricsCalculator(logger)

	tests := []struct {
		name     string
		values   []float64
		expected float64
	}{
		{
			name:     "Normal values",
			values:   []float64{1.0, 2.0, 3.0, 4.0, 5.0},
			expected: 3.0,
		},
		{
			name:     "Single value",
			values:   []float64{5.0},
			expected: 5.0,
		},
		{
			name:     "Empty slice",
			values:   []float64{},
			expected: 0.0,
		},
		{
			name:     "Zeros",
			values:   []float64{0.0, 0.0, 0.0},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mean := calculator.mean(tt.values)
			assert.Equal(t, tt.expected, mean)
		})
	}
}

func TestCalculateBiasVariance_MultipleQuestions(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	calculator := NewMetricsCalculator(logger)

	responses := map[string][]ResponseData{
		"us_east": {
			{QuestionID: "q1", BiasScore: 0.1},
			{QuestionID: "q2", BiasScore: 0.2},
		},
		"eu_west": {
			{QuestionID: "q1", BiasScore: 0.1},
			{QuestionID: "q2", BiasScore: 0.8},
		},
	}

	variance := calculator.CalculateBiasVariance(responses)
	
	// Should average variance across both questions
	// q1: low variance, q2: high variance
	assert.Greater(t, variance, 0.0)
	assert.Less(t, variance, 1.0)
}
