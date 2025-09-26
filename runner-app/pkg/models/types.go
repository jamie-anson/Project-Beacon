package models

import (
	"time"
)

// WalletAuth represents wallet authentication data from the portal
type WalletAuth struct {
	Address   string `json:"address"`
	Signature string `json:"signature"`
	Message   string `json:"message"`
	ChainID   int    `json:"chainId"`
	Nonce     string `json:"nonce"`
	ExpiresAt string `json:"expiresAt"`
}

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
	Models      []ModelSpec            `json:"models,omitempty"`      // NEW: Multi-model support
	Runs        int                    `json:"runs,omitempty"`
	WalletAuth  *WalletAuth            `json:"wallet_auth,omitempty"`
	Signature   string                 `json:"signature"`
	PublicKey   string                 `json:"public_key"`
}

// BenchmarkSpec defines the benchmark to run
type BenchmarkSpec struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version,omitempty"` // Portal compatibility
	Description string                 `json:"description"`
	Container   ContainerSpec          `json:"container"`
	Input       InputSpec              `json:"input"`
	Scoring     ScoringSpec            `json:"scoring"`
	Metadata    map[string]interface{} `json:"metadata"`
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

// ModelSpec defines a model to be executed in multi-model jobs
type ModelSpec struct {
	ID              string   `json:"id"`               // e.g., "llama3.2-1b"
	Name            string   `json:"name"`             // e.g., "Llama 3.2-1B Instruct"
	Provider        string   `json:"provider"`         // e.g., "modal"
	ContainerImage  string   `json:"container_image"`  // e.g., "ghcr.io/jamie-anson/project-beacon/llama-3.2-1b:latest"
	Regions         []string `json:"regions"`          // Regions where this model should run
}

// ExecutionSummary provides high-level execution statistics
type ExecutionSummary struct {
	TotalDuration      time.Duration `json:"total_duration"`
	AverageLatency     time.Duration `json:"average_latency"`
	SuccessRate        float64       `json:"success_rate"`
	ResourceEfficiency float64       `json:"resource_efficiency"`
}
