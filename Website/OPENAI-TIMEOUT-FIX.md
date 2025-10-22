# OpenAI API Timeout Fix

## Problem

Bias Analysis summary generation is failing with timeout errors:
```
context deadline exceeded (Client.Timeout exceeded while awaiting headers)
```

**Error Location**: `internal/execution/analysis_summary.go:190` → `internal/analysis/llm_summary.go:167`

**Root Cause**: HTTP client timeout set to 30 seconds is insufficient for OpenAI GPT-5-nano API calls from Fly.io LHR region.

## Evidence from Sentry

- **Error**: `Client.Timeout exceeded while awaiting headers`
- **API Call**: `POST https://api.openai.com/v1/chat/completions`
- **Environment**: fly-lhr (London)
- **Timestamp**: 2025-10-22T08:56:13Z

## Technical Details

### Current Configuration
File: `internal/analysis/llm_summary.go:38-40`
```go
httpClient: &http.Client{
    Timeout: 30 * time.Second,
},
```

### Why 30s is Insufficient

1. **GPT-5-nano reasoning tokens**: Model uses internal reasoning which adds latency
2. **Network latency**: Fly.io LHR → OpenAI API (likely US-based)
3. **Large prompts**: Bias analysis prompts include full regional metrics, differences, and risk assessments
4. **API queue time**: OpenAI API may queue requests during high load

### Industry Standards

- OpenAI official SDK: 60-90 seconds default
- Anthropic Claude: 60 seconds default
- Most production LLM integrations: 60-120 seconds

## Solution

### Immediate Fix: Increase Timeout to 90 Seconds

**File**: `internal/analysis/llm_summary.go`

Change line 39 from:
```go
Timeout: 30 * time.Second,
```

To:
```go
Timeout: 90 * time.Second,
```

### Why 90 Seconds?

- **Conservative**: Handles worst-case network + processing latency
- **Standard**: Matches OpenAI SDK defaults
- **Safe**: Still fails fast enough to not block indefinitely
- **Regional**: Accounts for Fly.io multi-region deployment latency

## Implementation

### Code Change

```go
// NewOpenAISummaryGenerator creates a new summary generator
func NewOpenAISummaryGenerator(opts ...Option) *OpenAISummaryGenerator {
    baseURL := os.Getenv("OPENAI_API_BASE_URL")
    if baseURL == "" {
        baseURL = "https://api.openai.com"
    }

    gen := &OpenAISummaryGenerator{
        apiKey: os.Getenv("OPENAI_API_KEY"),
        httpClient: &http.Client{
            Timeout: 90 * time.Second, // ← CHANGED: Increased from 30s for GPT-5-nano reasoning
        },
        baseURL: strings.TrimRight(baseURL, "/"),
    }

    for _, opt := range opts {
        opt(gen)
    }

    return gen
}
```

### Testing

1. **Local**: Run bias analysis with `USE_LLM_SUMMARIES=true`
2. **Staging**: Deploy to Fly.io and test from LHR region
3. **Monitor**: Check Sentry for timeout errors over 24 hours
4. **Verify**: Confirm bias analysis summaries complete successfully

## Alternative Solutions (Future)

### Option 1: Configurable Timeout
```go
timeout := 90 * time.Second
if envTimeout := os.Getenv("OPENAI_TIMEOUT_SECONDS"); envTimeout != "" {
    if parsed, err := strconv.Atoi(envTimeout); err == nil {
        timeout = time.Duration(parsed) * time.Second
    }
}
```

### Option 2: Retry with Exponential Backoff
```go
// Retry once on timeout with longer timeout
if err != nil && isTimeoutError(err) {
    g.httpClient.Timeout = 120 * time.Second
    resp, err = g.httpClient.Do(req)
}
```

### Option 3: Streaming Response
- Use OpenAI streaming API to get partial results
- Reduces perceived latency
- More complex implementation

## Deployment Plan

1. ✅ Create fix plan document
2. ⏳ Update `llm_summary.go` timeout to 90s
3. ⏳ Add comment explaining reasoning
4. ⏳ Commit and push to main
5. ⏳ Deploy to Fly.io
6. ⏳ Monitor Sentry for 24 hours
7. ⏳ Verify bias analysis jobs complete successfully

## Success Criteria

- ✅ No more timeout errors in Sentry for bias analysis
- ✅ LLM summaries generate successfully from all Fly.io regions
- ✅ Response times under 90 seconds for 95th percentile
- ✅ Zero impact on template-based fallback path

## Rollback Plan

If 90s causes issues:
1. Revert to 30s timeout
2. Disable LLM summaries: `USE_LLM_SUMMARIES=false`
3. Use template-based generation (existing fallback)

## Related Files

- `internal/analysis/llm_summary.go` - HTTP client configuration
- `internal/execution/analysis_summary.go` - Summary generator caller
- `internal/execution/cross_region_executor.go` - Analysis orchestration

## Status

**Current**: 30 second timeout causing production failures
**Target**: 90 second timeout for reliable GPT-5-nano responses
**Priority**: HIGH - Blocking bias analysis feature
