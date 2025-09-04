package store

import (
	"github.com/lib/pq"
)

// GetBundlesByJobID retrieves all IPFS bundles for a specific job ID
func (r *IPFSRepo) GetBundlesByJobID(jobID string) ([]IPFSBundle, error) {
	query := `
		SELECT id, job_id, cid, bundle_size, execution_count, regions, 
		       created_at, pinned_at, gateway_url
		FROM ipfs_bundles 
		WHERE job_id = $1
		ORDER BY created_at DESC
	`
	
	rows, err := r.db.Query(query, jobID)
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
