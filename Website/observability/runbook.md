# Project Beacon Error Recovery Runbook

## Overview
This runbook provides step-by-step procedures for handling common error scenarios in Project Beacon runner infrastructure.

## Alert Response Procedures

### 🚨 Critical Alerts

#### Runner Down
**Alert**: `RunnerDown`
**Symptoms**: Service unavailable, health checks failing

**Immediate Actions**:
1. Check Fly.io app status: `flyctl status --app beacon-runner-production`
2. Check recent logs: `flyctl logs --app beacon-runner-production --region lhr`
3. Restart if needed: `flyctl restart --app beacon-runner-production`
4. Verify health endpoint: `curl https://beacon-runner-production.fly.dev/health/ready`

**Escalation**: If restart doesn't resolve, check for infrastructure issues

#### High Job Failure Rate
**Alert**: `HighJobFailureRate`
**Symptoms**: >10% job failure rate over 5 minutes

**Investigation Steps**:
1. Check job processing metrics: `curl https://beacon-runner-production.fly.dev/metrics | grep jobs_`
2. Review recent job logs for error patterns
3. Check Golem network connectivity
4. Verify IPFS daemon status

**Recovery Actions**:
1. Reset circuit breakers if needed
2. Check provider availability
3. Restart stuck services

#### Jobs Stuck in Queue
**Alert**: `JobsStuckInQueue`
**Symptoms**: >10 jobs unpublished for >5 minutes

**Diagnosis**:
1. Check outbox publisher status in logs
2. Verify Redis connectivity
3. Check database connection

**Recovery**:
1. Restart outbox publisher worker
2. Clear stuck Redis queues if safe
3. Manually republish critical jobs

### ⚠️ Warning Alerts

#### High HTTP Latency
**Alert**: `HighHTTPLatency`
**Symptoms**: 95th percentile >2s

**Investigation**:
1. Check system resource usage
2. Review database query performance
3. Check external service response times

**Mitigation**:
1. Scale resources if needed
2. Enable circuit breakers for slow services
3. Optimize database queries

#### Low Negotiation Success Rate
**Alert**: `LowNegotiationSuccessRate`
**Symptoms**: <70% probe success rate

**Actions**:
1. Check Golem network status
2. Verify provider requirements
3. Review negotiation constraints
4. Consider fallback regions

## Service Recovery Procedures

### IPFS Recovery
**Symptoms**: IPFS connection errors, bundle failures

**Steps**:
1. Check IPFS daemon: `curl http://localhost:5001/api/v0/version`
2. Restart IPFS if needed: `ipfs daemon --init`
3. Verify IPFS configuration
4. Test bundle operations

### Golem Network Recovery
**Symptoms**: No offers, negotiation failures

**Steps**:
1. Check yagna status: `yagna --version`
2. Verify GLM balance: `yagna payment status`
3. Restart yagna service
4. Check network connectivity

### Database Recovery
**Symptoms**: Connection errors, query timeouts

**Steps**:
1. Check database connectivity
2. Review slow query logs
3. Restart database connections
4. Verify schema integrity

### Redis Recovery
**Symptoms**: Queue processing stopped, connection errors

**Steps**:
1. Check Redis connectivity
2. Review queue depths
3. Clear problematic queues
4. Restart Redis connections

## Circuit Breaker Management

### Viewing Circuit Breaker Status
```bash
curl -s https://beacon-runner-production.fly.dev/health/ready | jq '.services[].circuit_breaker_stats'
```

### Resetting Circuit Breakers
```bash
# Via API (requires admin token)
curl -X POST https://beacon-runner-production.fly.dev/api/v1/admin/circuit-breakers/reset \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

### Circuit Breaker States
- **Closed**: Normal operation
- **Open**: Service unavailable, requests failing fast
- **Half-Open**: Testing if service recovered

## Job Recovery Procedures

### Stuck Job Recovery
1. Identify stuck jobs: Check `outbox_oldest_unpublished_age_seconds` metric
2. Review job logs for errors
3. Manually retry or move to dead letter queue
4. Update job status in database

### Failed Job Analysis
1. Check job execution logs
2. Verify input validation
3. Review provider selection
4. Test with different regions

### Dead Letter Queue Processing
1. Review failed jobs in dead letter queue
2. Identify common failure patterns
3. Fix underlying issues
4. Requeue recoverable jobs

## Performance Optimization

### High Memory Usage
1. Check Go memory metrics
2. Review goroutine count
3. Look for memory leaks
4. Restart if necessary

### High CPU Usage
1. Identify CPU-intensive operations
2. Check for infinite loops
3. Review algorithm efficiency
4. Scale resources if needed

## Monitoring and Alerting

### Key Metrics to Watch
- `jobs_enqueued_total` vs `jobs_processed_total`
- `outbox_unpublished_count`
- `http_request_duration_seconds`
- `negotiation_offers_seen_total`
- Circuit breaker states

### Log Analysis
```bash
# Check for errors in last hour
flyctl logs --app beacon-runner-production --region lhr | grep -i error

# Monitor job processing
flyctl logs --app beacon-runner-production --region lhr | grep "job.*processed"

# Check circuit breaker events
flyctl logs --app beacon-runner-production --region lhr | grep "circuit.*breaker"
```

## Escalation Procedures

### Level 1: Automated Recovery
- Circuit breakers activate
- Retry mechanisms engage
- Fallback regions used

### Level 2: Manual Intervention
- Restart services
- Reset circuit breakers
- Clear queues

### Level 3: Engineering Escalation
- Infrastructure changes needed
- Code fixes required
- Architecture review

## Emergency Contacts

- **Primary On-Call**: Project Beacon Team
- **Secondary**: Infrastructure Team
- **Escalation**: Engineering Manager

## Post-Incident Actions

1. Document incident timeline
2. Identify root cause
3. Implement preventive measures
4. Update runbooks
5. Review alert thresholds
