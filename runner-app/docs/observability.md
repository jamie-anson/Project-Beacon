# Observability: Health and Metrics

This runner exposes health probes and Prometheus metrics. All examples target http://localhost:8090.

## Health Endpoints

- /health — general status
- /health/live — liveness probe (process up)
- /health/ready — readiness probe (dependencies OK)

Examples:

```bash
# General health
curl -s http://localhost:8090/health | jq .

# Liveness
curl -s http://localhost:8090/health/live | jq .

# Readiness (503 if a dependency is down)
curl -s -i http://localhost:8090/health/ready | sed -n '1,20p'
```

Typical responses:

```json
{"status":"ok","timestamp":"2025-08-23T15:04:05Z"}
{"status":"live","timestamp":"2025-08-23T15:04:05Z"}
{"status":"ready","timestamp":"2025-08-23T15:04:05Z"}
```

## Prometheus Metrics

Metrics are exposed in Prometheus text exposition format at two paths:

- /metrics
- /api/v1/metrics (alias under API namespace)

Examples:

```bash
# Check headers
curl -sI http://localhost:8090/metrics | sed -n '1,10p'

# Fetch first lines
curl -s http://localhost:8090/api/v1/metrics | head -n 20
```

## Notes

- Tracing: HTTP spans are instrumented via Gin's OpenTelemetry middleware (`otelgin`).
- Exporter is enabled only when `OTEL_EXPORTER_OTLP_ENDPOINT` is set.
- Port preference: examples use :8090.
