# Distributed Tracing Risk Assessment & Pre-Implementation Checklist

**Date:** 2025-10-22  
**Status:** 🛡️ SAFETY REVIEW COMPLETE  
**Priority:** CRITICAL - Read before implementation

---

## Executive Summary

✅ **GOOD NEWS: Existing Infrastructure Found**
- Runner already has tracing foundation (internal/logging/tracing.go)
- Sentry integration already present (sentry_hook.go)
- Structured logging working (zerolog + slog)
- Hybrid router has Python logging configured

🎯 **Strategy: Extend, Don't Replace**
- Build on existing tracing.go infrastructure
- Integrate with existing Sentry hook
- Add database persistence layer
- Zero disruption to current operations

---

## Existing Infrastructure Analysis

### ✅ What We Already Have

#### Runner App (Go)
```
/internal/logging/
├── tracing.go       ← Already has Span, TraceContext, NewSpan()!
├── sentry_hook.go   ← Sentry already integrated!
├── structured.go    ← Structured logging ready
└── logger.go        ← Base logger
```

**Key Finding**: The runner **already has in-memory tracing**
- `TraceContext` with TraceID, SpanID, RequestID
- `Span` struct with StartTime, EndTime, Tags, Logs
- `NewSpan()`, `Span.Finish()`, `Span.FinishWithError()`
- `JobTracer` for job-specific tracing
- Trace ID propagation via HTTP headers (X-Trace-ID, X-Request-ID)

**What's Missing**: Database persistence for gap detection

#### Hybrid Router (Python)
```python
import logging
logger = logging.getLogger(__name__)
# Extensive use of logger.info(), logger.error(), logger.warning()
```

**Key Finding**: Standard Python logging already in use
- 46+ logger calls across router codebase
- Structured logging patterns consistent
- Error handling with proper logging

---

## Risk Assessment

### 🔴 HIGH RISKS (Must Mitigate)

#### 1. **Database Write Failures Breaking Job Execution**

**Risk**: If trace span INSERT fails, could crash job execution

**Mitigation**:
```go
func (w *JobRunner) recordSpan(...) uuid.UUID {
    spanID := uuid.New()
    
    _, err := w.DB.ExecContext(ctx, `INSERT INTO trace_spans...`)
    
    if err != nil {
        // CRITICAL: Log but DON'T crash
        log.Error().Err(err).Msg("failed to record trace span")
        // Still return spanID so execution continues
    }
    
    return spanID
}
```

**Status**: ✅ Already handled in plan (lines 254-256)

#### 2. **Database Connection Pool Exhaustion**

**Risk**: Extra DB writes could exhaust connection pool

**Mitigation**:
- **Use existing DB connection** (already pooled)
- **Async writes**: Don't wait for trace writes to complete
- **Monitor**: Add metrics for trace_spans table size
- **Retention**: Auto-delete traces older than 30 days

**Implementation**:
```sql
-- Add to migration
CREATE INDEX idx_trace_spans_created_at ON trace_spans(created_at);

-- Cleanup job (run daily)
DELETE FROM trace_spans WHERE created_at < NOW() - INTERVAL '30 days';
```

#### 3. **Modal Function Latency from DB Writes**

**Risk**: Database writes in Modal could add 50-200ms latency

**Mitigation**:
- **Fire-and-forget pattern**: Don't wait for confirmation
- **Connection pooling**: Use asyncpg pool (not sync)
- **Fallback**: If DB unavailable, skip tracing (don't fail inference)

**Implementation**:
```python
try:
    async with db_pool.acquire() as conn:
        await conn.execute("INSERT INTO trace_spans...")
except Exception as e:
    logger.warning(f"Failed to record trace span: {e}")
    # Continue with inference - tracing is optional
```

#### 4. **Trace ID Collision Across Services**

**Risk**: Different trace ID formats could break correlation

**Current State**:
- Runner: `generateID()` using crypto/rand (8 bytes = 16 hex chars)
- Plan: Uses `uuid.New()` (36 chars with dashes)

**Mitigation**:
- **Decision**: Use existing runner `generateID()` format for consistency
- **Alternative**: Migrate runner to UUIDs in Phase 1
- **Don't mix**: Pick one format and use everywhere

---

### 🟡 MEDIUM RISKS (Monitor)

#### 5. **Sentry Rate Limits**

**Risk**: Too many events → Sentry drops/throttles

**Current State**: Sentry already integrated, but not heavily used

**Mitigation**:
- Start with `TracesSampleRate: 1.0` (100%) in dev/staging
- Reduce to `0.1` (10%) in production after validation
- Use Sentry's `BeforeSend` to filter noisy errors

#### 6. **Performance Overhead**

**Risk**: Tracing adds latency to each operation

**Measurements Needed**:
- Baseline: Current execution time without tracing
- With tracing: Measure overhead per span
- Target: <5ms overhead per span

**Mitigation**:
- **Week 4 validation**: Performance testing required
- If overhead >5ms, reduce span granularity

#### 7. **Schema Migration Risk**

**Risk**: Migration could lock tables

**Mitigation**:
- Run migration during low-traffic window
- Tables are new (no ALTER on existing tables)
- Use `IF NOT EXISTS` clauses
- Test on staging first

---

### 🟢 LOW RISKS (Acceptable)

#### 8. **Storage Growth**

**Risk**: trace_spans table could grow large

**Calculation**:
- ~10 spans per execution
- ~1KB per span
- 1000 executions/day = 10MB/day = 300MB/month
- With indexes: ~500MB/month

**Mitigation**: 30-day retention policy keeps storage <15GB

#### 9. **MCP Server Availability**

**Risk**: LLM queries fail if MCP server down

**Impact**: Low - only affects debugging, not production
**Mitigation**: MCP server runs locally, restarts automatically

---

## Pre-Implementation Checklist

### Phase 0: Investigation (Before Any Code)

- [x] **Check existing tracing** - FOUND in internal/logging/tracing.go
- [x] **Check existing Sentry** - FOUND in internal/logging/sentry_hook.go
- [x] **Check existing logging** - FOUND (zerolog + slog + Python logging)
- [x] **Review database schema** - OK (new table, no conflicts)
- [ ] **Check database connection pool size** - NEED TO VERIFY
- [ ] **Review current DB query performance** - NEED TO BASELINE
- [ ] **Check Modal DATABASE_URL access** - NEED TO VERIFY

### Phase 1: Safe Foundation (Zero Risk)

- [ ] **Create migration file** (don't run yet)
- [ ] **Test migration on local Postgres**
- [ ] **Verify indexes don't slow existing queries**
- [ ] **Add trace_spans table to staging DB only**
- [ ] **Test helper functions in isolation**
- [ ] **Performance test DB writes** (should be <5ms)

**Rollback Plan**: Drop table if issues found

### Phase 2: Runner Integration (Low Risk)

**Strategy**: Extend existing tracing.go, don't replace

- [ ] **Add DB persistence to existing Span.Finish()**
- [ ] **Keep existing in-memory tracing working**
- [ ] **Add feature flag**: `ENABLE_DB_TRACING=false` (default off)
- [ ] **Test with flag OFF** - should work exactly as before
- [ ] **Test with flag ON** - adds DB writes
- [ ] **Deploy with flag OFF initially**

**Rollback Plan**: Set feature flag to OFF

### Phase 3: Router Integration (Low Risk)

- [ ] **Add TracingMiddleware as optional middleware**
- [ ] **Feature flag**: `ENABLE_DB_TRACING=false`
- [ ] **Test async DB writes don't block responses**
- [ ] **Measure latency impact** (<5ms target)
- [ ] **Deploy with flag OFF initially**

**Rollback Plan**: Remove middleware, set flag to OFF

### Phase 4: Modal Integration (Medium Risk)

**High Alert**: This affects inference latency

- [ ] **Test DB writes on Modal staging function**
- [ ] **Measure latency impact** (target <50ms)
- [ ] **Use connection pooling** (asyncpg.create_pool)
- [ ] **Add timeout** (max 100ms for DB write)
- [ ] **Test fallback** - inference works if DB unavailable
- [ ] **Deploy to 1 region first** (US only)
- [ ] **Monitor for 24 hours**
- [ ] **If stable, roll out to EU/APAC**

**Rollback Plan**: Deploy version without tracing

### Phase 5: LLM Query System (Zero Risk)

- [ ] **MCP server runs locally** - doesn't affect production
- [ ] **Test SQL functions return correct data**
- [ ] **Test MCP tools in Windsurf**
- [ ] **No production dependencies**

---

## Migration Strategy

### Option A: Extend Existing Tracing (Recommended)

**Benefits**:
- Minimal code changes
- Leverages existing infrastructure
- Lower risk of bugs

**Implementation**:
```go
// internal/logging/tracing.go

// Add to existing Span struct
func (s *Span) PersistToDB(db *sql.DB, executionID int64) error {
    if !isDBTracingEnabled() {
        return nil // Feature flag check
    }
    
    _, err := db.Exec(`
        INSERT INTO trace_spans 
        (trace_id, span_id, parent_span_id, service, operation, 
         started_at, completed_at, duration_ms, status, execution_id, metadata)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
    `, s.TraceID, s.SpanID, s.ParentSpanID, "runner", s.Operation,
       s.StartTime, s.EndTime, s.getDuration(), s.getStatus(), executionID, s.Tags)
    
    return err // Don't crash if fails
}
```

### Option B: Build Parallel System (Higher Risk)

**Not Recommended**: Creates duplicate tracing infrastructure

---

## Existing Logging We Can Leverage

### Runner (Go)

**Structured Fields Already Captured**:
```go
logger.Info("request completed",
    "method", c.Request.Method,
    "path", c.Request.URL.Path,
    "status", c.Writer.Status(),
    "duration_ms", duration.Milliseconds())
```

**We Can Add**:
```go
logger.Info("request completed",
    // ... existing fields ...
    "trace_id", traceID,  // Link to trace_spans table
    "execution_id", executionID)  // Link to executions table
```

### Router (Python)

**Current Logging Pattern**:
```python
logger.info(f"Routing inference to {selected_provider.name}")
logger.error(f"Provider {provider.name} failed: {str(e)}")
```

**We Can Enhance**:
```python
logger.info(f"Routing inference to {selected_provider.name}",
    extra={
        "trace_id": trace_id,
        "provider": selected_provider.name,
        "model": request.model
    })
```

---

## Performance Benchmarks to Establish

### Before Implementation

```bash
# Baseline current performance
scripts/benchmark-current-state.sh

# Measure:
1. Average execution time (current: varies by model)
2. Database query latency (current: ?)
3. Database connection pool usage (current: ?)
4. Modal cold start time (current: ~5-10s)
5. Modal inference time (current: 1-80s depending on model)
```

### After Each Phase

```bash
# Compare against baseline
scripts/benchmark-with-tracing.sh

# Red flag if:
- Execution time increases >5%
- Database pool >80% utilized
- Modal latency increases >100ms
- Any 500 errors appear
```

---

## Rollback Procedures

### Emergency Rollback (Something Breaks)

**Symptoms**:
- Jobs failing with DB errors
- Execution times 2x longer
- Database connection errors
- Modal functions timing out

**Immediate Action**:
```bash
# 1. Disable tracing via feature flag
flyctl secrets set ENABLE_DB_TRACING=false -a beacon-runner-change-me

# 2. If Sentry causing issues
flyctl secrets unset SENTRY_DSN -a beacon-runner-change-me

# 3. If database issues persist
# Drop trace_spans table (tracing data not critical)
psql $DATABASE_URL -c "DROP TABLE IF EXISTS trace_spans CASCADE;"

# 4. Redeploy previous version
git revert <commit-hash>
git push
```

### Gradual Rollback (Performance Issues)

**If overhead >5ms**:
1. Reduce span granularity (fewer spans)
2. Reduce Sentry sample rate (1.0 → 0.1)
3. Increase trace cleanup frequency (delete older data)
4. Disable Modal tracing (highest latency risk)

---

## Success Criteria (Don't Deploy Without These)

### Phase 1 (Database)
- [ ] Migration runs in <10 seconds
- [ ] Indexes don't slow existing queries
- [ ] No connection pool exhaustion

### Phase 2 (Runner)
- [ ] Feature flag works (ON/OFF)
- [ ] Execution time increase <5%
- [ ] No new errors in logs
- [ ] Existing jobs still complete

### Phase 3 (Router)
- [ ] Request latency increase <5ms
- [ ] No 500 errors introduced
- [ ] Traces successfully written to DB

### Phase 4 (Modal)
- [ ] Inference time increase <50ms
- [ ] DB writes complete in <100ms
- [ ] Fallback works (inference succeeds if DB unavailable)

### Phase 5 (LLM Queries)
- [ ] SQL functions return correct data
- [ ] MCP tools work in Windsurf
- [ ] LLM can diagnose test failure in <30s

---

## Final Safety Checks

### Before Merging to Main

- [ ] All tests pass
- [ ] Performance benchmarks acceptable
- [ ] Feature flags tested (ON/OFF)
- [ ] Rollback procedure documented
- [ ] Team notified of deployment window

### Before Deploying to Production

- [ ] Staging tested for 24 hours
- [ ] No errors in staging logs
- [ ] Performance metrics stable
- [ ] Database backup created
- [ ] Rollback tested on staging

### After Production Deployment

- [ ] Monitor for 1 hour continuously
- [ ] Check error rates every 15 minutes
- [ ] Verify execution times acceptable
- [ ] Test LLM query system works
- [ ] Document any issues found

---

## Questions to Answer Before Starting

### Database
- [ ] What's current DB connection pool size? (Check Neon dashboard)
- [ ] What's current query P95 latency? (Baseline needed)
- [ ] Does Modal have DATABASE_URL access? (Verify secrets)

### Performance
- [ ] What's acceptable execution time increase? (Recommend <5%)
- [ ] What's acceptable Modal latency increase? (Recommend <100ms)

### Operational
- [ ] Who monitors production during rollout? (You? Team?)
- [ ] What time window for deployment? (Low traffic period?)
- [ ] How quickly can we rollback? (Test on staging)

---

## Recommended Implementation Order

### Week 1: Foundation (Safest)
1. **Day 1-2**: Database migration on staging ONLY
2. **Day 3-4**: Test helper functions in isolation
3. **Day 5**: Performance benchmarks and analysis

### Week 2: Runner (Low Risk)
1. **Day 1-2**: Extend existing tracing.go with DB persistence
2. **Day 3**: Deploy to staging with flag OFF
3. **Day 4**: Enable flag ON staging, monitor 24h
4. **Day 5**: If stable, deploy to production with flag OFF

### Week 3: Router + Modal (Higher Risk)
1. **Day 1-2**: Router instrumentation on staging
2. **Day 3**: Modal US region only on staging
3. **Day 4**: If stable, deploy router to production
4. **Day 5**: Deploy Modal (US only), monitor 24h

### Week 4: Completion (Safe)
1. **Day 1**: Modal EU/APAC if US stable
2. **Day 2-3**: MCP server setup (local only)
3. **Day 4**: Test LLM queries
4. **Day 5**: Documentation and retrospective

---

**Status**: 🛡️ Risk assessment complete - Safe to proceed with caution  
**Recommendation**: Start with Phase 1 on staging, enable feature flags gradually  
**Emergency Contact**: Keep this doc open during deployment
