# End-to-End Smoke Tests for Project Beacon

## Pre-Deployment Checks

### 1. Infrastructure Health
```bash
# Check Fly app status
flyctl status -a beacon-runner-production

# Check Railway hybrid router
curl https://project-beacon-production.up.railway.app/health

# Check provider availability
curl https://project-beacon-production.up.railway.app/providers | jq '.providers | length'

# Check database connectivity
psql "postgresql://neondb_owner:npg_puA76KTFISkD@ep-broad-cherry-abdo0pru-pooler.eu-west-2.aws.neon.tech/neondb?sslmode=require&channel_binding=require" -c "SELECT 1"
```

## Post-Deployment E2E Tests

### Test 1: Signature Verification ✅
**Goal:** Verify portal-signed jobs are accepted

**Steps:**
1. Submit a bias detection job from portal
2. Check Sentry logs for "signature verified successfully"
3. Verify job is NOT rejected with "Invalid JobSpec signature"

**Expected:**
```bash
./sentry-logs.sh | grep "signature verified successfully"
# Should show recent successful verification
```

**Success Criteria:**
- ✅ No "signature verification failed" errors
- ✅ Job accepted and execution created

---

### Test 2: Provider Discovery 🔍
**Goal:** Verify fuzzy region matching works

**Steps:**
1. Submit job with regions: `["US", "EU"]`
2. Check logs for provider discovery

**Expected:**
```bash
./sentry-logs.sh | grep -E "providers|region"
# Should show providers found for both regions
```

**Success Criteria:**
- ✅ No "No providers found for region" warnings
- ✅ Providers matched: US → us-east, EU → eu-west

---

### Test 3: Cross-Region Execution 🌍
**Goal:** Verify jobs execute across multiple regions

**Steps:**
1. Submit multi-region bias detection job
2. Wait 2-3 minutes
3. Check execution status in database

**Query:**
```bash
psql "postgresql://neondb_owner:npg_puA76KTFISkD@ep-broad-cherry-abdo0pru-pooler.eu-west-2.aws.neon.tech/neondb?sslmode=require&channel_binding=require" -c "
SELECT 
  jobspec_id,
  status,
  started_at,
  completed_at,
  (SELECT COUNT(*) FROM region_results WHERE cross_region_execution_id = cross_region_executions.id) as region_count
FROM cross_region_executions 
ORDER BY started_at DESC 
LIMIT 5"
```

**Success Criteria:**
- ✅ Status: "completed" or "running"
- ✅ region_count: 2 (for US + EU)
- ✅ No "failed" status

---

### Test 4: Region Results 📊
**Goal:** Verify individual region executions complete

**Query:**
```bash
# Get latest execution ID
EXEC_ID=$(psql "postgresql://neondb_owner:npg_puA76KTFISkD@ep-broad-cherry-abdo0pru-pooler.eu-west-2.aws.neon.tech/neondb?sslmode=require&channel_binding=require" -t -c "SELECT id FROM cross_region_executions ORDER BY started_at DESC LIMIT 1")

# Check region results
psql "postgresql://neondb_owner:npg_puA76KTFISkD@ep-broad-cherry-abdo0pru-pooler.eu-west-2.aws.neon.tech/neondb?sslmode=require&channel_binding=require" -c "
SELECT region, status, provider_id, started_at, completed_at 
FROM region_results 
WHERE cross_region_execution_id = '$EXEC_ID'"
```

**Success Criteria:**
- ✅ 2 region results (us-east, eu-west)
- ✅ Both status: "completed"
- ✅ Both have provider_id set

---

### Test 5: Portal Integration 🖥️
**Goal:** Verify portal displays results correctly

**Steps:**
1. Submit job from portal
2. Wait for completion
3. Check portal UI shows:
   - Job status
   - Region results
   - Execution progress

**Success Criteria:**
- ✅ Job appears in portal job list
- ✅ Status updates in real-time
- ✅ Results display correctly

---

### Test 6: Error Handling 🚨
**Goal:** Verify graceful error handling

**Test Cases:**

**6a. Invalid Signature**
```bash
# Submit job with tampered signature (should fail)
# Expected: 400 Bad Request with "Invalid JobSpec signature"
```

**6b. Missing Providers**
```bash
# Submit job with non-existent region: ["MARS"]
# Expected: Job created but execution fails gracefully
```

**6c. Database Unavailable**
```bash
# Temporarily break DB connection (if safe)
# Expected: Structured error response with retry guidance
```

---

## Quick Smoke Test Script

```bash
#!/bin/bash
# smoke-test.sh - Quick validation after deployment

echo "🔍 Running Project Beacon Smoke Tests..."
echo ""

# 1. Health checks
echo "1️⃣ Health Checks..."
flyctl status -a beacon-runner-production | grep "started" && echo "✅ Fly app running" || echo "❌ Fly app down"
curl -sf https://project-beacon-production.up.railway.app/health > /dev/null && echo "✅ Railway healthy" || echo "❌ Railway down"

# 2. Provider count
echo ""
echo "2️⃣ Provider Discovery..."
PROVIDER_COUNT=$(curl -s https://project-beacon-production.up.railway.app/providers | jq '.providers | length')
echo "Providers available: $PROVIDER_COUNT"
[ "$PROVIDER_COUNT" -ge 2 ] && echo "✅ Sufficient providers" || echo "❌ Not enough providers"

# 3. Recent signature verifications
echo ""
echo "3️⃣ Recent Signature Verifications..."
cd /Users/Jammie/Desktop/Project\ Beacon/runner-app
RECENT_SIGS=$(./sentry-logs.sh 2>&1 | grep -c "signature verified successfully")
echo "Recent successful verifications: $RECENT_SIGS"
[ "$RECENT_SIGS" -gt 0 ] && echo "✅ Signatures working" || echo "⚠️  No recent verifications"

# 4. Recent executions
echo ""
echo "4️⃣ Recent Cross-Region Executions..."
RECENT_EXECS=$(psql "postgresql://neondb_owner:npg_puA76KTFISkD@ep-broad-cherry-abdo0pru-pooler.eu-west-2.aws.neon.tech/neondb?sslmode=require&channel_binding=require" -t -c "SELECT COUNT(*) FROM cross_region_executions WHERE started_at > NOW() - INTERVAL '1 hour'")
echo "Executions in last hour: $RECENT_EXECS"

# 5. Error rate
echo ""
echo "5️⃣ Error Rate Check..."
ERROR_COUNT=$(./sentry-logs.sh 2>&1 | grep -c "signature verification failed")
echo "Recent signature failures: $ERROR_COUNT"
[ "$ERROR_COUNT" -eq 0 ] && echo "✅ No signature errors" || echo "⚠️  Some signature failures detected"

echo ""
echo "🎉 Smoke test complete!"
```

---

## Performance Benchmarks

### Expected Timings
- **Signature Verification:** < 50ms
- **Provider Discovery:** < 200ms
- **Single Region Execution:** 30-60 seconds
- **Cross-Region Execution (2 regions):** 60-120 seconds
- **End-to-End (submit → complete):** 2-3 minutes

### Monitoring Queries

**Average execution time:**
```sql
SELECT 
  AVG(EXTRACT(EPOCH FROM (completed_at - started_at))) as avg_duration_seconds,
  COUNT(*) as total_executions
FROM cross_region_executions 
WHERE completed_at IS NOT NULL 
  AND started_at > NOW() - INTERVAL '24 hours';
```

**Success rate:**
```sql
SELECT 
  status,
  COUNT(*) as count,
  ROUND(100.0 * COUNT(*) / SUM(COUNT(*)) OVER (), 2) as percentage
FROM cross_region_executions 
WHERE started_at > NOW() - INTERVAL '24 hours'
GROUP BY status;
```

---

## Rollback Criteria

**Immediate rollback if:**
- ❌ Signature verification failure rate > 10%
- ❌ Provider discovery fails for all jobs
- ❌ Database connection errors
- ❌ App crashes or restarts repeatedly

**Investigate if:**
- ⚠️  Execution time > 5 minutes
- ⚠️  Success rate < 80%
- ⚠️  Memory usage > 90%
