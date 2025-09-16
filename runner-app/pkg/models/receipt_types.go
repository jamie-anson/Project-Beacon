package models

import (
	"fmt"
	"time"
)

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
	CompletedAt      time.Time              `json:"completed_at"`
	
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

// NewReceipt creates a new Receipt v0 with proper initialization
func NewReceipt(jobSpecID string, executionDetails ExecutionDetails, output ExecutionOutput, provenance ProvenanceInfo) *Receipt {
	return &Receipt{
		SchemaVersion:    "v0.1.0",
		ID:               fmt.Sprintf("receipt-%d", time.Now().UnixNano()),
		JobSpecID:        jobSpecID,
		ExecutionDetails: executionDetails,
		Output:           output,
		Provenance:       provenance,
		CreatedAt:        time.Now(),
	}
}

// AddAttestation adds a third-party attestation to the receipt
func (r *Receipt) AddAttestation(attestation ExecutionAttestation) {
	r.Attestations = append(r.Attestations, attestation)
}
