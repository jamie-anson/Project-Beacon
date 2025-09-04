-- Remove constraints
ALTER TABLE diffs DROP CONSTRAINT IF EXISTS chk_diffs_classification;
ALTER TABLE executions DROP CONSTRAINT IF EXISTS chk_executions_status;
ALTER TABLE jobs DROP CONSTRAINT IF EXISTS chk_jobs_status;

-- Remove indices for outbox table
DROP INDEX IF EXISTS idx_outbox_created_at;
DROP INDEX IF EXISTS idx_outbox_topic;
DROP INDEX IF EXISTS idx_outbox_published_at;

-- Remove indices for diffs table
DROP INDEX IF EXISTS idx_diffs_created_at;
DROP INDEX IF EXISTS idx_diffs_classification;
DROP INDEX IF EXISTS idx_diffs_job_id;

-- Remove indices for executions table
DROP INDEX IF EXISTS idx_executions_created_at;
DROP INDEX IF EXISTS idx_executions_region;
DROP INDEX IF EXISTS idx_executions_status;
DROP INDEX IF EXISTS idx_executions_job_id;

-- Remove indices for jobs table
DROP INDEX IF EXISTS idx_jobs_jobspec_data_gin;
DROP INDEX IF EXISTS idx_jobs_updated_at;
DROP INDEX IF EXISTS idx_jobs_created_at;
DROP INDEX IF EXISTS idx_jobs_status;
