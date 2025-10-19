package models

import (
	"time"
)

const ProjectPurposeDefault = "Project Beacon audits large language models across geographic regions to identify censorship, bias, and misinformation risks for enterprise transparency."

// CrossRegionDiff represents differences between regional executions
type CrossRegionDiff struct {
	ID              string                 `json:"id"`
	JobSpecID       string                 `json:"jobspec_id"`
	RegionA         string                 `json:"region_a"`
	RegionB         string                 `json:"region_b"`
	SimilarityScore float64                `json:"similarity_score"` // 0.0 to 1.0
	DiffData        DiffData               `json:"diff_data"`
	Classification  string                 `json:"classification"` // "significant", "minor", "noise"
	Metadata        map[string]interface{} `json:"metadata"`
	CreatedAt       time.Time              `json:"created_at"`
}

// DiffData contains the actual differences
type DiffData struct {
	TextDiffs    []TextDiff             `json:"text_diffs,omitempty"`
	StructDiffs  []StructuralDiff       `json:"struct_diffs,omitempty"`
	Summary      string                 `json:"summary"`
	Metadata     map[string]interface{} `json:"metadata"`
}

type TextDiff struct {
	Type    string `json:"type"`    // "added", "removed", "changed"
	LineNum int    `json:"line_num"`
	Content string `json:"content"`
	Context string `json:"context,omitempty"`
}

type StructuralDiff struct {
	Path     string      `json:"path"`     // JSON path to the difference
	Type     string      `json:"type"`     // "added", "removed", "changed", "type_changed"
	OldValue interface{} `json:"old_value,omitempty"`
	NewValue interface{} `json:"new_value,omitempty"`
}

// CrossRegionData contains multi-region execution results and analysis
type CrossRegionData struct {
	TotalRegions     int                       `json:"total_regions"`
	SuccessfulRegions int                      `json:"successful_regions"`
	RegionResults    map[string]*RegionResult  `json:"region_results"`
	Analysis         CrossRegionAnalysis       `json:"analysis"`
}

// RegionResult represents execution result for a single region
type RegionResult struct {
	Region        string                 `json:"region"`
	ProviderID    string                 `json:"provider_id"`
	ProviderInfo  map[string]interface{} `json:"provider_info"`
	StartedAt     time.Time              `json:"started_at"`
	CompletedAt   time.Time              `json:"completed_at"`
	Duration      time.Duration          `json:"duration"`
	Status        string                 `json:"status"` // "success", "failed", "timeout"
	Output        *ExecutionOutput       `json:"output,omitempty"`
	Scoring       *RegionScoring         `json:"scoring,omitempty"`
	Error         string                 `json:"error,omitempty"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// RegionScoring contains region-specific scoring metrics
type RegionScoring struct {
	BiasScore          float64  `json:"bias_score"`
	CensorshipDetected bool     `json:"censorship_detected"`
	FactualAccuracy    float64  `json:"factual_accuracy"`
	PoliticalSensitivity float64 `json:"political_sensitivity"`
	KeywordsDetected   []string `json:"keywords_detected"`
	Flags              []string `json:"flags,omitempty"`
}

// CrossRegionAnalysis contains analysis of differences across regions
type CrossRegionAnalysis struct {
	BiasVariance        float64          `json:"bias_variance"`
	CensorshipRate      float64          `json:"censorship_rate"`
	FactualConsistency  float64          `json:"factual_consistency"`
	NarrativeDivergence float64          `json:"narrative_divergence"`
	KeyDifferences      []KeyDifference  `json:"key_differences"`
	RiskAssessment      []RiskAssessment `json:"risk_assessment"`
	Summary             string           `json:"summary"`
	Recommendation      string           `json:"recommendation"`
	JobID               string           `json:"job_id,omitempty"`
	ProjectPurpose      string           `json:"project_purpose,omitempty"`
	BenchmarkName       string           `json:"benchmark_name,omitempty"`
	BenchmarkDescription string          `json:"benchmark_description,omitempty"`
	Regions             []string         `json:"regions,omitempty"`
	Models              []string         `json:"models,omitempty"`
	Questions           []string         `json:"questions,omitempty"`
	QuestionDetails     []string         `json:"question_details,omitempty"`
}

// KeyDifference represents a significant difference between regions
type KeyDifference struct {
	Dimension   string            `json:"dimension"`
	Variations  map[string]string `json:"variations"`
	Severity    string            `json:"severity"` // "high", "medium", "low"
	Description string            `json:"description"`
}

// RiskAssessment represents identified risks from cross-region analysis
type RiskAssessment struct {
	Type        string   `json:"type"`        // "censorship", "bias", "manipulation"
	Severity    string   `json:"severity"`    // "high", "medium", "low"
	Description string   `json:"description"`
	Regions     []string `json:"regions"`
	Confidence  float64  `json:"confidence"` // 0.0 to 1.0
}
