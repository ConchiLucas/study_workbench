#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "$ROOT_DIR/restart_service_common.sh"

SERVICE_NAME="word_agent"
SERVICE_DIR="$ROOT_DIR/word_select_dashboard/word-agent"
SERVICE_PORT=8010

restart_service "$SERVICE_NAME" "$SERVICE_PORT"

if have_cmd uv; then
  start_process "$SERVICE_NAME" "$SERVICE_DIR" uv run word-agent
elif [ -x "$SERVICE_DIR/.venv/bin/word-agent" ]; then
  start_process "$SERVICE_NAME" "$SERVICE_DIR" "$SERVICE_DIR/.venv/bin/word-agent"
else
  [ "$AUTO_INSTALL" = "1" ] || \
    die "uv is missing and $SERVICE_DIR/.venv is not ready. Install uv or rerun with AUTO_INSTALL=1."

  need_cmd python3
  log "Creating Python venv for word-agent"
  (cd "$SERVICE_DIR" && python3 -m venv .venv && .venv/bin/python -m pip install -e .)
  start_process "$SERVICE_NAME" "$SERVICE_DIR" "$SERVICE_DIR/.venv/bin/word-agent"
fi

wait_for_port "$SERVICE_NAME" "$SERVICE_PORT" 60
log "URL: http://127.0.0.1:$SERVICE_PORT"
