-- Migration: Add retry tracking for failed questions
-- Created: 2025-10-06
-- Description: Add retry attempt tracking to executions table for handling cold start timeouts

-- Add retry tracking columns to executions table
ALTER TABLE executions ADD COLUMN IF NOT EXISTS retry_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE executions ADD COLUMN IF NOT EXISTS max_retries INTEGER NOT NULL DEFAULT 3;
ALTER TABLE executions ADD COLUMN IF NOT EXISTS last_retry_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE executions ADD COLUMN IF NOT EXISTS retry_history JSONB DEFAULT '[]'::jsonb;
ALTER TABLE executions ADD COLUMN IF NOT EXISTS original_error TEXT;

-- Add index for querying retryable executions
CREATE INDEX IF NOT EXISTS idx_executions_retry_count ON executions(retry_count);
CREATE INDEX IF NOT EXISTS idx_executions_status_retry ON executions(status, retry_count) WHERE status IN ('failed', 'timeout', 'error');

-- Add comment for documentation
COMMENT ON COLUMN executions.retry_count IS 'Number of retry attempts for this execution';
COMMENT ON COLUMN executions.max_retries IS 'Maximum number of retries allowed (default 3)';
COMMENT ON COLUMN executions.last_retry_at IS 'Timestamp of the last retry attempt';
COMMENT ON COLUMN executions.retry_history IS 'JSON array of retry attempt details [{attempt: 1, timestamp: "...", status: "...", error: "..."}]';
COMMENT ON COLUMN executions.original_error IS 'Original error message before retries';
