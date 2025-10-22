# Distributed Tracing Pre-Flight Checklist

**Run these checks BEFORE starting implementation**

---

## 1. Database Health Check

```bash
# Check Neon database connection
psql $DATABASE_URL -c "SELECT version();"

# Check current connection pool usage
psql $DATABASE_URL -c "
SELECT 
    count(*) as total_connections,
    count(*) FILTER (WHERE state = 'active') as active,
    count(*) FILTER (WHERE state = 'idle') as idle
FROM pg_stat_activity 
WHERE datname = current_database();
"

# Check database size
psql $DATABASE_URL -c "
SELECT pg_size_pretty(pg_database_size(current_database())) as db_size;
"

# Verify Modal has DATABASE_URL access
modal secret list | grep DATABASE_URL
```

**Expected Results**:
- ✅ Connection successful
- ✅ Active connections <50% of pool size
- ✅ Database size reasonable (<1GB warning if >5GB)
- ✅ Modal has DATABASE_URL secret

**If Fails**: Don't proceed - fix database issues first

---

## 2. Current Performance Baseline

```bash
# Query current executions table
psql $DATABASE_URL -c "
SELECT 
    COUNT(*) as total_executions,
    AVG(EXTRACT(EPOCH FROM (completed_at - started_at))) as avg_duration_seconds,
    MAX(EXTRACT(EPOCH FROM (completed_at - started_at))) as max_duration_seconds
FROM executions 
WHERE started_at > NOW() - INTERVAL '7 days'
  AND completed_at IS NOT NULL;
"

# Check slow queries
psql $DATABASE_URL -c "
SELECT 
    query,
    calls,
    mean_exec_time,
    max_exec_time
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 5;
"
```

**Record These Numbers**:
- Average execution time: _______ seconds
- Max execution time: _______ seconds
- Slowest query time: _______ ms

**Purpose**: Compare after adding tracing to ensure <5% overhead

---

## 3. Existing Logging Verification

```bash
# Check runner logs have trace IDs
flyctl logs -a beacon-runner-change-me --lines 50 | grep -i "trace"

# Check if Sentry is configured
flyctl secrets list -a beacon-runner-change-me | grep SENTRY

# Check hybrid router logs
railway logs -s project-beacon | grep -i "trace\|span"
```

**Verify**:
- ✅ Runner has logging infrastructure
- ✅ Sentry DSN is configured (or not - we can add it)
- ✅ Logs are being captured

---

## 4. Test Database Write Performance

```bash
# Create test table
psql $DATABASE_URL -c "
CREATE TABLE IF NOT EXISTS test_trace_write (
    id BIGSERIAL PRIMARY KEY,
    trace_id UUID NOT NULL,
    data JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
"

# Test write speed (should be <5ms average)
time psql $DATABASE_URL -c "
INSERT INTO test_trace_write (trace_id, data)
SELECT 
    gen_random_uuid(),
    '{\"test\": \"data\"}'::jsonb
FROM generate_series(1, 100);
"

# Clean up
psql $DATABASE_URL -c "DROP TABLE test_trace_write;"
```

**Expected**: 100 inserts in <500ms (5ms per insert)

**If Slower**: Database may be overloaded - investigate before adding tracing

---

## 5. Feature Flag Infrastructure Check

```bash
# Check if runner supports environment variable feature flags
flyctl secrets list -a beacon-runner-change-me

# We'll add these in implementation:
# ENABLE_DB_TRACING=false
# SENTRY_DSN=https://...
# SENTRY_TRACES_SAMPLE_RATE=1.0
```

**Verify**: Can set secrets via flyctl (no errors)

---

## 6. Staging Environment Verification

```bash
# Verify you have a staging environment
# (If not, we'll test with feature flags OFF in production)

# Check staging database
echo $DATABASE_URL_STAGING

# Check staging runner
flyctl status -a beacon-runner-staging 2>/dev/null || echo "No staging app"
```

**Ideal**: Separate staging environment  
**Acceptable**: Production with feature flags OFF initially

---

## 7. Rollback Capability Check

```bash
# Verify you can rollback deployments
git log --oneline -n 5

# Verify you can update secrets quickly
flyctl secrets set TEST_FLAG=true -a beacon-runner-change-me
flyctl secrets unset TEST_FLAG -a beacon-runner-change-me

# Test time
# Should complete in <30 seconds
```

**Verify**: Can rollback quickly if needed

---

## 8. Monitoring Setup

```bash
# Check if you have access to:
# - Fly.io metrics dashboard
flyctl dashboard -a beacon-runner-change-me

# - Neon database dashboard
# https://console.neon.tech/

# - Railway logs
railway logs -s project-beacon --lines 10

# - Modal dashboard  
modal app list
```

**Verify**: Can access all monitoring dashboards

---

## 9. Team Communication

**Questions to Answer**:

1. **Deployment Window**: When can you deploy?
   - [ ] Low-traffic period identified
   - [ ] Team notified of deployment
   - [ ] Time allocated for monitoring (1-2 hours)

2. **Rollback Authority**: Who can authorize rollback?
   - [ ] You can rollback independently
   - [ ] Team member on standby
   - [ ] Emergency procedure documented

3. **Success Criteria**: What indicates success?
   - [ ] No new errors in logs for 1 hour
   - [ ] Execution times within 5% of baseline
   - [ ] Database connections stable
   - [ ] Test execution completes successfully

---

## 10. Code Readiness

```bash
# Ensure you're on latest main
git checkout main
git pull origin main

# Create feature branch
git checkout -b feature/distributed-tracing

# Verify runner app tests pass
cd /Users/Jammie/Desktop/Project\ Beacon/runner-app
go test ./... 2>&1 | head -20

# Verify no uncommitted changes
git status
```

**Verify**: Clean slate to start from

---

## Final Go/No-Go Decision

### ✅ GO if ALL true:
- [ ] Database healthy and responsive
- [ ] Performance baseline recorded
- [ ] Existing logging verified
- [ ] DB write performance acceptable (<5ms)
- [ ] Feature flag capability confirmed
- [ ] Rollback tested and ready
- [ ] Monitoring dashboards accessible
- [ ] Deployment window scheduled
- [ ] Code base clean and tests passing
- [ ] Risk assessment document read

### ⛔ NO-GO if ANY true:
- [ ] Database connection issues
- [ ] DB write performance >10ms
- [ ] Can't access monitoring dashboards
- [ ] No rollback capability
- [ ] No time allocated for monitoring
- [ ] Uncommitted changes in codebase
- [ ] Haven't read risk assessment

---

## Next Steps After Pre-Flight

If all checks pass:

1. **Read**: TRACING-RISK-ASSESSMENT.md completely
2. **Start**: Week 1, Day 1 (Database migration on staging)
3. **Monitor**: Every step, checking logs continuously
4. **Document**: Any issues or deviations from plan

---

**Date Completed**: _____________  
**Completed By**: _____________  
**Go/No-Go Decision**: _____________  
**Notes**: _____________________________________________
