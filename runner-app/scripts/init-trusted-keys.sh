#!/usr/bin/env bash
set -euo pipefail

: "${TRUSTED_KEYS_FILE:?TRUSTED_KEYS_FILE must be set (e.g., /secrets/trusted_keys.json)}"

# TRUSTED_KEYS_JSON is recommended; if absent, do nothing (allows bypass in dev)
if [[ -z "${TRUSTED_KEYS_JSON:-}" ]]; then
  echo "[init-trusted-keys] TRUSTED_KEYS_JSON not set; skipping materialization." >&2
  exit 0
fi

# Create parent directory
keys_dir="$(dirname "${TRUSTED_KEYS_FILE}")"
mkdir -p "${keys_dir}"

# Write atomically
tmpfile="${TRUSTED_KEYS_FILE}.tmp.$RANDOM"
printf '%s' "${TRUSTED_KEYS_JSON}" > "${tmpfile}"
# Basic sanity check: ensure begins with '{' or '['
if ! head -c 1 "${tmpfile}" | grep -qE '[\[{]'; then
  echo "[init-trusted-keys] Content does not look like JSON; aborting." >&2
  rm -f "${tmpfile}"
  exit 1
fi
mv -f "${tmpfile}" "${TRUSTED_KEYS_FILE}"
chmod 0640 "${TRUSTED_KEYS_FILE}"
echo "[init-trusted-keys] Wrote trusted keys to ${TRUSTED_KEYS_FILE}" >&2
