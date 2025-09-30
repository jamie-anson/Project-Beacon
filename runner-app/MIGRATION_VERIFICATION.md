# Database Migration Verification - 0009_add_response_classification

**Date**: 2025-09-29T21:04:51+01:00  
**Migration**: 0009_add_response_classification  
**Status**: ⏳ Verification in progress

---

## Quick Verification

### Option 1: Railway CLI (Recommended)

```bash
# Connect to Railway database
railway connect

# Run verification script
psql $DATABASE_URL -f scripts/verify_migration_0009.sql
```

### Option 2: Railway Dashboard

1. Go to Railway dashboard
2. Select your database service
3. Click "Connect" or "Query"
4. Run this query:

```sql
-- Quick check for new columns
SELECT column_name, data_type 
FROM information_schema.columns 
WHERE table_name = 'executions' 
AND column_name IN (
    'is_substantive',
    'is_content_refusal',
    'is_technical_error',
    'response_classification',
    'response_length',
    'system_prompt'
);
```

**Expected Result**: 6 rows returned

### Option 3: Direct psql Connection

```bash
# Get DATABASE_URL from Railway
railway variables

# Connect directly
psql "postgresql://user:pass@host:port/dbname"

# Run verification
\i scripts/verify_migration_0009.sql
```

---

## Detailed Verification Steps

### Step 1: Check Columns Exist

```sql
SELECT column_name, data_type, is_nullable, column_default
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
```

**Expected Output:**
```
column_name              | data_type          | is_nullable | column_default
-------------------------+--------------------+-------------+----------------
is_content_refusal       | boolean            | YES         | false
is_substantive           | boolean            | YES         | false
is_technical_error       | boolean            | YES         | false
response_classification  | character varying  | YES         | NULL
response_length          | integer            | YES         | NULL
system_prompt            | text               | YES         | NULL
```

✅ **Success**: All 6 columns present  
❌ **Failure**: Missing columns → Run migration manually

---

### Step 2: Check Indexes Created

```sql
SELECT indexname, indexdef
FROM pg_indexes
WHERE tablename = 'executions'
AND (
    indexname LIKE '%classification%'
    OR indexname LIKE '%substantive%'
    OR indexname LIKE '%refusal%'
    OR indexname LIKE '%response_length%'
)
ORDER BY indexname;
```

**Expected Output:**
```
indexname                          | indexdef
-----------------------------------+--------------------------------------------------
idx_executions_classification      | CREATE INDEX ... ON executions(response_classification)
idx_executions_refusal             | CREATE INDEX ... ON executions(is_content_refusal)
idx_executions_response_length     | CREATE INDEX ... ON executions(response_length)
idx_executions_substantive         | CREATE INDEX ... ON executions(is_substantive)
```

✅ **Success**: All 4 indexes present  
❌ **Failure**: Missing indexes → Run migration manually

---

### Step 3: Test Table Structure

```sql
\d executions
```

**Look for these columns in output:**
- `is_substantive` (boolean)
- `is_content_refusal` (boolean)
- `is_technical_error` (boolean)
- `response_classification` (varchar)
- `response_length` (integer)
- `system_prompt` (text)

---

### Step 4: Check Existing Data

```sql
SELECT 
    COUNT(*) as total_executions,
    COUNT(response_classification) as with_classification,
    COUNT(CASE WHEN is_substantive THEN 1 END) as substantive_count,
    COUNT(CASE WHEN is_content_refusal THEN 1 END) as refusal_count
FROM executions;
```

**Expected for existing data:**
- `total_executions`: > 0 (if you have existing data)
- `with_classification`: 0 (old executions won't have classifications)
- `substantive_count`: 0
- `refusal_count`: 0

**This is normal** - existing executions won't have classification data. Only new executions will be classified.

---

## If Migration Not Applied

### Manual Migration Steps

**1. Get the migration file:**
```bash
cd /Users/Jammie/Desktop/Project\ Beacon/runner-app
cat migrations/0009_add_response_classification.up.sql
```

**2. Connect to Railway database:**
```bash
railway connect
```

**3. Run migration:**
```bash
psql $DATABASE_URL -f migrations/0009_add_response_classification.up.sql
```

**4. Verify success:**
```bash
psql $DATABASE_URL -c "SELECT column_name FROM information_schema.columns WHERE table_name = 'executions' AND column_name = 'response_classification';"
```

**Expected**: One row with `response_classification`

---

## Rollback (If Needed)

**Only if something goes wrong:**

```bash
# Run down migration
psql $DATABASE_URL -f migrations/0009_add_response_classification.down.sql

# Verify rollback
psql $DATABASE_URL -c "\d executions"
```

---

## Verification Checklist

### ✅ Migration Applied Successfully If:

- [x] All 6 new columns exist
- [x] All 4 indexes created
- [x] No errors in migration logs
- [x] Table structure correct
- [x] Existing data intact

### ⚠️ Migration Needs Manual Application If:

- [ ] Columns missing
- [ ] Indexes missing
- [ ] Error messages in logs
- [ ] Auto-migration disabled

---

## Common Issues

### Issue: Columns Don't Exist

**Cause**: Migration didn't run automatically  
**Solution**: Run migration manually (see above)

### Issue: Permission Denied

**Cause**: Database user lacks ALTER TABLE permission  
**Solution**: Use Railway dashboard SQL console (has admin access)

### Issue: Column Already Exists

**Cause**: Migration ran partially or multiple times  
**Solution**: Check if all columns exist, may need to complete manually

---

## Next Steps After Verification

### If Migration Successful ✅

1. **Test API endpoint:**
```bash
curl https://fabulous-renewal-production.up.railway.app/health
```

2. **Check portal:**
- Open: https://project-beacon.netlify.app
- Navigate to active job
- Verify Classification column appears

3. **Submit test job:**
- Create job with 3 models
- Monitor execution
- Check classifications stored

### If Migration Failed ❌

1. Run migration manually
2. Verify success
3. Restart Railway service if needed
4. Retest

---

## Support

**Railway Issues**: Check Railway logs  
**Database Issues**: Check PostgreSQL logs  
**Migration Issues**: Review migration SQL file

---

**Status**: ⏳ Awaiting verification  
**Action**: Run verification queries above  
**Expected**: All checks pass ✅
