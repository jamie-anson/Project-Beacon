package anchor

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/store"
)

// Service manages anchoring operations for the transparency log
type Service struct {
	strategy         AnchorStrategy
	transparencyRepo *store.TransparencyRepo
}

// NewService creates a new anchoring service
func NewService(transparencyRepo *store.TransparencyRepo) *Service {
	strategy := createAnchorStrategy()
	
	return &Service{
		strategy:         strategy,
		transparencyRepo: transparencyRepo,
	}
}

// createAnchorStrategy creates the appropriate anchoring strategy based on configuration
func createAnchorStrategy() AnchorStrategy {
	anchorType := os.Getenv("ANCHOR_STRATEGY")
	if anchorType == "" {
		anchorType = "timestamp" // Default to timestamp for MVP
	}

	switch anchorType {
	case "ethereum":
		rpcUrl := os.Getenv("ETHEREUM_RPC_URL")
		privateKey := os.Getenv("ETHEREUM_PRIVATE_KEY")
		contractAddr := os.Getenv("ETHEREUM_CONTRACT_ADDR")
		return NewEthereumStrategy(rpcUrl, privateKey, contractAddr)
		
	case "multi":
		// Use both timestamp and Ethereum for redundancy
		timestamp := NewTimestampStrategy("")
		ethereum := NewEthereumStrategy(
			os.Getenv("ETHEREUM_RPC_URL"),
			os.Getenv("ETHEREUM_PRIVATE_KEY"),
			os.Getenv("ETHEREUM_CONTRACT_ADDR"),
		)
		return NewMultiStrategy(timestamp, ethereum)
		
	default:
		return NewTimestampStrategy("")
	}
}

// AnchorLogEntry anchors a specific transparency log entry
func (s *Service) AnchorLogEntry(ctx context.Context, logIndex int64) error {
	// Get the log entry
	entry, err := s.transparencyRepo.GetEntryByIndex(logIndex)
	if err != nil {
		return fmt.Errorf("failed to get log entry: %w", err)
	}
	
	if entry == nil {
		return fmt.Errorf("log entry not found")
	}

	// Anchor the merkle leaf hash
	result, err := s.strategy.Anchor(ctx, entry.MerkleLeafHash)
	if err != nil {
		return fmt.Errorf("failed to anchor log entry: %w", err)
	}

	// Update the entry with anchor information
	err = s.updateEntryWithAnchor(entry, result)
	if err != nil {
		return fmt.Errorf("failed to update entry with anchor: %w", err)
	}

	return nil
}

// AnchorMerkleRoot anchors a Merkle tree root for a batch of entries
func (s *Service) AnchorMerkleRoot(ctx context.Context, rootHash string, startIndex, endIndex int64) error {
	// Anchor the root hash
	result, err := s.strategy.Anchor(ctx, rootHash)
	if err != nil {
		return fmt.Errorf("failed to anchor merkle root: %w", err)
	}

	// Update all entries in the range with the anchor information
	err = s.updateRangeWithAnchor(startIndex, endIndex, result)
	if err != nil {
		return fmt.Errorf("failed to update entries with anchor: %w", err)
	}

	return nil
}

// VerifyAnchor verifies an existing anchor
func (s *Service) VerifyAnchor(ctx context.Context, txHash string) (bool, error) {
	// Get anchor status
	status, err := s.strategy.GetStatus(ctx, txHash)
	if err != nil {
		return false, fmt.Errorf("failed to get anchor status: %w", err)
	}

	// Check if anchor is confirmed
	return status.Status == "confirmed", nil
}

// updateEntryWithAnchor updates a single entry with anchor information
func (s *Service) updateEntryWithAnchor(entry *store.TransparencyLogEntry, result *AnchorResult) error {
	// In a real implementation, you would update the database directly
	// For now, we'll simulate the update
	entry.AnchorTxHash = &result.TxHash
	entry.AnchorBlockNumber = &result.BlockNumber
	entry.AnchorTimestamp = &result.Timestamp
	
	return nil
}

// updateRangeWithAnchor updates a range of entries with anchor information
func (s *Service) updateRangeWithAnchor(startIndex, endIndex int64, result *AnchorResult) error {
	// In a real implementation, you would update the database with a batch operation
	// This would update all entries in the range with the anchor information
	return nil
}

// ScheduledAnchoringJob runs periodic anchoring of transparency log entries
func (s *Service) ScheduledAnchoringJob(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour) // Anchor every hour
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := s.performScheduledAnchoring(ctx)
			if err != nil {
				// Log error but continue
				fmt.Printf("Scheduled anchoring failed: %v\n", err)
			}
		}
	}
}

// performScheduledAnchoring anchors recent transparency log entries
func (s *Service) performScheduledAnchoring(ctx context.Context) error {
	// Get the latest entry
	latest, err := s.transparencyRepo.GetLatestEntry()
	if err != nil {
		return fmt.Errorf("failed to get latest entry: %w", err)
	}

	if latest == nil {
		return nil // No entries to anchor
	}

	// For MVP, anchor the latest entry if it doesn't have an anchor
	if latest.AnchorTxHash == nil {
		err = s.AnchorLogEntry(ctx, latest.LogIndex)
		if err != nil {
			return fmt.Errorf("failed to anchor latest entry: %w", err)
		}
	}

	return nil
}

// GetAnchorInfo returns anchor information for a log entry
func (s *Service) GetAnchorInfo(ctx context.Context, logIndex int64) (*AnchorInfo, error) {
	entry, err := s.transparencyRepo.GetEntryByIndex(logIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to get log entry: %w", err)
	}

	if entry == nil {
		return nil, fmt.Errorf("log entry not found")
	}

	info := &AnchorInfo{
		LogIndex:    entry.LogIndex,
		HasAnchor:   entry.AnchorTxHash != nil,
		LeafHash:    entry.MerkleLeafHash,
	}

	if entry.AnchorTxHash != nil {
		info.TxHash = *entry.AnchorTxHash
		info.BlockNumber = entry.AnchorBlockNumber
		info.Timestamp = entry.AnchorTimestamp

		// Get current status
		status, err := s.strategy.GetStatus(ctx, *entry.AnchorTxHash)
		if err == nil {
			info.Status = status.Status
			info.Confirmations = status.Confirmations
		}
	}

	return info, nil
}

// AnchorInfo contains information about an anchor
type AnchorInfo struct {
	LogIndex      int64      `json:"log_index"`
	HasAnchor     bool       `json:"has_anchor"`
	LeafHash      string     `json:"leaf_hash"`
	TxHash        string     `json:"tx_hash,omitempty"`
	BlockNumber   *int64     `json:"block_number,omitempty"`
	Timestamp     *time.Time `json:"timestamp,omitempty"`
	Status        string     `json:"status,omitempty"`
	Confirmations int        `json:"confirmations,omitempty"`
}
