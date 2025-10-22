# 🔬 Execution Failure Debug Plan - ENHANCED

**Date**: 2025-10-22  
**Issue**: All jobs failing in 10-30ms with empty output/receipt data  
**Priority**: CRITICAL - Production is broken  
**Enhancement**: Additional systematic diagnostic layers

---

## 🎯 Key Insight

**10-30ms failures = Pre-execution failure**
- Not enough time for network round-trip to hybrid router
- Not enough time for Modal cold start
- Not enough time for actual inference
- **Likely**: Configuration issue, nil pointer, or immediate error return

---

## 🔍 **PHASE 0: Pre-Flight Configuration Audit (NEW - 15 minutes)**

These checks MUST run first to verify basic setup:

### **Step 0.1: Verify Hybrid Client Initialization**

```bash
# Check if hybrid router env vars are set
flyctl ssh console -a beacon-runner-production -C '
  echo "=== HYBRID CONFIG ==="
  echo "HYBRID_BASE: $HYBRID_BASE"
  echo "HYBRID_ROUTER_URL: $HYBRID_ROUTER_URL"
  echo "ENABLE_HYBRID_DEFAULT: $ENABLE_HYBRID_DEFAULT"
  echo "HYBRID_ROUTER_DISABLE: $HYBRID_ROUTER_DISABLE"
  echo "HYBRID_ROUTER_TIMEOUT: $HYBRID_ROUTER_TIMEOUT"
  echo "HYBRID_TIMEOUT: $HYBRID_TIMEOUT"
'
```

**Expected**: At least ONE of `HYBRID_BASE` or `HYBRID_ROUTER_URL` should be set  
**If empty**: Hybrid client will NOT be initialized → executor will be Golem (which may be misconfigured)

### **Step 0.2: Check Startup Logs for Hybrid Initialization**

```bash
# Check logs from last deployment
flyctl logs -a beacon-runner-production | grep -i "hybrid"

# Look for:
# ✅ "Hybrid Router enabled" with base URL
# ❌ "Hybrid Router disabled"
# ✅ "[HYBRID_CLIENT_INIT]" with timeout settings
```

**Critical Flags**:
- ✅ `Hybrid Router enabled hybrid_base=https://...` = Good
- ❌ `Hybrid Router disabled` = Executor is NOT using hybrid router
- ❌ No hybrid logs = Client never initialized

### **Step 0.3: Verify Executor Type in Memory**

Add temporary endpoint to check executor type:

```go
// In cmd/runner/main.go after jr.SetHybridClient():
r.GET("/debug/executor", func(c *gin.Context) {
    executorType := "nil"
    if jr.Executor != nil {
        executorType = fmt.Sprintf("%T", jr.Executor)
    }
    c.JSON(200, gin.H{
        "executor_type": executorType,
        "hybrid_client_set": jr.Hybrid != nil,
        "hybrid_base": func() string {
            if jr.Hybrid != nil {
                return jr.Hybrid.BaseURL
            }
            return "n/a"
        }(),
    })
})
```

Then test:
```bash
curl https://beacon-runner-production.fly.dev/debug/executor
```

**Expected**: `"executor_type": "*worker.HybridExecutor"`  
**If**: `"executor_type": "*worker.GolemExecutor"` → Hybrid not configured

---

## 🔍 **PHASE 1: Identify Failure Point (15 minutes)**

Your existing Phase 1 is good, but add these:

### **Step 1.5: Check Circuit Breaker State**

```bash
# Check if circuit breaker is blocking requests
curl https://beacon-runner-production.fly.dev/api/v1/metrics | grep -E "circuit|breaker"

# Look for: hybrid_circuit_breaker_state{state="open"}
# open = 1 means circuit is BLOCKING requests
# closed = 1 means circuit is allowing requests
```

### **Step 1.6: Test Hybrid Router from Runner Container**

```bash
# SSH into runner and test connectivity
flyctl ssh console -a beacon-runner-production -C '
  # Test DNS resolution
  nslookup project-beacon-production.up.railway.app
  
  # Test HTTP connectivity (should return quickly)
  time curl -I https://project-beacon-production.up.railway.app/health
  
  # Test inference endpoint (might timeout, that's OK)
  timeout 5 curl -X POST https://project-beacon-production.up.railway.app/inference \
    -H "Content-Type: application/json" \
    -d "{\"model\":\"llama3.2-1b\",\"prompt\":\"test\",\"temperature\":0.1,\"max_tokens\":100,\"cost_priority\":true}"
'
```

**Expected**: DNS resolution and /health should work  
**If DNS fails**: Network configuration issue  
**If /health times out**: Router is down or blocked

---

## 🔍 **PHASE 2: Enhanced Diagnostic Logging (30 minutes)**

Your existing Phase 2 is good. Add these enhancements:

### **Step 2.1-ENHANCED: Add Nil Checks with Panic Recovery**

Modify `internal/worker/job_runner.go` `executeQuestion()`:

```go
func (w *JobRunner) executeQuestion(ctx context.Context, jobID string, spec *models.JobSpec, region string, modelID string, questionID string, executor Executor) ExecutionResult {
    l := logging.FromContext(ctx)
    
    // 🔍 CRITICAL: Add panic recovery
    defer func() {
        if r := recover(); r != nil {
            l.Error().
                Interface("panic", r).
                Str("job_id", jobID).
                Str("region", region).
                Msg("🚨 PANIC RECOVERED in executeQuestion")
            
            // Capture stack trace
            buf := make([]byte, 4096)
            n := runtime.Stack(buf, false)
            l.Error().Str("stack", string(buf[:n])).Msg("panic stack trace")
        }
    }()
    
    // 🔍 TRACE: Entry point
    l.Info().
        Str("job_id", jobID).
        Str("region", region).
        Str("model_id", modelID).
        Str("question_id", questionID).
        Str("executor_type", fmt.Sprintf("%T", executor)).
        Bool("executor_nil", executor == nil).
        Msg("🔍 TRACE: executeQuestion ENTRY")
    
    regionStart := time.Now()
    
    // 🔍 CRITICAL: Check executor is not nil
    if executor == nil {
        l.Error().
            Str("job_id", jobID).
            Str("region", region).
            Msg("🚨 CRITICAL: executor is NIL!")
        
        return ExecutionResult{
            Region:      region,
            Status:      "failed",
            ModelID:     modelID,
            QuestionID:  questionID,
            StartedAt:   regionStart.UTC(),
            CompletedAt: time.Now().UTC(),
            Error:       fmt.Errorf("executor is nil - hybrid client not initialized"),
        }
    }
    
    // ... rest of function continues at line 840 ...
```

### **Step 2.2-ENHANCED: Add Detailed Executor Call Logging**

Replace line 914 in `internal/worker/job_runner.go`:

```go
    // 🔍 TRACE: Before executor call
    l.Info().
        Str("job_id", jobID).
        Str("region", region).
        Str("model_id", modelID).
        Str("question_id", questionID).
        Int("prompt_len", len(singleQuestionSpec.Benchmark.Prompts)).
        Time("execution_start", executionStart).
        Msg("🔍 TRACE: Calling executor.Execute()")
    
    executionStart := time.Now()
    
    // Execute job in this region
    providerID, status, outputJSON, receiptJSON, err := executor.Execute(ctx, &singleQuestionSpec, region)
    
    executionEnd := time.Now()
    executionDuration := executionEnd.Sub(executionStart)
    
    // 🔍 TRACE: After executor call
    l.Info().
        Str("job_id", jobID).
        Str("region", region).
        Str("model_id", modelID).
        Str("question_id", questionID).
        Str("provider_id", providerID).
        Str("status", status).
        Dur("duration_ms", executionDuration).
        Int64("duration_nanosec", executionDuration.Nanoseconds()).
        Int("output_len", len(outputJSON)).
        Int("receipt_len", len(receiptJSON)).
        Bool("has_error", err != nil).
        Err(err).
        Msg("🔍 TRACE: executor.Execute() RETURNED")
    
    // 🔍 CRITICAL: Check for suspiciously fast execution
    if executionDuration < 100*time.Millisecond {
        l.Warn().
            Dur("duration", executionDuration).
            Str("status", status).
            Msg("🚨 SUSPICIOUS: Execution completed in <100ms - likely immediate failure")
    }
```

### **Step 2.3-ENHANCED: Add Hybrid Client Logging**

Modify `internal/worker/executor_hybrid.go` `Execute()`:

```go
func (e *HybridExecutor) Execute(ctx context.Context, spec *models.JobSpec, region string) (string, string, []byte, []byte, error) {
    l := logging.FromContext(ctx)
    
    // 🔍 TRACE: Entry with full context
    l.Info().
        Str("region", region).
        Str("client_type", fmt.Sprintf("%T", e.Client)).
        Bool("client_nil", e.Client == nil).
        Str("base_url", func() string {
            if e.Client != nil {
                return e.Client.BaseURL
            }
            return "<client_nil>"
        }()).
        Int("num_questions", len(spec.Questions)).
        Int("num_models", len(spec.Models)).
        Msg("🔍 TRACE: HybridExecutor.Execute ENTRY")
    
    // 🔍 CRITICAL: Check client is not nil
    if e.Client == nil {
        l.Error().Msg("🚨 CRITICAL: hybrid client is NIL!")
        return "", "failed", nil, nil, fmt.Errorf("hybrid client is nil")
    }
    
    // ... rest of function continues ...
```

### **Step 2.4-ENHANCED: Add HTTP Client Logging**

Modify `internal/hybrid/client.go` `RunInference()` method (around line 120):

```go
func (c *Client) RunInference(ctx context.Context, req InferenceRequest) (*InferenceResponse, error) {
    url := c.baseURL + "/inference"
    
    // 🔍 TRACE: HTTP request details
    fmt.Printf("[HYBRID_CLIENT] 🔍 Starting inference request\n")
    fmt.Printf("[HYBRID_CLIENT]   URL: %s\n", url)
    fmt.Printf("[HYBRID_CLIENT]   Model: %s\n", req.Model)
    fmt.Printf("[HYBRID_CLIENT]   Region: %s\n", req.RegionPreference)
    fmt.Printf("[HYBRID_CLIENT]   Timeout: %v\n", c.httpClient.Timeout)
    fmt.Printf("[HYBRID_CLIENT]   Prompt length: %d chars\n", len(req.Prompt))
    
    startTime := time.Now()
    
    payload, err := json.Marshal(req)
    if err != nil {
        fmt.Printf("[HYBRID_CLIENT] ❌ JSON marshal error: %v\n", err)
        return nil, &HybridError{Type: ErrorTypeJSON, Message: err.Error()}
    }
    
    httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
    if err != nil {
        fmt.Printf("[HYBRID_CLIENT] ❌ Request creation error: %v\n", err)
        return nil, &HybridError{Type: ErrorTypeNetwork, Message: err.Error()}
    }
    httpReq.Header.Set("Content-Type", "application/json")
    
    fmt.Printf("[HYBRID_CLIENT] 🚀 Sending HTTP request...\n")
    
    resp, err := c.httpClient.Do(httpReq)
    
    elapsed := time.Since(startTime)
    
    if err != nil {
        fmt.Printf("[HYBRID_CLIENT] ❌ HTTP error after %v: %v\n", elapsed, err)
        
        // Categorize error
        if ctx.Err() == context.DeadlineExceeded {
            return nil, &HybridError{Type: ErrorTypeTimeout, Message: err.Error(), URL: url}
        }
        return nil, &HybridError{Type: ErrorTypeNetwork, Message: err.Error(), URL: url}
    }
    defer resp.Body.Close()
    
    fmt.Printf("[HYBRID_CLIENT] ✅ HTTP response received after %v\n", elapsed)
    fmt.Printf("[HYBRID_CLIENT]   Status: %d\n", resp.StatusCode)
    
    // ... rest of function continues ...
```

---

## 🔍 **PHASE 3: Analyze Results (10 minutes)**

Add these scenarios to your existing Phase 3:

**Scenario F: "executor is NIL" in logs**
- **Root Cause**: `SetHybridClient()` never called OR called with nil
- **Fix**: Check `HYBRID_BASE` / `HYBRID_ROUTER_URL` environment variables
- **Action**: Set proper env vars in Fly.io secrets

**Scenario G: "hybrid client is NIL" in logs**
- **Root Cause**: Executor created but client field is nil
- **Fix**: Check `NewHybridExecutor()` implementation
- **Action**: Verify hybrid.New() returns non-nil client

**Scenario H: Execution < 100ms with no HTTP logs**
- **Root Cause**: Executor.Execute() returns immediately without making HTTP call
- **Fix**: Check for early return paths in executor_hybrid.go
- **Action**: Review error handling in HybridExecutor

**Scenario I: HTTP logs show connection refused**
- **Root Cause**: Hybrid router URL is wrong or service is down
- **Fix**: Verify hybrid router deployment and DNS
- **Action**: Check Railway deployment status

**Scenario J: HTTP logs show timeout**
- **Root Cause**: Hybrid router is slow or unresponsive
- **Fix**: Check hybrid router logs for errors
- **Action**: Investigate Railway service health

---

## 🔍 **PHASE 4: Root Cause Specific Fixes (30 minutes)**

Add these fixes to your existing Phase 4:

**Fix F: Missing Hybrid Configuration**
```bash
# Set hybrid router URL
flyctl secrets set HYBRID_BASE=https://project-beacon-production.up.railway.app -a beacon-runner-production

# Or use environment variable
flyctl secrets set HYBRID_ROUTER_URL=https://project-beacon-production.up.railway.app -a beacon-runner-production

# Verify
flyctl secrets list -a beacon-runner-production | grep HYBRID
```

**Fix G: Executor Not Initialized**
```go
// Verify in cmd/runner/main.go around line 243:
if base != "" {
    hybridClient := hybrid.New(base)
    if hybridClient != nil {
        jr.SetHybridClient(hybridClient)
        logger.Info().Str("hybrid_base", base).Msg("Hybrid Router enabled")
    } else {
        logger.Error().Str("hybrid_base", base).Msg("Failed to create hybrid client")
    }
}
```

**Fix H: Circuit Breaker Stuck Open**
```bash
# Check circuit breaker metrics
curl https://beacon-runner-production.fly.dev/api/v1/metrics | grep circuit_breaker

# If stuck open, restart to reset
flyctl apps restart beacon-runner-production
```

**Fix I: HTTP Client Timeout Too Short**
```bash
# Set longer timeout for Modal cold starts
flyctl secrets set HYBRID_ROUTER_TIMEOUT=300 -a beacon-runner-production

# Or use alternative env var
flyctl secrets set HYBRID_TIMEOUT=300 -a beacon-runner-production
```

---

## 🔬 **PHASE 5: Deep Dive Analysis (NEW - 20 minutes)**

If failures persist, investigate these:

### **Step 5.1: Check for Context Cancellation**

Add context cancellation tracking:

```go
// In executeQuestion(), before executor.Execute():
go func() {
    <-ctx.Done()
    l.Warn().
        Str("job_id", jobID).
        Str("region", region).
        Err(ctx.Err()).
        Msg("🚨 Context cancelled during execution")
}()
```

### **Step 5.2: Check for Goroutine Leaks**

```bash
# Check goroutine count
curl https://beacon-runner-production.fly.dev/debug/pprof/goroutine?debug=1 | head -20

# If > 1000 goroutines: resource leak
```

### **Step 5.3: Check Database Transaction State**

```bash
# Check for long-running transactions
psql "$DATABASE_URL" -c "
SELECT pid, now() - pg_stat_activity.query_start AS duration, query, state
FROM pg_stat_activity
WHERE state != 'idle'
ORDER BY duration DESC
LIMIT 10;
"
```

### **Step 5.4: Check Memory Usage**

```bash
# Check if runner is OOM'ing
flyctl ssh console -a beacon-runner-production -C 'free -m'

# Check Go heap allocations
curl https://beacon-runner-production.fly.dev/debug/pprof/heap?debug=1 | head -50
```

---

## 📊 **Diagnostic Decision Tree**

```
Job fails in 10-30ms with empty receipt?
│
├─ Check executor initialization logs
│  │
│  ├─ "executor is NIL" → Missing HYBRID_BASE env var
│  │   └─ Fix: Set HYBRID_BASE or HYBRID_ROUTER_URL
│  │
│  ├─ "hybrid client is NIL" → Client creation failed
│  │   └─ Fix: Check hybrid.New() implementation
│  │
│  ├─ Executor type is GolemExecutor → Wrong executor
│  │   └─ Fix: Set hybrid configuration
│  │
│  └─ No executor logs → Never reached executeQuestion()
│      └─ Fix: Check job queue processing
│
├─ Check HTTP client logs
│  │
│  ├─ No HTTP logs → Never made request
│  │   └─ Check for early return in executor
│  │
│  ├─ "connection refused" → Router down
│  │   └─ Fix: Check Railway deployment
│  │
│  ├─ "timeout" → Router too slow
│  │   └─ Fix: Increase HYBRID_TIMEOUT
│  │
│  └─ HTTP 5xx errors → Router errors
│      └─ Check: Railway service logs
│
└─ Check circuit breaker state
   │
   ├─ Circuit open → Auto-blocked
   │   └─ Fix: Reset or increase threshold
   │
   └─ Circuit closed → Should be working
       └─ Check: Provider health
```

---

## ⚡ **Quick Win Checklist**

Run these in order for fastest diagnosis:

1. ✅ **Check hybrid env vars** (30 seconds)
   ```bash
   flyctl ssh console -a beacon-runner-production -C 'env | grep HYBRID'
   ```

2. ✅ **Check executor type** (1 minute)
   ```bash
   flyctl logs -a beacon-runner-production | grep -E "(Hybrid|executor)" | tail -10
   ```

3. ✅ **Test hybrid router** (1 minute)
   ```bash
   curl https://project-beacon-production.up.railway.app/health
   ```

4. ✅ **Check recent job logs** (2 minutes)
   ```bash
   flyctl logs -a beacon-runner-production | grep "🔍 TRACE" | tail -20
   ```

5. ✅ **Check circuit breaker** (1 minute)
   ```bash
   curl https://beacon-runner-production.fly.dev/api/v1/metrics | grep circuit
   ```

**If ALL pass**: Problem is deeper, proceed with full Phase 2 logging

---

## 🎯 **Success Criteria**

- ✅ **Phase 0 Complete**: Hybrid client is initialized and executor type is correct
- ✅ **Phase 1 Complete**: Know which component is failing
- ✅ **Phase 2 Complete**: Have detailed trace logs showing exact failure point
- ✅ **Phase 3 Complete**: Understand root cause
- ✅ **Phase 4 Complete**: Fix applied and jobs execute successfully
- ✅ **Phase 5 Complete**: System is stable with no memory/context leaks

---

## 📈 **Expected Outcomes**

### **Most Likely Root Causes** (ordered by probability):

1. **80% chance**: Missing `HYBRID_BASE` or `HYBRID_ROUTER_URL` environment variable
2. **10% chance**: Hybrid router is down or unreachable
3. **5% chance**: Circuit breaker stuck in open state
4. **3% chance**: HTTP client timeout too short
5. **2% chance**: Actual code bug in executor

### **Timeline**:

- **Phase 0**: 15 minutes (configuration audit)
- **Phase 1**: 15 minutes (health checks)
- **Phase 2**: 30 minutes (add logging + deploy)
- **Phase 3**: 10 minutes (analyze logs)
- **Phase 4**: 30 minutes (implement fix)
- **Phase 5**: 20 minutes (deep dive if needed)

**Total**: 2 hours maximum, but likely resolved in first 30-60 minutes

---

## 🚀 **Deployment Commands**

```bash
# Quick deployment workflow
cd /Users/Jammie/Desktop/Project\ Beacon/runner-app

# Test build locally
go build ./...

# Deploy to Fly
flyctl deploy

# Watch logs
flyctl logs -a beacon-runner-production -f | grep -E "(TRACE|CRITICAL|HYBRID)"
```

---

## 📝 **Next Steps**

1. **START HERE**: Run "Quick Win Checklist" (5 minutes)
2. If inconclusive: Run **Phase 0** configuration audit
3. If still failing: Add **Phase 2** enhanced logging
4. Deploy and analyze logs
5. Apply targeted fix from **Phase 4**
6. Verify with test job submission

---

**Status**: ENHANCED - Ready for systematic debugging  
**Confidence**: VERY HIGH - comprehensive diagnostic coverage  
**Estimated Resolution**: 30-120 minutes
