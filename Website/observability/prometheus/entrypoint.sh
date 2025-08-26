#!/usr/bin/env sh
set -euo pipefail

# Required secrets for Grafana Cloud remote_write
: "${GC_REMOTE_WRITE_URL:?GC_REMOTE_WRITE_URL is required}"
: "${GC_USERNAME:?GC_USERNAME is required}"
: "${GC_PASSWORD:?GC_PASSWORD is required}"

# Optional scrape configuration
: "${SCRAPE_TARGET:=beacon-runner.fly.dev:443}"
: "${SCRAPE_SCHEME:=https}"
: "${SCRAPE_METRICS_PATH:=/metrics}"
: "${SCRAPE_JOB_NAME:=beacon-runner}"
: "${SCRAPE_INSTANCE_LABEL:=beacon-runner}"

# Render config from template
mkdir -p /etc/prometheus
envsubst < /etc/prometheus/agent.yml.tmpl > /etc/prometheus/prometheus.yml

# Exec Prometheus in agent mode
exec "$@"
