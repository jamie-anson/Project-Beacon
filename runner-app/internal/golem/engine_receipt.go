package golem

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// generateReceipt creates a cryptographically signed receipt for the execution.
func (e *ExecutionEngine) generateReceipt(jobspec *models.JobSpec, execution *TaskExecution, provider *Provider) (*models.Receipt, error) {
	// Compute output hash (SHA256 of canonical JSON of execution.Output)
	var outputHash string
	if execution != nil {
		if b, err := json.Marshal(execution.Output); err == nil {
			sum := sha256.Sum256(b)
			outputHash = hex.EncodeToString(sum[:])
		}
	}

	receipt := &models.Receipt{
		ID:        fmt.Sprintf("receipt_%s_%s", execution.ID, provider.Region),
		JobSpecID: jobspec.ID,
		ExecutionDetails: models.ExecutionDetails{
			TaskID:      execution.ID,
			ProviderID:  provider.ID,
			Region:      provider.Region,
			StartedAt:   execution.StartedAt,
			CompletedAt: execution.CompletedAt,
			Duration:    execution.CompletedAt.Sub(execution.StartedAt),
			Status:      execution.Status,
		},
		Output: models.ExecutionOutput{
			Data:     execution.Output,
			Hash:     outputHash,
			Metadata: execution.Metadata,
		},
		Provenance: models.ProvenanceInfo{
			BenchmarkHash: jobspec.Benchmark.Input.Hash,
			ProviderInfo: map[string]interface{}{
				"id":        provider.ID,
				"name":      provider.Name,
				"region":    provider.Region,
				"score":     provider.Score,
				"resources": provider.Resources,
			},
			ExecutionEnv: map[string]interface{}{
				"container_image": jobspec.Benchmark.Container.Image,
				"timeout":        jobspec.Constraints.Timeout.String(),
				"network":        e.service.Network(),
			},
		},
		CreatedAt: time.Now(),
	}

	// Sign the receipt if a signing key is configured
	if e.service.SigningKey() != nil {
		if err := receipt.Sign(e.service.SigningKey()); err != nil {
			return nil, fmt.Errorf("sign receipt: %w", err)
		}
	}

	return receipt, nil
}
