#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

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

ROOT_DIR="$(cd "$SCRIPT_DIR/../../../.." && pwd)"
PROJECT_DIR="$ROOT_DIR/rob_english_word_back"
TARGET_DIR="$PROJECT_DIR/target"
POM_FILE="$PROJECT_DIR/pom.xml"
POM_HASH_FILE="$TARGET_DIR/.pom.sha256"
MODE="${1:-incremental}"

log_info()  { echo "[INFO] $1"; }
log_warn()  { echo "[WARN] $1"; }
log_step()  { echo "[STEP] $1"; }
log_error() { echo "[ERROR] $1"; }

pom_hash() {
    shasum -a 256 "$POM_FILE" | awk '{print $1}'
}

run_package() {
    mkdir -p "$TARGET_DIR"
    CURRENT_POM_HASH="$(pom_hash)"
    PREVIOUS_POM_HASH=""

    if [ -f "$POM_HASH_FILE" ]; then
        PREVIOUS_POM_HASH="$(<"$POM_HASH_FILE")"
    fi

    case "$MODE" in
        full)
            log_step "full build: mvn clean package -DskipTests"
            (cd "$PROJECT_DIR" && mvn clean package -DskipTests)
            ;;
        compile)
            if [ "$CURRENT_POM_HASH" = "$PREVIOUS_POM_HASH" ] && [ -n "$PREVIOUS_POM_HASH" ]; then
                log_step "offline compile: mvn clean package -o -DskipTests"
                (cd "$PROJECT_DIR" && mvn clean package -o -DskipTests)
            else
                log_warn "pom.xml changed, refreshing dependencies"
                (cd "$PROJECT_DIR" && mvn clean package -DskipTests)
            fi
            log_info "compile completed"
            ;;
        incremental)
            if [ "$CURRENT_POM_HASH" = "$PREVIOUS_POM_HASH" ] && [ -n "$PREVIOUS_POM_HASH" ]; then
                log_step "pom unchanged, offline package"
                (cd "$PROJECT_DIR" && mvn clean package -o -DskipTests)
            else
                log_warn "pom.xml changed or first deploy, full package"
                (cd "$PROJECT_DIR" && mvn clean package -DskipTests)
            fi
            ;;
        *)
            log_error "unsupported mode: $MODE"
            exit 1
            ;;
    esac

    printf '%s' "$CURRENT_POM_HASH" > "$POM_HASH_FILE"
}

run_package

if [ "$MODE" = "compile" ]; then
    exit 0
fi

docker build \
  -f "$SCRIPT_DIR/Dockerfile.run" \
  -t rob-english-word:1.0.0 \
  "$ROOT_DIR"

docker compose \
  -p rob-english-word-back \
  -f "$SCRIPT_DIR/docker-compose.yml" \
  up -d --no-deps --no-build --force-recreate app
