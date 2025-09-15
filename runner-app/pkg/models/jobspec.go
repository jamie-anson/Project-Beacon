package models

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/crypto"
)

// JobSpec represents a signed benchmark execution specification
type JobSpec struct {
	ID          string                 `json:"id,omitempty"`
	JobSpecID   string                 `json:"jobspec_id,omitempty"`
	Version     string                 `json:"version"`
	Benchmark   BenchmarkSpec          `json:"benchmark"`
	Constraints ExecutionConstraints   `json:"constraints"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	Questions   []string               `json:"questions,omitempty"`
	Signature   string                 `json:"signature"`
	PublicKey   string                 `json:"public_key"`
}

// BenchmarkSpec defines the benchmark to be executed
type BenchmarkSpec struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Container   ContainerSpec     `json:"container"`
	Input       InputSpec         `json:"input"`
	Scoring     ScoringSpec       `json:"scoring"`
	Metadata    map[string]string `json:"metadata"`
}

// ContainerSpec defines the execution environment
type ContainerSpec struct {
	Image       string            `json:"image"`
	Tag         string            `json:"tag"`
	Command     []string          `json:"command,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Resources   ResourceSpec      `json:"resources"`
}

// ResourceSpec defines resource requirements
type ResourceSpec struct {
	CPU    string `json:"cpu"`    // e.g., "1000m" for 1 CPU
	Memory string `json:"memory"` // e.g., "512Mi"
	GPU    string `json:"gpu,omitempty"`
}

// InputSpec defines the benchmark input
type InputSpec struct {
	Type string                 `json:"type"` // "prompt", "dataset", "file"
	Data map[string]interface{} `json:"data"`
	Hash string                 `json:"hash"` // SHA256 hash for integrity
}

// ScoringSpec defines how results should be evaluated
type ScoringSpec struct {
	Method     string                 `json:"method"`     // "similarity", "accuracy", "custom"
	Parameters map[string]interface{} `json:"parameters"`
}

// ExecutionConstraints define where and how the benchmark should run
type ExecutionConstraints struct {
	Regions         []string          `json:"regions"`         // Required regions: ["US", "EU", "APAC"]
	MinRegions      int               `json:"min_regions"`     // Minimum number of regions (default: 3)
	MinSuccessRate  float64           `json:"min_success_rate"` // Minimum success rate (0.0-1.0, default: 0.67)
	Timeout         time.Duration     `json:"timeout"`         // Max execution time per region
	ProviderTimeout time.Duration     `json:"provider_timeout"` // Max time per provider attempt
	MaxCost         float64           `json:"max_cost,omitempty"` // Maximum total cost in GLM
	Providers       []ProviderFilter  `json:"providers,omitempty"`
}

// ProviderFilter defines Golem provider selection criteria
type ProviderFilter struct {
	Region     string  `json:"region"`
	MinScore   float64 `json:"min_score,omitempty"`
	MaxPrice   float64 `json:"max_price,omitempty"`
	Whitelist  []string `json:"whitelist,omitempty"`
	Blacklist  []string `json:"blacklist,omitempty"`
}

// Receipt v0 represents a cryptographically signed execution proof
type Receipt struct {
	// Schema metadata
	SchemaVersion    string                 `json:"schema_version"`    // "v0.1.0"
	
	// Core identifiers
	ID               string                 `json:"id"`               // Receipt UUID
	JobSpecID        string                 `json:"jobspec_id"`       // Reference to JobSpec
	
	// Execution information
	ExecutionDetails ExecutionDetails       `json:"execution_details"`
	Output           ExecutionOutput        `json:"output"`
	
	// Cross-region execution data (optional)
	CrossRegionData  *CrossRegionData       `json:"cross_region_data,omitempty"`
	
	// Verification and provenance
	Provenance       ProvenanceInfo         `json:"provenance"`
	Attestations     []ExecutionAttestation `json:"attestations,omitempty"`
	
	// Metadata
	CreatedAt        time.Time              `json:"created_at"`
	
	// Cryptographic fields (excluded from signing)
	Signature        string                 `json:"signature"`
	PublicKey        string                 `json:"public_key"`
}

// ExecutionAttestation represents third-party verification of execution
type ExecutionAttestation struct {
	Type        string    `json:"type"`         // "provider", "network", "third_party"
	Source      string    `json:"source"`       // Identifier of attesting party
	Timestamp   time.Time `json:"timestamp"`
	Statement   string    `json:"statement"`    // What is being attested
	Evidence    string    `json:"evidence"`     // Hash or reference to evidence
	Signature   string    `json:"signature"`    // Attestation signature
	PublicKey   string    `json:"public_key"`   // Attester's public key
}

// ExecutionDetails contains execution metadata
type ExecutionDetails struct {
	TaskID      string        `json:"task_id"`
	ProviderID  string        `json:"provider_id"`
	Region      string        `json:"region"`
	StartedAt   time.Time     `json:"started_at"`
	CompletedAt time.Time     `json:"completed_at"`
	Duration    time.Duration `json:"duration"`
	Status      string        `json:"status"`
}

// ProvenanceInfo contains provenance and integrity data
type ProvenanceInfo struct {
	BenchmarkHash string                 `json:"benchmark_hash"`
	ProviderInfo  map[string]interface{} `json:"provider_info"`
	ExecutionEnv  map[string]interface{} `json:"execution_env"`
}

// ExecutionOutput contains the benchmark results
type ExecutionOutput struct {
	Data     interface{}            `json:"data"`
	Hash     string                 `json:"hash"`     // SHA256 of output data
	Metadata map[string]interface{} `json:"metadata"`
}

// ResourceUsage tracks actual resource consumption
type ResourceUsage struct {
	CPUTime    time.Duration `json:"cpu_time"`
	MemoryPeak int64         `json:"memory_peak"` // Peak memory usage in bytes
	NetworkIO  NetworkIO     `json:"network_io"`
	DiskIO     DiskIO        `json:"disk_io"`
}

type NetworkIO struct {
	BytesIn  int64 `json:"bytes_in"`
	BytesOut int64 `json:"bytes_out"`
}

type DiskIO struct {
	BytesRead    int64 `json:"bytes_read"`
	BytesWritten int64 `json:"bytes_written"`
}

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
	Analysis         *CrossRegionAnalysis      `json:"analysis,omitempty"`
	ExecutionSummary ExecutionSummary          `json:"execution_summary"`
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
	Error         string                 `json:"error,omitempty"`
	Scoring       *RegionScoring         `json:"scoring,omitempty"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// RegionScoring contains region-specific scoring metrics
type RegionScoring struct {
	BiasScore          float64  `json:"bias_score"`
	CensorshipDetected bool     `json:"censorship_detected"`
	FactualAccuracy    float64  `json:"factual_accuracy"`
	PoliticalSensitivity float64 `json:"political_sensitivity"`
	KeywordsDetected   []string `json:"keywords_detected"`
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

// ExecutionSummary provides high-level execution statistics
type ExecutionSummary struct {
	TotalDuration      time.Duration `json:"total_duration"`
	AverageLatency     time.Duration `json:"average_latency"`
	SuccessRate        float64       `json:"success_rate"`
	RegionDistribution map[string]int `json:"region_distribution"`
	ProviderTypes      map[string]int `json:"provider_types"`
}

// Validation methods
func (js *JobSpec) Validate() error {
	// Auto-generate ID if missing (for job creation)
	if js.ID == "" && js.JobSpecID == "" {
		// Generate ID from benchmark name and timestamp
		timestamp := time.Now().Unix()
		if js.Benchmark.Name != "" {
			js.ID = fmt.Sprintf("%s-%d", js.Benchmark.Name, timestamp)
		} else {
			js.ID = fmt.Sprintf("job-%d", timestamp)
		}
	}
	// Normalize: if JobSpecID is provided but ID is empty, copy it over
	if js.ID == "" && js.JobSpecID != "" {
		js.ID = js.JobSpecID
	}
	if js.Version == "" {
		return fmt.Errorf("jobspec version is required")
	}
	if js.Benchmark.Name == "" {
		return fmt.Errorf("benchmark name is required")
	}
	if js.Benchmark.Container.Image == "" {
		return fmt.Errorf("container image is required")
	}
	if len(js.Constraints.Regions) == 0 {
		return fmt.Errorf("at least one region constraint is required")
	}
	if js.Constraints.MinRegions < 1 {
		js.Constraints.MinRegions = 3 // Default to 3 regions
	}
	if js.Constraints.MinSuccessRate == 0 {
		js.Constraints.MinSuccessRate = 0.67 // Default to 67% success rate
	}
	if js.Constraints.Timeout == 0 {
		js.Constraints.Timeout = 10 * time.Minute // Default timeout
	}
	if js.Constraints.ProviderTimeout == 0 {
		js.Constraints.ProviderTimeout = 2 * time.Minute // Default provider timeout
	}
	if js.Benchmark.Input.Hash == "" {
		return fmt.Errorf("input hash is required for integrity verification")
	}
	// Enforce questions for bias-detection v1
	if strings.EqualFold(js.Version, "v1") {
		name := strings.ToLower(js.Benchmark.Name)
		if strings.Contains(name, "bias") {
			if len(js.Questions) == 0 {
				return fmt.Errorf("questions are required for bias-detection v1 jobspec")
			}
		}
	}
	
	return nil
}

func (js *JobSpec) VerifySignature() error {
	if js.Signature == "" {
		return fmt.Errorf("signature is required")
	}
	if js.PublicKey == "" {
		return fmt.Errorf("public key is required")
	}

	// Parse public key
	publicKey, err := crypto.PublicKeyFromBase64(js.PublicKey)
	if err != nil {
		return fmt.Errorf("invalid public key: %w", err)
	}

	// Create signable data (without signature and public_key fields)
	signableData, err := crypto.CreateSignableJobSpec(js)
	if err != nil {
		return fmt.Errorf("failed to create signable data: %w", err)
	}

	// Verify signature
	if err := crypto.VerifyJSONSignature(signableData, js.Signature, publicKey); err != nil {
        // Fallback: accept v1 canonicalization if it verifies (temporary compatibility)
        if canonV1, cErr := crypto.CanonicalizeJobSpecV1(js); cErr == nil {
            if sigBytes, sErr := base64.StdEncoding.DecodeString(js.Signature); sErr == nil {
                if ed25519.Verify(publicKey, canonV1, sigBytes) {
                    fmt.Printf("COMPAT: JobSpec %s verified with v1 canonicalization (accepting; please re-sign/update)\n", js.ID)
                    return nil
                }
            }
        }
        
        // Backward compatibility: try v0 canonicalization
        if canonV0, cErr := crypto.CanonicalizeJobSpecV0(js); cErr == nil {
            if sigBytes, sErr := base64.StdEncoding.DecodeString(js.Signature); sErr == nil {
                if ed25519.Verify(publicKey, canonV0, sigBytes) {
                    // Log deprecation warning using fmt.Printf for now
                    // TODO: Replace with proper structured logging when available
                    fmt.Printf("DEPRECATED: JobSpec %s signed with v0 canonicalization - please re-sign with current method\n", js.ID)
                    return nil // Accept v0 signature with warning
                }
            }
        }
        
        return fmt.Errorf("signature verification failed: %w", err)
    }

	return nil
}

func (js *JobSpec) Sign(privateKey ed25519.PrivateKey) error {
	// Set public key from private key
	publicKey := privateKey.Public().(ed25519.PublicKey)
	js.PublicKey = crypto.PublicKeyToBase64(publicKey)

	// Create signable data (without signature and public_key fields)
	signableData, err := crypto.CreateSignableJobSpec(js)
	if err != nil {
		return fmt.Errorf("failed to create signable data: %w", err)
	}

	// Sign the data
	signature, err := crypto.SignJSON(signableData, privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign jobspec: %w", err)
	}

	js.Signature = signature
	return nil
}

// NewReceipt creates a new Receipt v0 with proper initialization
func NewReceipt(jobSpecID string, executionDetails ExecutionDetails, output ExecutionOutput, provenance ProvenanceInfo) *Receipt {
	return &Receipt{
		SchemaVersion:    "v0.1.0",
		ID:               fmt.Sprintf("receipt-%d", time.Now().UnixNano()),
		JobSpecID:        jobSpecID,
		ExecutionDetails: executionDetails,
		Output:           output,
		Provenance:       provenance,
		Attestations:     []ExecutionAttestation{},
		CreatedAt:        time.Now(),
	}
}

// AddAttestation adds a third-party attestation to the receipt
func (r *Receipt) AddAttestation(attestation ExecutionAttestation) {
	r.Attestations = append(r.Attestations, attestation)
}

func (r *Receipt) Sign(privateKey ed25519.PrivateKey) error {
	// Set public key from private key
	publicKey := privateKey.Public().(ed25519.PublicKey)
	r.PublicKey = crypto.PublicKeyToBase64(publicKey)

	// Create signable data (without signature and public_key fields)
	signableData, err := crypto.CreateSignableReceipt(r)
	if err != nil {
		return fmt.Errorf("failed to create signable data: %w", err)
	}

	// Sign the data
	signature, err := crypto.SignJSON(signableData, privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign receipt: %w", err)
	}

	r.Signature = signature
	return nil
}

func (r *Receipt) VerifySignature() error {
	if r.Signature == "" {
		return fmt.Errorf("signature is required")
	}
	if r.PublicKey == "" {
		return fmt.Errorf("public key is required")
	}

	// Parse public key
	publicKey, err := crypto.PublicKeyFromBase64(r.PublicKey)
	if err != nil {
		return fmt.Errorf("invalid public key: %w", err)
	}

	// Create signable data (without signature and public_key fields)
	signableData, err := crypto.CreateSignableReceipt(r)
	if err != nil {
		return fmt.Errorf("failed to create signable data: %w", err)
	}

	// Verify signature
	if err := crypto.VerifyJSONSignature(signableData, r.Signature, publicKey); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}
