-- Drop indexes created in 0005 up migration
DROP INDEX IF EXISTS idx_executions_job_created_desc;
DROP INDEX IF EXISTS idx_executions_job_status_created;
DROP INDEX IF EXISTS idx_jobs_status_created;
