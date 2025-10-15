-- FORCE CLEANUP - Immediately cancel all stuck jobs
-- WARNING: This will cancel jobs that may still be running
-- Use with caution!

-- Show what will be affected
SELECT 
    'WILL_BE_CANCELLED' as action,
    jobspec_id,
    status,
    created_at,
    updated_at,
    EXTRACT(EPOCH FROM (NOW() - COALESCE(updated_at, created_at)))/60 as age_minutes
FROM jobs
WHERE status IN ('created', 'processing', 'running', 'queued')
AND (
    (status = 'created' AND created_at < NOW() - INTERVAL '5 minutes')
    OR (status = 'processing' AND updated_at < NOW() - INTERVAL '10 minutes')
    OR (status = 'running' AND COALESCE(started_at, created_at) < NOW() - INTERVAL '30 minutes')
    OR (status = 'queued' AND created_at < NOW() - INTERVAL '5 minutes')
)
ORDER BY COALESCE(updated_at, created_at) ASC;

-- Count before cleanup
SELECT 
    'BEFORE_CLEANUP' as stage,
    COUNT(*) as total_stuck_jobs
FROM jobs
WHERE status IN ('created', 'processing', 'running', 'queued')
AND (
    (status = 'created' AND created_at < NOW() - INTERVAL '5 minutes')
    OR (status = 'processing' AND updated_at < NOW() - INTERVAL '10 minutes')
    OR (status = 'running' AND COALESCE(started_at, created_at) < NOW() - INTERVAL '30 minutes')
    OR (status = 'queued' AND created_at < NOW() - INTERVAL '5 minutes')
);

-- EXECUTE CLEANUP
-- Uncomment the following block to actually cancel stuck jobs:

/*
BEGIN;

-- Cancel all stuck jobs
UPDATE jobs 
SET status = 'cancelled', 
    updated_at = NOW()
WHERE status IN ('created', 'processing', 'running', 'queued')
AND (
    (status = 'created' AND created_at < NOW() - INTERVAL '5 minutes')
    OR (status = 'processing' AND updated_at < NOW() - INTERVAL '10 minutes')
    OR (status = 'running' AND COALESCE(started_at, created_at) < NOW() - INTERVAL '30 minutes')
    OR (status = 'queued' AND created_at < NOW() - INTERVAL '5 minutes')
);

-- Mark stuck executions as failed
UPDATE executions
SET status = 'failed',
    completed_at = NOW()
WHERE status IN ('running', 'pending', 'created')
AND job_id IN (
    SELECT id FROM jobs WHERE status = 'cancelled'
);

COMMIT;
*/

-- Count after cleanup
SELECT 
    'AFTER_CLEANUP' as stage,
    COUNT(*) as remaining_stuck_jobs
FROM jobs
WHERE status IN ('created', 'processing', 'running', 'queued')
AND (
    (status = 'created' AND created_at < NOW() - INTERVAL '5 minutes')
    OR (status = 'processing' AND updated_at < NOW() - INTERVAL '10 minutes')
    OR (status = 'running' AND COALESCE(started_at, created_at) < NOW() - INTERVAL '30 minutes')
    OR (status = 'queued' AND created_at < NOW() - INTERVAL '5 minutes')
);

-- Show final status distribution
SELECT 
    status,
    COUNT(*) as count
FROM jobs
GROUP BY status
ORDER BY 
    CASE status
        WHEN 'completed' THEN 1
        WHEN 'failed' THEN 2
        WHEN 'cancelled' THEN 3
        WHEN 'running' THEN 4
        WHEN 'processing' THEN 5
        WHEN 'queued' THEN 6
        WHEN 'created' THEN 7
        ELSE 8
    END;
