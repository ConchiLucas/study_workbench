#!/bin/zsh
set -euo pipefail

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
WORKSPACE_ROOT=${WORKSPACE_HOST_ROOT:-$(CDPATH= cd -- "$SCRIPT_DIR/../../../../.." && pwd)}

exec "$WORKSPACE_ROOT/deploy-compose-full.sh" --project word-select-dashboard-server
