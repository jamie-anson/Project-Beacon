# Distributed Tracing - Runner Implementation

## Overview

The runner app now includes database-backed distributed tracing for debugging request flow across services.

## Quick Start

### 1. Enable Tracing

```bash
# Add to Fly.io secrets
flyctl secrets set ENABLE_DB_TRACING=true -a beacon-runner-production

# Or in .env for local development
ENABLE_DB_TRACING=true
```

### 2. Use in Code

```go
import "github.com/jamie-anson/project-beacon-runner/internal/logging"

// Initialize (usually in main.go)
dbTracer := logging.NewDBTracer(db)

// Start a span
ctx := context.Background()
traceID := uuid.New()
span, _ := dbTracer.StartSpan(ctx, traceID, nil, "runner", "execute_question", map[string]interface{}{
    "model_id": "llama-3.2-1b",
    "region": "us-east",
})

// Add execution context
span.SetExecutionContext("job-123", 456, "llama-3.2-1b", "us-east")

// Complete successfully
defer dbTracer.CompleteSpan(ctx, span, "completed")

// Or complete with error
if err != nil {
    dbTracer.CompleteSpanWithError(ctx, span, err, "execution_failure")
}
```

## Integration Example

### Job Execution Tracing

```go
func (w *JobRunner) executeQuestion(...) ExecutionResult {
    // Generate trace ID for this execution
    traceID := uuid.New()
    
    // Start execution span
    executionSpan, _ := w.dbTracer.StartSpan(ctx, traceID, nil, "runner", "execute_question", map[string]interface{}{
        "job_id": spec.ID,
        "model_id": modelID,
        "region": region,
    })
    
    defer func() {
        if result.Error != nil {
            w.dbTracer.CompleteSpanWithError(ctx, executionSpan, result.Error, "execution_failure")
        } else {
            w.dbTracer.CompleteSpan(ctx, executionSpan, "completed")
        }
    }()
    
    // Link to execution record
    if executionID > 0 {
        executionSpan.SetExecutionContext(spec.ID, executionID, modelID, region)
    }
    
    // Span for hybrid router call
    routerSpan, _ := w.dbTracer.StartSpan(ctx, traceID, &executionSpan.SpanID, "runner", "hybrid_router_call", map[string]interface{}{
        "router_url": executor.baseURL,
    })
    
    // Execute router call
    providerID, status, outputJSON, receiptJSON, err := executor.Execute(ctx, &singleQuestionSpec, region)
    
    // Complete router span
    if err != nil {
        w.dbTracer.CompleteSpanWithError(ctx, routerSpan, err, "router_timeout")
    } else {
        w.dbTracer.CompleteSpan(ctx, routerSpan, "completed")
    }
    
    // Continue with rest of execution...
}
```

## Querying Traces

### 1. View All Spans for an Execution

```sql
SELECT * FROM trace_waterfall WHERE execution_id = 123;
```

### 2. Diagnose Execution (Auto-detect anomalies)

```sql
SELECT * FROM diagnose_execution_trace(123);
```

This automatically flags:
- Operations 3x slower than average
- Gaps >5 seconds between services
- Failed or timeout operations

### 3. Identify Root Cause

```sql
SELECT * FROM identify_root_cause(123);
```

Returns pattern-matched root causes:
- `NETWORK_TIMEOUT` - Gap >60s between services
- `CONNECTION_FAILURE` - Connection errors
- `SERVICE_TIMEOUT` - Timeout errors
- `PERFORMANCE_DEGRADATION` - Operations >2 minutes
- `MODAL_EXECUTION_FAILURE` - Modal function failures

### 4. Health Monitoring

```sql
SELECT * FROM trace_spans_health;
```

Shows:
- Total spans and recent activity
- Table/index sizes
- Average duration
- Failed/timeout span counts

## Feature Flag

**Default**: OFF (safe to deploy without impact)

**Enable**: Set `ENABLE_DB_TRACING=true`

When disabled:
- Zero performance impact
- No database writes
- No errors thrown

When enabled:
- ~1-5ms overhead per span
- Database writes are non-blocking
- Errors logged but don't crash execution

## Architecture

```
Runner App
├── internal/logging/db_tracer.go     ← New: Database persistence
├── internal/logging/tracing.go       ← Existing: In-memory tracing
└── migrations/0011_add_trace_spans   ← Database schema

Database
├── trace_spans table                 ← Stores all spans
├── trace_waterfall view              ← Easy trace reconstruction
├── diagnose_execution_trace()        ← Auto anomaly detection
├── identify_root_cause()             ← Pattern matching
└── trace_spans_health view           ← Monitoring
```

## Performance

Based on Neon Free Tier analysis:

**Current Database**:
- Storage: 18 MB / 512 MB (3.5% used)
- Connections: 4 / 901 (0.4% used)
- Executions: ~26/day

**With Tracing**:
- Storage growth: ~200 KB/day
- Connection usage: Minimal (non-blocking writes)
- Performance impact: <5ms per span

**Monthly Impact**: ~6 MB storage, well within free tier limits

## Retention

Recommended cleanup (run daily):

```sql
-- 30-day retention (default)
DELETE FROM trace_spans WHERE created_at < NOW() - INTERVAL '30 days';

-- Or 7-day for aggressive cleanup
DELETE FROM trace_spans WHERE created_at < NOW() - INTERVAL '7 days';
```

## Troubleshooting

### No spans appearing in database

1. Check feature flag: `echo $ENABLE_DB_TRACING`
2. Check database connection
3. Check logs for "failed to persist trace span"

### Performance degradation

1. Check database size: `SELECT * FROM trace_spans_health;`
2. Run cleanup if >100 MB
3. Reduce retention period

### High storage usage

Enable aggressive retention:
```sql
DELETE FROM trace_spans WHERE created_at < NOW() - INTERVAL '7 days';
```

## Next Steps

1. **Week 2**: Add tracing to hybrid router (Python)
2. **Week 3**: Add tracing to Modal functions
3. **Week 3**: Create MCP server for LLM queries
4. **Week 4**: End-to-end testing and validation

## Status

✅ Database schema deployed
✅ Tracing helpers created
✅ Feature flag implemented
✅ Documentation complete

**Ready for**: Integration into job_runner.go
