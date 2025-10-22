# Database Considerations for Distributed Tracing

**Critical Checklist Before Implementation**

---

## Overview

Based on your existing infrastructure, here are **specific database considerations** you MUST address:

**Your Current Setup**:
- ✅ PostgreSQL (likely Neon based on memory)
- ✅ Migration system in place (`/runner-app/migrations/`)
- ✅ Already have 10 migrations (0001-0010)
- ✅ Complex schema with executions, transparency_log, idempotency_keys

---

## 1. **Migration Numbering & Ordering** 

### Issue
Your migrations are numbered 0001-0010. The tracing migration will be **0011**.

### Critical Points
```sql
-- File: /runner-app/migrations/0011_add_trace_spans.up.sql
-- File: /runner-app/migrations/0011_add_trace_spans.down.sql
```

**⚠️ Migration Order Dependency**:
- Depends on `executions` table (created in 0001)
- Must come AFTER 0010_add_retry_tracking
- NO foreign key to executions (intentional - trace_spans can exist without executions)

### Action Items
- [ ] Create 0011_add_trace_spans.up.sql
- [ ] Create 0011_add_trace_spans.down.sql
- [ ] Test migration on local database first
- [ ] Verify down migration works (rollback capability)

---

## 2. **Table Size & Growth Projections**

### Current Database Profile
Based on your migrations, you have:
- `jobs` table
- `executions` table (with model_id, question_id, region)
- `transparency_log` table
- `idempotency_keys` table
- `cross_region_diffs` table

### New Table Growth Analysis

**Assumptions** (update with your real numbers):
```
Daily executions: _______ (estimate from dashboard)
Regions per execution: 3 (US, EU, APAC)
Models per execution: 3 (llama, mistral, qwen)
Questions per execution: 1-10 (varies)

Total executions/day = daily_jobs × regions × models × questions
```

**Example Calculation**:
```
100 jobs/day × 3 regions × 3 models × 5 questions = 4,500 executions/day
```

### trace_spans Growth

**Per Execution Spans**:
```
1 execution creates approximately:
- 1 span: execute_question
- 1 span: hybrid_router_call
- 1 span: router.inference_request
- 1 span: router.modal_call
- 1 span: modal.inference
= 5 spans per execution
```

**Daily Growth**:
```
4,500 executions/day × 5 spans = 22,500 rows/day
22,500 rows × ~1KB per row = 22.5 MB/day (data)
22,500 rows × ~0.5KB per row = 11.25 MB/day (indexes)
= ~34 MB/day total

Monthly: ~1 GB
Yearly: ~12 GB
```

### Storage Recommendations

1. **Retention Policy** (CRITICAL):
```sql
-- Run daily via cron or scheduler
DELETE FROM trace_spans 
WHERE created_at < NOW() - INTERVAL '30 days';

-- For aggressive cleanup (7 days):
DELETE FROM trace_spans 
WHERE created_at < NOW() - INTERVAL '7 days'
  AND status = 'completed';  -- Keep failures longer for analysis
```

2. **Partitioning** (If scale is high):
```sql
-- Optional: Partition by month for faster cleanup
CREATE TABLE trace_spans_2025_10 PARTITION OF trace_spans
FOR VALUES FROM ('2025-10-01') TO ('2025-11-01');
```

---

## 3. **Index Strategy**

### Essential Indexes (From Plan)
```sql
CREATE INDEX idx_trace_spans_trace_id ON trace_spans(trace_id);
CREATE INDEX idx_trace_spans_job_id ON trace_spans(job_id);
CREATE INDEX idx_trace_spans_execution_id ON trace_spans(execution_id);
CREATE INDEX idx_trace_spans_started_at ON trace_spans(started_at);
CREATE INDEX idx_trace_spans_service_operation ON trace_spans(service, operation);
```

### Index Size Impact
- 5 indexes × 22,500 rows/day × ~40 bytes/row = ~4.5 MB/day in indexes
- Acceptable for PostgreSQL

### Index Considerations

**⚠️ Watch For**:
1. **Index Bloat**: Monitor with:
```sql
SELECT 
    schemaname,
    tablename,
    indexname,
    pg_size_pretty(pg_relation_size(indexrelid)) as index_size
FROM pg_stat_user_indexes
WHERE schemaname = 'public' 
  AND tablename = 'trace_spans'
ORDER BY pg_relation_size(indexrelid) DESC;
```

2. **Unused Indexes**: Check after 1 week:
```sql
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes
WHERE schemaname = 'public' 
  AND tablename = 'trace_spans'
  AND idx_scan = 0;  -- Never used
```

---

## 4. **Connection Pool Management**

### Your Existing Tables
Looking at your migrations, you have multiple tables competing for connections:
- jobs
- executions
- transparency_log
- idempotency_keys
- cross_region_diffs
- (NEW) trace_spans

### Connection Math

**Per Request**:
```
Normal job execution:
1. INSERT INTO jobs (1 connection)
2. INSERT INTO executions (1 connection)
3. INSERT INTO transparency_log (1 connection)
4. (NEW) INSERT INTO trace_spans (5-10 connections for all spans)

Total connections per request: 3 → 8-13
```

**Concurrent Requests**:
```
If you have 10 concurrent jobs:
Before: 10 × 3 = 30 connections
After:  10 × 13 = 130 connections

⚠️ DEFAULT NEON FREE TIER: 100 connections
⚠️ NEON SCALE TIER: 300-500 connections
```

### Critical Checks

1. **Current Pool Size**:
```sql
-- Check current max_connections
SHOW max_connections;

-- Check current usage
SELECT count(*) as active_connections
FROM pg_stat_activity 
WHERE datname = current_database()
  AND state = 'active';
```

2. **Runner App Pool Config**:
```go
// Check in runner-app code:
db, err := sql.Open("postgres", databaseURL)
db.SetMaxOpenConns(25)      // Current setting?
db.SetMaxIdleConns(10)      // Current setting?
db.SetConnMaxLifetime(5 * time.Minute)
```

### Mitigation Strategies

**Option 1: Batch Writes** (Recommended)
```go
// Instead of 5 separate INSERTs
// Use a single transaction with multiple spans

tx, _ := db.BeginTx(ctx, nil)
for _, span := range spans {
    tx.Exec("INSERT INTO trace_spans...")
}
tx.Commit()

// Reduces 5 connections → 1 connection
```

**Option 2: Async Buffer**
```go
// Write to channel, flush periodically
type SpanBuffer struct {
    spans []Span
    mu    sync.Mutex
}

func (sb *SpanBuffer) Flush(db *sql.DB) {
    // Bulk insert all buffered spans
    // Reduces connection pressure
}
```

**Option 3: Connection Pooling**
```go
// Increase pool size (if needed)
db.SetMaxOpenConns(50)  // Was 25
db.SetMaxIdleConns(25)  // Was 10
```

---

## 5. **Write Performance**

### Critical Performance Thresholds

**Target**: <5ms per span INSERT
**Alert**: >10ms per span INSERT
**Critical**: >50ms per span INSERT

### Benchmark Test

```sql
-- Run this BEFORE implementing tracing
-- Establishes baseline

EXPLAIN ANALYZE
INSERT INTO trace_spans 
(trace_id, span_id, parent_span_id, service, operation, 
 started_at, status, metadata, execution_id)
SELECT 
    gen_random_uuid(),
    gen_random_uuid(),
    NULL,
    'test',
    'test_operation',
    NOW(),
    'started',
    '{"test": true}'::jsonb,
    NULL
FROM generate_series(1, 1000);

-- Should complete in <5 seconds (5ms per row)
```

### Performance Factors

1. **JSONB Metadata Column**:
   - GIN index on JSONB is expensive
   - **Don't index metadata** (we search by trace_id/execution_id)
   - Keep metadata small (<1KB)

2. **UUID Generation**:
   - `gen_random_uuid()` is fast in PostgreSQL
   - Go's `uuid.New()` is also fast
   - No performance concern

3. **Timestamp Operations**:
   - `NOW()` in SQL is fast
   - Pre-compute in application if needed

---

## 6. **Transaction Isolation & Deadlocks**

### Your Existing Schema Pattern

Looking at your migrations, you use:
- Foreign keys (e.g., executions → jobs)
- Unique constraints (idempotency_keys)
- Multiple tables updated per transaction

### Potential Deadlock Scenario

```
Transaction A:
1. INSERT INTO executions
2. INSERT INTO trace_spans (execution_id = 123)

Transaction B:
1. SELECT FROM trace_spans WHERE execution_id = 123
2. INSERT INTO executions

= DEADLOCK if both run simultaneously
```

### Prevention Strategy

**Rule**: Trace writes should NOT block execution writes

```go
// GOOD: Non-blocking trace write
func recordTrace(ctx context.Context, span Span) {
    go func() {
        // Async write in separate goroutine
        // Won't block main execution path
        db.Exec("INSERT INTO trace_spans...")
    }()
}

// BAD: Blocking trace write
func recordTrace(ctx context.Context, span Span) error {
    _, err := db.Exec("INSERT INTO trace_spans...")
    if err != nil {
        return err // Blocks entire job execution!
    }
    return nil
}
```

---

## 7. **Neon-Specific Considerations**

### Neon's Autosuspend Feature

**Issue**: Neon databases auto-suspend after inactivity
**Impact**: First query after suspend has ~1-2s cold start

**For Tracing**:
```python
# In Modal functions
try:
    conn = await asyncpg.connect(DATABASE_URL)
    # First query might be slow (cold start)
    await conn.execute("INSERT INTO trace_spans...")
except asyncpg.TimeoutError:
    # Database suspended - this is OK
    # Tracing is optional, don't fail inference
    pass
```

### Neon's Branching Feature (Bonus)

**Use for Testing**:
```bash
# Create a branch for testing tracing
neonctl branches create --name tracing-test

# Test migration on branch
DATABASE_URL_BRANCH="postgresql://..." psql -f 0011_add_trace_spans.up.sql

# If works well, apply to main
# If issues, delete branch
neonctl branches delete tracing-test
```

### Neon's Storage Limits

**Free Tier**: 0.5 GB → ⚠️ trace_spans could fill this in 15 days
**Scale Tier**: 10 GB → trace_spans could fill this in 10 months

**Recommendation**: 
- Use 7-day retention on free tier
- Use 30-day retention on scale tier

---

## 8. **Query Performance Monitoring**

### Set Up Before Implementation

```sql
-- Enable pg_stat_statements (if not already)
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Baseline current slow queries
SELECT 
    query,
    calls,
    mean_exec_time,
    max_exec_time
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 10;
```

### After Implementation

```sql
-- Check trace_spans query performance
SELECT 
    query,
    calls,
    mean_exec_time,
    max_exec_time
FROM pg_stat_statements
WHERE query LIKE '%trace_spans%'
ORDER BY mean_exec_time DESC;

-- Red flag if mean_exec_time > 10ms
```

---

## 9. **Backup & Recovery**

### Migration Rollback

**CRITICAL**: Test rollback before production

```bash
# Test migration up
psql $DATABASE_URL < migrations/0011_add_trace_spans.up.sql

# Verify table created
psql $DATABASE_URL -c "\d trace_spans"

# Test migration down
psql $DATABASE_URL < migrations/0011_add_trace_spans.down.sql

# Verify table dropped
psql $DATABASE_URL -c "\d trace_spans"
# Should show: Did not find any relation named "trace_spans"
```

### Data Loss Acceptance

**Important Understanding**:
- Trace data is **diagnostic only**
- Losing trace data doesn't lose executions/jobs
- Safe to drop trace_spans table in emergency

```sql
-- Emergency cleanup (if needed)
DROP TABLE IF EXISTS trace_spans CASCADE;

-- Jobs and executions remain intact
SELECT count(*) FROM executions;  -- Still works
```

---

## 10. **Security & Access Control**

### Database User Permissions

**Check Current Permissions**:
```sql
-- What can your app user do?
SELECT 
    grantee,
    table_name,
    privilege_type
FROM information_schema.table_privileges
WHERE grantee = current_user;
```

**Required for Tracing**:
```sql
-- App user needs these permissions
GRANT SELECT, INSERT, UPDATE ON trace_spans TO app_user;
GRANT USAGE ON SEQUENCE trace_spans_id_seq TO app_user;

-- Don't need DELETE (cleanup via scheduled job with admin user)
```

### Modal Access Considerations

**Issue**: Modal functions need DATABASE_URL secret

**Security Check**:
```bash
# Verify Modal has DATABASE_URL
modal secret list | grep DATABASE_URL

# If not set:
modal secret create DATABASE_URL "postgresql://..."
```

**⚠️ Security Best Practice**:
- Use read-write user for Modal (needs INSERT)
- Consider separate "tracer" database user with limited permissions
- Rotate credentials regularly

---

## Pre-Implementation Checklist

### Must Answer These Questions:

1. **Current Database Tier**:
   - [ ] Neon Free (0.5 GB, 100 connections)
   - [ ] Neon Scale (10 GB, 300-500 connections)
   - [ ] Other: _________________

2. **Current Connection Pool**:
   - [ ] Max open connections: ______
   - [ ] Current average usage: ______
   - [ ] Peak usage: ______

3. **Current Storage**:
   - [ ] Total database size: ______
   - [ ] Executions table size: ______
   - [ ] Growth rate: ______/day

4. **Acceptable Overhead**:
   - [ ] Can accept 5% write latency increase? (Y/N)
   - [ ] Can accept 1 GB/month storage? (Y/N)
   - [ ] Can increase connection pool? (Y/N)

5. **Retention Policy**:
   - [ ] 7 days (aggressive, saves space)
   - [ ] 30 days (recommended)
   - [ ] 90 days (requires more storage)
   - [ ] Custom: ______

---

## Recommended Actions Before Implementation

### Step 1: Gather Metrics (30 minutes)
```bash
# Run all queries from this document
# Save results for comparison

psql $DATABASE_URL -f database_metrics.sql > baseline_metrics.txt
```

### Step 2: Test on Staging (2 hours)
```bash
# If you have staging database
DATABASE_URL_STAGING="..." 

# Run migration
psql $DATABASE_URL_STAGING < migrations/0011_add_trace_spans.up.sql

# Insert test data
psql $DATABASE_URL_STAGING < test_trace_data.sql

# Query performance
psql $DATABASE_URL_STAGING < test_queries.sql

# Rollback
psql $DATABASE_URL_STAGING < migrations/0011_add_trace_spans.down.sql
```

### Step 3: Create Monitoring Dashboard
```sql
-- Save this as a view for easy monitoring
CREATE VIEW trace_spans_health AS
SELECT 
    COUNT(*) as total_spans,
    COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '1 day') as spans_last_24h,
    COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '1 hour') as spans_last_hour,
    pg_size_pretty(pg_relation_size('trace_spans')) as table_size,
    pg_size_pretty(pg_total_relation_size('trace_spans')) as total_size_with_indexes,
    COUNT(DISTINCT trace_id) as unique_traces,
    AVG(duration_ms) as avg_duration_ms
FROM trace_spans;

-- Query daily
SELECT * FROM trace_spans_health;
```

---

## Red Flags to Watch For

### Week 1 (After Implementation)
- [ ] Database connection errors
- [ ] Query times >10ms
- [ ] Connection pool >80% utilized
- [ ] Storage growth >100 MB/day

### Week 2-4 (Stabilization)
- [ ] Index bloat >2x table size
- [ ] Slow queries appearing
- [ ] Autosuspend issues in Neon
- [ ] Any execution failures related to tracing

---

**Summary**: The biggest database risks are:
1. **Connection pool exhaustion** → Mitigate with batching/async writes
2. **Storage growth** → Mitigate with 30-day retention
3. **Write latency** → Target <5ms, alert >10ms

**All manageable with proper monitoring and the mitigations outlined above.**
