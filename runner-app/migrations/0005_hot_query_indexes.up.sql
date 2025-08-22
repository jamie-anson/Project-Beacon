-- Optimize hot queries with composite and partial indexes
-- Executions: latest receipt lookups per job
CREATE INDEX IF NOT EXISTS idx_executions_job_created_desc
ON executions(job_id, created_at DESC);

-- Executions: filter by job and status quickly (e.g., completed)
CREATE INDEX IF NOT EXISTS idx_executions_job_status_created
ON executions(job_id, status, created_at DESC);

-- Receipts table (if receipts are separate); if not present, this is a no-op in down migration
-- Jobs: status and created_at already indexed in 0004
-- Add covering index for recent jobs by status then created_at
CREATE INDEX IF NOT EXISTS idx_jobs_status_created
ON jobs(status, created_at DESC);
