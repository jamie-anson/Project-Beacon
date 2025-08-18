---
id: lgtm
title: Logs & Traces (LGTM)
sidebar_label: Logs & Traces (LGTM)
---

This page outlines adding Logs and Traces to complement metrics, using the LGTM stack (Loki, Grafana, Tempo) + Promtail.

Status: optional for MVP; recommended for staging and deeper triage.

## Why add logs and traces

- Troubleshoot slow requests with end‑to‑end traces (API → DB/Redis/IPFS → Golem interactions)
- Correlate errors across services (runner, workers) with structured logs
- Pivot from Grafana panels directly into logs and traces

## Components

- Loki: log database optimized for labels and fast queries
- Promtail: ships container logs to Loki
- Tempo: distributed tracing backend (Zipkin/OTLP compatible)
- Grafana: single pane of glass for metrics, logs, traces

## Proposed docker services (future)

In `runner-app/docker-compose.observability.yml` (to be added):

- `loki` (port 3100)
- `promtail` (reads Docker json logs, static targets, or file paths)
- `tempo` (port 3200; OTLP receiver 4317/4318)

Grafana datasource provisioning to add:

- Loki datasource at `http://loki:3100`
- Tempo datasource at `http://tempo:3200`

## Logging guidance

- Use structured logs (JSON) in the Go runner: include `trace_id`, `span_id`, `job_id`, `region`, `request_id`
- Label key logs in Promtail to enable fast filtering
- Add drop rules to avoid excessive noisy logs

## Tracing guidance

- Instrument HTTP middleware and key operations (DB calls, Redis, IPFS, Golem execution)
- Use OTLP exporter with batch span processor; send to Tempo
- Propagate context through worker/job execution pipeline

## Grafana examples

- Explore → Logs: filter by `{service="runner"} job_id="..."`
- Explore → Traces: search by trace ID from logs
- Dashboards: add drill‑downs from latency panels to logs/traces

## Next steps to enable

1. Add Loki/Tempo/Promtail services to `docker-compose.observability.yml`
2. Add Grafana datasources under `runner-app/observability/grafana/provisioning/datasources/`
3. Add Go logging/tracing libraries (e.g., zap + otel)
4. Create a minimal Logs/Traces dashboard

If you want, we can scaffold the compose services and Grafana provisioning next.
