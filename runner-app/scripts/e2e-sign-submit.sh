#!/usr/bin/env bash
set -euo pipefail

# E2E: Sign (fresh timestamp/nonce) and submit JobSpec, assert 202 enqueued
# Requirements: jq, curl, Go-built ./sigtool, API server on :8090, Postgres+Redis up

ROOT_DIR="$(cd "$(dirname "$0")"/.. && pwd)"
UNSIGNED="$ROOT_DIR/examples/jobspec-who-are-you.json"
PRIVATE_KEY_FILE="$ROOT_DIR/.dev/keys/private.key"
TMP_DIR="${TMPDIR:-/tmp}"
CLEAN_JSON="$TMP_DIR/who-are-you.clean.json"
SIGNED_JSON="$TMP_DIR/who-are-you.signed.json"
API_URL="http://localhost:8090/api/v1/jobs"

check_dep() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Error: required dependency '$1' not found in PATH" >&2
    exit 1
  fi
}

check_dep jq
check_dep curl

# Ensure sigtool exists or build it
if [ ! -x "$ROOT_DIR/sigtool" ]; then
  echo "Building sigtool..."
  (cd "$ROOT_DIR" && go build -o sigtool ./cmd/sigtool)
fi

if [ ! -f "$UNSIGNED" ]; then
  echo "Error: unsigned JobSpec not found: $UNSIGNED" >&2
  exit 1
fi
if [ ! -f "$PRIVATE_KEY_FILE" ]; then
  echo "Error: private key file not found: $PRIVATE_KEY_FILE" >&2
  exit 1
fi

# Remove timestamp and nonce so sigtool injects fresh values
jq 'del(.metadata.timestamp, .metadata.nonce)' "$UNSIGNED" > "$CLEAN_JSON"
echo "Cleaned unsigned spec -> $CLEAN_JSON"

# Sign
"$ROOT_DIR/sigtool" sign \
  --private-key "$(cat "$PRIVATE_KEY_FILE")" \
  --input "$CLEAN_JSON" \
  --output "$SIGNED_JSON"

echo "Signed spec -> $SIGNED_JSON"

# Submit and assert response
HTTP_OUT="$TMP_DIR/e2e.http.out"
BODY_OUT="$TMP_DIR/e2e.body.out"

# -S show errors, -s silent transfer, -i include headers
curl -S -s -i -H 'Content-Type: application/json' -d @"$SIGNED_JSON" "$API_URL" > "$HTTP_OUT"

# Extract status code
STATUS=$(awk 'BEGIN{RS="\r\n\r\n"} NR==1 {for(i=1;i<=NF;i++){if($i ~ /^HTTP/){code=$(i+1)}}} END{print code}' "$HTTP_OUT")
# Extract body (everything after first CRLFCRLF)
sed '1,/^\r$/d' "$HTTP_OUT" > "$BODY_OUT" || true

echo "HTTP status: $STATUS"
echo "Body: $(cat "$BODY_OUT" | tr -d '\r' | tr -d '\n')"

if [ "$STATUS" != "202" ]; then
  echo "E2E FAIL: expected HTTP 202, got $STATUS" >&2
  exit 2
fi

# Validate JSON fields
if ! jq -e '.id and .status == "enqueued"' "$BODY_OUT" >/dev/null 2>&1; then
  echo "E2E FAIL: response body missing expected fields {id, status='enqueued'}" >&2
  cat "$BODY_OUT" >&2
  exit 3
fi

echo "E2E PASS: received 202 and expected response body."
