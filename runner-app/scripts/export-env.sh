#!/usr/bin/env bash
set -euo pipefail

# Terminal D: Capture managed DB URLs into a local .env for reuse
# Usage:
#   ./scripts/export-env.sh "postgres://..." "redis://..."
# or export DATABASE_URL / REDIS_URL and run without args.

DB_URL_INPUT=${1:-${DATABASE_URL:-}}
REDIS_URL_INPUT=${2:-${REDIS_URL:-}}

if [[ -z "$DB_URL_INPUT" || -z "$REDIS_URL_INPUT" ]]; then
  echo "Usage: $0 POSTGRES_URL REDIS_URL" >&2
  echo "Example: $0 postgres://user:pass@host/db redis://default:pass@host:port" >&2
  exit 1
fi

cat > .env <<EOF
# Runner app environment
HTTP_PORT=:8090
JOBS_QUEUE_NAME=jobs
DATABASE_URL=$DB_URL_INPUT
REDIS_URL=$REDIS_URL_INPUT
# Optional extras
# GOLEM_API_KEY=
# GOLEM_NETWORK=testnet
# IPFS_NODE_URL=
# IPFS_GATEWAY=
EOF

echo ".env written with DATABASE_URL and REDIS_URL."
