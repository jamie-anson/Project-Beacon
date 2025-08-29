---
id: observability
title: Observability (Prometheus, Grafana, Alertmanager)
sidebar_label: Observability
---

Observability verifies MVP reliability and multi‑region execution. Metrics power dashboards and alerts for: HTTP 5xx/latency, job retries/dead‑letters, Golem execution failure rate, providers availability per region, and wallet balance.

- Purpose in MVP: uptime, latency SLOs, execution success/failure, capacity signals, alerts
- Who should run it: Dev/Staging — on by default; End users — optional (separate compose file)

## Ports

- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000 (admin / beacon123)
- Alertmanager: http://localhost:9093
- Runner API: http://localhost:8090
- Yagna metrics: http://localhost:7465/metrics

## How to run

- Terminal A (Yagna daemon)
  - Ensure Yagna is running and exposes `/metrics` on 7465

- Terminal D (Postgres + Redis + Docker Compose)
  - App stack: `docker compose up -d`
  - Observability stack: `docker compose -f docker-compose.observability.yml up -d`

- Terminal B (Go API server)
  - If running locally, listen on 8090 and expose `GET /api/v1/metrics`
  - If using Docker runner, no extra steps

- Terminal C (Actions)
  - Prometheus targets: http://localhost:9090/targets
  - Reload Prometheus after changes: `curl -X POST http://localhost:9090/-/reload`
  - Test metrics: `curl -s http://localhost:8090/api/v1/metrics | head`

## Dashboards

Grafana auto‑provisions the Prometheus datasource and a Runner dashboard.
- Location in repo: `runner-app/observability/grafana/provisioning/dashboards/dashboard-runner.json`
- In Grafana: Home → Dashboards → Runner

## Alerts

- Rules: `runner-app/observability/prometheus/alerts.yaml`
- Alertmanager config: `runner-app/observability/alertmanager/alertmanager.yml` (webhook default `http://host.docker.internal:5001/`)
- Restart Alertmanager after config updates:
  - `docker compose -f docker-compose.observability.yml restart alertmanager`

## Targets scraped

- Runner: `http://host.docker.internal:8090/api/v1/metrics`
- Yagna: `http://host.docker.internal:7465/metrics`

## Troubleshooting

- Port 8090 in use:
  - `lsof -i :8090 -P -n` then stop the process or container; rerun `docker compose up -d runner`
- Prometheus target DOWN:
  - Ensure runner listens on 8090 and `/api/v1/metrics` returns 200
- Webhook unreachable:
  - Ensure the receiver runs on host `:5001`; containers should call `http://host.docker.internal:5001/`

## GPU Acceleration Architecture

Project Beacon now uses GPU-accelerated inference with host delegation:

- **Host GPU**: Ollama runs on host with GPU acceleration (Metal/CUDA/ROCm)
- **Container clients**: HTTP-only containers call `host.docker.internal:11434`
- **Performance**: 25x improvement (1.25s vs 30s+ response times)
- **Models supported**: Llama 3.2:1b, Mistral 7b, Qwen 2.5:1.5b
- **Monitoring**: `ollama-metrics.py` tracks GPU utilization and inference stats

## Negotiation Telemetry

New metrics track provider negotiation and GeoIP verification:

- **Offers**: `negotiation_offers_seen_total{region}`, `negotiation_offers_matched_p0p2_total{region}`, `negotiation_offers_p3_total{region}`
- **Probes**: `negotiation_probes_passed_total{region}`, `negotiation_probes_failed_total{region}`
- **Duration**: `negotiation_duration_seconds{region,outcome}`
- **Alerts**: Zero progress, rising failures, high probe failure rates
- **Dashboard**: New "Negotiation Telemetry" row with offers/probes/duration panels

## References (repo)

- Prometheus: `runner-app/observability/prometheus/prometheus.yml`
- Alerts: `runner-app/observability/prometheus/alerts.yaml`
- Alertmanager: `runner-app/observability/alertmanager/alertmanager.yml`
- Grafana provisioning: `runner-app/observability/grafana/provisioning/`
- Runner port defaults: `runner-app/internal/config/config.go`
- GPU metrics: `runner-app/observability/ollama-metrics.py`
