-- Check if EU executions exist in database
-- Replace <job-id> with your actual jobspec_id

-- Count executions by region for the job
SELECT 
    region,
    status,
    COUNT(*) as count
FROM executions e
JOIN jobs j ON e.job_id = j.id
WHERE j.jobspec_id = '<job-id>'
GROUP BY region, status
ORDER BY region, status;

-- Show all executions for the job
SELECT 
    e.id,
    e.region,
    e.status,
    e.provider_id,
    e.model_id,
    e.question_id,
    e.created_at
FROM executions e
JOIN jobs j ON e.job_id = j.id
WHERE j.jobspec_id = '<job-id>'
ORDER BY e.created_at DESC;

-- Check for EU executions specifically
SELECT 
    e.id,
    e.region,
    e.status,
    e.provider_id,
    e.model_id,
    e.question_id,
    e.started_at,
    e.completed_at,
    e.created_at
FROM executions e
JOIN jobs j ON e.job_id = j.id
WHERE j.jobspec_id = '<job-id>'
  AND (e.region = 'eu-west' OR e.region = 'EU' OR e.region = 'eu')
ORDER BY e.created_at DESC;
