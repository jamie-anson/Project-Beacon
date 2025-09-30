-- Verification script for migration 0009_add_response_classification
-- Run this against your Railway database to verify the migration was applied

-- Check if new columns exist
SELECT 
    column_name,
    data_type,
    is_nullable,
    column_default
FROM information_schema.columns
WHERE table_name = 'executions'
AND column_name IN (
    'is_substantive',
    'is_content_refusal',
    'is_technical_error',
    'response_classification',
    'response_length',
    'system_prompt'
)
ORDER BY column_name;

-- Expected: 6 rows
-- is_content_refusal | boolean | YES | false
-- is_substantive | boolean | YES | false
-- is_technical_error | boolean | YES | false
-- response_classification | character varying | YES | NULL
-- response_length | integer | YES | NULL
-- system_prompt | text | YES | NULL

-- Check if indexes exist
SELECT 
    indexname,
    indexdef
FROM pg_indexes
WHERE tablename = 'executions'
AND indexname LIKE '%classification%'
OR indexname LIKE '%substantive%'
OR indexname LIKE '%refusal%'
ORDER BY indexname;

-- Expected: 4 indexes
-- idx_executions_classification
-- idx_executions_refusal
-- idx_executions_substantive
-- idx_executions_response_length

-- Check table structure
\d executions

-- Sample query to test new columns
SELECT 
    id,
    job_id,
    region,
    model_id,
    status,
    is_substantive,
    is_content_refusal,
    response_classification,
    response_length,
    LENGTH(system_prompt) as system_prompt_length,
    created_at
FROM executions
ORDER BY created_at DESC
LIMIT 5;

-- Check if any existing executions have classification data
SELECT 
    COUNT(*) as total_executions,
    COUNT(response_classification) as with_classification,
    COUNT(CASE WHEN is_substantive THEN 1 END) as substantive_count,
    COUNT(CASE WHEN is_content_refusal THEN 1 END) as refusal_count,
    COUNT(CASE WHEN is_technical_error THEN 1 END) as error_count
FROM executions;

-- Verify migration version (if you have a migrations table)
-- SELECT * FROM schema_migrations ORDER BY version DESC LIMIT 5;
