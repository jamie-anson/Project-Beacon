-- Migration: Add multi-model support
-- Created: 2025-09-27
-- Description: Add model_id column to executions table for multi-model job support

-- Add model_id column to executions table
ALTER TABLE executions ADD COLUMN IF NOT EXISTS model_id VARCHAR(50) DEFAULT 'llama3.2-1b';

-- Add index for efficient model-based queries
CREATE INDEX IF NOT EXISTS idx_executions_model_id ON executions(model_id);
CREATE INDEX IF NOT EXISTS idx_executions_model_region ON executions(model_id, region);

-- Update existing executions to have default model_id
UPDATE executions SET model_id = 'llama3.2-1b' WHERE model_id IS NULL;

-- Add model_id to region_results table as well for consistency
ALTER TABLE region_results ADD COLUMN IF NOT EXISTS model_id VARCHAR(50) DEFAULT 'llama3.2-1b';
CREATE INDEX IF NOT EXISTS idx_region_results_model_id ON region_results(model_id);
CREATE INDEX IF NOT EXISTS idx_region_results_model_region ON region_results(model_id, region);

-- Update existing region_results to have default model_id
UPDATE region_results SET model_id = 'llama3.2-1b' WHERE model_id IS NULL;
