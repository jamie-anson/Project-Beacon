-- Add IPFS CID storage to executions table
ALTER TABLE executions ADD COLUMN ipfs_cid VARCHAR(100);
ALTER TABLE executions ADD COLUMN ipfs_pinned_at TIMESTAMP;

-- Create index for CID lookups
CREATE INDEX idx_executions_ipfs_cid ON executions(ipfs_cid) WHERE ipfs_cid IS NOT NULL;

-- Add IPFS bundle metadata table
CREATE TABLE ipfs_bundles (
    id SERIAL PRIMARY KEY,
    job_id VARCHAR(100) NOT NULL,
    cid VARCHAR(100) NOT NULL UNIQUE,
    bundle_size BIGINT,
    execution_count INTEGER NOT NULL,
    regions TEXT[] NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    pinned_at TIMESTAMP,
    gateway_url TEXT,
    
    FOREIGN KEY (job_id) REFERENCES jobs(jobspec_id) ON DELETE CASCADE
);

-- Create indexes for bundle lookups
CREATE INDEX idx_ipfs_bundles_job_id ON ipfs_bundles(job_id);
CREATE INDEX idx_ipfs_bundles_cid ON ipfs_bundles(cid);
CREATE INDEX idx_ipfs_bundles_created_at ON ipfs_bundles(created_at);
