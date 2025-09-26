-- Migration rollback: Remove multi-model support
-- Created: 2025-09-27
-- Description: Remove model_id columns and indexes

-- Remove indexes
DROP INDEX IF EXISTS idx_executions_model_id;
DROP INDEX IF EXISTS idx_executions_model_region;
DROP INDEX IF EXISTS idx_region_results_model_id;
DROP INDEX IF EXISTS idx_region_results_model_region;

-- Remove columns
ALTER TABLE executions DROP COLUMN IF EXISTS model_id;
ALTER TABLE region_results DROP COLUMN IF EXISTS model_id;
