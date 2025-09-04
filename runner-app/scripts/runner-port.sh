#!/usr/bin/env bash
set -euo pipefail

ADDR_FILE="${RUNNER_HTTP_ADDR_FILE:-.runner-http.addr}"
MODE="base"

usage() {
  echo "Usage: $0 [--port|--addr|--base] [--file <addr-file>]" >&2
  exit 1
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --port) MODE="port"; shift ;;
    --addr) MODE="addr"; shift ;;
    --base) MODE="base"; shift ;;
    --file) shift; [[ $# -gt 0 ]] || usage; ADDR_FILE="$1"; shift ;;
    -h|--help) usage ;;
    *) usage ;;
  esac
done

if [[ ! -f "$ADDR_FILE" ]]; then
  echo "addr file not found: $ADDR_FILE" >&2
  exit 2
fi

ADDR=$(cat "$ADDR_FILE")
if [[ -z "$ADDR" ]]; then
  echo "addr file empty: $ADDR_FILE" >&2
  exit 3
fi

HOST=${ADDR%%:*}
PORT=${ADDR##*:}
if [[ -z "$PORT" || "$PORT" == "0" ]]; then
  echo "invalid addr in file: $ADDR" >&2
  exit 4
fi

case "$MODE" in
  port)
    echo "$PORT" ;;
  addr)
    echo "$ADDR" ;;
  base)
    if [[ "$HOST" == "" || "$HOST" == "0.0.0.0" || "$HOST" == "::" ]]; then
      echo "http://localhost:$PORT"
    else
      echo "http://$HOST:$PORT"
    fi
    ;;
  *) usage ;;
fi
