#!/usr/bin/env bash
set -euo pipefail

# Terminal D: Capture Upstash Redis URL (manual provisioning)
# Upstash currently recommends creating the Redis database in the dashboard.
# Steps:
#  1) Open https://console.upstash.com/ -> Redis -> Create Database
#  2) Choose region close to your Fly app (e.g., eu-west if Fly region lhr)
#  3) Copy the "UPSTASH_REDIS_REST_URL" or the standard Redis URL (rediss://...)
#  4) Paste the standard Redis URL below, then this script will write .env
#
# Usage:
#   ./scripts/provision-upstash.sh rediss://:<password>@<host>:<port>
# or set REDIS_URL in env and run without args.

REDIS_URL_INPUT=${1:-${REDIS_URL:-}}

if [[ -z "$REDIS_URL_INPUT" ]]; then
  echo "Usage: $0 REDIS_URL" >&2
  echo "Example: $0 redis://default:pass@eu1-enchanting-marmoset-12345.upstash.io:12345" >&2
  exit 1
fi

# If .env exists, preserve other vars and update/add REDIS_URL
if [[ -f .env ]]; then
  # Replace or append REDIS_URL
  if grep -q '^REDIS_URL=' .env; then
    sed -i.bak "s|^REDIS_URL=.*$|REDIS_URL=$REDIS_URL_INPUT|" .env
  else
    echo "REDIS_URL=$REDIS_URL_INPUT" >> .env
  fi
  echo "Updated .env with REDIS_URL" 
else
  cat > .env <<EOF
HTTP_PORT=:8090
JOBS_QUEUE_NAME=jobs
REDIS_URL=$REDIS_URL_INPUT
# DATABASE_URL=postgres://... (run provision-neon.sh or add manually)
EOF
  echo "Created .env with REDIS_URL"
fi

echo "REDIS_URL set. Next: set DATABASE_URL via Neon script, then run scripts/deploy-fly.sh"
