#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

# vibedeploy: migrate directory-derived compose project
cleanup_vibedeploy_legacy_compose_containers() {
  vibedeploy_legacy_compose_project=$(basename "$SCRIPT_DIR")
  vibedeploy_legacy_compose_config="$SCRIPT_DIR/docker-compose.yml"
  vibedeploy_legacy_compose_ids=$(docker ps -aq \
    --filter "label=com.docker.compose.project=$vibedeploy_legacy_compose_project" \
    --filter "label=com.docker.compose.project.config_files=$vibedeploy_legacy_compose_config")
  if [ -n "$vibedeploy_legacy_compose_ids" ]; then
    docker rm -f $vibedeploy_legacy_compose_ids
  fi
}
cleanup_vibedeploy_legacy_compose_containers


docker compose \
  -p word-select-dashboard-web \
  -f "$SCRIPT_DIR/docker-compose.yml" \
  up --build -d
