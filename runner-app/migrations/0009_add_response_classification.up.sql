-- Migration: Add response classification support
-- Created: 2025-09-29
-- Description: Add response classification fields to executions table for regional prompts MVP

-- Add response classification fields to executions table
ALTER TABLE executions ADD COLUMN IF NOT EXISTS is_substantive BOOLEAN DEFAULT FALSE;
ALTER TABLE executions ADD COLUMN IF NOT EXISTS is_content_refusal BOOLEAN DEFAULT FALSE;
ALTER TABLE executions ADD COLUMN IF NOT EXISTS is_technical_error BOOLEAN DEFAULT FALSE;
ALTER TABLE executions ADD COLUMN IF NOT EXISTS response_classification VARCHAR(50) DEFAULT 'unknown';
ALTER TABLE executions ADD COLUMN IF NOT EXISTS response_length INT DEFAULT 0;
ALTER TABLE executions ADD COLUMN IF NOT EXISTS system_prompt TEXT;

-- Add indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_executions_classification ON executions(response_classification);
CREATE INDEX IF NOT EXISTS idx_executions_substantive ON executions(is_substantive);
CREATE INDEX IF NOT EXISTS idx_executions_content_refusal ON executions(is_content_refusal);
CREATE INDEX IF NOT EXISTS idx_executions_response_length ON executions(response_length);

-- Update existing records with default classification
UPDATE executions SET response_classification = 'unknown' WHERE response_classification IS NULL;
UPDATE executions SET is_substantive = FALSE WHERE is_substantive IS NULL;
UPDATE executions SET is_content_refusal = FALSE WHERE is_content_refusal IS NULL;
UPDATE executions SET is_technical_error = FALSE WHERE is_technical_error IS NULL;
UPDATE executions SET response_length = 0 WHERE response_length IS NULL;

-- Add comment for documentation
COMMENT ON COLUMN executions.is_substantive IS 'True if response is substantive (>200 chars, no refusal patterns)';
COMMENT ON COLUMN executions.is_content_refusal IS 'True if response contains content refusal patterns';
COMMENT ON COLUMN executions.is_technical_error IS 'True if execution failed due to technical error';
COMMENT ON COLUMN executions.response_classification IS 'Classification: substantive, content_refusal, technical_failure, or unknown';
COMMENT ON COLUMN executions.response_length IS 'Length of response in characters';
COMMENT ON COLUMN executions.system_prompt IS 'System prompt used for this execution (for validation)';
