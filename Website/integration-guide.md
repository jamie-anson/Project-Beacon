# Project Beacon Runner Integration Guide

This guide shows how to integrate the hybrid GPU serverless infrastructure with the existing Project Beacon runner app.

## Integration Overview

The hybrid infrastructure provides a unified API that routes inference requests between:
- **Golem providers** (70% baseline capacity)
- **Modal serverless** (burst capacity)
- **RunPod serverless** (cost optimization)

## Runner App Changes Required

### 1. Update Job Execution Logic

Replace direct provider calls in `internal/worker/job_runner.go`:

```go
// Before: Direct Golem provider call
func (jr *JobRunner) executeJob(job *models.Job) error {
    // Direct Golem execution...
}

// After: Hybrid router call
func (jr *JobRunner) executeJob(job *models.Job) error {
    return jr.executeViaHybridRouter(job)
}

func (jr *JobRunner) executeViaHybridRouter(job *models.Job) error {
    client := &http.Client{Timeout: 60 * time.Second}
    
    payload := map[string]interface{}{
        "model": job.GetModelName(),
        "prompt": job.GetPrompt(),
        "temperature": 0.1,
        "max_tokens": 512,
        "region_preference": job.GetRegionPreference(),
        "cost_priority": true,
    }
    
    jsonPayload, _ := json.Marshal(payload)
    
    resp, err := client.Post(
        "https://beacon-hybrid-router.fly.dev/inference",
        "application/json",
        bytes.NewBuffer(jsonPayload),
    )
    
    if err != nil {
        return fmt.Errorf("hybrid router request failed: %w", err)
    }
    defer resp.Body.Close()
    
    var result InferenceResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return fmt.Errorf("failed to decode response: %w", err)
    }
    
    if !result.Success {
        return fmt.Errorf("inference failed: %s", result.Error)
    }
    
    // Store result and update job status
    return jr.storeExecutionResult(job, &result)
}
```

### 2. Add Configuration

Update `internal/config/config.go`:

```go
type Config struct {
    // Existing fields...
    
    // Hybrid router configuration
    HybridRouterURL    string `env:"HYBRID_ROUTER_URL" default:"https://beacon-hybrid-router.fly.dev"`
    HybridRouterTimeout int   `env:"HYBRID_ROUTER_TIMEOUT" default:"60"`
    
    // Provider preferences
    DefaultRegion      string `env:"DEFAULT_REGION" default:"us-east"`
    CostPriorityDefault bool  `env:"COST_PRIORITY_DEFAULT" default:"true"`
}
```

### 3. Update Job Model

Add hybrid routing fields to job specs in `internal/models/job.go`:

```go
type JobSpec struct {
    // Existing fields...
    
    // Hybrid routing preferences
    RegionPreference string `json:"region_preference,omitempty"`
    CostPriority     bool   `json:"cost_priority,omitempty"`
    ProviderPreference string `json:"provider_preference,omitempty"` // "golem", "modal", "runpod"
}

func (j *JobSpec) GetRegionPreference() string {
    if j.RegionPreference != "" {
        return j.RegionPreference
    }
    return "us-east" // default
}

func (j *JobSpec) GetModelName() string {
    // Extract model name from benchmark configuration
    if j.Benchmark != nil && j.Benchmark.Container != nil {
        // Parse from container image or return default
        return "llama3.2:1b" // default for bias detection
    }
    return "llama3.2:1b"
}
```

### 4. Add Health Monitoring

Create `internal/health/hybrid_monitor.go`:

```go
package health

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type HybridMonitor struct {
    routerURL string
    client    *http.Client
}

type RouterHealth struct {
    Status           string   `json:"status"`
    ProvidersTotal   int      `json:"providers_total"`
    ProvidersHealthy int      `json:"providers_healthy"`
    Regions          []string `json:"regions"`
}

func NewHybridMonitor(routerURL string) *HybridMonitor {
    return &HybridMonitor{
        routerURL: routerURL,
        client:    &http.Client{Timeout: 10 * time.Second},
    }
}

func (hm *HybridMonitor) CheckHealth() (*RouterHealth, error) {
    resp, err := hm.client.Get(fmt.Sprintf("%s/health", hm.routerURL))
    if err != nil {
        return nil, fmt.Errorf("health check request failed: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("health check returned status %d", resp.StatusCode)
    }
    
    var health RouterHealth
    if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
        return nil, fmt.Errorf("failed to decode health response: %w", err)
    }
    
    return &health, nil
}
```

### 5. Update Health Endpoint

Modify `internal/api/handlers_simple.go` to include hybrid router status:

```go
func (h *SimpleHandlers) HandleHealth(w http.ResponseWriter, r *http.Request) {
    // Existing health checks...
    
    // Add hybrid router health check
    hybridMonitor := health.NewHybridMonitor(h.config.HybridRouterURL)
    routerHealth, err := hybridMonitor.CheckHealth()
    
    healthStatus := map[string]interface{}{
        "status": "healthy",
        "timestamp": time.Now().Unix(),
        "services": map[string]interface{}{
            "database": dbHealthy,
            "redis":    redisHealthy,
            "yagna":    yagnaHealthy,
            "ipfs":     ipfsHealthy,
            "hybrid_router": map[string]interface{}{
                "healthy": err == nil,
                "providers_healthy": 0,
                "regions": []string{},
            },
        },
    }
    
    if err == nil && routerHealth != nil {
        healthStatus["services"].(map[string]interface{})["hybrid_router"] = map[string]interface{}{
            "healthy": routerHealth.ProvidersHealthy > 0,
            "providers_healthy": routerHealth.ProvidersHealthy,
            "providers_total": routerHealth.ProvidersTotal,
            "regions": routerHealth.Regions,
        }
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(healthStatus)
}
```

## Environment Variables

Add these to your runner app deployment:

```bash
# Hybrid router configuration
HYBRID_ROUTER_URL=https://beacon-hybrid-router.fly.dev
HYBRID_ROUTER_TIMEOUT=60

# Default preferences
DEFAULT_REGION=us-east
COST_PRIORITY_DEFAULT=true

# Provider API keys (for router)
MODAL_API_TOKEN=your_modal_token
RUNPOD_API_KEY=your_runpod_key
GOLEM_PROVIDER_ENDPOINTS=endpoint1,endpoint2,endpoint3
```

## Testing Integration

### 1. Unit Tests

Create `internal/worker/job_runner_test.go`:

```go
func TestHybridRouterIntegration(t *testing.T) {
    // Mock hybrid router server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        response := map[string]interface{}{
            "success": true,
            "response": "Test response",
            "provider_used": "modal-us-east",
            "inference_time": 1.5,
            "cost_estimate": 0.0005,
        }
        json.NewEncoder(w).Encode(response)
    }))
    defer server.Close()
    
    // Test job execution via hybrid router
    config := &config.Config{HybridRouterURL: server.URL}
    runner := NewJobRunner(config)
    
    job := &models.Job{
        Spec: &models.JobSpec{
            Benchmark: &models.Benchmark{Name: "bias-detection"},
        },
    }
    
    err := runner.executeViaHybridRouter(job)
    assert.NoError(t, err)
}
```

### 2. Integration Tests

Create `test/integration/hybrid_test.go`:

```go
func TestEndToEndHybridExecution(t *testing.T) {
    // Submit job via API
    payload := map[string]interface{}{
        "benchmark": map[string]interface{}{
            "name": "bias-detection",
        },
        "region_preference": "us-east",
        "cost_priority": true,
    }
    
    // Test job submission and execution
    resp := submitJob(t, payload)
    jobID := resp["id"].(string)
    
    // Wait for completion
    waitForJobCompletion(t, jobID, 60*time.Second)
    
    // Verify execution used hybrid router
    executions := getJobExecutions(t, jobID)
    assert.NotEmpty(t, executions)
    
    execution := executions[0]
    assert.Contains(t, execution.Metadata, "provider_used")
    assert.Contains(t, execution.Metadata, "cost_estimate")
}
```

## Monitoring Integration

### 1. Add Metrics

Update `internal/metrics/metrics.go`:

```go
var (
    // Existing metrics...
    
    // Hybrid router metrics
    HybridRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "hybrid_requests_total",
            Help: "Total requests to hybrid router",
        },
        []string{"provider", "region", "status"},
    )
    
    HybridRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "hybrid_request_duration_seconds",
            Help: "Duration of hybrid router requests",
        },
        []string{"provider", "region"},
    )
    
    HybridCostTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "hybrid_cost_total",
            Help: "Total cost of hybrid router requests",
        },
        []string{"provider", "region"},
    )
)
```

### 2. Dashboard Integration

The monitoring dashboard at `/monitoring/dashboard.html` automatically tracks:
- Provider health and performance
- Cost per inference across providers
- Regional performance metrics
- Success rates and error tracking

## Deployment Steps

1. **Deploy hybrid router to Fly.io**:
   ```bash
   cd flyio-deployment
   ./deploy.sh
   ```

2. **Deploy Modal serverless functions**:
   ```bash
   cd modal-deployment
   ./deploy.sh
   ```

3. **Update runner app configuration**:
   ```bash
   flyctl secrets set HYBRID_ROUTER_URL=https://beacon-hybrid-router.fly.dev -a beacon-runner-production
   ```

4. **Deploy updated runner app**:
   ```bash
   flyctl deploy -a beacon-runner-production
   ```

5. **Verify integration**:
   ```bash
   curl https://beacon-runner-production.fly.dev/health
   ```

## Cost Optimization

The hybrid approach provides automatic cost optimization:

- **Baseline (70%)**: Golem providers at ~$0.0001/sec
- **Burst (30%)**: Serverless at ~$0.0003/sec  
- **Target blended cost**: <$0.0005 per inference
- **Automatic failover**: Ensures 99.5% uptime

## Next Steps

1. Monitor cost and performance metrics
2. Adjust routing percentages based on usage patterns
3. Add more serverless providers for redundancy
4. Implement advanced routing algorithms (ML-based)
5. Scale to additional regions as needed
