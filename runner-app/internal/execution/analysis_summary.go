package execution

import (
	"fmt"
	"log/slog"
	"strings"
)

// SummaryGenerator creates human-readable analysis summaries
type SummaryGenerator struct {
	logger *slog.Logger
}

// NewSummaryGenerator creates a new summary generator
func NewSummaryGenerator(logger *slog.Logger) *SummaryGenerator {
	return &SummaryGenerator{
		logger: logger,
	}
}

// GenerateSummary creates human-readable analysis summary
func (sg *SummaryGenerator) GenerateSummary(
	biasVariance float64,
	censorshipRate float64,
	factualConsistency float64,
	narrativeDivergence float64,
	keyDifferences []KeyDifference,
	riskAssessments []RiskAssessment,
) string {
	var summary strings.Builder

	// Opening statement
	summary.WriteString(sg.generateOpeningStatement(biasVariance, censorshipRate, factualConsistency))
	summary.WriteString("\n\n")

	// Metrics overview
	summary.WriteString("**Key Metrics:**\n")
	summary.WriteString(fmt.Sprintf("- Bias Variance: %.2f (%.0f%% variation in responses)\n", biasVariance, biasVariance*100))
	summary.WriteString(fmt.Sprintf("- Censorship Rate: %.2f (%.0f%% of responses censored)\n", censorshipRate, censorshipRate*100))
	summary.WriteString(fmt.Sprintf("- Factual Consistency: %.2f (%.0f%% alignment)\n", factualConsistency, factualConsistency*100))
	summary.WriteString(fmt.Sprintf("- Narrative Divergence: %.2f (%.0f%% difference)\n", narrativeDivergence, narrativeDivergence*100))
	summary.WriteString("\n")

	// Key findings
	if len(keyDifferences) > 0 {
		summary.WriteString("**Key Findings:**\n")
		summary.WriteString(sg.summarizeKeyDifferences(keyDifferences))
		summary.WriteString("\n")
	}

	// Risk assessment summary
	if len(riskAssessments) > 0 {
		summary.WriteString("**Identified Risks:**\n")
		summary.WriteString(sg.summarizeRiskAssessments(riskAssessments))
		summary.WriteString("\n")
	}

	// Conclusion
	summary.WriteString(sg.generateConclusion(biasVariance, censorshipRate, riskAssessments))

	sg.logger.Info("Summary generated",
		"length", summary.Len(),
		"key_differences", len(keyDifferences),
		"risk_assessments", len(riskAssessments))

	return summary.String()
}

// GenerateRecommendation creates risk level recommendation
func (sg *SummaryGenerator) GenerateRecommendation(
	biasVariance float64,
	censorshipRate float64,
	riskAssessments []RiskAssessment,
) string {
	// Count risk severities
	criticalCount := 0
	highCount := 0
	mediumCount := 0

	for _, risk := range riskAssessments {
		switch risk.Severity {
		case "critical":
			criticalCount++
		case "high":
			highCount++
		case "medium":
			mediumCount++
		}
	}

	// Determine overall risk level
	if criticalCount > 0 || (censorshipRate >= 0.7 && biasVariance >= 0.7) {
		return "CRITICAL RISK"
	} else if highCount >= 2 || criticalCount > 0 || censorshipRate >= 0.5 || biasVariance >= 0.6 {
		return "HIGH RISK"
	} else if highCount > 0 || mediumCount >= 2 || censorshipRate >= 0.3 || biasVariance >= 0.4 {
		return "MEDIUM RISK"
	}

	return "LOW RISK"
}

// generateOpeningStatement creates the opening paragraph
func (sg *SummaryGenerator) generateOpeningStatement(biasVariance, censorshipRate, factualConsistency float64) string {
	if censorshipRate >= 0.7 && biasVariance >= 0.7 {
		return "**Critical Analysis:** This cross-region analysis reveals severe information control patterns. " +
			"High censorship rates combined with significant bias variance indicate systematic narrative manipulation across regions."
	} else if censorshipRate >= 0.5 || biasVariance >= 0.6 {
		return "**Concerning Findings:** This analysis identifies significant differences in how information is presented across regions. " +
			"Notable censorship patterns and bias variations warrant careful attention."
	} else if factualConsistency >= 0.8 && censorshipRate < 0.2 && biasVariance < 0.3 {
		return "**Positive Assessment:** This analysis shows strong consistency across regions. " +
			"Minimal censorship and low bias variance indicate reliable and consistent information delivery."
	} else {
		return "**Mixed Results:** This cross-region analysis reveals moderate variations in information presentation. " +
			"Some differences detected but overall patterns remain within acceptable ranges."
	}
}

// summarizeKeyDifferences creates a summary of key differences
func (sg *SummaryGenerator) summarizeKeyDifferences(differences []KeyDifference) string {
	var summary strings.Builder

	// Group by dimension
	dimensionCounts := make(map[string]int)
	for _, diff := range differences {
		dimensionType := sg.getDimensionType(diff.Dimension)
		dimensionCounts[dimensionType]++
	}

	// Summarize each dimension
	for dimension, count := range dimensionCounts {
		summary.WriteString(fmt.Sprintf("- %s: %d difference(s) detected\n", sg.formatDimensionName(dimension), count))
	}

	// Highlight critical differences
	criticalDiffs := []KeyDifference{}
	for _, diff := range differences {
		if diff.Severity == "critical" || diff.Severity == "high" {
			criticalDiffs = append(criticalDiffs, diff)
		}
	}

	if len(criticalDiffs) > 0 {
		summary.WriteString(fmt.Sprintf("\nCritical/High severity differences: %d\n", len(criticalDiffs)))
		for i, diff := range criticalDiffs {
			if i >= 3 {
				summary.WriteString(fmt.Sprintf("  ... and %d more\n", len(criticalDiffs)-3))
				break
			}
			summary.WriteString(fmt.Sprintf("  • %s (%s severity)\n", sg.formatDimensionName(diff.Dimension), diff.Severity))
		}
	}

	return summary.String()
}

// summarizeRiskAssessments creates a summary of risk assessments
func (sg *SummaryGenerator) summarizeRiskAssessments(assessments []RiskAssessment) string {
	var summary strings.Builder

	// Group by severity
	bySeverity := make(map[string][]RiskAssessment)
	for _, assessment := range assessments {
		bySeverity[assessment.Severity] = append(bySeverity[assessment.Severity], assessment)
	}

	// List critical risks first
	if critical, exists := bySeverity["critical"]; exists {
		summary.WriteString(fmt.Sprintf("**Critical Risks (%d):**\n", len(critical)))
		for _, risk := range critical {
			summary.WriteString(fmt.Sprintf("- %s: %s\n", sg.formatRiskType(risk.Type), risk.Description))
		}
		summary.WriteString("\n")
	}

	// Then high risks
	if high, exists := bySeverity["high"]; exists {
		summary.WriteString(fmt.Sprintf("**High Risks (%d):**\n", len(high)))
		for _, risk := range high {
			summary.WriteString(fmt.Sprintf("- %s: %s\n", sg.formatRiskType(risk.Type), risk.Description))
		}
		summary.WriteString("\n")
	}

	// Medium risks (summarized)
	if medium, exists := bySeverity["medium"]; exists {
		summary.WriteString(fmt.Sprintf("**Medium Risks (%d):** ", len(medium)))
		types := []string{}
		for _, risk := range medium {
			types = append(types, sg.formatRiskType(risk.Type))
		}
		summary.WriteString(strings.Join(types, ", "))
		summary.WriteString("\n")
	}

	return summary.String()
}

// generateConclusion creates the conclusion paragraph
func (sg *SummaryGenerator) generateConclusion(biasVariance, censorshipRate float64, riskAssessments []RiskAssessment) string {
	criticalCount := 0
	for _, risk := range riskAssessments {
		if risk.Severity == "critical" {
			criticalCount++
		}
	}

	if criticalCount > 0 || (censorshipRate >= 0.7 && biasVariance >= 0.7) {
		return "**Conclusion:** The analysis reveals critical concerns requiring immediate attention. " +
			"The combination of high censorship and significant bias variance suggests systematic information control. " +
			"Users should exercise extreme caution and seek multiple independent sources."
	} else if censorshipRate >= 0.5 || biasVariance >= 0.6 {
		return "**Conclusion:** Notable differences exist in how information is presented across regions. " +
			"Users should be aware of potential bias and censorship patterns when interpreting results. " +
			"Cross-referencing with multiple sources is recommended."
	} else if censorshipRate < 0.2 && biasVariance < 0.3 {
		return "**Conclusion:** The analysis shows generally consistent and reliable information delivery across regions. " +
			"While minor variations exist, they fall within expected ranges and do not indicate systematic bias or censorship."
	} else {
		return "**Conclusion:** The analysis reveals moderate variations in information presentation. " +
			"While some differences exist, they do not indicate severe systematic issues. " +
			"Standard critical thinking and source verification practices are recommended."
	}
}

// Helper methods

func (sg *SummaryGenerator) getDimensionType(dimension string) string {
	if strings.HasPrefix(dimension, "political_stance") {
		return "political"
	}
	if strings.Contains(dimension, "censorship") {
		return "censorship"
	}
	if strings.Contains(dimension, "factual") {
		return "factual"
	}
	if strings.Contains(dimension, "tone") {
		return "tone"
	}
	return "other"
}

func (sg *SummaryGenerator) formatDimensionName(dimension string) string {
	// Convert snake_case to Title Case
	words := strings.Split(dimension, "_")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

func (sg *SummaryGenerator) formatRiskType(riskType string) string {
	// Convert snake_case to Title Case
	words := strings.Split(riskType, "_")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}
