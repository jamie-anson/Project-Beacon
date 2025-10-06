-- Rollback migration: Remove retry tracking columns

DROP INDEX IF EXISTS idx_executions_status_retry;
DROP INDEX IF EXISTS idx_executions_retry_count;

ALTER TABLE executions DROP COLUMN IF EXISTS original_error;
ALTER TABLE executions DROP COLUMN IF EXISTS retry_history;
ALTER TABLE executions DROP COLUMN IF EXISTS last_retry_at;
ALTER TABLE executions DROP COLUMN IF EXISTS max_retries;
ALTER TABLE executions DROP COLUMN IF EXISTS retry_count;
