# Hybrid Router Initialization Plan

**Date:** 2025-10-08 → 2025-10-09  
**Issue:** Cross-region jobs fail with "hybrid router not initialized"  
**Severity:** HIGH - Blocks cross-region job execution  
**Status:** ✅ COMPLETE - v25 deployed, now fixing portal data structure issue

---

## ✅ IMPLEMENTATION COMPLETE

**Status:** All components implemented and deployed in v25!

**What Was Built:**
1. ✅ Provider discovery from Railway hybrid router
2. ✅ HybridRouterAdapter to convert provider types
3. ✅ HybridSingleRegionExecutor for job execution
4. ✅ ZerologAdapter for logging
5. ✅ Full wiring in cmd/server/main.go
6. ✅ Proper Receipt v0 schema generation

**Timeline:** Investigation to deployment in ~1 hour

---

## Original Problem Summary

After signature verification was fixed, cross-region jobs were accepted (202) but failed during execution with:
```
hybrid router not initialized - cross-region execution not available
```

**Original State:**
- ✅ Jobs submit successfully
- ✅ Jobs appear in database
- ❌ Jobs cannot execute (hybrid router nil)
- ❌ No provider discovery

---

## Root Cause Analysis

### Code Locations

1. **cmd/server/main.go:61-63**
   ```go
   // TODO: Initialize CrossRegionExecutor with proper hybrid router and single region executor
   crossRegionExecutor := execution.NewCrossRegionExecutor(nil, nil, nil)
   crossRegionHandlers := handlers.NewCrossRegionHandlers(crossRegionExecutor, crossRegionRepo, diffEngine, jobsRepo)
   ```

2. **internal/api/routes.go:73-75**
   ```go
   crossRegionExecutor := execution.NewCrossRegionExecutor(nil, nil, nil)
   biasAnalysisHandler = handlers.NewCrossRegionHandlers(crossRegionExecutor, crossRegionRepo, diffEngine, jobsRepo)
   ```

3. **internal/execution/cross_region_executor.go:190-193**
   ```go
   if cre.hybridRouter == nil {
       return nil, fmt.Errorf("hybrid router not initialized - cross-region execution not available")
   }
   ```

### Missing Components

1. **HybridRouterClient** - Not initialized
2. **SingleRegionExecutor** - Not initialized  
3. **Logger** - Not initialized

---

## Architecture Overview

### Component Dependencies

```
CrossRegionExecutor
├── SingleRegionExecutor (executes on one provider)
├── HybridRouterClient (discovers providers across regions)
└── Logger (logging interface)
```

### Hybrid Router Purpose

The hybrid router is responsible for:
- Provider discovery across multiple regions
- Load balancing across providers
- Failover and retry logic
- Region-aware provider selection

---

## Implementation Summary

### Phase 1: Investigation ✅ COMPLETE

**Findings:**
- ✅ Found `hybrid.Client` in `internal/hybrid/client.go`
- ✅ Railway hybrid router running at `https://project-beacon-production.up.railway.app`
- ✅ `/providers` endpoint returns `modal-us-east` and `modal-eu-west`
- ✅ `HybridRouterClient` interface defined in `cross_region_executor.go`
- ✅ `SingleRegionExecutor` interface defined
- ❌ No implementations existed - needed to create them

**Key Discovery:**
```bash
$ curl https://project-beacon-production.up.railway.app/providers
{"providers":[
  {"name":"modal-us-east","type":"modal","region":"us-east","healthy":true},
  {"name":"modal-eu-west","type":"modal","region":"eu-west","healthy":true}
]}
```

---

### Phase 2: Implementation ✅ COMPLETE

**Created Components:**

1. **hybrid/client.go** - Added `GetProviders()` method
   ```go
   func (c *Client) GetProviders(ctx context.Context) ([]Provider, error)
   ```

2. **execution/hybrid_adapter.go** - NEW FILE
   - Adapts `hybrid.Client` to `HybridRouterClient` interface
   - Converts `hybrid.Provider` to `execution.Provider`

3. **execution/single_region_executor.go** - NEW FILE
   - Implements `SingleRegionExecutor` interface
   - Uses hybrid client for inference
   - Creates Receipt v0 schema with proper fields

4. **execution/zerolog_adapter.go** - NEW FILE
   - Adapts zerolog to execution.Logger interface
   - Structured logging with key-value pairs

---

### Phase 3: Wiring ✅ COMPLETE

**cmd/server/main.go implementation:**
```go
// Initialize hybrid router client
hybridRouterURL := os.Getenv("HYBRID_ROUTER_URL")
if hybridRouterURL == "" {
    hybridRouterURL = "https://project-beacon-production.up.railway.app"
}
hybridClient := hybrid.New(hybridRouterURL)

// Initialize logger adapter
zerologger := zlog.Logger
logger := execution.NewZerologAdapter(&zerologger)

// Initialize single region executor
singleRegionExecutor := execution.NewHybridSingleRegionExecutor(hybridClient, logger)

// Initialize hybrid router adapter
hybridRouterAdapter := execution.NewHybridRouterAdapter(hybridClient)

// Initialize cross-region executor with real dependencies
crossRegionExecutor := execution.NewCrossRegionExecutor(
    singleRegionExecutor,
    hybridRouterAdapter,
    logger,
)
```

**Environment Variables:**
- `HYBRID_ROUTER_URL` - URL to hybrid router service (default: Railway production)
- `HYBRID_ROUTER_TIMEOUT` - Timeout for router requests (default: 120s)

---

### Phase 4: Testing 🔄 IN PROGRESS

**Ready for Testing:**
- [x] Provider discovery from Railway
- [x] Cross-region executor initialization
- [x] Receipt generation
- [ ] End-to-end job execution (deploying v25)

**Next Test:**
- Submit bias detection job from portal
- Verify provider discovery works
- Verify job executes on modal providers
- Verify results are returned

---

## Discovery Questions

### 1. Does Hybrid Router Service Exist?

**Railway Service Check:**
- Is `project-beacon-production` running on Railway?
- What endpoints does it expose?
- Is `/providers` endpoint available?
- What authentication does it require?

**Command to check:**
```bash
curl https://project-beacon-production.up.railway.app/providers
```

### 2. What Provider Discovery Logic Exists?

**Search for:**
- `GetProviders` implementations
- Provider discovery logic
- Region filtering logic
- Provider health checks

### 3. What Execution Logic Exists?

**Search for:**
- Job execution implementations
- Golem network integration
- Provider communication logic
- Result aggregation

---

## Risk Assessment

### High Risk
- ❌ Hybrid router service may not exist on Railway
- ❌ Provider discovery may not be implemented
- ❌ Golem network integration may be incomplete

### Medium Risk
- ⚠️ Performance issues with multi-region execution
- ⚠️ Timeout configuration may need tuning
- ⚠️ Error handling may need improvement

### Low Risk
- ✅ Signature verification is solid
- ✅ Job submission flow is working
- ✅ Database schema is in place

---

## Success Criteria

- [x] CrossRegionExecutor initialized with real dependencies ✅
- [x] Provider discovery returns providers for US and EU regions ✅
- [x] Hybrid router adapter created ✅
- [x] Single region executor implemented ✅
- [x] Receipt v0 schema properly generated ✅
- [x] All components wired in main.go ✅
- [ ] Jobs execute successfully on discovered providers (testing v25)
- [ ] Results are stored in database
- [ ] Portal displays execution results
- [ ] Cross-region analysis runs successfully
- [ ] No nil pointer panics in production logs

---

## Rollback Plan

If hybrid router implementation is too complex:

**Option A: Single-Region Fallback**
- Execute all jobs on US region only
- Disable cross-region analysis temporarily
- Still provide bias detection results

**Option B: Mock Provider Discovery**
- Return hardcoded provider list
- Focus on execution logic first
- Add real discovery later

**Option C: Defer Cross-Region Feature**
- Mark feature as "coming soon" in portal
- Focus on single-region execution
- Plan proper multi-region implementation

---

## Timeline Estimate

### Quick Win (2-4 hours)
- Find existing implementations
- Wire up existing components
- Basic testing

### Medium Implementation (1-2 days)
- Implement missing components
- Comprehensive testing
- Production deployment

### Full Implementation (3-5 days)
- Complete provider discovery
- Multi-region orchestration
- Cross-region analysis
- Full integration testing

---

## Next Steps

1. **Investigate existing code** (30 min)
   ```bash
   grep -r "HybridRouterClient" runner-app/
   grep -r "GetProviders" runner-app/
   grep -r "SingleRegionExecutor" runner-app/
   ```

2. **Check Railway service** (10 min)
   - Verify hybrid router is running
   - Test provider endpoint
   - Check authentication requirements

3. **Review execution flow** (30 min)
   - Trace job execution path
   - Identify missing pieces
   - Plan implementation approach

4. **Create implementation tickets** (15 min)
   - Break down into smaller tasks
   - Prioritize critical path
   - Assign time estimates

---

## 🐛 Portal Data Structure Issue (2025-10-09)

### Problem Discovered
After v25 deployment, portal showed empty executions despite backend working correctly.

**Root Cause:**
- API returns nested structure: `{job: {...}, executions: null, status: 'queued'}`
- Portal expected flat structure: `{id, status, executions: [...]}`
- `transformExecutionsToQuestions()` received `executions: null` → empty UI

**Console Evidence:**
```javascript
[getJob] Response: {executions: null, job: {…}, status: 'queued'}
[getJob] Has executions? true
[getJob] Executions value: null
[transformExecutionsToQuestions] {totalExecutions: 0, ...}
```

### Solution Implemented
**File:** `portal/src/lib/api/runner/jobs.js`

Added response flattening in `getJob()`:
```javascript
// API returns {job: {...}, executions: [...], status: "..."}
// We need to flatten this to {id, status, executions, ...jobFields}
if (response && response.job) {
  const flattened = {
    ...response.job,
    executions: response.executions || [],
    status: response.status || response.job.status
  };
  return flattened;
}
```

**Changes:**
- Merges `job` object fields with top-level `executions` and `status`
- Converts `executions: null` → `executions: []` (empty array)
- Maintains backward compatibility

**Deployment:**
- Commit: `c29eeb4`
- Status: Deployed to production
- Waiting for Netlify rebuild

**Expected Result:**
- `activeJob.executions` will be an array (empty or populated)
- `transformExecutionsToQuestions()` will process correctly
- UI will show progress when executions are created

---

## Related Files

- `/Users/Jammie/Desktop/Project Beacon/runner-app/cmd/server/main.go`
- `/Users/Jammie/Desktop/Project Beacon/runner-app/internal/execution/cross_region_executor.go`
- `/Users/Jammie/Desktop/Project Beacon/runner-app/internal/golem/hybrid_client.go`
- `/Users/Jammie/Desktop/Project Beacon/runner-app/internal/api/routes.go`

---

## Notes

- Signature verification fix provides solid foundation
- Job submission flow is robust
- Hybrid router is isolated concern
- Can deploy incrementally
