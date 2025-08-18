package ipfs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/store"
)

// Bundler creates IPFS bundles from execution results
type Bundler struct {
	client    *Client
	ipfsRepo  *store.IPFSRepo
}

// getRegionsFromBundle extracts regions from the prepared bundle receipts
func (b *Bundler) getRegionsFromBundle(bun *Bundle) []string {
    regionSet := make(map[string]bool)
    for _, r := range bun.Receipts {
        regionSet[r.Region] = true
    }
    regions := make([]string, 0, len(regionSet))
    for region := range regionSet {
        regions = append(regions, region)
    }
    return regions
}

// NewBundler creates a new IPFS bundler
func NewBundler(client *Client, ipfsRepo *store.IPFSRepo) *Bundler {
	return &Bundler{
		client:   client,
		ipfsRepo: ipfsRepo,
	}
}

// CreateBundle creates a complete bundle from execution results
func (b *Bundler) CreateBundle(ctx context.Context, jobID string) (*Bundle, error) {
	// Get all executions for this job from IPFS repo (by jobspec_id)
	executions, err := b.ipfsRepo.GetExecutionsByJobSpecID(jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get executions for job %s: %w", jobID, err)
	}

	if len(executions) == 0 {
		return nil, fmt.Errorf("no executions found for job %s", jobID)
	}

	// Create bundle structure
	bundle := &Bundle{
		JobID:     jobID,
		Timestamp: time.Now().UTC(),
		Receipts:  make([]Receipt, 0, len(executions)),
		Outputs:   make(map[string]string),
		Metadata:  make(map[string]interface{}),
	}

	// Use the first execution ID as the primary execution ID
	bundle.ExecutionID = fmt.Sprintf("%d", executions[0].ID)

	// Process each execution
	for _, exec := range executions {
		// Extract output data from JSON field
		outputData := ""
		if exec.OutputData.Valid {
			outputData = exec.OutputData.String
		}

		// Create receipt
		receipt := Receipt{
			ExecutionID: fmt.Sprintf("%d", exec.ID),
			JobID:       fmt.Sprintf("%d", exec.JobID),
			Region:      exec.Region,
			ProviderID:  exec.ProviderID,
			Output:      outputData,
			OutputHash:  b.hashOutput(outputData),
			StartedAt:   exec.StartedAt.Time,
			CompletedAt: exec.CompletedAt.Time,
		}

		// For now, skip cryptographic signing - can be added later
		receipt.Signature = "unsigned"
		receipt.PublicKey = "none"

		bundle.Receipts = append(bundle.Receipts, receipt)
		bundle.Outputs[exec.Region] = outputData
	}

	// Add metadata
	bundle.Metadata = map[string]interface{}{
		"total_executions": len(executions),
		"regions":          b.getRegions(executions),
		"bundle_version":   "1.0.0",
		"created_by":       "project-beacon-runner",
	}

	// For now, skip bundle signing - can be added later
	bundle.Signature = "unsigned"
	bundle.PublicKey = "none"

	return bundle, nil
}

// StoreBundle creates a bundle and stores it in IPFS
func (b *Bundler) StoreBundle(ctx context.Context, jobID string) (string, error) {
	// Create the bundle
	bundle, err := b.CreateBundle(ctx, jobID)
	if err != nil {
		return "", err
	}

	// Add and pin to IPFS
	cid, err := b.client.AddAndPin(ctx, bundle)
	if err != nil {
		return "", fmt.Errorf("failed to store bundle in IPFS: %w", err)
	}

	// Persist bundle metadata and update executions' CID if repo is available
	if b.ipfsRepo != nil {
		now := time.Now().UTC()
		gw := b.client.GetGatewayURL(cid)
		regions := b.getRegionsFromBundle(bundle)

		rec := &store.IPFSBundle{
			JobID:          jobID,
			CID:            cid,
			BundleSize:     nil,
			ExecutionCount: len(bundle.Receipts),
			Regions:        regions,
			PinnedAt:       &now,
			GatewayURL:     &gw,
		}
		// best-effort insert
		_ = b.ipfsRepo.CreateBundle(rec)

		// update executions missing CID
		if execs, err := b.ipfsRepo.GetExecutionsByJobSpecID(jobID); err == nil {
			for _, e := range execs {
				if !e.IPFSCid.Valid || e.IPFSCid.String == "" {
					_ = b.ipfsRepo.UpdateExecutionCIDByID(e.ID, cid)
				}
			}
		}
	}

	return cid, nil
}

// hashOutput creates a SHA256 hash of the execution output
func (b *Bundler) hashOutput(output string) string {
	hash := sha256.Sum256([]byte(output))
	return hex.EncodeToString(hash[:])
}

// getRegions extracts unique regions from executions
func (b *Bundler) getRegions(executions []store.Execution) []string {
	regionSet := make(map[string]bool)
	for _, exec := range executions {
		regionSet[exec.Region] = true
	}
	
	regions := make([]string, 0, len(regionSet))
	for region := range regionSet {
		regions = append(regions, region)
	}
	return regions
}
