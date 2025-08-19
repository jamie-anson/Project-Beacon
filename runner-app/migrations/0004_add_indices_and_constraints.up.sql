-- Add indices for better query performance
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs(created_at);
CREATE INDEX IF NOT EXISTS idx_jobs_updated_at ON jobs(updated_at);

-- GIN index for JSONB queries on jobspec_data
CREATE INDEX IF NOT EXISTS idx_jobs_jobspec_data_gin ON jobs USING GIN(jobspec_data);

-- Add indices for executions table
CREATE INDEX IF NOT EXISTS idx_executions_job_id ON executions(job_id);
CREATE INDEX IF NOT EXISTS idx_executions_status ON executions(status);
CREATE INDEX IF NOT EXISTS idx_executions_region ON executions(region);
CREATE INDEX IF NOT EXISTS idx_executions_created_at ON executions(created_at);

-- Add indices for diffs table
CREATE INDEX IF NOT EXISTS idx_diffs_job_id ON diffs(job_id);
CREATE INDEX IF NOT EXISTS idx_diffs_classification ON diffs(classification);
CREATE INDEX IF NOT EXISTS idx_diffs_created_at ON diffs(created_at);

-- Add indices for outbox table
CREATE INDEX IF NOT EXISTS idx_outbox_published_at ON outbox(published_at);
CREATE INDEX IF NOT EXISTS idx_outbox_topic ON outbox(topic);
CREATE INDEX IF NOT EXISTS idx_outbox_created_at ON outbox(created_at);

-- Add status constraint to jobs table
ALTER TABLE jobs ADD CONSTRAINT chk_jobs_status 
CHECK (status IN ('created', 'queued', 'running', 'completed', 'failed', 'cancelled'));

-- Add status constraint to executions table
ALTER TABLE executions ADD CONSTRAINT chk_executions_status 
CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled'));

-- Add classification constraint to diffs table
ALTER TABLE diffs ADD CONSTRAINT chk_diffs_classification 
CHECK (classification IN ('identical', 'minor', 'moderate', 'major', 'critical'));
