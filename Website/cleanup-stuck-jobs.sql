-- Cleanup Stuck Jobs Script
-- Run this to identify and fix stuck jobs

-- ============================================
-- STEP 1: IDENTIFY STUCK JOBS
-- ============================================

-- 1a. Jobs stuck in "created" (>5 minutes old)
SELECT 
    'STUCK_CREATED' as issue_type,
    jobspec_id, 
    status,
    created_at,
    EXTRACT(EPOCH FROM (NOW() - created_at))/60 as age_minutes
FROM jobs 
WHERE status = 'created' 
AND created_at < NOW() - INTERVAL '5 minutes'
ORDER BY created_at ASC;

-- 1b. Jobs stuck in "processing" (>10 minutes old)
SELECT 
    'STUCK_PROCESSING' as issue_type,
    jobspec_id, 
    status,
    updated_at,
    EXTRACT(EPOCH FROM (NOW() - updated_at))/60 as age_minutes
FROM jobs 
WHERE status = 'processing' 
AND updated_at < NOW() - INTERVAL '10 minutes'
ORDER BY updated_at ASC;

-- 1c. Jobs stuck in "running" (>30 minutes old)
SELECT 
    'STUCK_RUNNING' as issue_type,
    jobspec_id, 
    status,
    started_at,
    EXTRACT(EPOCH FROM (NOW() - COALESCE(started_at, created_at)))/60 as age_minutes
FROM jobs 
WHERE status = 'running' 
AND (
    started_at < NOW() - INTERVAL '30 minutes'
    OR (started_at IS NULL AND created_at < NOW() - INTERVAL '30 minutes')
)
ORDER BY COALESCE(started_at, created_at) ASC;

-- 1d. Jobs stuck in "queued" (>5 minutes old)
SELECT 
    'STUCK_QUEUED' as issue_type,
    jobspec_id, 
    status,
    created_at,
    EXTRACT(EPOCH FROM (NOW() - created_at))/60 as age_minutes
FROM jobs 
WHERE status = 'queued' 
AND created_at < NOW() - INTERVAL '5 minutes'
ORDER BY created_at ASC;

-- ============================================
-- STEP 2: COUNT STUCK JOBS BY TYPE
-- ============================================

SELECT 
    'Summary' as report_type,
    COUNT(*) FILTER (WHERE status = 'created' AND created_at < NOW() - INTERVAL '5 minutes') as stuck_created,
    COUNT(*) FILTER (WHERE status = 'processing' AND updated_at < NOW() - INTERVAL '10 minutes') as stuck_processing,
    COUNT(*) FILTER (WHERE status = 'running' AND started_at < NOW() - INTERVAL '30 minutes') as stuck_running,
    COUNT(*) FILTER (WHERE status = 'queued' AND created_at < NOW() - INTERVAL '5 minutes') as stuck_queued,
    COUNT(*) FILTER (WHERE status IN ('created', 'processing', 'running', 'queued')) as total_active_jobs
FROM jobs;

-- ============================================
-- STEP 3: FIX STUCK JOBS (UNCOMMENT TO RUN)
-- ============================================

-- 3a. Cancel stuck "created" jobs (>1 hour old)
-- UPDATE jobs 
-- SET status = 'cancelled', 
--     updated_at = NOW()
-- WHERE status = 'created' 
-- AND created_at < NOW() - INTERVAL '1 hour';

-- 3b. Cancel stuck "processing" jobs (>30 minutes old)
-- UPDATE jobs 
-- SET status = 'cancelled', 
--     updated_at = NOW()
-- WHERE status = 'processing' 
-- AND updated_at < NOW() - INTERVAL '30 minutes';

-- 3c. Cancel stuck "running" jobs (>1 hour old)
-- UPDATE jobs 
-- SET status = 'cancelled', 
--     updated_at = NOW()
-- WHERE status = 'running' 
-- AND (
--     started_at < NOW() - INTERVAL '1 hour'
--     OR (started_at IS NULL AND created_at < NOW() - INTERVAL '1 hour')
-- );

-- 3d. Cancel stuck "queued" jobs (>1 hour old)
-- UPDATE jobs 
-- SET status = 'cancelled', 
--     updated_at = NOW()
-- WHERE status = 'queued' 
-- AND created_at < NOW() - INTERVAL '1 hour';

-- ============================================
-- STEP 4: CLEANUP STUCK EXECUTIONS
-- ============================================

-- 4a. Find executions stuck in "running" (>15 minutes old)
SELECT 
    'STUCK_EXECUTION' as issue_type,
    e.id,
    e.job_id,
    j.jobspec_id,
    e.region,
    e.model_id,
    e.status,
    e.started_at,
    EXTRACT(EPOCH FROM (NOW() - e.started_at))/60 as age_minutes
FROM executions e
JOIN jobs j ON e.job_id = j.id
WHERE e.status = 'running'
AND e.started_at < NOW() - INTERVAL '15 minutes'
ORDER BY e.started_at ASC;

-- 4b. Mark stuck executions as failed (UNCOMMENT TO RUN)
-- UPDATE executions
-- SET status = 'failed',
--     completed_at = NOW()
-- WHERE status = 'running'
-- AND started_at < NOW() - INTERVAL '15 minutes';

-- ============================================
-- STEP 5: VERIFY CLEANUP
-- ============================================

-- Check remaining active jobs
SELECT 
    status,
    COUNT(*) as count,
    MIN(created_at) as oldest,
    MAX(created_at) as newest
FROM jobs
WHERE status IN ('created', 'processing', 'running', 'queued')
GROUP BY status
ORDER BY status;
