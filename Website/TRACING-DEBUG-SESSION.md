# Tracing Debug Session - 2025-10-22

**Status**: 🔍 DEBUGGING - Tracing code deployed but not executing

---

## 🎯 Goal

Get distributed tracing working in production to diagnose job failures.

---

## ✅ What We've Done

### 1. **Database Schema** (Complete)
- ✅ Created `trace_spans` table
- ✅ Added diagnostic functions
- ✅ Deployed to Neon database

### 2. **Tracing Code** (Complete)
- ✅ Created `internal/logging/db_tracer.go`
- ✅ Integrated into `job_runner.go`
- ✅ Added `DBTracer` field to `JobRunner`
- ✅ Calls `StartSpan()` in `executeQuestion()`

### 3. **Configuration** (Complete)
- ✅ Set `ENABLE_DB_TRACING=true` secret
- ✅ Deployed code to production (multiple times)
- ✅ Runner restarted

### 4. **Query Tools** (Complete)
- ✅ Created `query-traces.sh` helper script
- ✅ Created `watch-traces.sh` monitor
- ✅ Created `test-tracing.sh` pre-flight checks

---

## ❌ The Problem

**Multiple jobs have failed, but ZERO trace spans appear in database:**

| Job ID | Time | Executions | Traces |
|--------|------|------------|--------|
| 469 | 11:06 AM | 4 failed | 0 |
| 470 | 11:19 AM | 4 failed | 0 |
| 471 | 11:36 AM | 4 failed | 0 |
| 472 | 12:15 PM | 4 failed | 0 |

All executions fail in ~10-20ms with empty `output_data` and `receipt_data`.

---

## 🔍 Debugging Steps Taken

### Attempt 1: Check Secret
```bash
flyctl secrets list -a beacon-runner-production | grep ENABLE_DB_TRACING
# Result: Secret exists (digest: d8c5ac2e11c8e492)
```

### Attempt 2: Reset Secret
```bash
flyctl secrets unset ENABLE_DB_TRACING
flyctl secrets set ENABLE_DB_TRACING=true
# Result: Runner restarted, still no traces
```

### Attempt 3: Add Debug Logging
Added debug output to:
- `NewDBTracer()` - Shows if tracing is enabled on startup
- `StartSpan()` - Shows when span creation is attempted

**Current deployment**: Deploying with debug logging to see what's happening

---

## 🤔 Possible Root Causes

### Theory 1: Environment Variable Not Set
- Secret exists but may not be loading correctly
- Debug logging will show: `⚠️ DBTracer: DISABLED`

### Theory 2: DBTracer is Nil
- `NewJobRunner()` creates DBTracer
- But maybe it's not being called correctly?
- Debug logging will show: panic or no output

### Theory 3: Code Path Not Executing
- `executeQuestion()` might not be called
- Or early return before `StartSpan()`
- Debug logging will show: no "StartSpan called" message

### Theory 4: Silent Failure
- `RecordSpan()` returns early if `!dt.enabled`
- No error logged, just silently skips
- Debug logging will show: "StartSpan called" but "DISABLED"

---

## 📊 Expected Debug Output

### If Tracing is Working:
```
🔍 DBTracer: ENABLED (ENABLE_DB_TRACING=true)
🔍 StartSpan called: service=runner, operation=execute_question, enabled=true
✅ StartSpan RecordSpan success
```

### If Tracing is Disabled:
```
⚠️  DBTracer: DISABLED (ENABLE_DB_TRACING=)
🔍 StartSpan called: service=runner, operation=execute_question, enabled=false
```

### If Code Not Executing:
```
(no output at all)
```

---

## 🎯 Next Steps

1. **Wait for deployment** to complete
2. **Submit new test job** or wait for organic job
3. **Check Fly.io logs** for debug output:
   ```bash
   flyctl logs -a beacon-runner-production | grep -E "(DBTracer|StartSpan)"
   ```
4. **Diagnose based on output**:
   - If "DISABLED": Environment variable issue
   - If "ENABLED" but no RecordSpan: Database issue
   - If no output: Code path issue

---

## 💡 Lessons Learned

1. **Feature flags need verification** - Secret exists doesn't mean it's loaded
2. **Silent failures are hard to debug** - Need explicit logging
3. **Production debugging is slow** - Each deploy takes ~2 minutes
4. **Trace retroactively impossible** - Old jobs can't be traced

---

## 📝 Files Modified

```
runner-app/
├── internal/logging/db_tracer.go     [Added debug logging]
├── internal/worker/job_runner.go     [Integrated tracing]
└── migrations/0011_add_trace_spans.up.sql [Already deployed]
```

---

**Status**: Waiting for deployment with debug logging...
**Time Invested**: ~2 hours
**Deployments**: 4
**Jobs Tested**: 4 (all failed, none traced)
