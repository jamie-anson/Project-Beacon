# Sentry Setup for Project Beacon Runner

## Overview

Sentry log forwarding is integrated using **Fly.io's native Sentry extension**. All application logs are automatically forwarded to Sentry with zero code changes required.

## Configuration

### Setup (Already Complete ✅)

The Sentry integration was created with:
```bash
flyctl ext sentry create
```

This automatically:
- Created a Sentry project linked to your Fly app
- Configured log forwarding for all stdout/stderr
- Injected the `SENTRY_DSN` secret into your app
- Started shipping logs to Sentry

### View Sentry Dashboard

```bash
flyctl ext sentry dashboard -a beacon-runner-production
```

Or visit directly: https://sentry.io/issues/?project=4510159975153664

### Check Integration Status

```bash
flyctl ext sentry list
```

## Features Enabled

### Error Tracking
- Automatic panic recovery with stack traces
- HTTP request context (URL, method, headers, body)
- User context (IP, user agent)
- Breadcrumbs for debugging

### Performance Monitoring
- 20% sample rate for transaction tracing
- HTTP endpoint performance metrics
- Database query performance
- External API call tracking

### Filtered Errors
The following expected errors are **not** reported to Sentry:
- `context canceled` - Graceful shutdowns
- `context deadline exceeded` - Expected timeouts

## Alert Configuration

Based on your Sentry setup:
- **Alert Frequency**: 10 occurrences in 1 minute
- **Framework**: Gin (Go web framework)

This prevents alert fatigue from transient errors while catching sustained issues.

## Key Metrics to Monitor

### Job Processing Pipeline
- `job_runner.go` - Job execution errors
- `outbox_publisher.go` - Queue publishing failures
- `redis_queue.go` - Redis connection issues

### API Endpoints
- `POST /api/v1/jobs` - Job submission failures
- `GET /api/v1/executions/:id` - Execution query errors
- `/ws` - WebSocket connection issues

### Infrastructure
- Database connection failures
- Redis queue timeouts
- IPFS bundling errors
- Hybrid router communication failures

### Multi-Region Execution
- `executeMultiRegion()` - Cross-region coordination failures
- Region-specific provider errors
- Signature verification issues

## Testing Sentry Integration

### Local Testing

```bash
# Set DSN in environment
export SENTRY_DSN="your-dsn-here"

# Run the app
go run cmd/runner/main.go
```

You should see in logs:
```
{"level":"info","message":"Sentry initialized","environment":"development"}
```

### Trigger Test Error

```bash
# Create a test panic endpoint (for testing only)
curl -X POST http://localhost:8090/admin/test-sentry \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

### Production Testing

After deployment, check Sentry dashboard for:
1. **Issues** tab - Captured errors
2. **Performance** tab - Transaction traces
3. **Releases** tab - Deployment tracking

## Sentry Dashboard

### Issue Grouping
Errors are grouped by:
- Error type/message
- Stack trace fingerprint
- Request URL pattern

### Context Available
Each error includes:
- **Request**: Method, URL, headers, body
- **User**: IP address, user agent
- **Environment**: Fly region, app version
- **Breadcrumbs**: Recent logs and events
- **Stack Trace**: Full Go stack with source code

### Performance Monitoring
Track:
- HTTP endpoint latency (p50, p95, p99)
- Database query performance
- External API call duration
- Transaction throughput

## Cost Considerations

Sentry pricing is based on:
- **Events**: Errors captured (filtered by BeforeSend)
- **Transactions**: Performance traces (20% sample rate)

With current configuration:
- Expected errors filtered out (context canceled, deadline exceeded)
- 20% sampling reduces transaction volume by 80%
- Alert threshold (10/min) prevents duplicate notifications

## Troubleshooting

### Sentry Not Initializing

Check logs for:
```
{"level":"info","message":"Sentry disabled (no SENTRY_DSN)"}
```

Solution: Verify `SENTRY_DSN` secret is set in Fly.io

### No Errors Appearing

1. Check Sentry project DSN is correct
2. Verify errors are not being filtered by `BeforeSend`
3. Check Sentry project quota limits
4. Ensure app is actually encountering errors

### Too Many Alerts

Adjust alert threshold in Sentry dashboard:
- **Settings** → **Alerts** → Edit alert rule
- Increase threshold (e.g., 20 occurrences in 1 minute)
- Add filters for specific error types

## Next Steps

1. **Set up Sentry DSN** in Fly.io secrets
2. **Deploy** the updated runner app
3. **Configure alerts** in Sentry dashboard
4. **Monitor** for production issues
5. **Create custom alerts** for critical paths:
   - Job processing failures
   - Database connection issues
   - Multi-region execution errors

## Resources

- [Sentry Go SDK Docs](https://docs.sentry.io/platforms/go/)
- [Sentry Gin Integration](https://docs.sentry.io/platforms/go/guides/gin/)
- [Sentry Performance Monitoring](https://docs.sentry.io/product/performance/)
