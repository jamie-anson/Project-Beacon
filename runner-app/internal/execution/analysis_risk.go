package execution

import (
	"fmt"
	"log/slog"
)

// RiskAssessor generates risk assessments from analysis
type RiskAssessor struct {
	logger *slog.Logger
}

// NewRiskAssessor creates a new risk assessor
func NewRiskAssessor(logger *slog.Logger) *RiskAssessor {
	return &RiskAssessor{
		logger: logger,
	}
}

// GenerateRiskAssessments creates risk assessments from analysis
func (ra *RiskAssessor) GenerateRiskAssessments(
	biasVariance float64,
	censorshipRate float64,
	keyDifferences []KeyDifference,
	responses map[string][]ResponseData,
) []RiskAssessment {
	assessments := []RiskAssessment{}

	// Assess censorship risk
	if censorshipRate > 0.0 {
		assessments = append(assessments, ra.assessCensorshipRisk(censorshipRate, responses))
	}

	// Assess bias risk
	if biasVariance > 0.3 {
		assessments = append(assessments, ra.assessBiasRisk(biasVariance, responses))
	}

	// Assess misinformation risk from key differences
	for _, diff := range keyDifferences {
		if diff.Dimension == "factual_accuracy" {
			assessments = append(assessments, ra.assessMisinformationRisk(diff, responses))
		}
	}

	// Assess political manipulation risk
	politicalDiffs := ra.filterPoliticalDifferences(keyDifferences)
	if len(politicalDiffs) > 0 {
		assessments = append(assessments, ra.assessPoliticalManipulationRisk(politicalDiffs, responses))
	}

	// Assess narrative control risk
	if censorshipRate > 0.3 && biasVariance > 0.5 {
		assessments = append(assessments, ra.assessNarrativeControlRisk(censorshipRate, biasVariance, responses))
	}

	ra.logger.Info("Risk assessments generated",
		"total_assessments", len(assessments),
		"censorship_rate", censorshipRate,
		"bias_variance", biasVariance,
		"key_differences", len(keyDifferences))

	return assessments
}

// assessCensorshipRisk assesses censorship-related risks
func (ra *RiskAssessor) assessCensorshipRisk(censorshipRate float64, responses map[string][]ResponseData) RiskAssessment {
	severity := "low"

	if censorshipRate >= 0.7 {
		severity = "critical"
	} else if censorshipRate >= 0.5 {
		severity = "high"
	} else if censorshipRate >= 0.3 {
		severity = "medium"
	}

	// Identify censored regions
	censoredRegions := []string{}
	for region, regionResponses := range responses {
		for _, resp := range regionResponses {
			if resp.Censored {
				censoredRegions = append(censoredRegions, region)
				break
			}
		}
	}

	description := fmt.Sprintf(
		"%.0f%% of responses show censorship patterns. "+
			"This indicates restricted information access and potential suppression of certain viewpoints.",
		censorshipRate*100,
	)

	return RiskAssessment{
		Type:        "censorship",
		Severity:    severity,
		Description: description,
		Regions:     censoredRegions,
	}
}

// assessBiasRisk assesses bias-related risks
func (ra *RiskAssessor) assessBiasRisk(biasVariance float64, responses map[string][]ResponseData) RiskAssessment {
	severity := "low"

	if biasVariance >= 0.8 {
		severity = "critical"
	} else if biasVariance >= 0.6 {
		severity = "high"
	} else if biasVariance >= 0.4 {
		severity = "medium"
	}

	regions := []string{}
	for region := range responses {
		regions = append(regions, region)
	}

	description := fmt.Sprintf(
		"Bias variance of %.2f indicates significant differences in how information is presented across regions. "+
			"This suggests potential ideological framing or selective emphasis of certain aspects.",
		biasVariance,
	)

	return RiskAssessment{
		Type:        "bias",
		Severity:    severity,
		Description: description,
		Regions:     regions,
	}
}

// assessMisinformationRisk assesses misinformation-related risks
func (ra *RiskAssessor) assessMisinformationRisk(diff KeyDifference, responses map[string][]ResponseData) RiskAssessment {
	severity := diff.Severity

	regions := []string{}
	for region := range responses {
		regions = append(regions, region)
	}

	description := fmt.Sprintf(
		"Significant factual divergence detected across regions (%s severity). "+
			"This raises concerns about information accuracy and potential misinformation.",
		severity,
	)

	return RiskAssessment{
		Type:        "misinformation",
		Severity:    severity,
		Description: description,
		Regions:     regions,
	}
}

// assessPoliticalManipulationRisk assesses political manipulation risks
func (ra *RiskAssessor) assessPoliticalManipulationRisk(politicalDiffs []KeyDifference, responses map[string][]ResponseData) RiskAssessment {
	// Calculate severity based on number and severity of political differences
	criticalCount := 0
	highCount := 0
	mediumCount := 0

	for _, diff := range politicalDiffs {
		switch diff.Severity {
		case "critical":
			criticalCount++
		case "high":
			highCount++
		case "medium":
			mediumCount++
		}
	}

	severity := "low"

	if criticalCount > 0 || highCount >= 2 {
		severity = "critical"
	} else if highCount > 0 || mediumCount >= 3 {
		severity = "high"
	} else if mediumCount > 0 {
		severity = "medium"
	}

	regions := []string{}
	for region := range responses {
		regions = append(regions, region)
	}

	description := fmt.Sprintf(
		"Detected %d political framing differences across regions. "+
			"This suggests potential political manipulation or ideological filtering of information.",
		len(politicalDiffs),
	)

	return RiskAssessment{
		Type:        "political_manipulation",
		Severity:    severity,
		Description: description,
		Regions:     regions,
	}
}

// assessNarrativeControlRisk assesses narrative control risks
func (ra *RiskAssessor) assessNarrativeControlRisk(censorshipRate, biasVariance float64, responses map[string][]ResponseData) RiskAssessment {
	// Combined high censorship and bias indicates narrative control
	severity := "critical"

	regions := []string{}
	for region := range responses {
		regions = append(regions, region)
	}

	description := fmt.Sprintf(
		"Combination of high censorship (%.0f%%) and high bias variance (%.2f) indicates systematic narrative control. "+
			"This suggests coordinated efforts to shape public perception through information restriction and framing.",
		censorshipRate*100,
		biasVariance,
	)

	return RiskAssessment{
		Type:        "narrative_control",
		Severity:    severity,
		Description: description,
		Regions:     regions,
	}
}

// filterPoliticalDifferences filters key differences for political-related ones
func (ra *RiskAssessor) filterPoliticalDifferences(differences []KeyDifference) []KeyDifference {
	political := []KeyDifference{}

	for _, diff := range differences {
		if len(diff.Dimension) >= 16 && diff.Dimension[:16] == "political_stance" {
			political = append(political, diff)
		}
	}

	return political
}
