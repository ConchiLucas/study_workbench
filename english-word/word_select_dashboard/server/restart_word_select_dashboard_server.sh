#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "$ROOT_DIR/restart_service_common.sh"

SERVICE_NAME="word_select_dashboard_server"
SERVICE_DIR="$ROOT_DIR/word_select_dashboard/server"
SERVICE_PORT=8009

need_cmd go

restart_service "$SERVICE_NAME" "$SERVICE_PORT"
start_process "$SERVICE_NAME" "$SERVICE_DIR" go run .
wait_for_port "$SERVICE_NAME" "$SERVICE_PORT" 90
log "URL: http://127.0.0.1:$SERVICE_PORT"
