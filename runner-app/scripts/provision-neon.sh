#!/usr/bin/env bash
set -euo pipefail

# Terminal D: Provision Neon Postgres via API
# Requirements:
# - export NEON_API_KEY=...   (https://neon.tech/docs/manage/api-keys)
# - jq installed (brew install jq)
# Args (or env): PROJECT_NAME, NEON_REGION
#   REGION examples: aws-us-east-1, aws-eu-central-1, gcp-europe-west1
# Usage:
#   PROJECT_NAME=beacon-runner NEON_REGION=aws-eu-central-1 ./scripts/provision-neon.sh
# Output:
#   Prints DATABASE_URL you can export or write into .env

: "${NEON_API_KEY:?NEON_API_KEY is required}"
command -v jq >/dev/null 2>&1 || { echo "jq not found. Install with: brew install jq" >&2; exit 1; }

PROJECT_NAME=${PROJECT_NAME:-beacon-runner}
NEON_REGION=${NEON_REGION:-aws-eu-central-1}

API=https://api.neon.tech/v2

echo "Creating Neon project: $PROJECT_NAME in $NEON_REGION"
RESP=$(curl -sS -X POST "$API/projects" \
  -H "Authorization: Bearer $NEON_API_KEY" \
  -H "Content-Type: application/json" \
  -d "{\n    \"project\": {\n      \"name\": \"$PROJECT_NAME\",\n      \"region_id\": \"$NEON_REGION\"\n    }\n  }")

# Check for errors
if [[ $(echo "$RESP" | jq -r '.code // empty') != "" ]]; then
  echo "Neon API error:" >&2
  echo "$RESP" | jq >&2
  exit 1
fi

# Extract default connection string (direct psql)
DATABASE_URL=$(echo "$RESP" | jq -r '.project.connection_uris[0].connection_uri // empty')
if [[ -z "$DATABASE_URL" || "$DATABASE_URL" == "null" ]]; then
  # Fallback: compose from roles/endpoints if needed
  DB_HOST=$(echo "$RESP" | jq -r '.endpoints[0].host')
  DB_NAME=$(echo "$RESP" | jq -r '.databases[0].name')
  DB_USER=$(echo "$RESP" | jq -r '.roles[0].name')
  DB_PASS=$(echo "$RESP" | jq -r '.roles[0].password')
  DATABASE_URL="postgres://$DB_USER:$DB_PASS@$DB_HOST/$DB_NAME"
fi

echo "DATABASE_URL=$DATABASE_URL"
echo
cat <<EOF
Next steps:
  # Option A: export for current shell
  export DATABASE_URL="$DATABASE_URL"

  # Option B: write into .env alongside REDIS_URL
  ./scripts/export-env.sh "$DATABASE_URL" "<your_redis_url>"
EOF
