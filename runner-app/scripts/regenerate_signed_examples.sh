#!/usr/bin/env bash
set -euo pipefail

# Regenerate signed example JobSpecs using current signing rules.
# Requirements: go (to build sigtool), jq, and .dev/keys/private.key
# Usage:
#   scripts/regenerate_signed_examples.sh                # default examples
#   scripts/regenerate_signed_examples.sh file1.json ... # custom list
#   EXAMPLES="a.json b.json" scripts/regenerate_signed_examples.sh

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
EXAMPLES_DIR="$ROOT_DIR/examples"
KEY_FILE="${KEY_FILE:-$ROOT_DIR/.dev/keys/private.key}"
KEY_B64=""
SIGTOOL="${SIGTOOL:-$ROOT_DIR/sigtool}"

if ! command -v jq >/dev/null 2>&1; then
  echo "Error: jq is required"
  exit 1
fi

if [ ! -f "$KEY_FILE" ]; then
  echo "Error: private key not found at $KEY_FILE"
  exit 1
fi

# Load base64-encoded private key
KEY_B64="$(tr -d '\n\r\t ' < "$KEY_FILE")"
if [ -z "$KEY_B64" ]; then
  echo "Error: private key file is empty at $KEY_FILE"
  exit 1
fi

# Build sigtool if missing
if [ ! -x "$SIGTOOL" ]; then
  echo "Building sigtool..."
  (cd "$ROOT_DIR" && go build -o sigtool cmd/sigtool/main.go)
fi

echo "Regenerating signed examples..."

tmpdir="$(mktemp -d)"
cleanup() { rm -rf "$tmpdir"; }
trap cleanup EXIT

sign_example() {
  local input_json="$1"
  local output_signed="$2"
  local cleaned="$tmpdir/clean.json"

  # Remove timestamp and nonce so sigtool refreshes them
  jq '(.metadata.timestamp? |= empty) | (.metadata.nonce? |= empty)' "$input_json" > "$cleaned"

  # Sign
  "$SIGTOOL" sign \
    --private-key "$KEY_B64" \
    --input "$cleaned" \
    --output "$output_signed"

  echo "Signed: $output_signed"

  # Verify
  "$SIGTOOL" verify --input "$output_signed" >/dev/null && echo "Verified: $output_signed"

  # Print signer pubkey (once)
  if [ -z "${PRINTED_PUBKEY_ONCE:-}" ]; then
    echo -n "Signer public key: "
    "$SIGTOOL" extract-pubkey --input "$output_signed"
    PRINTED_PUBKEY_ONCE=1
  fi
}

# Determine example list
if [ "$#" -gt 0 ]; then
  # From args
  EXAMPLES=("$@")
elif [ -n "${EXAMPLES:-}" ]; then
  # From env var EXAMPLES (space-separated)
  # shellcheck disable=SC2206
  EXAMPLES=($EXAMPLES)
else
  # Default examples
  EXAMPLES=(
    "jobspec-who-are-you.json"
  )
fi

for f in "${EXAMPLES[@]}"; do
  in="$EXAMPLES_DIR/$f"
  if [ -f "$in" ]; then
    out="$EXAMPLES_DIR/${f}.signed"
    sign_example "$in" "$out"
  fi
done

echo "Done."
