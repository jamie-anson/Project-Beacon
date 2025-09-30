# Project Beacon Timeout System Documentation

## Overview

Project Beacon implements a comprehensive timeout system across multiple components to ensure reliable operation, prevent resource exhaustion, and provide predictable response times. This document details all timeout configurations and their interactions.

## Component Timeout Matrix

| Component | Operation | Default Timeout | Configurable | Environment Variable |
|-----------|-----------|----------------|--------------|---------------------|
| Hybrid Router | HTTP Client | 120s | No | - |
| Hybrid Router | Provider Health Check | 5s | No | - |
| Hybrid Router | Modal Retry Backoff | 2s × attempt | No | - |
| Portal | Fetch Requests | Browser Default (~30s) | No | - |
| Docker Health Check | Health Probe | 10s | Yes | HEALTHCHECK timeout |
| Docker Health Check | Start Period | 5s | Yes | HEALTHCHECK start-period |
| Docker Health Check | Interval | 30s | Yes | HEALTHCHECK interval |

## Hybrid Router Timeouts

### HTTP Client Configuration
```python
# hybrid_router/core/router.py:21
self.client = httpx.AsyncClient(timeout=120.0)
```

**Purpose**: Global timeout for all HTTP requests made by the hybrid router
**Impact**: Prevents hanging requests to providers
**Rationale**: 120 seconds allows for model loading and inference completion

### Provider Health Checks
```python
# hybrid_router/core/router.py:165,181
response = await self.client.get(f"{provider.endpoint}/health", timeout=5.0)
response = await self.client.get(health_endpoint, timeout=5.0)
```

**Purpose**: Quick health verification without blocking startup
**Impact**: Providers marked unhealthy if they don't respond within 5 seconds
**Rationale**: Health checks should be fast; slow responses indicate provider issues

### Modal Retry Logic
```python
# hybrid_router/core/router.py:384-391
if response.status_code == 404 and "app for invoked web endpoint is stopped" in response.text:
    await asyncio.sleep(2 * (attempt + 1))  # 2s, 4s, 6s backoff
```

**Purpose**: Handle Modal serverless cold starts
**Impact**: Automatic retry with exponential backoff
**Rationale**: Modal apps may need time to spin up from stopped state

## Portal Timeout Behavior

### Fetch API Defaults
The portal uses the browser's native `fetch()` API without explicit timeouts:

```javascript
// portal/src/lib/api/http.js:133
const res = await fetch(url, fetchOptions);
```

**Default Behavior**: 
- Browser timeout (typically 30-120 seconds depending on browser)
- No custom timeout override
- Relies on browser's network stack timeout handling

**Error Handling**:
```javascript
// portal/src/lib/api/http.js:98-102
if (!code && err?.name === 'AbortError') {
  code = 'ABORT_ERROR';
} else if (!code && (message.includes('Failed to fetch') || message.includes('Load failed'))) {
  code = 'NETWORK_ERROR';
}
```

## Docker Health Check Timeouts

### Configuration
```dockerfile
# Dockerfile:22-23
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD sh -c 'curl -fsS http://localhost:${PORT:-8000}/health || curl -fsS http://localhost:${PORT:-8000}/ready || exit 1'
```

**Parameters**:
- **interval**: 30s - Time between health checks
- **timeout**: 10s - Maximum time for health check command
- **start-period**: 5s - Grace period before health checks start
- **retries**: 3 - Failed attempts before marking unhealthy

**Fallback Strategy**: Try `/health` endpoint first, then `/ready` endpoint

## Timeout Interaction Patterns

### Provider Selection Flow
1. **Health Check**: 5s timeout per provider
2. **Inference Request**: 120s timeout for actual inference
3. **Retry Logic**: Up to 3 attempts with 2s × attempt backoff

### End-to-End Request Flow
```
Portal Request → Hybrid Router → Provider
     ∞              120s          varies
```

**Cascade Behavior**:
- Portal waits indefinitely (browser default)
- Hybrid router enforces 120s limit
- Provider timeout depends on implementation

### Failure Scenarios

#### Provider Timeout
```python
# When provider exceeds 120s timeout
failure = self._build_failure(
    code="ROUTER_INTERNAL_ERROR",
    stage="router_inference",
    message=str(e),
    transient=True,
)
```

#### Health Check Timeout
```python
# When health check exceeds 5s timeout
provider.healthy = False
logger.warning(f"Health check failed for {provider.name}: {e}")
```

## Configuration Guidelines

### Recommended Timeout Values

| Use Case | Recommended Timeout | Rationale |
|----------|-------------------|-----------|
| Health Checks | 5s | Fast failure detection |
| Model Inference | 60-120s | Allow for model loading |
| File Operations | 30s | Reasonable for small files |
| Network Requests | 30s | Standard web timeout |

### Environment-Specific Adjustments

#### Development
- Longer timeouts for debugging
- More verbose timeout logging
- Graceful degradation on timeout

#### Production
- Strict timeout enforcement
- Circuit breaker patterns
- Monitoring and alerting on timeouts

## Monitoring and Observability

### Timeout Metrics to Track
1. **Provider Response Times**: Track 95th percentile
2. **Health Check Failures**: Count timeout-related failures
3. **Request Timeout Rate**: Percentage of requests timing out
4. **Retry Success Rate**: Effectiveness of retry logic

### Logging Patterns
```python
# Timeout logging example
logger.warning(
    "Modal endpoint reported stopped app; retrying",
    extra={"provider": provider.name, "attempt": attempt + 1}
)
```

## Troubleshooting Common Timeout Issues

### Modal Cold Start Timeouts
**Symptoms**: 404 errors with "app stopped" message
**Solution**: Automatic retry with backoff (implemented)
**Prevention**: Keep Modal apps warm with periodic requests

### Provider Health Check Failures
**Symptoms**: Providers marked unhealthy despite being functional
**Solution**: Check provider startup time and network latency
**Prevention**: Ensure providers respond to `/health` within 5s

### Browser Request Timeouts
**Symptoms**: Portal shows network errors after ~30-120s
**Solution**: Implement custom timeout with AbortController
**Prevention**: Set reasonable expectations for long-running operations

## Future Improvements

### Planned Enhancements
1. **Configurable Portal Timeouts**: Add custom timeout support
2. **Circuit Breaker Pattern**: Implement provider circuit breakers
3. **Adaptive Timeouts**: Adjust timeouts based on provider performance
4. **Timeout Telemetry**: Comprehensive timeout metrics collection

### Configuration Externalization
Consider moving timeout values to environment variables:
```python
HYBRID_ROUTER_TIMEOUT = int(os.getenv("HYBRID_ROUTER_TIMEOUT", "120"))
HEALTH_CHECK_TIMEOUT = int(os.getenv("HEALTH_CHECK_TIMEOUT", "5"))
```

## Best Practices

### For Developers
1. **Always set explicit timeouts** for network operations
2. **Implement retry logic** with exponential backoff
3. **Log timeout events** with sufficient context
4. **Test timeout scenarios** in development

### For Operations
1. **Monitor timeout rates** across all components
2. **Set up alerts** for timeout threshold breaches
3. **Review timeout values** based on performance data
4. **Document timeout dependencies** between services

---

*This documentation is maintained as part of Project Beacon's operational runbooks. Update when timeout configurations change.*
