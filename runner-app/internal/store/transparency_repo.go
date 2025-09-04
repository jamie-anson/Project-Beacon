package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/merkle"
)

// TransparencyLogEntry represents a single entry in the transparency log
type TransparencyLogEntry struct {
	ID                int64     `json:"id"`
	LogIndex          int64     `json:"log_index"`
	ExecutionID       int       `json:"execution_id"`
	JobID             string    `json:"job_id"`
	Region            string    `json:"region"`
	ProviderID        string    `json:"provider_id"`
	Status            string    `json:"status"`
	StartedAt         *time.Time `json:"started_at,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	OutputHash        *string   `json:"output_hash,omitempty"`
	ReceiptHash       *string   `json:"receipt_hash,omitempty"`
	IPFSCid           *string   `json:"ipfs_cid,omitempty"`
	MerkleLeafHash    string    `json:"merkle_leaf_hash"`
	MerkleTreeRoot    *string   `json:"merkle_tree_root,omitempty"`
	MerkleProof       *merkle.Proof `json:"merkle_proof,omitempty"`
	LoggedAt          time.Time `json:"logged_at"`
	AnchorTxHash      *string   `json:"anchor_tx_hash,omitempty"`
	AnchorBlockNumber *int64    `json:"anchor_block_number,omitempty"`
	AnchorTimestamp   *time.Time `json:"anchor_timestamp,omitempty"`
	PreviousLogHash   string    `json:"previous_log_hash"`
	Signature         *string   `json:"signature,omitempty"`
}

// TransparencyRepo handles transparency log database operations
type TransparencyRepo struct {
	db *sql.DB
}

// NewTransparencyRepo creates a new transparency repository
func NewTransparencyRepo(db *sql.DB) *TransparencyRepo {
	return &TransparencyRepo{db: db}
}

// AppendEntry adds a new entry to the transparency log
func (r *TransparencyRepo) AppendEntry(entry *TransparencyLogEntry) error {
	// Compute the merkle leaf hash
	entry.MerkleLeafHash = merkle.ComputeLeafHash(
		entry.LogIndex,
		entry.ExecutionID,
		entry.JobID,
		entry.Region,
		entry.ProviderID,
		entry.Status,
		stringOrEmpty(entry.OutputHash),
		stringOrEmpty(entry.ReceiptHash),
		stringOrEmpty(entry.IPFSCid),
		entry.PreviousLogHash,
		entry.LoggedAt.Format(time.RFC3339),
	)

	// Serialize merkle proof if present
	var merkleProofJSON []byte
	if entry.MerkleProof != nil {
		var err error
		merkleProofJSON, err = entry.MerkleProof.SerializeProof()
		if err != nil {
			return fmt.Errorf("failed to serialize merkle proof: %w", err)
		}
	}

	query := `
		INSERT INTO transparency_log (
			execution_id, job_id, region, provider_id, status,
			started_at, completed_at, output_hash, receipt_hash, ipfs_cid,
			merkle_leaf_hash, merkle_tree_root, merkle_proof,
			anchor_tx_hash, anchor_block_number, anchor_timestamp, signature
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id, log_index, logged_at, previous_log_hash`

	err := r.db.QueryRow(
		query,
		entry.ExecutionID,
		entry.JobID,
		entry.Region,
		entry.ProviderID,
		entry.Status,
		entry.StartedAt,
		entry.CompletedAt,
		entry.OutputHash,
		entry.ReceiptHash,
		entry.IPFSCid,
		entry.MerkleLeafHash,
		entry.MerkleTreeRoot,
		merkleProofJSON,
		entry.AnchorTxHash,
		entry.AnchorBlockNumber,
		entry.AnchorTimestamp,
		entry.Signature,
	).Scan(&entry.ID, &entry.LogIndex, &entry.LoggedAt, &entry.PreviousLogHash)

	return err
}

// GetEntryByIndex retrieves a transparency log entry by its log index
func (r *TransparencyRepo) GetEntryByIndex(logIndex int64) (*TransparencyLogEntry, error) {
	query := `
		SELECT id, log_index, execution_id, job_id, region, provider_id, status,
		       started_at, completed_at, output_hash, receipt_hash, ipfs_cid,
		       merkle_leaf_hash, merkle_tree_root, merkle_proof, logged_at,
		       anchor_tx_hash, anchor_block_number, anchor_timestamp,
		       previous_log_hash, signature
		FROM transparency_log
		WHERE log_index = $1`

	entry := &TransparencyLogEntry{}
	var merkleProofJSON []byte

	err := r.db.QueryRow(query, logIndex).Scan(
		&entry.ID,
		&entry.LogIndex,
		&entry.ExecutionID,
		&entry.JobID,
		&entry.Region,
		&entry.ProviderID,
		&entry.Status,
		&entry.StartedAt,
		&entry.CompletedAt,
		&entry.OutputHash,
		&entry.ReceiptHash,
		&entry.IPFSCid,
		&entry.MerkleLeafHash,
		&entry.MerkleTreeRoot,
		&merkleProofJSON,
		&entry.LoggedAt,
		&entry.AnchorTxHash,
		&entry.AnchorBlockNumber,
		&entry.AnchorTimestamp,
		&entry.PreviousLogHash,
		&entry.Signature,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Deserialize merkle proof if present
	if len(merkleProofJSON) > 0 {
		proof, err := merkle.DeserializeProof(merkleProofJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize merkle proof: %w", err)
		}
		entry.MerkleProof = proof
	}

	return entry, nil
}

// GetEntriesByJobID retrieves all transparency log entries for a specific job
func (r *TransparencyRepo) GetEntriesByJobID(jobID string) ([]TransparencyLogEntry, error) {
	query := `
		SELECT id, log_index, execution_id, job_id, region, provider_id, status,
		       started_at, completed_at, output_hash, receipt_hash, ipfs_cid,
		       merkle_leaf_hash, merkle_tree_root, merkle_proof, logged_at,
		       anchor_tx_hash, anchor_block_number, anchor_timestamp,
		       previous_log_hash, signature
		FROM transparency_log
		WHERE job_id = $1
		ORDER BY log_index ASC`

	rows, err := r.db.Query(query, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []TransparencyLogEntry
	for rows.Next() {
		var entry TransparencyLogEntry
		var merkleProofJSON []byte

		err := rows.Scan(
			&entry.ID,
			&entry.LogIndex,
			&entry.ExecutionID,
			&entry.JobID,
			&entry.Region,
			&entry.ProviderID,
			&entry.Status,
			&entry.StartedAt,
			&entry.CompletedAt,
			&entry.OutputHash,
			&entry.ReceiptHash,
			&entry.IPFSCid,
			&entry.MerkleLeafHash,
			&entry.MerkleTreeRoot,
			&merkleProofJSON,
			&entry.LoggedAt,
			&entry.AnchorTxHash,
			&entry.AnchorBlockNumber,
			&entry.AnchorTimestamp,
			&entry.PreviousLogHash,
			&entry.Signature,
		)
		if err != nil {
			return nil, err
		}

		// Deserialize merkle proof if present
		if len(merkleProofJSON) > 0 {
			proof, err := merkle.DeserializeProof(merkleProofJSON)
			if err != nil {
				return nil, fmt.Errorf("failed to deserialize merkle proof: %w", err)
			}
			entry.MerkleProof = proof
		}

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// ListEntries retrieves transparency log entries with pagination
func (r *TransparencyRepo) ListEntries(limit, offset int) ([]TransparencyLogEntry, error) {
	query := `
		SELECT id, log_index, execution_id, job_id, region, provider_id, status,
		       started_at, completed_at, output_hash, receipt_hash, ipfs_cid,
		       merkle_leaf_hash, merkle_tree_root, merkle_proof, logged_at,
		       anchor_tx_hash, anchor_block_number, anchor_timestamp,
		       previous_log_hash, signature
		FROM transparency_log
		ORDER BY log_index ASC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []TransparencyLogEntry
	for rows.Next() {
		var entry TransparencyLogEntry
		var merkleProofJSON []byte

		err := rows.Scan(
			&entry.ID,
			&entry.LogIndex,
			&entry.ExecutionID,
			&entry.JobID,
			&entry.Region,
			&entry.ProviderID,
			&entry.Status,
			&entry.StartedAt,
			&entry.CompletedAt,
			&entry.OutputHash,
			&entry.ReceiptHash,
			&entry.IPFSCid,
			&entry.MerkleLeafHash,
			&entry.MerkleTreeRoot,
			&merkleProofJSON,
			&entry.LoggedAt,
			&entry.AnchorTxHash,
			&entry.AnchorBlockNumber,
			&entry.AnchorTimestamp,
			&entry.PreviousLogHash,
			&entry.Signature,
		)
		if err != nil {
			return nil, err
		}

		// Deserialize merkle proof if present
		if len(merkleProofJSON) > 0 {
			proof, err := merkle.DeserializeProof(merkleProofJSON)
			if err != nil {
				return nil, fmt.Errorf("failed to deserialize merkle proof: %w", err)
			}
			entry.MerkleProof = proof
		}

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// GetLatestEntry retrieves the most recent transparency log entry
func (r *TransparencyRepo) GetLatestEntry() (*TransparencyLogEntry, error) {
	query := `
		SELECT id, log_index, execution_id, job_id, region, provider_id, status,
		       started_at, completed_at, output_hash, receipt_hash, ipfs_cid,
		       merkle_leaf_hash, merkle_tree_root, merkle_proof, logged_at,
		       anchor_tx_hash, anchor_block_number, anchor_timestamp,
		       previous_log_hash, signature
		FROM transparency_log
		ORDER BY log_index DESC
		LIMIT 1`

	entry := &TransparencyLogEntry{}
	var merkleProofJSON []byte

	err := r.db.QueryRow(query).Scan(
		&entry.ID,
		&entry.LogIndex,
		&entry.ExecutionID,
		&entry.JobID,
		&entry.Region,
		&entry.ProviderID,
		&entry.Status,
		&entry.StartedAt,
		&entry.CompletedAt,
		&entry.OutputHash,
		&entry.ReceiptHash,
		&entry.IPFSCid,
		&entry.MerkleLeafHash,
		&entry.MerkleTreeRoot,
		&merkleProofJSON,
		&entry.LoggedAt,
		&entry.AnchorTxHash,
		&entry.AnchorBlockNumber,
		&entry.AnchorTimestamp,
		&entry.PreviousLogHash,
		&entry.Signature,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Deserialize merkle proof if present
	if len(merkleProofJSON) > 0 {
		proof, err := merkle.DeserializeProof(merkleProofJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize merkle proof: %w", err)
		}
		entry.MerkleProof = proof
	}

	return entry, nil
}

// UpdateMerkleRoot updates the merkle tree root for entries in a specific range
func (r *TransparencyRepo) UpdateMerkleRoot(startIndex, endIndex int64, rootHash string) error {
	query := `
		UPDATE transparency_log 
		SET merkle_tree_root = $1
		WHERE log_index >= $2 AND log_index <= $3`

	_, err := r.db.Exec(query, rootHash, startIndex, endIndex)
	return err
}

// GetLogSize returns the total number of entries in the transparency log
func (r *TransparencyRepo) GetLogSize() (int64, error) {
	var count int64
	err := r.db.QueryRow("SELECT COUNT(*) FROM transparency_log").Scan(&count)
	return count, err
}

// VerifyLogIntegrity checks the integrity of the transparency log chain
func (r *TransparencyRepo) VerifyLogIntegrity() (bool, error) {
	query := `
		SELECT log_index, merkle_leaf_hash, previous_log_hash
		FROM transparency_log
		ORDER BY log_index ASC`

	rows, err := r.db.Query(query)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	var previousHash string = "0000000000000000000000000000000000000000000000000000000000000000"
	
	for rows.Next() {
		var logIndex int64
		var leafHash, prevHash string
		
		err := rows.Scan(&logIndex, &leafHash, &prevHash)
		if err != nil {
			return false, err
		}
		
		// Verify chain integrity
		if prevHash != previousHash {
			return false, nil
		}
		
		previousHash = leafHash
	}
	
	return true, rows.Err()
}

// stringOrEmpty returns the string value or empty string if nil
func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
