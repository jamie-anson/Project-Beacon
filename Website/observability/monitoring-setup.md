# Project Beacon Monitoring and Alerting Setup

## Overview
Comprehensive monitoring and alerting infrastructure for Project Beacon runner using Prometheus, Grafana, and Alloy.

## Components

### 1. Metrics Collection
- **Prometheus metrics endpoint**: `https://beacon-runner-production.fly.dev/metrics`
- **Scrape interval**: 15 seconds
- **Metrics categories**:
  - HTTP request metrics (latency, throughput, errors)
  - Job processing metrics (enqueued, processed, failed, retried)
  - Queue health metrics (unpublished count, age)
  - Golem negotiation metrics (offers, probes, success rates)
  - System metrics (memory, goroutines, WebSocket connections)

### 2. Alerting Rules
Critical alerts configured for:
- **Runner Down**: Service unavailability
- **High Job Failure Rate**: >10% failure rate over 5 minutes
- **Jobs Stuck in Queue**: >10 jobs unpublished for >5 minutes
- **Old Unpublished Jobs**: Jobs >5 minutes old in queue
- **High Latency**: 95th percentile >2s for HTTP, >300s for execution
- **Low Negotiation Success**: <70% probe success rate
- **No Offers Available**: No Golem offers for >10 minutes
- **Resource Issues**: High memory (>512MB), goroutines (>100)

### 3. Grafana Dashboard
8-panel dashboard covering:
- System health overview
- Job processing rates and queue health
- HTTP request latency percentiles
- Execution duration by region
- Golem network negotiation metrics
- Resource usage (memory, goroutines, WebSocket)
- Error rates and failure analysis

## Deployment Instructions

### Option 1: Grafana Cloud (Recommended)
1. Configure Alloy agent with `observability/alloy/config.alloy`
2. Set `GC_PASSWORD` environment variable
3. Import dashboard from `observability/grafana-dashboard.json`
4. Import alerting rules from `observability/alerting-rules.yml`

### Option 2: Self-Hosted Prometheus
1. Use `observability/prometheus/agent.yml.tmpl` as template
2. Set environment variables:
   - `GC_REMOTE_WRITE_URL`
   - `GC_USERNAME` 
   - `GC_PASSWORD`
   - `SCRAPE_TARGET=beacon-runner-production.fly.dev:443`

## Key Metrics to Monitor

### Critical Business Metrics
- `jobs_enqueued_total` - Jobs submitted to system
- `jobs_processed_total` - Successfully completed jobs
- `jobs_failed_total` - Failed job executions
- `outbox_unpublished_count` - Jobs stuck in queue

### Performance Metrics
- `http_request_duration_seconds` - API response times
- `runner_execution_duration_seconds` - Job execution times
- `runner_queue_latency_seconds` - Time in queue before processing

### Golem Network Health
- `negotiation_offers_seen_total` - Available providers
- `negotiation_probes_passed_total` - Successful provider connections
- `negotiation_probes_failed_total` - Failed provider attempts

### System Health
- `go_memstats_alloc_bytes` - Memory usage
- `go_goroutines` - Concurrent operations
- `websocket_connections` - Active client connections

## Alert Thresholds

| Alert | Threshold | Duration | Severity |
|-------|-----------|----------|----------|
| Runner Down | up == 0 | 1m | Critical |
| High Job Failure | >10% failure rate | 2m | Critical |
| Jobs Stuck | >10 unpublished | 5m | Warning |
| Old Jobs | >300s age | 1m | Critical |
| High HTTP Latency | >2s (95th) | 3m | Warning |
| High Execution Latency | >300s (95th) | 5m | Warning |
| Low Negotiation Success | <70% | 5m | Warning |
| No Offers | 0 offers/10m | 5m | Critical |
| High Memory | >512MB | 5m | Warning |

## Next Steps
1. Deploy monitoring infrastructure
2. Configure notification channels (Slack, PagerDuty, email)
3. Set up runbooks for alert response
4. Implement automated remediation for common issues
5. Add custom business logic alerts based on usage patterns
