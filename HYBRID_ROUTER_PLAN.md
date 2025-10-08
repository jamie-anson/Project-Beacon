# Hybrid Router Initialization Plan

**Date:** 2025-10-08  
**Issue:** Cross-region jobs fail with "hybrid router not initialized"  
**Severity:** HIGH - Blocks cross-region job execution  
**Status:** 🔍 PLANNING

---

## Problem Summary

After signature verification was fixed, cross-region jobs are now accepted (202) but fail during execution with:
```
hybrid router not initialized - cross-region execution not available
```

**Current State:**
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

## Implementation Plan

### Phase 1: Identify Existing Implementations

**Goals:**
- Find if HybridRouterClient implementation exists
- Find if SingleRegionExecutor implementation exists
- Understand current provider discovery mechanism

**Tasks:**
1. Search for HybridRouterClient interface definition
2. Search for HybridRouterClient implementations
3. Search for SingleRegionExecutor implementations
4. Review provider discovery endpoints
5. Check Railway hybrid-router service status

**Deliverables:**
- List of existing implementations
- Architecture diagram
- Gap analysis

---

### Phase 2: Create Missing Implementations

**If HybridRouterClient doesn't exist:**
- [ ] Define HybridRouterClient interface
- [ ] Implement basic provider discovery
- [ ] Add region filtering
- [ ] Add health checks
- [ ] Add connection pooling

**If SingleRegionExecutor doesn't exist:**
- [ ] Define SingleRegionExecutor interface
- [ ] Implement job execution on single provider
- [ ] Add timeout handling
- [ ] Add error handling
- [ ] Add retry logic

**Logger:**
- [ ] Use existing zerolog logger from config
- [ ] Add structured logging fields
- [ ] Add tracing integration

---

### Phase 3: Wire Up Dependencies

**cmd/server/main.go updates:**
```go
// Initialize hybrid router client
hybridRouterURL := os.Getenv("HYBRID_ROUTER_URL") // e.g., https://project-beacon-production.up.railway.app
if hybridRouterURL == "" {
    hybridRouterURL = "https://project-beacon-production.up.railway.app"
}
hybridClient := golem.NewHybridClient(hybridRouterURL, 120*time.Second)

// Initialize logger
logger := log.Logger // Use zerolog

// Initialize single region executor
singleRegionExecutor := execution.NewSingleRegionExecutor(/* deps */)

// Initialize cross-region executor with proper dependencies
crossRegionExecutor := execution.NewCrossRegionExecutor(
    singleRegionExecutor,
    hybridClient,
    logger,
)
```

**Environment Variables:**
- `HYBRID_ROUTER_URL` - URL to hybrid router service
- `HYBRID_ROUTER_TIMEOUT` - Timeout for router requests (default: 120s)

---

### Phase 4: Testing

**Unit Tests:**
- [ ] Test HybridRouterClient with mock responses
- [ ] Test SingleRegionExecutor with mock providers
- [ ] Test CrossRegionExecutor orchestration
- [ ] Test error handling paths

**Integration Tests:**
- [ ] Test provider discovery from hybrid router
- [ ] Test cross-region job execution
- [ ] Test failover scenarios
- [ ] Test timeout handling

**End-to-End Test:**
- [ ] Submit bias detection job from portal
- [ ] Verify job executes across regions
- [ ] Verify results are stored correctly
- [ ] Verify portal displays results

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

- [ ] CrossRegionExecutor initialized with real dependencies
- [ ] Provider discovery returns providers for US and EU regions
- [ ] Jobs execute successfully on discovered providers
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
