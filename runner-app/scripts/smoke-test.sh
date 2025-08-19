#!/usr/bin/env bash
set -euo pipefail

# Terminal C: Smoke tests for Runner API
# Usage:
#   ./scripts/smoke-test.sh https://<app>.fly.dev
#   ./scripts/smoke-test.sh http://localhost:8090

BASE_URL=${1:-http://localhost:8090}
JOBSPEC=${2:-examples/jobspec-who-are-you.json}

echo "Checking readiness..."
curl -fsS "$BASE_URL/health/ready" | sed -e 's/.*/READY OK/'

echo "Submitting job..."
curl -fsS -H 'content-type: application/json' -d @"$JOBSPEC" "$BASE_URL/api/v1/jobs" | tee /tmp/smoke-job.json

sleep 1

echo "Listing jobs..."
curl -fsS "$BASE_URL/api/v1/jobs" | jq '.[0] // .'

echo "Metrics (first lines):"
curl -fsS "$BASE_URL/metrics" | head -n 10

echo "Smoke tests completed against $BASE_URL"
