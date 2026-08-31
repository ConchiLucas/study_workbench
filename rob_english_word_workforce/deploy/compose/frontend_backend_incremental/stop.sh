#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/../../.." && pwd)

echo "[STEP] stop dashboard React frontend"
sh "$ROOT_DIR/deploy/frontend/word_select_dashboard/local_full/stop.sh"

echo "[STEP] stop React cloze frontend"
sh "$ROOT_DIR/deploy/frontend/rob_english_word_cloze_web/local_full/stop.sh"

echo "[STEP] stop Vue main frontend"
sh "$ROOT_DIR/deploy/frontend/rob_english_word_front/local_full/stop.sh"

echo "[STEP] stop dashboard Go backend"
sh "$ROOT_DIR/deploy/backend/word_select_dashboard/local_incremental/stop.sh"

echo "[STEP] stop Java backend"
sh "$ROOT_DIR/deploy/backend/rob_english_word/local_incremental/stop.sh"

echo "[STEP] stop word-agent"
sh "$ROOT_DIR/deploy/backend/word_agent/local_full/stop.sh"

echo "[INFO] rob_english_word_workforce compose services stopped"
