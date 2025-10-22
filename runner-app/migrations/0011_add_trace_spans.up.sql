-- Migration: Add distributed tracing support
-- Created: 2025-10-22
-- Description: Create trace_spans table and diagnostic functions for distributed tracing

-- Main tracing table
CREATE TABLE IF NOT EXISTS trace_spans (
    id BIGSERIAL PRIMARY KEY,
    trace_id UUID NOT NULL,
    span_id UUID NOT NULL,
    parent_span_id UUID,
    
    -- Service identification
    service VARCHAR(50) NOT NULL,  -- 'runner', 'router', 'modal', 'backend'
    operation VARCHAR(100) NOT NULL, -- 'execute_job', 'http_request', 'inference'
    
    -- Timing
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    duration_ms INTEGER,
    
    -- Status
    status VARCHAR(20) NOT NULL, -- 'started', 'completed', 'failed', 'timeout'
    
    -- Context (link to existing tables)
    job_id VARCHAR(255),
    execution_id BIGINT,
    model_id VARCHAR(100),
    region VARCHAR(50),
    
    -- Metadata (flexible JSON for service-specific data)
    metadata JSONB,
    
    -- Error tracking
    error_message TEXT,
    error_type VARCHAR(100),
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for fast queries
CREATE INDEX IF NOT EXISTS idx_trace_spans_trace_id ON trace_spans(trace_id);
CREATE INDEX IF NOT EXISTS idx_trace_spans_job_id ON trace_spans(job_id);
CREATE INDEX IF NOT EXISTS idx_trace_spans_execution_id ON trace_spans(execution_id);
CREATE INDEX IF NOT EXISTS idx_trace_spans_started_at ON trace_spans(started_at);
CREATE INDEX IF NOT EXISTS idx_trace_spans_service_operation ON trace_spans(service, operation);
CREATE INDEX IF NOT EXISTS idx_trace_spans_created_at ON trace_spans(created_at);

-- View for easy trace reconstruction
CREATE OR REPLACE VIEW trace_waterfall AS
SELECT 
    trace_id,
    span_id,
    parent_span_id,
    service,
    operation,
    started_at,
    completed_at,
    duration_ms,
    status,
    job_id,
    execution_id,
    model_id,
    region,
    metadata,
    error_message
FROM trace_spans
ORDER BY trace_id, started_at;

-- Diagnostic function: Enhanced trace query with automatic anomaly detection
CREATE OR REPLACE FUNCTION diagnose_execution_trace(p_execution_id BIGINT)
RETURNS TABLE (
    service VARCHAR,
    operation VARCHAR,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    duration_ms INTEGER,
    gap_after_ms NUMERIC,
    status VARCHAR,
    error_message TEXT,
    is_anomaly BOOLEAN,
    anomaly_reason TEXT
) AS $$
WITH trace_data AS (
    SELECT 
        ts.service,
        ts.operation,
        ts.started_at,
        ts.completed_at,
        ts.duration_ms,
        ts.status,
        ts.error_message,
        LEAD(ts.started_at) OVER (ORDER BY ts.started_at) as next_started_at,
        AVG(ts.duration_ms) OVER (PARTITION BY ts.service, ts.operation) as avg_duration
    FROM trace_spans ts
    WHERE ts.execution_id = p_execution_id
)
SELECT 
    service,
    operation,
    started_at,
    completed_at,
    duration_ms,
    EXTRACT(EPOCH FROM (next_started_at - completed_at)) * 1000 as gap_after_ms,
    status,
    error_message,
    -- Flag anomalies
    CASE 
        WHEN duration_ms > avg_duration * 3 THEN TRUE
        WHEN EXTRACT(EPOCH FROM (next_started_at - completed_at)) * 1000 > 5000 THEN TRUE
        WHEN status IN ('failed', 'timeout') THEN TRUE
        ELSE FALSE
    END as is_anomaly,
    -- Explain anomaly
    CASE 
        WHEN duration_ms > avg_duration * 3 THEN 
            'Operation took ' || duration_ms || 'ms (3x longer than average ' || ROUND(avg_duration) || 'ms)'
        WHEN EXTRACT(EPOCH FROM (next_started_at - completed_at)) * 1000 > 5000 THEN
            'Gap of ' || ROUND(EXTRACT(EPOCH FROM (next_started_at - completed_at)) * 1000) || 'ms before next operation'
        WHEN status IN ('failed', 'timeout') THEN
            'Operation failed: ' || COALESCE(error_message, 'Unknown error')
        ELSE NULL
    END as anomaly_reason
FROM trace_data
ORDER BY started_at;
$$ LANGUAGE sql;

-- Diagnostic function: Automatically identify root cause of failures
CREATE OR REPLACE FUNCTION identify_root_cause(p_execution_id BIGINT)
RETURNS TABLE (
    root_cause_type VARCHAR,
    affected_service VARCHAR,
    affected_operation VARCHAR,
    evidence TEXT,
    recommendation TEXT
) AS $$
WITH trace_analysis AS (
    SELECT 
        ts.service,
        ts.operation,
        ts.status,
        ts.duration_ms,
        ts.error_message,
        ts.started_at,
        ts.completed_at,
        LEAD(ts.started_at) OVER (ORDER BY ts.started_at) as next_started_at
    FROM trace_spans ts
    WHERE ts.execution_id = p_execution_id
)
SELECT 
    CASE 
        WHEN EXTRACT(EPOCH FROM (next_started_at - completed_at)) > 60 THEN 'NETWORK_TIMEOUT'
        WHEN status = 'failed' AND error_message LIKE '%connection%' THEN 'CONNECTION_FAILURE'
        WHEN status = 'failed' AND error_message LIKE '%timeout%' THEN 'SERVICE_TIMEOUT'
        WHEN duration_ms > 120000 THEN 'PERFORMANCE_DEGRADATION'
        WHEN service = 'modal' AND status = 'failed' THEN 'MODAL_EXECUTION_FAILURE'
        ELSE 'UNKNOWN'
    END as root_cause_type,
    service as affected_service,
    operation as affected_operation,
    CASE 
        WHEN EXTRACT(EPOCH FROM (next_started_at - completed_at)) > 60 THEN
            'Service ' || service || ' completed at ' || completed_at || 
            ' but next service started at ' || next_started_at || 
            ' (gap: ' || ROUND(EXTRACT(EPOCH FROM (next_started_at - completed_at))) || 's)'
        WHEN status = 'failed' THEN
            'Service ' || service || ' failed with: ' || COALESCE(error_message, 'No error message')
        WHEN duration_ms > 120000 THEN
            'Operation took ' || duration_ms || 'ms (>2 minutes)'
        ELSE 'See trace_spans for details'
    END as evidence,
    CASE 
        WHEN EXTRACT(EPOCH FROM (next_started_at - completed_at)) > 60 THEN
            'Check network connectivity between ' || service || ' and next service. Review firewall rules and timeout configurations.'
        WHEN status = 'failed' AND error_message LIKE '%connection%' THEN
            'Verify service health and network connectivity. Check if service is running and accessible.'
        WHEN service = 'modal' AND status = 'failed' THEN
            'Check Modal dashboard for function logs. Verify GPU availability and model loading.'
        WHEN duration_ms > 120000 THEN
            'Investigate performance bottleneck in ' || service || '. Check resource utilization and scaling.'
        ELSE 'Review detailed trace spans for more information'
    END as recommendation
FROM trace_analysis
WHERE status = 'failed' 
   OR EXTRACT(EPOCH FROM (next_started_at - completed_at)) > 60
   OR duration_ms > 120000
ORDER BY started_at
LIMIT 1;
$$ LANGUAGE sql;

-- Diagnostic function: Find similar execution patterns for comparison
CREATE OR REPLACE FUNCTION find_similar_traces(
    p_execution_id BIGINT,
    p_limit INTEGER DEFAULT 5
)
RETURNS TABLE (
    similar_execution_id BIGINT,
    similarity_score FLOAT,
    status VARCHAR,
    total_duration_ms INTEGER,
    failure_point VARCHAR
) AS $$
WITH target_trace AS (
    SELECT 
        job_id,
        model_id,
        region,
        array_agg(service || '.' || operation ORDER BY started_at) as operation_sequence,
        SUM(duration_ms) as total_duration
    FROM trace_spans
    WHERE execution_id = p_execution_id
    GROUP BY job_id, model_id, region
)
SELECT 
    ts.execution_id as similar_execution_id,
    -- Calculate similarity (simplified - use levenshtein in production)
    1.0 as similarity_score,
    MAX(CASE WHEN ts.status = 'failed' THEN 'failed' ELSE 'completed' END) as status,
    SUM(ts.duration_ms)::INTEGER as total_duration_ms,
    MAX(CASE WHEN ts.status = 'failed' THEN ts.service || '.' || ts.operation ELSE NULL END) as failure_point
FROM trace_spans ts
CROSS JOIN target_trace tt
WHERE ts.execution_id != p_execution_id
  AND ts.model_id = tt.model_id
  AND ts.region = tt.region
GROUP BY ts.execution_id
ORDER BY SUM(ts.duration_ms) DESC
LIMIT p_limit;
$$ LANGUAGE sql;

-- Health monitoring view
CREATE OR REPLACE VIEW trace_spans_health AS
SELECT 
    COUNT(*) as total_spans,
    COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '1 day') as spans_last_24h,
    COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '1 hour') as spans_last_hour,
    pg_size_pretty(pg_relation_size('trace_spans')) as table_size,
    pg_size_pretty(pg_total_relation_size('trace_spans')) as total_size_with_indexes,
    COUNT(DISTINCT trace_id) as unique_traces,
    ROUND(AVG(duration_ms)) as avg_duration_ms,
    COUNT(*) FILTER (WHERE status = 'failed') as failed_spans,
    COUNT(*) FILTER (WHERE status = 'timeout') as timeout_spans
FROM trace_spans;

-- Add comments for documentation
COMMENT ON TABLE trace_spans IS 'Distributed tracing spans for debugging request flow across services';
COMMENT ON COLUMN trace_spans.trace_id IS 'Unique ID for entire request flow (shared across all services)';
COMMENT ON COLUMN trace_spans.span_id IS 'Unique ID for this specific operation';
COMMENT ON COLUMN trace_spans.parent_span_id IS 'Links to parent operation (for nested calls)';
COMMENT ON COLUMN trace_spans.execution_id IS 'Links to executions table for correlation';
COMMENT ON COLUMN trace_spans.metadata IS 'Service-specific data (JSONB for flexibility)';
COMMENT ON FUNCTION diagnose_execution_trace IS 'Auto-detect anomalies in execution trace (gaps >5s, duration >3x avg)';
COMMENT ON FUNCTION identify_root_cause IS 'Pattern-match failures to identify root cause';
COMMENT ON FUNCTION find_similar_traces IS 'Find similar executions for comparison';
