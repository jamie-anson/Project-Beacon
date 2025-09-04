# Production Logging and Observability Deployment Guide

## Overview
Complete deployment guide for production-grade logging and observability infrastructure for Project Beacon runner.

## Environment Variables

### Required Environment Variables
```bash
# Logging Configuration
export LOG_LEVEL="info"
export LOG_FORMAT="json"
export ENVIRONMENT="production"
export APP_VERSION="1.0.0"

# Grafana Cloud / Loki
export LOKI_URL="https://logs-prod-us-central1.grafana.net"
export LOKI_USERNAME="your-username"
export LOKI_PASSWORD="your-api-key"

# Datadog (Optional)
export DATADOG_API_KEY="your-datadog-api-key"

# Elasticsearch (Optional)
export ELASTICSEARCH_URL="https://your-elasticsearch-cluster.com"
export ES_USERNAME="your-username"
export ES_PASSWORD="your-password"
```

### Fly.io Secrets Configuration
```bash
# Set production logging secrets
flyctl secrets set LOG_LEVEL=info --app beacon-runner-change-me
flyctl secrets set LOG_FORMAT=json --app beacon-runner-change-me
flyctl secrets set ENVIRONMENT=production --app beacon-runner-change-me
flyctl secrets set APP_VERSION=1.0.0 --app beacon-runner-change-me

# Optional: Add log forwarding
flyctl secrets set LOKI_URL="your-loki-url" --app beacon-runner-change-me
flyctl secrets set LOKI_USERNAME="your-username" --app beacon-runner-change-me
flyctl secrets set LOKI_PASSWORD="your-password" --app beacon-runner-change-me
```

## Log Aggregation Setup

### Option 1: Grafana Cloud (Recommended)
1. **Create Grafana Cloud Account**
   - Sign up at https://grafana.com/
   - Create a new stack
   - Note your Loki endpoint and credentials

2. **Configure Promtail**
   ```bash
   # Install Promtail on your log aggregation server
   wget https://github.com/grafana/loki/releases/download/v2.9.0/promtail-linux-amd64.zip
   unzip promtail-linux-amd64.zip
   sudo mv promtail-linux-amd64 /usr/local/bin/promtail
   ```

3. **Deploy Promtail Configuration**
   - Use `observability/log-aggregation.yml` promtail section
   - Update with your Loki credentials
   - Start promtail service

### Option 2: Self-Hosted ELK Stack
1. **Deploy Elasticsearch**
   ```bash
   docker run -d \
     --name elasticsearch \
     -p 9200:9200 \
     -e "discovery.type=single-node" \
     elasticsearch:8.8.0
   ```

2. **Deploy Logstash**
   ```bash
   docker run -d \
     --name logstash \
     -p 5044:5044 \
     --link elasticsearch:elasticsearch \
     logstash:8.8.0
   ```

3. **Deploy Kibana**
   ```bash
   docker run -d \
     --name kibana \
     -p 5601:5601 \
     --link elasticsearch:elasticsearch \
     kibana:8.8.0
   ```

## Dashboard Setup

### Grafana Dashboard Import
1. **Access Grafana**
   - Navigate to your Grafana instance
   - Go to Dashboards > Import

2. **Import Dashboard**
   - Upload `observability/observability-dashboard.json`
   - Configure data sources (Loki, Prometheus)
   - Save dashboard

3. **Configure Alerts**
   - Import `observability/alerting-rules.yml`
   - Set up notification channels
   - Test alert delivery

## Log Forwarding Configuration

### Vector Setup (Recommended)
1. **Install Vector**
   ```bash
   curl --proto '=https' --tlsv1.2 -sSf https://sh.vector.dev | bash
   ```

2. **Configure Vector**
   - Use `observability/log-aggregation.yml` vector section
   - Update endpoints and credentials
   - Start vector service

### Fluent Bit Alternative
1. **Install Fluent Bit**
   ```bash
   curl https://raw.githubusercontent.com/fluent/fluent-bit/master/install.sh | sh
   ```

2. **Configure Fluent Bit**
   - Use `observability/log-aggregation.yml` fluent_bit section
   - Update output destinations
   - Start fluent-bit service

## Structured Logging Integration

### Update Runner Code
1. **Import New Logging Package**
   ```go
   import "github.com/project-beacon/runner-app/internal/logging"
   ```

2. **Initialize Structured Logger**
   ```go
   logger := logging.NewStructuredLogger("beacon-runner")
   ```

3. **Add Tracing Middleware**
   ```go
   router.Use(logging.TracingMiddleware())
   ```

4. **Use Job Logger**
   ```go
   jobLogger := logging.NewJobLogger(jobID)
   jobLogger.LogJobStart(region, provider)
   ```

## Monitoring and Alerting

### Key Metrics to Monitor
- Log volume by level
- Error rate by component
- Request trace latency
- Job execution traces
- Circuit breaker states

### Alert Configuration
1. **High Error Rate**
   - Threshold: >5% error rate over 5 minutes
   - Action: Page on-call engineer

2. **Log Volume Spike**
   - Threshold: >2x normal log volume
   - Action: Investigate potential issues

3. **Missing Logs**
   - Threshold: No logs for >5 minutes
   - Action: Check log forwarding

## Security and Compliance

### Data Privacy
- Sensitive field masking enabled
- PII redaction configured
- Audit trail retention: 365 days

### Access Control
- Role-based access to logs
- Audit log access logging
- Retention policies enforced

## Performance Optimization

### Log Sampling
- Health checks: 10% sampling
- Metrics collection: 5% sampling
- Job events: 100% sampling
- Errors: 100% sampling

### Resource Management
- Log rotation: 100MB max size
- Local retention: 7 days
- Remote retention: 30-365 days

## Troubleshooting

### Common Issues
1. **Logs Not Appearing**
   - Check log forwarding configuration
   - Verify credentials
   - Check network connectivity

2. **High Log Volume**
   - Increase sampling rates
   - Filter noisy components
   - Optimize log levels

3. **Missing Traces**
   - Verify tracing middleware
   - Check trace ID propagation
   - Validate span completion

### Debug Commands
```bash
# Check log forwarding status
curl http://localhost:2020/api/v1/metrics

# Verify Loki connectivity
curl -G -s "$LOKI_URL/loki/api/v1/query" \
  --data-urlencode 'query={service="beacon-runner"}'

# Test log ingestion
echo '{"level":"info","message":"test"}' | \
  curl -X POST "$LOKI_URL/loki/api/v1/push" \
  -H "Content-Type: application/json" \
  --data-binary @-
```

## Deployment Checklist

- [ ] Environment variables configured
- [ ] Log aggregation service deployed
- [ ] Dashboard imported and configured
- [ ] Alerting rules configured
- [ ] Notification channels tested
- [ ] Structured logging integrated
- [ ] Tracing middleware enabled
- [ ] Security policies applied
- [ ] Performance optimization configured
- [ ] Troubleshooting procedures documented

## Next Steps

1. Deploy log aggregation infrastructure
2. Configure Grafana dashboards
3. Set up alerting and notifications
4. Integrate structured logging in runner code
5. Test end-to-end observability pipeline
6. Train team on log analysis and troubleshooting
