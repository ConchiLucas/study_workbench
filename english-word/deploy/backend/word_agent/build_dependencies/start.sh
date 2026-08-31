#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
ROOT_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/../../../.." && pwd)
PROJECT_START="$ROOT_DIR/deploy/backend/word_agent/build_project/start.sh"

if ! docker image inspect word-agent:1.0.0 >/dev/null 2>&1; then
  sh "$PROJECT_START"
fi

docker build \
  -f "$SCRIPT_DIR/Dockerfile.deps" \
  --build-arg PROJECT_IMAGE=word-agent:1.0.0 \
  -t word-agent:1.0.0 \
  "$ROOT_DIR"
