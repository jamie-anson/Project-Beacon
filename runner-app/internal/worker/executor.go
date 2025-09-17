package worker

import (
	"context"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// Executor defines the interface for job execution strategies
type Executor interface {
	// Execute runs a job in a specific region and returns execution details
	// Returns: providerID, status, outputJSON, receiptJSON, error
	Execute(ctx context.Context, spec *models.JobSpec, region string) (providerID, status string, outputJSON, receiptJSON []byte, err error)
}
