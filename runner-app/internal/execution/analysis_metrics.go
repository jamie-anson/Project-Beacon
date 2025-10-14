package execution

import (
	"log/slog"
	"math"
	"strings"
)

// MetricsCalculator calculates various analysis metrics
type MetricsCalculator struct {
	logger *slog.Logger
}

// NewMetricsCalculator creates a new metrics calculator
func NewMetricsCalculator(logger *slog.Logger) *MetricsCalculator {
	return &MetricsCalculator{
		logger: logger,
	}
}

// CalculateBiasVariance measures response variation across regions
// Returns 0.0-1.0 where higher = more variance
func (mc *MetricsCalculator) CalculateBiasVariance(responses map[string][]ResponseData) float64 {
	if len(responses) < 2 {
		return 0.0
	}

	// Group responses by question to compare same questions across regions
	byQuestion := make(map[string][]ResponseData)
	for _, regionResponses := range responses {
		for _, resp := range regionResponses {
			byQuestion[resp.QuestionID] = append(byQuestion[resp.QuestionID], resp)
		}
	}

	if len(byQuestion) == 0 {
		return 0.0
	}

	totalVariance := 0.0
	questionCount := 0

	for questionID, questionResponses := range byQuestion {
		if len(questionResponses) < 2 {
			continue
		}

		// Calculate variance for this question
		variance := mc.calculateQuestionVariance(questionResponses)
		totalVariance += variance
		questionCount++

		mc.logger.Debug("Question variance calculated",
			"question_id", questionID,
			"variance", variance,
			"response_count", len(questionResponses))
	}

	if questionCount == 0 {
		return 0.0
	}

	avgVariance := totalVariance / float64(questionCount)
	return math.Min(1.0, avgVariance)
}

// calculateQuestionVariance calculates variance for a single question across regions
func (mc *MetricsCalculator) calculateQuestionVariance(responses []ResponseData) float64 {
	if len(responses) < 2 {
		return 0.0
	}

	// Use bias scores if available
	biasScores := make([]float64, 0, len(responses))
	for _, resp := range responses {
		biasScores = append(biasScores, resp.BiasScore)
	}

	// Calculate standard deviation of bias scores
	mean := mc.mean(biasScores)
	variance := 0.0
	for _, score := range biasScores {
		diff := score - mean
		variance += diff * diff
	}
	variance /= float64(len(biasScores))
	stdDev := math.Sqrt(variance)

	// Normalize to 0-1 range (assuming bias scores are 0-1)
	// Standard deviation of 0.5 would indicate maximum variance
	return math.Min(1.0, stdDev*2.0)
}

// CalculateCensorshipRate measures censorship across regions
// Returns 0.0-1.0 where higher = more censorship
func (mc *MetricsCalculator) CalculateCensorshipRate(responses map[string][]ResponseData) float64 {
	totalResponses := 0
	censoredResponses := 0

	for _, regionResponses := range responses {
		for _, resp := range regionResponses {
			totalResponses++
			if resp.Censored || mc.detectCensorshipInText(resp.Response) {
				censoredResponses++
			}
		}
	}

	if totalResponses == 0 {
		return 0.0
	}

	rate := float64(censoredResponses) / float64(totalResponses)
	
	mc.logger.Debug("Censorship rate calculated",
		"total_responses", totalResponses,
		"censored_responses", censoredResponses,
		"rate", rate)

	return rate
}

// detectCensorshipInText checks if response text indicates censorship
func (mc *MetricsCalculator) detectCensorshipInText(text string) bool {
	lowerText := strings.ToLower(text)
	
	censorshipPhrases := []string{
		"i cannot",
		"i can't",
		"i'm not able to",
		"i am not able to",
		"i'm unable to",
		"i am unable to",
		"i don't have information",
		"i do not have information",
		"i cannot provide",
		"i can't provide",
		"i'm not allowed",
		"i am not allowed",
		"this topic is sensitive",
		"i cannot discuss",
		"i can't discuss",
		"i'm not permitted",
		"i am not permitted",
		"i decline to answer",
		"i must decline",
		"i cannot answer",
		"i can't answer",
	}

	for _, phrase := range censorshipPhrases {
		if strings.Contains(lowerText, phrase) {
			return true
		}
	}

	return false
}

// CalculateFactualConsistency measures factual alignment across regions
// Returns 0.0-1.0 where higher = more consistent
func (mc *MetricsCalculator) CalculateFactualConsistency(responses map[string][]ResponseData) float64 {
	if len(responses) < 2 {
		return 1.0 // Single region is consistent with itself
	}

	// Group responses by question
	byQuestion := make(map[string][]ResponseData)
	for _, regionResponses := range responses {
		for _, resp := range regionResponses {
			byQuestion[resp.QuestionID] = append(byQuestion[resp.QuestionID], resp)
		}
	}

	if len(byQuestion) == 0 {
		return 1.0
	}

	totalConsistency := 0.0
	questionCount := 0

	for _, questionResponses := range byQuestion {
		if len(questionResponses) < 2 {
			continue
		}

		// Calculate pairwise similarity
		similarities := []float64{}
		for i := 0; i < len(questionResponses); i++ {
			for j := i + 1; j < len(questionResponses); j++ {
				sim := mc.calculateTextSimilarity(
					questionResponses[i].Response,
					questionResponses[j].Response,
				)
				similarities = append(similarities, sim)
			}
		}

		if len(similarities) > 0 {
			avgSimilarity := mc.mean(similarities)
			totalConsistency += avgSimilarity
			questionCount++
		}
	}

	if questionCount == 0 {
		return 1.0
	}

	consistency := totalConsistency / float64(questionCount)
	return math.Min(1.0, math.Max(0.0, consistency))
}

// CalculateNarrativeDivergence measures narrative differences across regions
// Returns 0.0-1.0 where higher = more divergence
func (mc *MetricsCalculator) CalculateNarrativeDivergence(responses map[string][]ResponseData) float64 {
	// Narrative divergence is inverse of factual consistency
	consistency := mc.CalculateFactualConsistency(responses)
	return 1.0 - consistency
}

// calculateTextSimilarity calculates similarity between two texts
// Simple implementation using word overlap (Jaccard similarity)
func (mc *MetricsCalculator) calculateTextSimilarity(text1, text2 string) float64 {
	words1 := mc.tokenize(text1)
	words2 := mc.tokenize(text2)

	if len(words1) == 0 && len(words2) == 0 {
		return 1.0
	}
	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	// Calculate Jaccard similarity
	set1 := make(map[string]bool)
	for _, word := range words1 {
		set1[word] = true
	}

	set2 := make(map[string]bool)
	for _, word := range words2 {
		set2[word] = true
	}

	intersection := 0
	for word := range set1 {
		if set2[word] {
			intersection++
		}
	}

	union := len(set1) + len(set2) - intersection
	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// tokenize splits text into words
func (mc *MetricsCalculator) tokenize(text string) []string {
	// Convert to lowercase and split on whitespace and punctuation
	text = strings.ToLower(text)
	
	// Replace punctuation with spaces
	replacer := strings.NewReplacer(
		".", " ",
		",", " ",
		"!", " ",
		"?", " ",
		";", " ",
		":", " ",
		"(", " ",
		")", " ",
		"[", " ",
		"]", " ",
		"{", " ",
		"}", " ",
		"\"", " ",
		"'", " ",
	)
	text = replacer.Replace(text)

	// Split on whitespace
	words := strings.Fields(text)

	// Filter out very short words (< 3 chars)
	filtered := make([]string, 0, len(words))
	for _, word := range words {
		if len(word) >= 3 {
			filtered = append(filtered, word)
		}
	}

	return filtered
}

// mean calculates the mean of a slice of float64
func (mc *MetricsCalculator) mean(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}
