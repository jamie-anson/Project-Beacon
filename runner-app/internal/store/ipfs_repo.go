package store

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
)

// Execution represents an execution record with IPFS fields
type Execution struct {
	ID           int            `json:"id"`
	JobID        int            `json:"job_id"`
	Region       string         `json:"region"`
	ProviderID   string         `json:"provider_id"`
	Status       string         `json:"status"`
	StartedAt    sql.NullTime   `json:"started_at"`
	CompletedAt  sql.NullTime   `json:"completed_at"`
	CreatedAt    time.Time      `json:"created_at"`
	OutputData   sql.NullString `json:"output_data"`
	ReceiptData  sql.NullString `json:"receipt_data"`
	IPFSCid      sql.NullString `json:"ipfs_cid"`
	IPFSPinnedAt sql.NullTime   `json:"ipfs_pinned_at"`
}

// UpdateExecutionCIDByID updates an execution's CID by its primary key ID
func (r *IPFSRepo) UpdateExecutionCIDByID(executionID int, cid string) error {
    query := `
        UPDATE executions 
        SET ipfs_cid = $1, ipfs_pinned_at = CURRENT_TIMESTAMP
        WHERE id = $2`

    _, err := r.db.Exec(query, cid, executionID)
    return err
}

// IPFSBundle represents an IPFS bundle record
type IPFSBundle struct {
	ID             int       `json:"id"`
	JobID          string    `json:"job_id"`
	CID            string    `json:"cid"`
	BundleSize     *int64    `json:"bundle_size,omitempty"`
	ExecutionCount int       `json:"execution_count"`
	Regions        []string  `json:"regions"`
	CreatedAt      time.Time `json:"created_at"`
	PinnedAt       *time.Time `json:"pinned_at,omitempty"`
	GatewayURL     *string   `json:"gateway_url,omitempty"`
}

// IPFSRepo handles IPFS bundle database operations
type IPFSRepo struct {
	db *sql.DB
}

// NewIPFSRepo creates a new IPFS repository
func NewIPFSRepo(db *sql.DB) *IPFSRepo {
	return &IPFSRepo{db: db}
}

// CreateBundle stores a new IPFS bundle record
func (r *IPFSRepo) CreateBundle(bundle *IPFSBundle) error {
	query := `
		INSERT INTO ipfs_bundles (job_id, cid, bundle_size, execution_count, regions, pinned_at, gateway_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`

	err := r.db.QueryRow(
		query,
		bundle.JobID,
		bundle.CID,
		bundle.BundleSize,
		bundle.ExecutionCount,
		pq.Array(bundle.Regions),
		bundle.PinnedAt,
		bundle.GatewayURL,
	).Scan(&bundle.ID, &bundle.CreatedAt)

	return err
}

// GetBundleByJobID retrieves an IPFS bundle by job ID
func (r *IPFSRepo) GetBundleByJobID(jobID string) (*IPFSBundle, error) {
	query := `
		SELECT id, job_id, cid, bundle_size, execution_count, regions, created_at, pinned_at, gateway_url
		FROM ipfs_bundles 
		WHERE job_id = $1
		ORDER BY created_at DESC
		LIMIT 1`

	bundle := &IPFSBundle{}
	err := r.db.QueryRow(query, jobID).Scan(
		&bundle.ID,
		&bundle.JobID,
		&bundle.CID,
		&bundle.BundleSize,
		&bundle.ExecutionCount,
		pq.Array(&bundle.Regions),
		&bundle.CreatedAt,
		&bundle.PinnedAt,
		&bundle.GatewayURL,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return bundle, err
}

// GetBundleByCID retrieves an IPFS bundle by CID
func (r *IPFSRepo) GetBundleByCID(cid string) (*IPFSBundle, error) {
	query := `
		SELECT id, job_id, cid, bundle_size, execution_count, regions, created_at, pinned_at, gateway_url
		FROM ipfs_bundles 
		WHERE cid = $1`

	bundle := &IPFSBundle{}
	err := r.db.QueryRow(query, cid).Scan(
		&bundle.ID,
		&bundle.JobID,
		&bundle.CID,
		&bundle.BundleSize,
		&bundle.ExecutionCount,
		pq.Array(&bundle.Regions),
		&bundle.CreatedAt,
		&bundle.PinnedAt,
		&bundle.GatewayURL,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return bundle, err
}

// ListBundles retrieves all IPFS bundles with pagination
func (r *IPFSRepo) ListBundles(limit, offset int) ([]IPFSBundle, error) {
	query := `
		SELECT id, job_id, cid, bundle_size, execution_count, regions, created_at, pinned_at, gateway_url
		FROM ipfs_bundles 
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bundles []IPFSBundle
	for rows.Next() {
		var bundle IPFSBundle
		err := rows.Scan(
			&bundle.ID,
			&bundle.JobID,
			&bundle.CID,
			&bundle.BundleSize,
			&bundle.ExecutionCount,
			pq.Array(&bundle.Regions),
			&bundle.CreatedAt,
			&bundle.PinnedAt,
			&bundle.GatewayURL,
		)
		if err != nil {
			return nil, err
		}
		bundles = append(bundles, bundle)
	}

	return bundles, rows.Err()
}

// UpdateExecutionCID updates an execution with its IPFS CID
func (r *IPFSRepo) UpdateExecutionCID(executionID, cid string) error {
	query := `
		UPDATE executions 
		SET ipfs_cid = $1, ipfs_pinned_at = CURRENT_TIMESTAMP
		WHERE id = $2`

	_, err := r.db.Exec(query, cid, executionID)
	return err
}

// GetExecutionsByJobSpecID retrieves all executions for a specific JobSpec ID
func (r *IPFSRepo) GetExecutionsByJobSpecID(jobspecID string) ([]Execution, error) {
	query := `
		SELECT e.id, e.job_id, e.region, e.provider_id, e.status, e.started_at, e.completed_at,
		       e.created_at, e.output_data, e.receipt_data, e.ipfs_cid, e.ipfs_pinned_at
		FROM executions e
		JOIN jobs j ON e.job_id = j.id
		WHERE j.jobspec_id = $1
		ORDER BY e.created_at DESC
	`

	rows, err := r.db.Query(query, jobspecID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var executions []Execution
	for rows.Next() {
		var exec Execution
		err := rows.Scan(
			&exec.ID,
			&exec.JobID,
			&exec.Region,
			&exec.ProviderID,
			&exec.Status,
			&exec.StartedAt,
			&exec.CompletedAt,
			&exec.CreatedAt,
			&exec.OutputData,
			&exec.ReceiptData,
			&exec.IPFSCid,
			&exec.IPFSPinnedAt,
		)
		if err != nil {
			return nil, err
		}
		executions = append(executions, exec)
	}

	return executions, rows.Err()
}

// GetExecutionsByJobID retrieves all executions for a specific job ID
func (r *IPFSRepo) GetExecutionsByJobID(jobID string) ([]Execution, error) {
	query := `
		SELECT id, job_id, region, provider_id, status, started_at, completed_at, 
		       created_at, output_data, receipt_data, ipfs_cid, ipfs_pinned_at
		FROM executions 
		WHERE job_id = $1
		ORDER BY created_at DESC
	`
	
	rows, err := r.db.Query(query, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var executions []Execution
	for rows.Next() {
		var exec Execution
		err := rows.Scan(
			&exec.ID,
			&exec.JobID,
			&exec.Region,
			&exec.ProviderID,
			&exec.Status,
			&exec.StartedAt,
			&exec.CompletedAt,
			&exec.CreatedAt,
			&exec.OutputData,
			&exec.ReceiptData,
			&exec.IPFSCid,
			&exec.IPFSPinnedAt,
		)
		if err != nil {
			return nil, err
		}
		executions = append(executions, exec)
	}
	
	return executions, rows.Err()
}

// GetExecutionsByCID retrieves executions associated with an IPFS CID
func (r *IPFSRepo) GetExecutionsByCID(cid string) ([]Execution, error) {
	query := `
		SELECT id, job_id, region, provider_id, status, started_at, completed_at, 
		       created_at, output_data, receipt_data, ipfs_cid, ipfs_pinned_at
		FROM executions 
		WHERE ipfs_cid = $1
		ORDER BY created_at DESC
	`
	
	rows, err := r.db.Query(query, cid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var executions []Execution
	for rows.Next() {
		var exec Execution
		err := rows.Scan(
			&exec.ID,
			&exec.JobID,
			&exec.Region,
			&exec.ProviderID,
			&exec.Status,
			&exec.StartedAt,
			&exec.CompletedAt,
			&exec.CreatedAt,
			&exec.OutputData,
			&exec.ReceiptData,
			&exec.IPFSCid,
			&exec.IPFSPinnedAt,
		)
		if err != nil {
			return nil, err
		}
		executions = append(executions, exec)
	}
	
	return executions, rows.Err()
}
