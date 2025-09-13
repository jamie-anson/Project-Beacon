-- Migration: Add cross-region execution support
-- Created: 2025-09-13
-- Description: Add tables for multi-region job execution and cross-region analysis

-- Cross-region execution tracking table
CREATE TABLE IF NOT EXISTS cross_region_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    jobspec_id VARCHAR(255) NOT NULL,
    total_regions INTEGER NOT NULL,
    success_count INTEGER NOT NULL DEFAULT 0,
    failure_count INTEGER NOT NULL DEFAULT 0,
    min_regions_required INTEGER NOT NULL DEFAULT 3,
    min_success_rate DECIMAL(3,2) NOT NULL DEFAULT 0.67,
    status VARCHAR(50) NOT NULL DEFAULT 'running', -- 'running', 'completed', 'partial', 'failed'
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms BIGINT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Individual region execution results
CREATE TABLE IF NOT EXISTS region_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cross_region_execution_id UUID NOT NULL REFERENCES cross_region_executions(id) ON DELETE CASCADE,
    region VARCHAR(100) NOT NULL,
    provider_id VARCHAR(255),
    provider_info JSONB,
    status VARCHAR(50) NOT NULL, -- 'success', 'failed', 'timeout', 'running'
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms BIGINT,
    execution_output JSONB,
    error_message TEXT,
    scoring JSONB, -- bias_score, censorship_detected, factual_accuracy, etc.
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Cross-region analysis results
CREATE TABLE IF NOT EXISTS cross_region_analyses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cross_region_execution_id UUID NOT NULL REFERENCES cross_region_executions(id) ON DELETE CASCADE,
    bias_variance DECIMAL(5,4),
    censorship_rate DECIMAL(3,2),
    factual_consistency DECIMAL(3,2),
    narrative_divergence DECIMAL(3,2),
    key_differences JSONB, -- Array of key difference objects
    risk_assessment JSONB, -- Array of risk assessment objects
    summary TEXT,
    recommendation TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_cross_region_executions_jobspec_id ON cross_region_executions(jobspec_id);
CREATE INDEX IF NOT EXISTS idx_cross_region_executions_status ON cross_region_executions(status);
CREATE INDEX IF NOT EXISTS idx_cross_region_executions_created_at ON cross_region_executions(created_at);

CREATE INDEX IF NOT EXISTS idx_region_results_cross_region_execution_id ON region_results(cross_region_execution_id);
CREATE INDEX IF NOT EXISTS idx_region_results_region ON region_results(region);
CREATE INDEX IF NOT EXISTS idx_region_results_status ON region_results(status);
CREATE INDEX IF NOT EXISTS idx_region_results_provider_id ON region_results(provider_id);

CREATE INDEX IF NOT EXISTS idx_cross_region_analyses_cross_region_execution_id ON cross_region_analyses(cross_region_execution_id);

-- Add foreign key constraint to link executions table with cross-region executions
ALTER TABLE executions ADD COLUMN IF NOT EXISTS cross_region_execution_id UUID REFERENCES cross_region_executions(id);
CREATE INDEX IF NOT EXISTS idx_executions_cross_region_execution_id ON executions(cross_region_execution_id);

-- Update trigger for updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply update triggers
DROP TRIGGER IF EXISTS update_cross_region_executions_updated_at ON cross_region_executions;
CREATE TRIGGER update_cross_region_executions_updated_at
    BEFORE UPDATE ON cross_region_executions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_region_results_updated_at ON region_results;
CREATE TRIGGER update_region_results_updated_at
    BEFORE UPDATE ON region_results
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_cross_region_analyses_updated_at ON cross_region_analyses;
CREATE TRIGGER update_cross_region_analyses_updated_at
    BEFORE UPDATE ON cross_region_analyses
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
