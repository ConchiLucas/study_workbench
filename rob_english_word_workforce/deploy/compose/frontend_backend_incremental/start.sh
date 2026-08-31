#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/../../.." && pwd)

echo "[STEP] update word-agent dependency image"
sh "$ROOT_DIR/deploy/backend/word_agent/build_dependencies/start.sh"

echo "[STEP] restart word-agent"
sh "$ROOT_DIR/deploy/backend/word_agent/local_full/start.sh"

echo "[STEP] incrementally deploy Java backend"
bash "$ROOT_DIR/deploy/backend/rob_english_word/local_incremental/start.sh" incremental

echo "[STEP] incrementally deploy dashboard Go backend"
sh "$ROOT_DIR/deploy/backend/word_select_dashboard/local_incremental/start.sh"

echo "[STEP] rebuild Vue main frontend"
sh "$ROOT_DIR/deploy/frontend/rob_english_word_front/local_full/start.sh"

echo "[STEP] rebuild React cloze frontend"
sh "$ROOT_DIR/deploy/frontend/rob_english_word_cloze_web/local_full/start.sh"

echo "[STEP] rebuild dashboard React frontend"
sh "$ROOT_DIR/deploy/frontend/word_select_dashboard/local_full/start.sh"

echo "[INFO] rob_english_word_workforce incremental compose deploy completed"
