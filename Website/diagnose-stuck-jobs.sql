-- Diagnose stuck jobs issue

-- 1. Check if stuck jobs have ANY executions at all
SELECT 
    j.jobspec_id,
    j.status as job_status,
    j.success_count,
    j.failure_count,
    COUNT(e.id) as actual_execution_count
FROM jobs j
LEFT JOIN executions e ON e.job_id = j.id
WHERE j.status = 'running'
  AND j.started_at < NOW() - INTERVAL '1 hour'  -- Running for more than 1 hour
GROUP BY j.id, j.jobspec_id, j.status, j.success_count, j.failure_count
ORDER BY j.started_at DESC
LIMIT 20;

-- 2. Check one specific stuck job in detail
-- Replace with one of your stuck jobspec_ids
SELECT 
    j.id as job_table_id,
    j.jobspec_id,
    j.status,
    j.started_at,
    j.completed_at,
    j.success_count,
    j.failure_count
FROM jobs j
WHERE j.jobspec_id = 'bias-detection-1760458397970'
LIMIT 1;

-- 3. Check if executions exist but job_id doesn't match
SELECT 
    e.id,
    e.job_id,
    e.region,
    e.status,
    e.model_id,
    e.question_id,
    e.created_at
FROM executions e
WHERE e.job_id = (
    SELECT id FROM jobs WHERE jobspec_id = 'bias-detection-1760458397970'
)
ORDER BY e.created_at DESC;

-- 4. Check if executions were written with wrong job_id
-- (This would happen if runner is using jobspec_id instead of numeric id)
SELECT 
    e.id,
    e.job_id,
    e.region,
    e.status,
    e.model_id,
    e.created_at
FROM executions e
WHERE e.job_id::text LIKE '%1760458397970%'
ORDER BY e.created_at DESC;
