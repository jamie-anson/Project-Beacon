# Week 2: Router Tracing Implementation Complete ✅

**Date**: 2025-10-22  
**Status**: READY TO DEPLOY - Router instrumented with Sentry + Custom Tracing  

---

## 🎯 What We Implemented

### 1. **Custom Database Tracing** (`hybrid_router/tracing.py`)
- `DBTracer` class for database-backed tracing
- `start_span()` - Creates trace spans in database
- `complete_span()` - Marks spans as completed with duration
- `create_db_pool()` - Async connection pool for tracing
- Respects `ENABLE_DB_TRACING` environment variable

### 2. **Sentry Integration** (`hybrid_router/main.py`)
- Sentry SDK initialization with FastAPI integration
- Environment: `RAILWAY_ENVIRONMENT`
- Release tracking: `router@{git_commit_sha}`
- 20% transaction sampling
- Service tag: `router`

### 3. **Inference Endpoint Instrumentation** (`hybrid_router/api/inference.py`)
- **Sentry transactions** for performance monitoring
- **Breadcrumbs** for execution flow
- **Custom trace spans** stored in database
- **Error capture** with full context
- **Success tracking** with transaction status

---

## 📊 What Gets Traced

### Sentry Transaction
```python
Transaction: router.inference
Duration: <execution_time>ms
Status: ok | internal_error
Tags:
  - model: llama3.2-1b
  - region: US
  - service: router
```

### Breadcrumbs
```
1. Inference request received
   - model: llama3.2-1b
   - region: US
   - has_prompt: true
```

### Database Span
```sql
INSERT INTO trace_spans (
  trace_id,
  span_id,
  service,
  operation,
  started_at,
  status,
  metadata
) VALUES (
  '<uuid>',
  '<uuid>',
  'router',
  'inference_request',
  NOW(),
  'started',
  '{"model": "llama3.2-1b", "region": "US"}'
)
```

---

## 🔧 Files Created/Modified

### New Files
```
hybrid_router/
└── tracing.py          [NEW] Database tracing module
```

### Modified Files
```
hybrid_router/
├── main.py             [MODIFIED] Added Sentry init + DB tracer
├── api/
│   └── inference.py    [MODIFIED] Added tracing to /inference endpoint
└── ../requirements.txt [MODIFIED] Added sentry-sdk, asyncpg
```

---

## 🚀 Deployment Steps

### 1. Install Dependencies
```bash
cd /Users/Jammie/Desktop/Project\ Beacon/Website
pip install -r requirements.txt
```

### 2. Set Environment Variables (Railway)
```bash
# Sentry (required for error tracking)
SENTRY_DSN=<your_sentry_dsn>

# Database (required for custom tracing)
DATABASE_URL=<neon_postgres_url>

# Tracing toggle (optional, defaults to false)
ENABLE_DB_TRACING=true
```

### 3. Deploy to Railway
```bash
# Railway will auto-deploy on git push
git add .
git commit -m "Add Week 2 router tracing"
git push origin main
```

### 4. Verify Deployment
```bash
# Check router health
curl https://beacon-hybrid-router.railway.app/health

# Check Sentry initialization in logs
railway logs -a beacon-hybrid-router | grep Sentry
```

---

## ✅ Verification Checklist

After deployment, verify:

- [ ] Router starts without errors
- [ ] Sentry shows "✅ Sentry initialized for router" in logs
- [ ] DB tracer shows "🔍 DBTracer: ENABLED" in logs (if `ENABLE_DB_TRACING=true`)
- [ ] `/inference` endpoint responds successfully
- [ ] Sentry dashboard shows router transactions
- [ ] Database shows router trace spans (if tracing enabled)

---

## 🔍 Testing Tracing

### Test 1: Submit Inference Request
```bash
curl -X POST https://beacon-hybrid-router.railway.app/inference \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.2-1b",
    "prompt": "What is 2+2?",
    "region_preference": "US"
  }'
```

**Expected**:
- Sentry transaction appears in dashboard
- Database span created (if `ENABLE_DB_TRACING=true`)
- Breadcrumb shows "Inference request received"

### Test 2: Check Database Traces
```bash
./query-traces.sh recent
```

**Expected**:
```
📋 Recent Trace Spans (Last 10)

 id | trace | service | operation          | status    | duration_ms | time 
----+-------+---------+--------------------+-----------+-------------+------
  1 | abc123| router  | inference_request  | completed | 2500        | 14:50
```

### Test 3: Check Sentry Dashboard
Go to: https://sentry.io/organizations/project-beacon/issues/

**Expected**:
- Performance tab shows `router.inference` transactions
- Tags: `model`, `region`, `service=router`
- Breadcrumbs visible in transaction details

---

## 🎯 What This Enables

### 1. **End-to-End Tracing**
```
Runner → Router → Modal Provider
  ↓        ↓          ↓
 span    span       span
```

### 2. **Gap Detection**
Can now detect delays between:
- Runner calling router
- Router receiving request
- Router calling provider

### 3. **Performance Monitoring**
- Track router latency
- Identify slow requests
- Compare performance across regions

### 4. **Error Diagnosis**
- See exact error in router
- Know which provider failed
- Get full request context

---

## 📈 Next Steps (Week 3)

### Modal Provider Tracing
- Add lightweight tracing to Modal functions
- Link Modal spans to router spans
- Complete end-to-end trace chain

### LLM Query System
- Create MCP server for trace queries
- Enable LLM to diagnose failures
- Autonomous debugging capabilities

---

## 🎓 Key Learnings

### What Worked
1. **Async tracing** - Non-blocking database writes
2. **Conditional tracing** - `ENABLE_DB_TRACING` toggle
3. **Sentry integration** - FastAPI integration is seamless
4. **Breadcrumbs** - Provide excellent execution context

### Challenges
1. **Async context** - Need to pass tracer through request state
2. **Transaction context** - Sentry transactions need proper scope
3. **Error handling** - Must capture errors in both systems

### Best Practices
1. **Always use try/except** around tracing code
2. **Make tracing optional** - Don't break app if tracing fails
3. **Use breadcrumbs liberally** - They're cheap and valuable
4. **Tag everything** - Makes filtering in Sentry easier

---

## 🔗 Integration with Week 1

### Runner → Router Flow
```
1. Runner starts span (service=runner, operation=execute_question)
2. Runner calls router /inference
3. Router starts span (service=router, operation=inference_request)
4. Router completes span
5. Runner completes span
```

### Trace Continuity
- Runner generates `trace_id`
- Router uses same `trace_id` (future: pass via header)
- All spans linked by `trace_id`
- Can query full execution path

---

## 📝 Environment Variables Summary

| Variable | Required | Default | Purpose |
|----------|----------|---------|---------|
| `SENTRY_DSN` | Yes | - | Sentry error tracking |
| `DATABASE_URL` | Yes | - | Postgres for tracing |
| `ENABLE_DB_TRACING` | No | `false` | Toggle custom tracing |
| `RAILWAY_ENVIRONMENT` | No | `development` | Sentry environment |
| `RAILWAY_GIT_COMMIT_SHA` | No | `dev` | Sentry release |

---

## 🎉 Status

**Week 2**: ✅ COMPLETE - Router instrumented and ready to deploy  
**Week 1**: ✅ COMPLETE - Runner instrumented (pending execution bug fix)  
**Week 3**: ⏳ PENDING - Modal tracing + LLM query system  

**Blocker**: Production execution bug (separate debug plan)  
**Ready**: Router tracing can be deployed and tested independently

---

**Next Action**: Deploy router to Railway and verify tracing works! 🚀
