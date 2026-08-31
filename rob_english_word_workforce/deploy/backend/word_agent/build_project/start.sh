#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
ROOT_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/../../../.." && pwd)

docker build \
  -f "$SCRIPT_DIR/Dockerfile.project" \
  -t word-agent:1.0.0 \
  "$ROOT_DIR"
