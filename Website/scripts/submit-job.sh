#!/usr/bin/env bash
set -euo pipefail

# Submit a job to the Runner API using a demand constraints template.
# Usage:
#   scripts/submit-job.sh <tier> <model> [job_id]
# Tiers:
#   cpu | nvidia-8gib | nvidia-24gib | amd-8gib
# Example:
#   scripts/submit-job.sh nvidia-8gib llama3.2 my-job-001
#
# Notes:
# - The GPU templates include placeholders (${GPU_VENDOR_EXPR}, ${GPU_VRAM_EXPR}).
#   Replace these with validated expressions before expecting GPU selection.

TIER=${1:-}
MODEL=${2:-}
JOB_ID=${3:-}

if [[ -z "$TIER" || -z "$MODEL" ]]; then
  echo "Usage: $0 <tier> <model> [job_id]" >&2
  exit 1
fi

case "$TIER" in
  cpu)
    TEMPLATE="golem-provider/market/demand.cpu.json";;
  nvidia-8gib)
    TEMPLATE="golem-provider/market/demand.gpu.nvidia.8gib.json";;
  nvidia-24gib)
    TEMPLATE="golem-provider/market/demand.gpu.nvidia.24gib.json";;
  amd-8gib)
    TEMPLATE="golem-provider/market/demand.gpu.amd.8gib.json";;
  *)
    echo "Unknown tier: $TIER" >&2
    exit 1;;
 esac

if [[ ! -f "$TEMPLATE" ]]; then
  echo "Template not found: $TEMPLATE" >&2
  exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required" >&2
  exit 1
fi

CONSTRAINTS=$(jq .constraints "$TEMPLATE")

# Default JOB_ID if unset
if [[ -z "${JOB_ID}" ]]; then
  JOB_ID="${TIER//[^a-zA-Z0-9_-]/-}-$(date +%s)"
fi

RUNNER_URL=${RUNNER_URL:-http://localhost:8090}

BODY=$(jq -n \
  --arg id "$JOB_ID" \
  --argjson constraints "$CONSTRAINTS" \
  --arg model "$MODEL" \
  '{
     id: $id,
     constraints: $constraints,
     benchmark: { name: "text-generation", model: $model },
     regions: ["testnet"]
   }')

set -x
curl -sS -X POST "$RUNNER_URL/api/v1/jobs" \
  -H 'Content-Type: application/json' \
  -d "$BODY" | jq .
set +x
