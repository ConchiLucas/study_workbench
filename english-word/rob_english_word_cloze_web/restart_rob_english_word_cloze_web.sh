#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$ROOT_DIR/restart_service_common.sh"

SERVICE_NAME="rob_english_word_cloze_web"
SERVICE_DIR="$ROOT_DIR/rob_english_word_cloze_web"
SERVICE_PORT=7003

need_cmd npm
ensure_node_modules "$SERVICE_DIR"

restart_service "$SERVICE_NAME" "$SERVICE_PORT"
start_process "$SERVICE_NAME" "$SERVICE_DIR" npm run dev -- --host 0.0.0.0
wait_for_port "$SERVICE_NAME" "$SERVICE_PORT" 60
log "URL: http://127.0.0.1:$SERVICE_PORT"
