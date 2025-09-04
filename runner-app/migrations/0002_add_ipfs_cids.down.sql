-- Remove IPFS bundle metadata table
DROP TABLE IF EXISTS ipfs_bundles;

-- Remove IPFS columns from executions table
ALTER TABLE executions DROP COLUMN IF EXISTS ipfs_cid;
ALTER TABLE executions DROP COLUMN IF EXISTS ipfs_pinned_at;
