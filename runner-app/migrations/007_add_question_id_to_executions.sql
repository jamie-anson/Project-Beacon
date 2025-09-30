-- Migration: Add question_id column to executions table for per-question execution tracking
-- Date: 2025-09-30
-- Purpose: Enable granular tracking of individual question responses for bias detection

-- Add question_id column to executions table
ALTER TABLE executions 
ADD COLUMN IF NOT EXISTS question_id VARCHAR(255);

-- Create composite index for deduplication (includes question_id)
-- This index is used by the auto-stop check to prevent duplicate executions
CREATE INDEX IF NOT EXISTS idx_executions_dedup_with_question 
ON executions(job_id, region, model_id, question_id);

-- Add comment for documentation
COMMENT ON COLUMN executions.question_id IS 'Question ID for per-question execution tracking. NULL or empty for legacy batch executions.';

-- Note: Existing executions will have NULL question_id (backward compatible)
-- New executions will populate this field when questions are provided in the job spec
