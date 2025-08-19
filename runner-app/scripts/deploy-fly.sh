#!/usr/bin/env bash
set -euo pipefail

# Terminal B: Deploy runner to Fly.io
# Prereqs:
# - brew install flyctl
# - flyctl auth login
# - You have DATABASE_URL and REDIS_URL ready (from Neon / Upstash)
# - File fly.toml exists and app name is set

APP_NAME=${FLY_APP_NAME:-beacon-runner-change-me}
REGION=${FLY_REGION:-lhr}

if ! command -v flyctl >/dev/null 2>&1; then
  echo "flyctl not found. Install with: brew install flyctl" >&2
  exit 1
fi

if [[ ${DATABASE_URL:-} == "" || ${REDIS_URL:-} == "" ]]; then
  echo "ERROR: Please export DATABASE_URL and REDIS_URL in your shell before running this script." >&2
  echo "Example: export DATABASE_URL=postgres://...  export REDIS_URL=redis://..." >&2
  exit 1
fi

# Ensure app exists (idempotent)
if ! flyctl apps list --json | grep -q "\"Name\": \"$APP_NAME\""; then
  echo "Creating Fly app: $APP_NAME in region $REGION"
  flyctl apps create "$APP_NAME" --region "$REGION"
fi

# Set secrets
flyctl secrets set \
  DATABASE_URL="$DATABASE_URL" \
  REDIS_URL="$REDIS_URL" \
  HTTP_PORT=":8090" \
  JOBS_QUEUE_NAME="jobs" \
  --app "$APP_NAME"

# Optional secrets (uncomment and set if you use them)
# flyctl secrets set GOLEM_API_KEY="${GOLEM_API_KEY:-}" --app "$APP_NAME"
# flyctl secrets set GOLEM_NETWORK="${GOLEM_NETWORK:-testnet}" --app "$APP_NAME"
# flyctl secrets set IPFS_NODE_URL="${IPFS_NODE_URL:-}" --app "$APP_NAME"
# flyctl secrets set IPFS_GATEWAY="${IPFS_GATEWAY:-}" --app "$APP_NAME"

# Deploy
flyctl deploy --app "$APP_NAME" --build-only=false

echo "Deploy complete. App status:"
flyctl status --app "$APP_NAME"

echo "If you need the URL:"
flyctl info --app "$APP_NAME" | sed -n 's/^Hostname:\s*//p' | awk '{print "https://" $1}'
