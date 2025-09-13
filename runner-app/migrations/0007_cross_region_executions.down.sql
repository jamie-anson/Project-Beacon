-- Migration: Remove cross-region execution support
-- Created: 2025-09-13
-- Description: Drop tables for multi-region job execution and cross-region analysis

-- Drop triggers first
DROP TRIGGER IF EXISTS update_cross_region_analyses_updated_at ON cross_region_analyses;
DROP TRIGGER IF EXISTS update_region_results_updated_at ON region_results;
DROP TRIGGER IF EXISTS update_cross_region_executions_updated_at ON cross_region_executions;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_executions_cross_region_execution_id;
DROP INDEX IF EXISTS idx_cross_region_analyses_cross_region_execution_id;
DROP INDEX IF EXISTS idx_region_results_provider_id;
DROP INDEX IF EXISTS idx_region_results_status;
DROP INDEX IF EXISTS idx_region_results_region;
DROP INDEX IF EXISTS idx_region_results_cross_region_execution_id;
DROP INDEX IF EXISTS idx_cross_region_executions_created_at;
DROP INDEX IF EXISTS idx_cross_region_executions_status;
DROP INDEX IF EXISTS idx_cross_region_executions_jobspec_id;

-- Remove foreign key column
ALTER TABLE executions DROP COLUMN IF EXISTS cross_region_execution_id;

-- Drop tables
DROP TABLE IF EXISTS cross_region_analyses;
DROP TABLE IF EXISTS region_results;
DROP TABLE IF EXISTS cross_region_executions;
