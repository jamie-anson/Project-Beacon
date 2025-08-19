package diff

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// Engine handles cross-region diff computation
type Engine struct {
	similarityThreshold float64
}

// NewEngine creates a new diff engine with default settings
func NewEngine() *Engine {
	return &Engine{
		similarityThreshold: 0.95,
	}
}

// NewEngineWithThreshold creates a new diff engine with custom similarity threshold
func NewEngineWithThreshold(threshold float64) *Engine {
	return &Engine{
		similarityThreshold: threshold,
	}
}

// ComputeDiff compares outputs from two regions and generates a diff
func (e *Engine) ComputeDiff(jobSpecID, regionA, regionB string, outputA, outputB map[string]interface{}) (*models.CrossRegionDiff, error) {
	diff := &models.CrossRegionDiff{
		JobSpecID:    jobSpecID,
		RegionA:      regionA,
		RegionB:      regionB,
		CreatedAt:    time.Now(),
		DiffData:     models.DiffData{},
	}

	// Compute similarity score
	similarity, err := e.computeSimilarity(outputA, outputB)
	if err != nil {
		return nil, fmt.Errorf("failed to compute similarity: %w", err)
	}
	diff.SimilarityScore = similarity

	// Classify the diff
	diff.Classification = e.classifyDiff(similarity)

	// Generate detailed diffs
	textDiffs := e.computeTextDiffs(outputA, outputB)
	structuralDiffs := e.computeStructuralDiffs(outputA, outputB)

	diff.DiffData.TextDiffs = textDiffs
	diff.DiffData.StructDiffs = structuralDiffs

	return diff, nil
}

// computeSimilarity calculates similarity score between two outputs
func (e *Engine) computeSimilarity(outputA, outputB map[string]interface{}) (float64, error) {
	if reflect.DeepEqual(outputA, outputB) {
		return 1.0, nil
	}

	// Convert to JSON for comparison
	jsonA, err := json.Marshal(outputA)
	if err != nil {
		return 0, err
	}

	jsonB, err := json.Marshal(outputB)
	if err != nil {
		return 0, err
	}

	// Simple similarity based on string comparison
	strA := string(jsonA)
	strB := string(jsonB)

	if strA == strB {
		return 1.0, nil
	}

	// Calculate Levenshtein-based similarity
	return e.calculateStringSimilarity(strA, strB), nil
}

// calculateStringSimilarity computes similarity between two strings
func (e *Engine) calculateStringSimilarity(a, b string) float64 {
	if a == b {
		return 1.0
	}

	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}

	if maxLen == 0 {
		return 1.0
	}

	distance := levenshteinDistance(a, b)
	return 1.0 - float64(distance)/float64(maxLen)
}

// levenshteinDistance computes the edit distance between two strings
func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	matrix := make([][]int, len(a)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(b)+1)
		matrix[i][0] = i
	}

	for j := 1; j <= len(b); j++ {
		matrix[0][j] = j
	}

	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(a)][len(b)]
}

func min(a, b, c int) int {
	if a < b && a < c {
		return a
	}
	if b < c {
		return b
	}
	return c
}

// classifyDiff determines the classification based on similarity score
func (e *Engine) classifyDiff(similarity float64) string {
	if similarity >= e.similarityThreshold {
		return "identical"
	} else if similarity >= 0.8 {
		return "minor_diff"
	} else if similarity >= 0.5 {
		return "moderate_diff"
	}
	return "major_diff"
}

// computeTextDiffs finds textual differences between outputs
func (e *Engine) computeTextDiffs(outputA, outputB map[string]interface{}) []models.TextDiff {
	var diffs []models.TextDiff

	// Compare text fields
	for key, valueA := range outputA {
		if valueB, exists := outputB[key]; exists {
			strA := fmt.Sprintf("%v", valueA)
			strB := fmt.Sprintf("%v", valueB)

			if strA != strB {
				diffs = append(diffs, models.TextDiff{
					Type:    "changed",
					Content: fmt.Sprintf("A: %s\nB: %s", strA, strB),
					Context: "field_comparison",
				})
			}
		} else {
			diffs = append(diffs, models.TextDiff{
				Type:    "removed",
				Content: fmt.Sprintf("%s: %v", key, valueA),
				Context: "missing_in_b",
			})
		}
	}

	// Check for fields only in B
	for key, valueB := range outputB {
		if _, exists := outputA[key]; !exists {
			diffs = append(diffs, models.TextDiff{
				Type:    "added",
				Content: fmt.Sprintf("%s: %v", key, valueB),
				Context: "missing_in_a",
			})
		}
	}

	return diffs
}

// computeStructuralDiffs finds structural differences between outputs
func (e *Engine) computeStructuralDiffs(outputA, outputB map[string]interface{}) []models.StructuralDiff {
	var diffs []models.StructuralDiff

	// Compare structure
	keysA := getKeys(outputA)
	keysB := getKeys(outputB)

	if !equalStringSlices(keysA, keysB) {
		diffs = append(diffs, models.StructuralDiff{
			Path:     "root",
			Type:     "key_mismatch",
			OldValue: keysA,
			NewValue: keysB,
		})
	}

	// Compare data types
	for key, valueA := range outputA {
		if valueB, exists := outputB[key]; exists {
			typeA := reflect.TypeOf(valueA).String()
			typeB := reflect.TypeOf(valueB).String()

			if typeA != typeB {
				diffs = append(diffs, models.StructuralDiff{
					Path:     key,
					Type:     "added",
					NewValue: outputB[key],
				})
			}
		}
	}

	return diffs
}

// getKeys extracts and sorts keys from a map
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// equalStringSlices checks if two string slices are equal (order-independent)
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	countA := make(map[string]int)
	countB := make(map[string]int)

	for _, s := range a {
		countA[s]++
	}
	for _, s := range b {
		countB[s]++
	}

	return reflect.DeepEqual(countA, countB)
}

// SetSimilarityThreshold updates the similarity threshold
func (e *Engine) SetSimilarityThreshold(threshold float64) {
	e.similarityThreshold = threshold
}

// GetSimilarityThreshold returns the current similarity threshold
func (e *Engine) GetSimilarityThreshold() float64 {
	return e.similarityThreshold
}
