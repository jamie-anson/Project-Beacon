# How to Run Database Migration

**Migration**: `007_add_question_id_to_executions.sql`  
**Purpose**: Add question_id column for per-question execution tracking

---

## Option 1: Via Fly.io SSH (Recommended)

### Step 1: Get DATABASE_URL

```bash
flyctl ssh console --app beacon-runner-production
```

Once inside the container:
```bash
echo $DATABASE_URL
```

Copy the DATABASE_URL (it will look like: `postgres://user:pass@host:5432/dbname`)

### Step 2: Exit and Run Migration Locally

```bash
# Exit the SSH session
exit

# Run migration (replace <DATABASE_URL> with the actual URL from step 1)
psql "<DATABASE_URL>" -f migrations/007_add_question_id_to_executions.sql
```

### Step 3: Verify Migration

```bash
psql "<DATABASE_URL>" -c "\d executions" | grep question_id
```

**Expected output**:
```
question_id | character varying(255) |
```

---

## Option 2: Via Railway/Database Dashboard

If the database is on Railway:

1. Go to Railway dashboard
2. Find the database service
3. Click "Connect" → "psql"
4. Copy the connection command
5. Run it in your terminal
6. Once connected, run:

```sql
\i /Users/Jammie/Desktop/Project\ Beacon/runner-app/migrations/007_add_question_id_to_executions.sql
```

---

## Option 3: Copy-Paste SQL (Easiest)

If you have database access, just run this SQL:

```sql
-- Add question_id column to executions table
ALTER TABLE executions 
ADD COLUMN IF NOT EXISTS question_id VARCHAR(255);

-- Create composite index for deduplication
CREATE INDEX IF NOT EXISTS idx_executions_dedup_with_question 
ON executions(job_id, region, model_id, question_id);

-- Add comment
COMMENT ON COLUMN executions.question_id IS 'Question ID for per-question execution tracking. NULL or empty for legacy batch executions.';
```

---

## Verification

After running the migration, verify it worked:

```sql
-- Check column exists
\d executions

-- Check index exists
\di idx_executions_dedup_with_question

-- Check existing data (should all have NULL question_id)
SELECT COUNT(*) as total_executions,
       COUNT(question_id) as with_question_id,
       COUNT(*) - COUNT(question_id) as without_question_id
FROM executions;
```

**Expected**:
- Column `question_id` exists
- Index `idx_executions_dedup_with_question` exists
- All existing executions have NULL question_id (backward compatible)

---

## After Migration

Once migration is complete:

1. ✅ Deploy the updated runner app
2. ✅ Test with a sample job
3. ✅ Verify new executions have question_id populated

---

## Rollback (if needed)

If something goes wrong:

```sql
-- Remove index
DROP INDEX IF EXISTS idx_executions_dedup_with_question;

-- Remove column
ALTER TABLE executions DROP COLUMN IF EXISTS question_id;
```

---

## Quick Command Reference

```bash
# Get DATABASE_URL
flyctl ssh console --app beacon-runner-production -C "echo \$DATABASE_URL"

# Run migration (replace <URL>)
psql "<URL>" -f migrations/007_add_question_id_to_executions.sql

# Verify
psql "<URL>" -c "\d executions"
```

---

## Need Help?

If you encounter issues:

1. Check if psql is installed: `psql --version`
2. Install if needed: `brew install postgresql`
3. Verify database connection: `psql "<DATABASE_URL>" -c "SELECT 1"`
4. Check migration file exists: `ls -la migrations/007_add_question_id_to_executions.sql`

**The migration file is ready at**:
`/Users/Jammie/Desktop/Project Beacon/runner-app/migrations/007_add_question_id_to_executions.sql`

**Once migration is complete, we can deploy!** 🚀
