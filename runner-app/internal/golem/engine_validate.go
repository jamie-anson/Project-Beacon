package golem

import (
	"context"
	"fmt"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// GetExecutionStatus returns the current status of an execution.
func (e *ExecutionEngine) GetExecutionStatus(jobspecID string) (*ExecutionSummary, error) {
	// In a real implementation, this would query the database
	// For now, return a placeholder
	return &ExecutionSummary{
		JobSpecID:    jobspecID,
		TotalRegions: 0,
		SuccessCount: 0,
		FailureCount: 0,
		Results:      []*ExecutionResult{},
	}, nil
}

// CancelExecution cancels a running execution.
func (e *ExecutionEngine) CancelExecution(ctx context.Context, jobspecID string) error {
	// In a real implementation, this would cancel running tasks
	// For now, return success
	return nil
}

// ValidateExecution validates that an execution meets the JobSpec requirements.
func (e *ExecutionEngine) ValidateExecution(summary *ExecutionSummary, jobspec *models.JobSpec) error {
	// Check minimum regions requirement
	if summary.SuccessCount < jobspec.Constraints.MinRegions {
		return fmt.Errorf("insufficient successful executions: got %d, need %d",
			summary.SuccessCount, jobspec.Constraints.MinRegions)
	}

	// Check that all required regions have results
	requiredRegions := make(map[string]bool)
	for _, region := range jobspec.Constraints.Regions {
		requiredRegions[region] = false
	}

	for _, result := range summary.Results {
		if result.Error == nil {
			requiredRegions[result.Region] = true
		}
	}

	for region, satisfied := range requiredRegions {
		if !satisfied {
			return fmt.Errorf("no successful execution in required region: %s", region)
		}
	}

	return nil
}
