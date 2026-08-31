#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/../../.." && pwd)

echo "[STEP] build word-agent project image"
sh "$ROOT_DIR/deploy/backend/word_agent/build_project/start.sh"

echo "[STEP] start word-agent"
sh "$ROOT_DIR/deploy/backend/word_agent/local_full/start.sh"

echo "[STEP] start Java backend"
sh "$ROOT_DIR/deploy/backend/rob_english_word/local_full/start.sh"

echo "[STEP] start dashboard Go backend"
sh "$ROOT_DIR/deploy/backend/word_select_dashboard/local_full/start.sh"

echo "[STEP] start Vue main frontend"
sh "$ROOT_DIR/deploy/frontend/rob_english_word_front/local_full/start.sh"

echo "[STEP] start React cloze frontend"
sh "$ROOT_DIR/deploy/frontend/rob_english_word_cloze_web/local_full/start.sh"

echo "[STEP] start dashboard React frontend"
sh "$ROOT_DIR/deploy/frontend/word_select_dashboard/local_full/start.sh"

echo "[INFO] rob_english_word_workforce full compose deploy completed"
