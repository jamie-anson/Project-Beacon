package golem

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
	"github.com/jamie-anson/project-beacon-runner/pkg/crypto"
	"github.com/jamie-anson/project-beacon-runner/internal/transparency"
)

// ExecuteSingleRegion executes a JobSpec in a single target region using the best available provider.
// Returns the execution result and a generated (mock) receipt.
func ExecuteSingleRegion(ctx context.Context, svc *Service, jobspec *models.JobSpec, region string) (*ExecutionResult, error) {
	// Validate region is requested by the JobSpec
	allowed := false
	for _, r := range jobspec.Constraints.Regions {
		if r == region {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, fmt.Errorf("region %s not in jobspec constraints", region)
	}

	// Discover providers and pick the best in-region provider by score
	providers, err := svc.DiscoverProviders(ctx, jobspec.Constraints)
	if err != nil {
		return nil, fmt.Errorf("discover providers: %w", err)
	}

	var inRegion []*Provider
	for _, p := range providers {
		if p.Region == region {
			inRegion = append(inRegion, p)
		}
	}
	if len(inRegion) == 0 {
		return nil, fmt.Errorf("no providers available in region %s", region)
	}

	sort.Slice(inRegion, func(i, j int) bool { return inRegion[i].Score > inRegion[j].Score })
	best := inRegion[0]

	// Execute task
	execCtx, cancel := context.WithTimeout(ctx, jobspec.Constraints.Timeout)
	defer cancel()

	exec, err := svc.ExecuteTask(execCtx, best, jobspec)
	if err != nil {
		return &ExecutionResult{
			JobSpecID:  jobspec.ID,
			Region:     region,
			ProviderID: best.ID,
			Execution:  exec,
			Error:      err,
			ExecutedAt: time.Now(),
		}, nil
	}

	// Build a minimal receipt using existing helper
	receipt, _ := NewExecutionEngine(svc).generateReceipt(jobspec, exec, best)

	// Append to transparency log (best-effort, non-fatal)
	var (
		outputHash  string
		receiptHash string
	)
	if b, err := crypto.CanonicalJSON(exec.Output); err == nil {
		h := sha256.Sum256(b)
		outputHash = hex.EncodeToString(h[:])
	}
	if b, err := crypto.CanonicalJSON(receipt); err == nil {
		h := sha256.Sum256(b)
		receiptHash = hex.EncodeToString(h[:])
	}

	entry := transparency.LogEntry{
		// LogIndex is assigned in Append
		ExecutionID: hashStringToInt(exec.ID),
		JobID:       jobspec.ID,
		Region:      region,
		ProviderID:  best.ID,
		Status:      exec.Status,
		OutputHash:  outputHash,
		ReceiptHash: receiptHash,
		// IPFSCID can be set later when bundling occurs
		Timestamp:   time.Now().UTC(),
	}
	transparency.DefaultWriter.Append(entry)

	return &ExecutionResult{
		JobSpecID:  jobspec.ID,
		Region:     region,
		ProviderID: best.ID,
		Execution:  exec,
		Receipt:    receipt,
		ExecutedAt: time.Now(),
	}, nil
}
