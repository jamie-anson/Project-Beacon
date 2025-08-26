#!/usr/bin/env bash
set -euo pipefail

# Git pre-commit hook: validate signed examples are fresh and valid.
# Install with: make install-pre-commit

ROOT_DIR="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$ROOT_DIR"

# Build sigtool if needed and regenerate examples
if [ -f "examples/jobspec-who-are-you.json" ]; then
  echo "[pre-commit] Regenerating signed examples..."
  make -s regen-examples
fi

# Validate JSON in .signed files
echo "[pre-commit] Validating examples..."
make -s validate-examples

# Fail commit if diffs remain in examples/
if ! git diff --quiet -- examples; then
  echo "[pre-commit] ERROR: Signed examples changed. Changes have been staged."
  echo "Please review and re-run the commit."
  exit 1
fi

echo "[pre-commit] OK"
