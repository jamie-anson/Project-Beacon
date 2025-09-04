-- Create transparency log table for immutable execution records
CREATE TABLE transparency_log (
    id SERIAL PRIMARY KEY,
    log_index BIGINT NOT NULL UNIQUE,
    execution_id INTEGER NOT NULL,
    job_id VARCHAR(100) NOT NULL,
    region VARCHAR(50) NOT NULL,
    provider_id VARCHAR(100) NOT NULL,
    
    -- Execution metadata
    status VARCHAR(20) NOT NULL,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    
    -- Content hashes for tamper detection
    output_hash VARCHAR(64), -- SHA-256 of output data
    receipt_hash VARCHAR(64), -- SHA-256 of receipt data
    ipfs_cid VARCHAR(100),
    
    -- Merkle tree structure
    merkle_leaf_hash VARCHAR(64) NOT NULL, -- Hash of this log entry
    merkle_tree_root VARCHAR(64), -- Root hash at time of insertion
    merkle_proof JSONB, -- Merkle proof for verification
    
    -- Timestamps and anchoring
    logged_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    anchor_tx_hash VARCHAR(66), -- Blockchain transaction hash (optional)
    anchor_block_number BIGINT, -- Block number for anchoring
    anchor_timestamp TIMESTAMP, -- When anchored to blockchain
    
    -- Integrity fields
    previous_log_hash VARCHAR(64), -- Hash of previous log entry
    signature VARCHAR(128), -- Digital signature of log entry
    
    FOREIGN KEY (execution_id) REFERENCES executions(id) ON DELETE RESTRICT
);

-- Create indexes for efficient querying
CREATE INDEX idx_transparency_log_index ON transparency_log(log_index);
CREATE INDEX idx_transparency_log_execution_id ON transparency_log(execution_id);
CREATE INDEX idx_transparency_log_job_id ON transparency_log(job_id);
CREATE INDEX idx_transparency_log_logged_at ON transparency_log(logged_at);
CREATE INDEX idx_transparency_log_merkle_root ON transparency_log(merkle_tree_root);

-- Create sequence for log_index to ensure monotonic ordering
CREATE SEQUENCE transparency_log_index_seq START 1;

-- Create function to automatically set log_index and compute hashes
CREATE OR REPLACE FUNCTION set_transparency_log_fields()
RETURNS TRIGGER AS $$
DECLARE
    prev_hash VARCHAR(64);
    entry_data TEXT;
BEGIN
    -- Set the log index
    NEW.log_index = nextval('transparency_log_index_seq');
    
    -- Get previous log entry hash for chaining
    SELECT merkle_leaf_hash INTO prev_hash 
    FROM transparency_log 
    WHERE log_index = NEW.log_index - 1;
    
    NEW.previous_log_hash = COALESCE(prev_hash, '0000000000000000000000000000000000000000000000000000000000000000');
    
    -- Create canonical representation of log entry for hashing
    entry_data = NEW.log_index || '|' || 
                 NEW.execution_id || '|' || 
                 NEW.job_id || '|' || 
                 NEW.region || '|' || 
                 NEW.provider_id || '|' || 
                 NEW.status || '|' || 
                 COALESCE(NEW.output_hash, '') || '|' || 
                 COALESCE(NEW.receipt_hash, '') || '|' || 
                 COALESCE(NEW.ipfs_cid, '') || '|' || 
                 NEW.previous_log_hash || '|' || 
                 NEW.logged_at;
    
    -- Compute merkle leaf hash (SHA-256 would be computed in application layer)
    -- For now, use a placeholder that will be updated by the application
    NEW.merkle_leaf_hash = 'PLACEHOLDER_' || NEW.log_index;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to automatically set fields on insert
CREATE TRIGGER trigger_set_transparency_log_fields
    BEFORE INSERT ON transparency_log
    FOR EACH ROW
    EXECUTE FUNCTION set_transparency_log_fields();

-- Create view for public transparency log access (excludes sensitive internal fields)
CREATE VIEW public_transparency_log AS
SELECT 
    log_index,
    job_id,
    region,
    provider_id,
    status,
    started_at,
    completed_at,
    output_hash,
    receipt_hash,
    ipfs_cid,
    merkle_leaf_hash,
    merkle_tree_root,
    logged_at,
    anchor_tx_hash,
    anchor_block_number,
    anchor_timestamp
FROM transparency_log
ORDER BY log_index;
