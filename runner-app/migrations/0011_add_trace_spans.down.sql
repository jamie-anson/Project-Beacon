-- Migration rollback: Remove distributed tracing support
-- Created: 2025-10-22

-- Drop views
DROP VIEW IF EXISTS trace_spans_health;
DROP VIEW IF EXISTS trace_waterfall;

-- Drop functions
DROP FUNCTION IF EXISTS find_similar_traces(BIGINT, INTEGER);
DROP FUNCTION IF EXISTS identify_root_cause(BIGINT);
DROP FUNCTION IF EXISTS diagnose_execution_trace(BIGINT);

-- Drop indexes (will be dropped automatically with table, but explicit for clarity)
DROP INDEX IF EXISTS idx_trace_spans_created_at;
DROP INDEX IF EXISTS idx_trace_spans_service_operation;
DROP INDEX IF EXISTS idx_trace_spans_started_at;
DROP INDEX IF EXISTS idx_trace_spans_execution_id;
DROP INDEX IF EXISTS idx_trace_spans_job_id;
DROP INDEX IF EXISTS idx_trace_spans_trace_id;

-- Drop table
DROP TABLE IF EXISTS trace_spans;
