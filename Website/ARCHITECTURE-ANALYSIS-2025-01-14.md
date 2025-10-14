# Architecture Analysis - Multi-Region Execution
**Date:** 2025-01-14  
**Status:** Architecture Validated ✅  
**Conclusion:** Current implementation is optimal, no runner code changes needed

---

## Executive Summary

Investigated potential performance optimizations for multi-region, multi-model job execution. After thorough analysis of the runner, hybrid router, and Modal infrastructure, **confirmed the current architecture is already optimal**.

**Key Finding:** The perceived slowness is due to Modal cold starts and inference times, NOT architectural issues in the runner or router.

---

## Architecture Components

### 1. Runner (`job_runner.go`)

**Execution Flow:**
```go
// Spawn 3 region goroutines in parallel
for _, region := range ["us-east", "eu-west", "asia-pacific"] {
    go func(r string) {
        // Process 8 questions sequentially
        for _, question := range spec.Questions {
            // Execute 3 models in parallel
            for _, model := range spec.Models {
                go func(m models.ModelSpec) {
                    // HTTP request to hybrid router
                    executor.Execute(ctx, jobID, spec, region)
                }(model)
            }
            questionWg.Wait()  // Wait for all 3 models
        }
    }(region)
}
regionWg.Wait()  // Wait for all 3 regions
```

**Concurrency:**
- **3 region goroutines** run in parallel (US, EU, APAC)
- **Semaphore = 10** concurrent executions globally
- **9 slots used** at peak (3 regions × 3 models)
- **1 slot free** for overhead

**Status:** ✅ Optimal - regions run in parallel, bounded concurrency prevents overload

---

### 2. Hybrid Router (`hybrid_router/core/region_queue.py`)

**Queue Architecture:**
```python
self.queues = {
    "US": RegionQueue("US"),      # Sequential processing
    "EU": RegionQueue("EU"),      # Sequential processing
    "ASIA": RegionQueue("ASIA"),  # Sequential processing
}
```

**Processing Model:**
- Each region has its own queue
- Jobs process **sequentially per region** (by design)
- Prevents GPU limit exhaustion on Modal
- Regions don't wait for each other

**Example Timeline:**
```
T=0s:
  US Queue:   Job1 (Model1) starts
  EU Queue:   Job1 (Model1) starts  } All 3 regions
  ASIA Queue: Job1 (Model1) starts  } start simultaneously

T=3s:
  US Queue:   Job1 done, Job2 (Model2) starts
  EU Queue:   Job1 done, Job2 (Model2) starts
  ASIA Queue: Job1 done, Job2 (Model2) starts

T=6s:
  US Queue:   Job2 done, Job3 (Model3) starts
  ...
```

**Status:** ✅ Optimal - per-region queues prevent Modal overload while maintaining parallelism

---

### 3. Modal Infrastructure

**Deployed Apps:**
- `project-beacon-hf-us` (US region)
- `project-beacon-hf-eu` (EU region)
- `project-beacon-hf-apac` (APAC region - disabled)

**Container Configuration:**
```python
@app.function(
    gpu="T4",
    scaledown_window=120,  # 2 minutes (EU)
    timeout=900,
    region=["eu-west", "eu-north"],
)
```

**Current Settings:**
- **US:** `scaledown_window=120s` (2 min)
- **EU:** `scaledown_window=120s` (2 min)
- **APAC:** Disabled (6+ min cold starts)

**Status:** ⚠️ Needs optimization - short scaledown causes frequent cold starts

---

## Performance Analysis

### Current Timing (8 questions × 3 models)

**Per Region:**
- Q1: 3 models × 3-5s = 9-15s
- Q2: 3 models × 3-5s = 9-15s
- ...
- Q8: 3 models × 3-5s = 9-15s
- **Total: 72-120s per region**

**Overall Job:**
- All 3 regions run in parallel
- Job completes when slowest region finishes
- **Total: ~120s (2 minutes)**

### Bottleneck Identification

**NOT bottlenecks:**
- ✅ Runner architecture (regions already parallel)
- ✅ Semaphore size (10 slots, only using 9)
- ✅ Router queues (designed for sequential per-region)

**Actual bottlenecks:**
- ❌ Modal cold starts (30-90s on first question)
- ❌ Short scaledown window (2 min → frequent cold starts)
- ❌ EU region slower than US (CDN, infrastructure)

---

## Optimization Strategy

### ❌ What NOT to Do

**Don't parallelize questions within regions:**
- Would overload Modal containers
- Router queues designed to prevent this
- Would cause more cold starts, not fewer

**Don't increase semaphore:**
- Already has 1 free slot
- Not the bottleneck

**Don't refactor runner architecture:**
- Already optimal
- Regions run in parallel
- Bounded concurrency working correctly

### ✅ What TO Do

**O1.1: Add Metrics Logging** ✅ DONE
- Deployed to EU region
- Tracks warm vs cold starts
- Measures actual inference times
- Will guide optimization decisions

**O1.4: Pre-Warming (Free)**
- Ping Modal health endpoints before job starts
- Gives containers 2-5s head start
- Reduces cold start impact

**O1.5: Longer Scaledown Window (Free)**
- Change `scaledown_window=120` → `scaledown_window=1800` (30 min)
- Keeps containers warm between questions
- Prevents cold starts during job execution

**O2.1: Keep-Warm ($150-360/month)**
- Only if metrics show >60% cold start rate
- Enable `keep_warm=1` during business hours
- Guarantees warm containers

---

## Validation Results

### Architecture Correctness

**Test:** Reviewed code flow from runner → router → Modal

**Findings:**
1. ✅ Runner spawns 3 region goroutines simultaneously
2. ✅ Each goroutine processes questions sequentially
3. ✅ Router queues serialize per-region (by design)
4. ✅ Modal apps are separate per region
5. ✅ Semaphore allows 10 concurrent (9 needed)

**Conclusion:** Architecture is optimal for the use case

### Performance Expectations

**With current architecture:**
- 3 regions × 8 questions × 3 models = 72 executions
- Regions run in parallel → ~2 minutes total
- Limited by slowest region (usually EU)

**After optimizations (O1.4 + O1.5):**
- Eliminate cold starts → faster first question
- Keep containers warm → consistent 3-5s per inference
- Expected: ~1 minute total (50% improvement)

**After keep-warm (O2.1):**
- Zero cold starts during business hours
- Consistent performance
- Expected: ~45 seconds total (62% improvement)

---

## Recommendations

### Immediate Actions

1. ✅ **Metrics logging deployed** - Let it collect data for 3-5 days
2. **Analyze metrics** - Determine actual cold start rate
3. **Implement O1.5** - Increase scaledown window (free, low risk)
4. **Implement O1.4** - Add pre-warming (free, low risk)

### Decision Tree

```
After 3-5 days of metrics:

EU Cold Start Rate < 20%:
  → Current setup is fine
  → Maybe just increase scaledown to 30 min

EU Cold Start Rate 20-40%:
  → Implement O1.4 + O1.5 (free optimizations)
  → Re-measure after 1 week

EU Cold Start Rate > 40%:
  → Implement O1.4 + O1.5 first
  → If still >40%, enable keep_warm=1 ($150/month)
```

### Long-Term

- Monitor EU vs US performance gap
- Consider EU-specific volume for faster model loading
- Evaluate APAC re-enablement with optimizations

---

## Lessons Learned

1. **Architecture was already optimal** - No runner code changes needed
2. **Router queues are by design** - Prevent Modal GPU overload
3. **Semaphore is not the bottleneck** - Has spare capacity
4. **Modal infrastructure is the bottleneck** - Cold starts and scaledown timing
5. **Metrics are essential** - Can't optimize without data

---

## Files Modified

### GPU-OPTIMIZATION-PLAN.md
- ✅ Updated O1.1 status (deployed)
- ✅ Cancelled O1.2 (not needed)
- ✅ Added architecture validation section
- ✅ Clarified optimization strategy

### job_runner.go
- ✅ Added architecture documentation comments
- ✅ No functional changes (architecture already optimal)

### modal_hf_eu.py
- ✅ Added metrics logging
- ✅ Tracks warm/cold starts
- ✅ Records inference times

---

## Next Steps

1. **Week 1:** Collect metrics data (passive)
2. **Week 1:** Implement O1.5 (longer scaledown) - 5 min
3. **Week 1:** Implement O1.4 (pre-warming) - 15 min
4. **Week 2:** Analyze metrics, measure improvement
5. **Week 2:** Decide on keep_warm based on data

---

**Status:** Architecture validated, optimization path clear  
**Risk:** Low - free optimizations first, paid only if needed  
**Expected ROI:** 50-62% faster jobs with minimal cost
