package golem

import (
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// ExecutionEngine orchestrates multi-region benchmark execution
// Note: keep this lean; behavior lives in engine_*.go files.
type ExecutionEngine struct {
	service EngineService
	results chan *ExecutionResult
}

// ExecutionResult represents the result of a benchmark execution
type ExecutionResult struct {
	JobSpecID  string          `json:"jobspec_id"`
	Region     string          `json:"region"`
	ProviderID string          `json:"provider_id"`
	Execution  *TaskExecution  `json:"execution"`
	Receipt    *models.Receipt `json:"receipt"`
	Error      error           `json:"error,omitempty"`
	ExecutedAt time.Time       `json:"executed_at"`
}

// ExecutionSummary provides an overview of multi-region execution
type ExecutionSummary struct {
	JobSpecID     string                      `json:"jobspec_id"`
	TotalRegions  int                         `json:"total_regions"`
	SuccessCount  int                         `json:"success_count"`
	FailureCount  int                         `json:"failure_count"`
	Results       []*ExecutionResult          `json:"results"`
	RegionResults map[string]*ExecutionResult `json:"region_results"`
	StartedAt     time.Time                   `json:"started_at"`
	CompletedAt   time.Time                   `json:"completed_at"`
	TotalDuration time.Duration               `json:"total_duration"`
	TotalCost     float64                     `json:"total_cost"`
	MaxCost       float64                     `json:"max_cost"`
	SuccessRate   float64                     `json:"success_rate"`
	PartialSuccess bool                       `json:"partial_success"`
	Errors        []string                    `json:"errors,omitempty"`
}

// NewExecutionEngine creates a new execution engine
func NewExecutionEngine(service EngineService) *ExecutionEngine {
	return &ExecutionEngine{service: service, results: make(chan *ExecutionResult, 100)}
}
