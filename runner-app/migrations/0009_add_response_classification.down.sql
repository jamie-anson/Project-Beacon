-- Migration rollback: Remove response classification support
-- Created: 2025-09-29

-- Drop indexes
DROP INDEX IF EXISTS idx_executions_classification;
DROP INDEX IF EXISTS idx_executions_substantive;
DROP INDEX IF EXISTS idx_executions_content_refusal;
DROP INDEX IF EXISTS idx_executions_response_length;

-- Remove columns from executions table
ALTER TABLE executions DROP COLUMN IF EXISTS is_substantive;
ALTER TABLE executions DROP COLUMN IF EXISTS is_content_refusal;
ALTER TABLE executions DROP COLUMN IF EXISTS is_technical_error;
ALTER TABLE executions DROP COLUMN IF EXISTS response_classification;
ALTER TABLE executions DROP COLUMN IF EXISTS response_length;
ALTER TABLE executions DROP COLUMN IF EXISTS system_prompt;
