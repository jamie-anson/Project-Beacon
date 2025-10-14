package execution

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDifferenceAnalyzer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	simEngine := NewSimilarityEngine(logger)
	analyzer := NewDifferenceAnalyzer(logger, simEngine)

	assert.NotNil(t, analyzer)
	assert.NotNil(t, analyzer.logger)
	assert.NotNil(t, analyzer.similarityEngine)
}

func TestFindKeyDifferences_PoliticalStance(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	simEngine := NewSimilarityEngine(logger)
	analyzer := NewDifferenceAnalyzer(logger, simEngine)

	responses := map[string][]ResponseData{
		"us_east": {
			{
				QuestionID: "tiananmen",
				Response:   "The Tiananmen Square protests were a pro-democracy movement in 1989 advocating for freedom and human rights.",
			},
		},
		"asia_pacific": {
			{
				QuestionID: "tiananmen",
				Response:   "The government handled the situation to maintain stability and order.",
			},
		},
	}

	differences := analyzer.FindKeyDifferences(responses)

	require.NotEmpty(t, differences)
	
	// Should detect political stance differences
	foundPolitical := false
	for _, diff := range differences {
		if diff.Dimension == "political_stance_democracy" || 
		   diff.Dimension == "political_stance_human_rights" ||
		   diff.Dimension == "political_stance_government" {
			foundPolitical = true
			assert.NotEmpty(t, diff.Variations)
			assert.Contains(t, []string{"low", "medium", "high", "critical"}, diff.Severity)
		}
	}
	assert.True(t, foundPolitical, "Should detect political stance differences")
}

func TestFindKeyDifferences_Censorship(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	simEngine := NewSimilarityEngine(logger)
	analyzer := NewDifferenceAnalyzer(logger, simEngine)

	responses := map[string][]ResponseData{
		"us_east": {
			{
				QuestionID: "sensitive-topic",
				Response:   "This is a detailed response about the topic with factual information.",
				Censored:   false,
			},
		},
		"asia_pacific": {
			{
				QuestionID: "sensitive-topic",
				Response:   "I cannot provide information on this sensitive topic.",
				Censored:   true,
			},
		},
	}

	differences := analyzer.FindKeyDifferences(responses)

	require.NotEmpty(t, differences)
	
	// Should detect censorship difference
	foundCensorship := false
	for _, diff := range differences {
		if diff.Dimension == "censorship" {
			foundCensorship = true
			assert.Len(t, diff.Variations, 2)
			assert.Contains(t, diff.Variations["censored"], "Censored in")
			assert.Contains(t, diff.Variations["uncensored"], "Uncensored in")
			assert.Contains(t, []string{"medium", "high"}, diff.Severity)
		}
	}
	assert.True(t, foundCensorship, "Should detect censorship differences")
}

func TestFindKeyDifferences_FactualAccuracy(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	simEngine := NewSimilarityEngine(logger)
	analyzer := NewDifferenceAnalyzer(logger, simEngine)

	responses := map[string][]ResponseData{
		"us_east": {
			{
				QuestionID: "capital-france",
				Response:   "The capital of France is Paris, a major European city.",
			},
		},
		"eu_west": {
			{
				QuestionID: "capital-france",
				Response:   "Paris is the capital city of France in Western Europe.",
			},
		},
	}

	differences := analyzer.FindKeyDifferences(responses)

	// Should NOT detect factual differences for similar responses
	foundFactual := false
	for _, diff := range differences {
		if diff.Dimension == "factual_accuracy" {
			foundFactual = true
		}
	}
	assert.False(t, foundFactual, "Should not detect factual differences for similar responses")
}

func TestFindKeyDifferences_FactualDivergence(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	simEngine := NewSimilarityEngine(logger)
	analyzer := NewDifferenceAnalyzer(logger, simEngine)

	responses := map[string][]ResponseData{
		"us_east": {
			{
				QuestionID: "historical-event",
				Response:   "The event was a peaceful demonstration advocating for democratic reforms and civil liberties.",
			},
		},
		"asia_pacific": {
			{
				QuestionID: "historical-event",
				Response:   "I cannot discuss this topic as it is politically sensitive.",
			},
		},
	}

	differences := analyzer.FindKeyDifferences(responses)

	require.NotEmpty(t, differences)
	
	// Should detect factual divergence
	foundFactual := false
	for _, diff := range differences {
		if diff.Dimension == "factual_accuracy" {
			foundFactual = true
			assert.NotEmpty(t, diff.Variations)
			assert.Contains(t, []string{"high", "critical"}, diff.Severity)
		}
	}
	assert.True(t, foundFactual, "Should detect factual divergence")
}

func TestFindKeyDifferences_ToneSentiment(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	simEngine := NewSimilarityEngine(logger)
	analyzer := NewDifferenceAnalyzer(logger, simEngine)

	responses := map[string][]ResponseData{
		"us_east": {
			{
				QuestionID: "policy-evaluation",
				Response:   "The policy has been successful and beneficial, leading to positive progress and excellent improvements in freedom and democracy.",
			},
		},
		"asia_pacific": {
			{
				QuestionID: "policy-evaluation",
				Response:   "The policy has caused terrible problems, harmful effects, and negative consequences with failed implementation and crisis situations.",
			},
		},
	}

	differences := analyzer.FindKeyDifferences(responses)

	require.NotEmpty(t, differences)
	
	// Should detect tone differences
	foundTone := false
	for _, diff := range differences {
		if diff.Dimension == "tone_sentiment" {
			foundTone = true
			assert.NotEmpty(t, diff.Variations)
			assert.Contains(t, []string{"medium", "high"}, diff.Severity)
		}
	}
	assert.True(t, foundTone, "Should detect tone/sentiment differences")
}

func TestFindKeyDifferences_IdenticalResponses(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	simEngine := NewSimilarityEngine(logger)
	analyzer := NewDifferenceAnalyzer(logger, simEngine)

	responses := map[string][]ResponseData{
		"us_east": {
			{
				QuestionID: "simple-fact",
				Response:   "The sky is blue during the day.",
			},
		},
		"eu_west": {
			{
				QuestionID: "simple-fact",
				Response:   "The sky is blue during the day.",
			},
		},
	}

	differences := analyzer.FindKeyDifferences(responses)

	// Should find no or minimal differences
	assert.Empty(t, differences, "Should find no differences for identical responses")
}

func TestFindKeyDifferences_SingleRegion(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	simEngine := NewSimilarityEngine(logger)
	analyzer := NewDifferenceAnalyzer(logger, simEngine)

	responses := map[string][]ResponseData{
		"us_east": {
			{
				QuestionID: "test",
				Response:   "Some response",
			},
		},
	}

	differences := analyzer.FindKeyDifferences(responses)

	// Cannot detect differences with only one region
	assert.Empty(t, differences)
}

func TestDetectCensorshipDifferences_Mixed(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	simEngine := NewSimilarityEngine(logger)
	analyzer := NewDifferenceAnalyzer(logger, simEngine)

	responses := map[string][]ResponseData{
		"us_east": {
			{QuestionID: "q1", Response: "Full response with details", Censored: false},
		},
		"eu_west": {
			{QuestionID: "q1", Response: "Another full response", Censored: false},
		},
		"asia_pacific": {
			{QuestionID: "q1", Response: "I cannot answer this", Censored: true},
		},
	}

	differences := analyzer.detectCensorshipDifferences(responses)

	require.Len(t, differences, 1)
	assert.Equal(t, "censorship", differences[0].Dimension)
	assert.Contains(t, differences[0].Variations["censored"], "Censored in")
	assert.Contains(t, differences[0].Variations["uncensored"], "Uncensored in")
}

func TestCalculateToneScore(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	simEngine := NewSimilarityEngine(logger)
	analyzer := NewDifferenceAnalyzer(logger, simEngine)

	tests := []struct {
		name     string
		text     string
		minScore float64
		maxScore float64
	}{
		{
			name:     "Positive text",
			text:     "This is great and excellent with positive progress and successful improvements",
			minScore: 0.5,
			maxScore: 1.0,
		},
		{
			name:     "Negative text",
			text:     "This is terrible and bad with negative problems and failed crisis situations",
			minScore: -1.0,
			maxScore: -0.5,
		},
		{
			name:     "Neutral text",
			text:     "The weather today is cloudy with some wind",
			minScore: -0.1,
			maxScore: 0.1,
		},
		{
			name:     "Mixed text",
			text:     "Some good progress but also bad problems",
			minScore: -0.5,
			maxScore: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := analyzer.calculateToneScore(tt.text)
			assert.GreaterOrEqual(t, score, tt.minScore)
			assert.LessOrEqual(t, score, tt.maxScore)
		})
	}
}

func TestNewSimilarityEngine(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	engine := NewSimilarityEngine(logger)

	assert.NotNil(t, engine)
	assert.NotNil(t, engine.logger)
}

func TestCalculateJaccardSimilarity(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	engine := NewSimilarityEngine(logger)

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
			similarity := engine.CalculateJaccardSimilarity(tt.text1, tt.text2)
			assert.GreaterOrEqual(t, similarity, tt.minSim)
			assert.LessOrEqual(t, similarity, tt.maxSim)
		})
	}
}

func TestDetectCensorship(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	engine := NewSimilarityEngine(logger)

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
			name:     "Sensitive topic",
			text:     "This topic is sensitive and I cannot discuss it",
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
			result := engine.DetectCensorship(tt.text)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSimilarityEngineTokenize(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	engine := NewSimilarityEngine(logger)

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
			tokens := engine.tokenize(tt.text)
			assert.Equal(t, tt.expected, tokens)
		})
	}
}

func TestCalculateSeverity(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	simEngine := NewSimilarityEngine(logger)
	analyzer := NewDifferenceAnalyzer(logger, simEngine)

	tests := []struct {
		name      string
		variations []string
		responses  map[string]ResponseData
		expected   string
	}{
		{
			name:       "Single variation",
			variations: []string{"variation1"},
			responses: map[string]ResponseData{
				"us": {Censored: false},
			},
			expected: "low",
		},
		{
			name:       "With censorship",
			variations: []string{"var1", "var2"},
			responses: map[string]ResponseData{
				"us":   {Censored: false},
				"asia": {Censored: true},
			},
			expected: "critical",
		},
		{
			name:       "Three variations",
			variations: []string{"var1", "var2", "var3"},
			responses: map[string]ResponseData{
				"us": {Censored: false},
				"eu": {Censored: false},
			},
			expected: "high",
		},
		{
			name:       "Two variations",
			variations: []string{"var1", "var2"},
			responses: map[string]ResponseData{
				"us": {Censored: false},
				"eu": {Censored: false},
			},
			expected: "medium",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			severity := analyzer.calculateSeverity(tt.variations, tt.responses)
			assert.Equal(t, tt.expected, severity)
		})
	}
}
