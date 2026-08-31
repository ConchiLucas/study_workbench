#!/usr/bin/env sh
set -e

ROOT_DIR=$(pwd)

stop_compose() {
  project_dir="$1"
  if [ -f "$ROOT_DIR/$project_dir/docker-compose.yml" ] || [ -f "$ROOT_DIR/$project_dir/docker-compose.yaml" ]; then
    echo "[STEP] stop $project_dir"
    cd "$ROOT_DIR/$project_dir"
    docker compose down || true
  fi
}

stop_compose "word_select_dashboard/web-react"
stop_compose "rob_english_word_cloze_web"
stop_compose "rob_english_word_front"
stop_compose "word_select_dashboard/server"
stop_compose "rob_english_word_back"
stop_compose "word_select_dashboard/word-agent"

echo "[INFO] rob_english_word_workforce compose services stopped"
