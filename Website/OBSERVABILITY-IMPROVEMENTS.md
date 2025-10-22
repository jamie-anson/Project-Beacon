# Observability Improvements for Project Beacon

## Current Pain Points
1. ❌ **Deployment failures are hard to diagnose** - Railway tests fail with cryptic errors
2. ❌ **No visibility into health check failures** - US provider fails but we don't know why
3. ❌ **Limited error context** - Sentry not available in test environment
4. ❌ **No deployment metrics** - Can't track deployment success rate over time

## Recommended Observability Stack

### 1. **Structured Logging** (Quick Win - 1 hour)
**Problem**: Current logs are unstructured, hard to search/filter
**Solution**: Add structured logging with context

```python
# Current
logger.info("Health check failed")

# Improved
logger.info(
    "Health check failed",
    extra={
        "provider": provider.name,
        "region": provider.region,
        "endpoint": provider.endpoint,
        "error_type": type(e).__name__,
        "duration_ms": duration
    }
)
```

**Benefits**:
- Easy to search logs by provider/region
- Can track error patterns
- Better debugging context

**Implementation**:
- Add to `hybrid_router/core/router.py` health check methods
- Add to `hybrid_router/api/inference.py` error handlers
- Use JSON formatter for Railway logs

---

### 2. **Health Check Diagnostics** (Already Done! ✅)
**Status**: Implemented in commit `7af1ed1`
**Features**:
- 6 debug API endpoints (`/debug/*`)
- Automated testing script (`scripts/diagnose-router-health.sh`)
- Real-time provider status with timing
- Manual health check triggering

**Usage**:
```bash
./scripts/diagnose-router-health.sh
curl https://project-beacon-production.up.railway.app/debug/providers
```

---

### 3. **Deployment Status Dashboard** (Medium - 2-3 hours)
**Problem**: No visibility into deployment health over time
**Solution**: Create a simple status page

**Option A: GitHub Pages (Free)**
- Use GitHub Actions to update a static JSON file
- Display deployment history, success rate, current status
- Example: https://status.github.com style

**Option B: Better Uptime (Free tier)**
- Monitor `/health` and `/ready` endpoints
- Email/Slack alerts on failures
- Public status page
- Free for 10 monitors

**Recommended**: Better Uptime (easier, more features)

---

### 4. **Error Tracking Improvements** (Quick - 30 mins)
**Current**: Sentry only works in production
**Solution**: Add fallback error tracking

```python
# In hybrid_router/main.py
if not SENTRY_AVAILABLE:
    # Log errors to a structured format
    logging.config.dictConfig({
        'formatters': {
            'json': {
                'class': 'pythonjsonlogger.jsonlogger.JsonFormatter',
                'format': '%(asctime)s %(name)s %(levelname)s %(message)s'
            }
        }
    })
```

**Benefits**:
- Structured error logs even without Sentry
- Easy to parse and analyze
- Works in test environment

---

### 5. **Deployment Metrics** (Medium - 2 hours)
**Problem**: No metrics on deployment success/failure
**Solution**: Track deployment events

**Implementation**:
```yaml
# .github/workflows/railway.yml
- name: Report deployment status
  if: always()
  run: |
    curl -X POST https://api.github.com/repos/${{ github.repository }}/statuses/${{ github.sha }} \
      -H "Authorization: token ${{ secrets.GITHUB_TOKEN }}" \
      -d '{
        "state": "${{ job.status }}",
        "context": "deployment/railway",
        "description": "Railway deployment"
      }'
```

**Track**:
- Deployment duration
- Success/failure rate
- Time to recovery
- Error patterns

---

### 6. **Provider Health Monitoring** (High Value - 3-4 hours)
**Problem**: Don't know when providers go down until jobs fail
**Solution**: Continuous health monitoring

**Implementation**:
```python
# Add to hybrid_router/core/router.py
async def continuous_health_monitoring(self):
    """Run health checks every 30 seconds"""
    while True:
        await self.health_check_providers()
        await asyncio.sleep(30)

# In lifespan startup:
asyncio.create_task(router_instance.continuous_health_monitoring())
```

**Benefits**:
- Proactive detection of provider issues
- Historical health data
- Can alert before jobs fail

---

### 7. **Request Tracing** (Already Implemented! ✅)
**Status**: DB-backed distributed tracing in `hybrid_router/tracing.py`
**Features**:
- Trace ID for each request
- Span tracking across services
- Performance metrics
- Error correlation

**Enable**: Set `ENABLE_DB_TRACING=true` and `DATABASE_URL`

---

## Priority Implementation Plan

### Phase 1: Quick Wins (1-2 hours)
1. ✅ **Health check diagnostics** (Done)
2. 🔄 **Structured logging** (Add JSON formatter)
3. 🔄 **Error tracking fallback** (Add pythonjsonlogger)

### Phase 2: Monitoring (2-3 hours)
1. 🔄 **Better Uptime integration** (Free monitoring)
2. 🔄 **Deployment status tracking** (GitHub Actions)
3. 🔄 **Provider health monitoring** (Continuous checks)

### Phase 3: Advanced (Optional)
1. ⏳ **Custom metrics dashboard** (Grafana + Prometheus)
2. ⏳ **Alert rules** (PagerDuty/Slack)
3. ⏳ **Performance profiling** (py-spy)

---

## Immediate Action Items

### 1. Add Structured Logging (30 mins)
```bash
pip install python-json-logger
```

Update `hybrid_router/main.py`:
```python
import logging.config
from pythonjsonlogger import jsonlogger

# Configure JSON logging for Railway
if os.getenv("RAILWAY_ENVIRONMENT"):
    handler = logging.StreamHandler()
    formatter = jsonlogger.JsonFormatter(
        '%(asctime)s %(name)s %(levelname)s %(message)s'
    )
    handler.setFormatter(formatter)
    logging.root.addHandler(handler)
    logging.root.setLevel(logging.INFO)
```

### 2. Setup Better Uptime (15 mins)
1. Sign up at https://betteruptime.com
2. Add monitors:
   - `https://project-beacon-production.up.railway.app/health`
   - `https://project-beacon-production.up.railway.app/ready`
   - `https://project-beacon-production.up.railway.app/providers`
3. Configure alerts (email/Slack)
4. Share status page URL

### 3. Enable Continuous Health Checks (15 mins)
Add to `hybrid_router/main.py` lifespan:
```python
# Start continuous health monitoring
asyncio.create_task(router_instance.continuous_health_monitoring())
logger.info("Started continuous health monitoring (30s interval)")
```

---

## Expected Outcomes

### Before
- ❌ Deployment failures discovered by users
- ❌ No visibility into provider health
- ❌ Hard to debug issues
- ❌ No metrics on system health

### After
- ✅ Proactive alerts before users affected
- ✅ Real-time provider health visibility
- ✅ Structured logs for easy debugging
- ✅ Metrics dashboard showing trends
- ✅ Historical data for analysis

---

## Cost Analysis

| Tool | Cost | Value |
|------|------|-------|
| **Structured Logging** | Free | High |
| **Better Uptime** | Free (10 monitors) | High |
| **GitHub Actions** | Free (public repo) | Medium |
| **Sentry** | Free (5k events/mo) | High |
| **Database Tracing** | Free (uses existing DB) | Medium |

**Total Monthly Cost**: $0
**Setup Time**: 3-4 hours
**Maintenance**: <1 hour/month

---

## Next Steps

1. ✅ **Fix current deployment** (db_pool error) - DONE
2. 🔄 **Add structured logging** - 30 mins
3. 🔄 **Setup Better Uptime** - 15 mins
4. 🔄 **Enable continuous health checks** - 15 mins
5. 📊 **Review metrics after 1 week**

---

## Resources

- **Better Uptime**: https://betteruptime.com
- **Python JSON Logger**: https://github.com/madzak/python-json-logger
- **Sentry Docs**: https://docs.sentry.io/platforms/python/guides/fastapi/
- **FastAPI Logging**: https://fastapi.tiangolo.com/tutorial/logging/
