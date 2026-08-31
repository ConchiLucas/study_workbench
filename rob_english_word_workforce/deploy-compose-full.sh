#!/bin/zsh
set -euo pipefail
unsetopt BG_NICE

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
CONFIG_FILE=${WORKSPACE_ENV_FILE:-"$ROOT_DIR/.env.local"}
RUNTIME_CONFIG_DIR="$ROOT_DIR/.runtime-config"
RUNTIME_ENV="$RUNTIME_CONFIG_DIR/full-compose.env"
PROJECT_RUNTIME_DIR="$ROOT_DIR/.runtime"
RUNNER_ENV="$PROJECT_RUNTIME_DIR/cli-runner.env"
RUNNER_PID="$PROJECT_RUNTIME_DIR/cli-runner.pid"
RUNNER_LOG="$PROJECT_RUNTIME_DIR/cli-runner.log"
WORD_AGENT_DIR="$ROOT_DIR/word_select_dashboard/word-agent"
WORD_AGENT_PYTHON="$WORD_AGENT_DIR/.venv/bin/python"
RUNNER_MARKER="word-select-dashboard:$ROOT_DIR"
RUNNER_LAUNCH_LABEL="com.rob-english-word-workforce.word-select-dashboard-cli-runner"
CHECK_ONLY=0
TARGET_PROJECT=""
START_COMPLETE=0

source "$ROOT_DIR/scripts/word_select_dashboard_runtime.sh"

case "${1:-}" in
  "")
    (( $# == 0 )) || exit 2
    ;;
  --check)
    (( $# == 1 )) || {
      print -u2 -- "用法: $0 [--check | --project PROJECT]"
      exit 2
    }
    CHECK_ONLY=1
    ;;
  --project)
    (( $# == 2 )) || {
      print -u2 -- "用法: $0 [--check | --project PROJECT]"
      exit 2
    }
    TARGET_PROJECT="$2"
    case "$TARGET_PROJECT" in
      word-select-dashboard-server | word-agent | rob-english-word-back | \
        word-select-dashboard-web | rob-english-word-front | rob-english-word-cloze-web)
        ;;
      *)
        print -u2 -- "未知项目: $TARGET_PROJECT"
        exit 2
        ;;
    esac
    ;;
  *)
    print -u2 -- "用法: $0 [--check | --project PROJECT]"
    exit 2
    ;;
esac

log() {
  print -- "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

die() {
  print -u2 -- "ERROR: $*"
  exit 1
}

on_exit() {
  local exit_code="$1"
  release_lifecycle_lock
  if (( exit_code != 0 && START_COMPLETE == 0 )); then
    if [[ -n "$TARGET_PROJECT" ]]; then
      print -u2 -- "项目 $TARGET_PROJECT 增量更新未完成；已启动的容器和数据不会被自动删除。"
    else
      print -u2 -- "全量启动未完成；已启动的容器和数据不会被自动删除。"
    fi
    print -u2 -- "检查命令: docker ps -a --filter label=easy-deploy.project"
    print -u2 -- "Word Agent 日志: docker logs --tail=100 word-agent"
  fi
  return "$exit_code"
}
trap 'on_exit $?' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM HUP

require_command() {
  command -v "$1" >/dev/null 2>&1 || die "缺少命令: $1"
}

load_runtime_config() {
  [[ -f "$CONFIG_FILE" ]] || die "缺少根配置 $CONFIG_FILE；请复制 .env.example 为 .env.local 后填写。"
  [[ ! -L "$CONFIG_FILE" ]] || die "根配置不能是符号链接: $CONFIG_FILE"

  set -a
  source "$CONFIG_FILE"
  set +a

  mkdir -p "$RUNTIME_CONFIG_DIR"
  chmod 700 "$RUNTIME_CONFIG_DIR"
  python3 "$ROOT_DIR/scripts/workspace_runtime_config.py" --output "$RUNTIME_ENV"

  set -a
  source "$RUNTIME_ENV"
  set +a
}

compose_config() {
  local project="$1"
  local compose_file="$2"
  docker compose --project-name "$project" -f "$compose_file" config -q
}

check_all_compose() {
  compose_config word-agent "$ROOT_DIR/word_select_dashboard/word-agent/docker-compose.yml"
  compose_config word-select-dashboard-server "$ROOT_DIR/word_select_dashboard/server/docker-compose.yml"
  compose_config rob-english-word-back "$ROOT_DIR/rob_english_word_back/docker-compose.yml"
  compose_config rob-english-word-front "$ROOT_DIR/rob_english_word_front/docker-compose.yml"
  compose_config rob-english-word-cloze-web "$ROOT_DIR/rob_english_word_cloze_web/docker-compose.yml"
  compose_config word-select-dashboard-web "$ROOT_DIR/word_select_dashboard/web-react/docker-compose.yml"
}

check_tcp() {
  local label="$1"
  local host="$2"
  local port="$3"
  if [[ "${WORKSPACE_SKIP_DEPENDENCY_CHECK:-0}" == "1" ]]; then
    return 0
  fi
  if ! python3 - "$host" "$port" <<'PY'
import socket
import sys

host, raw_port = sys.argv[1], sys.argv[2]
try:
    with socket.create_connection((host, int(raw_port)), timeout=3):
        pass
except (OSError, ValueError):
    raise SystemExit(1)
PY
  then
    die "$label 不可达: $host:$port"
  fi
}

check_dependencies() {
  check_tcp "select_english_word PostgreSQL" "$SELECT_DB_HOST_RUNTIME" "$SELECT_DB_PORT"
  check_tcp "rob_english_word PostgreSQL" "$ROB_WORD_DB_HOST_RUNTIME" "$ROB_WORD_DB_PORT"
  check_tcp "Redis" "$REDIS_HOST_RUNTIME" "$REDIS_PORT"
  check_tcp "MinIO" "$MINIO_HOST_RUNTIME" "$MINIO_PORT"
}

check_cli_paths() {
  local configured_path
  for configured_path in "${CODEX_COMMAND_PATH:-}" "${GEMINI_COMMAND_PATH:-}"; do
    [[ -z "$configured_path" || -x "$configured_path" ]] || die "CLI 命令不可执行: $configured_path"
  done
}

preflight() {
  require_command python3
  require_command docker
  require_command curl
  require_command zsh
  load_runtime_config
  docker compose version >/dev/null
  check_all_compose
  check_dependencies
  check_cli_paths
  log "统一运行配置与六个 Compose 校验通过。"
}

ensure_word_agent_runtime() {
  if [[ ! -x "$WORD_AGENT_PYTHON" ]]; then
    log "创建 Word Agent Python 环境"
    python3 -m venv "$WORD_AGENT_DIR/.venv"
    "$WORD_AGENT_PYTHON" -m pip install -e "$WORD_AGENT_DIR"
  fi
}

wait_for_url() {
  local name="$1"
  local url="$2"
  local attempts="${3:-90}"
  local attempt
  for attempt in {1..$attempts}; do
    if curl --fail --silent --show-error --max-time 2 "$url" >/dev/null 2>&1; then
      log "$name 已就绪"
      return 0
    fi
    sleep 1
  done
  die "等待 $name 超时: $url"
}

wait_for_port() {
  local name="$1"
  local port="$2"
  local attempts="${3:-120}"
  local attempt
  for attempt in {1..$attempts}; do
    if python3 - "$port" <<'PY' >/dev/null 2>&1
import socket
import sys
try:
    with socket.create_connection(("127.0.0.1", int(sys.argv[1])), timeout=1):
        pass
except (OSError, ValueError):
    raise SystemExit(1)
PY
    then
      log "$name 已监听端口 $port"
      return 0
    fi
    sleep 1
  done
  die "等待 $name 端口 $port 超时"
}

wait_for_container() {
  local name="$1"
  local attempts="${2:-60}"
  local attempt state
  for attempt in {1..$attempts}; do
    state="$(docker inspect --format '{{.State.Running}} {{.State.Restarting}}' "$name" 2>/dev/null || true)"
    if [[ "$state" == "true false" ]]; then
      log "$name 容器运行正常"
      return 0
    fi
    sleep 1
  done
  die "容器未稳定运行: $name"
}

compose_up() {
  local project="$1"
  local compose_file="$2"
  docker compose --project-name "$project" -f "$compose_file" up --build -d
}

start_dashboard_server() {
  log "启动 Go 管理后端"
  compose_up word-select-dashboard-server "$ROOT_DIR/word_select_dashboard/server/docker-compose.yml"
  wait_for_url "Go 管理后端" "$DASHBOARD_HOST_URL/health" 120
}

stop_existing_cli_runner() {
  if [[ "$(uname -s)" == "Darwin" ]] && project_launchd_job_exists "$RUNNER_LAUNCH_LABEL"; then
    stop_project_runner "$RUNNER_PID" "$WORD_AGENT_PYTHON" "$RUNNER_MARKER" "$RUNNER_LAUNCH_LABEL"
    return
  fi
  if [[ -s "$RUNNER_PID" && ! -L "$RUNNER_PID" && -O "$RUNNER_PID" ]]; then
    local pid="$(<"$RUNNER_PID")"
    if project_runner_process_is_valid "$pid" "$WORD_AGENT_PYTHON" "$RUNNER_MARKER"; then
      stop_project_runner "$RUNNER_PID" "$WORD_AGENT_PYTHON" "$RUNNER_MARKER"
    else
      rm -f "$RUNNER_PID"
    fi
  fi
}

start_cli_runner() {
  log "初始化并启动宿主机 CLI Runner"
  "$WORD_AGENT_PYTHON" "$ROOT_DIR/scripts/bootstrap_sentence_cli_configs.py" \
    --project-root "$ROOT_DIR" \
    --runner-env-file "$RUNNER_ENV"
  chmod 600 "$RUNNER_ENV"
  stop_existing_cli_runner
  launch_project_runner \
    "$WORD_AGENT_PYTHON" \
    "$WORD_AGENT_DIR" \
    "$RUNNER_MARKER" \
    "$RUNNER_LOG" \
    "$RUNNER_PID" \
    "$RUNNER_ENV" \
    "$RUNNER_LAUNCH_LABEL"
  wait_for_url "CLI Runner" "$CLI_RUNNER_HOST_URL/health" 30
}

start_word_agent() {
  log "构建项目基础镜像并启动 Word Agent"
  (cd "$WORD_AGENT_DIR" && make build-project)
  compose_up word-agent "$ROOT_DIR/word_select_dashboard/word-agent/docker-compose.yml"
  wait_for_url "Word Agent" "$WORD_AGENT_HOST_URL/health" 90
}

start_dashboard_web() {
  log "启动 React 管理前端"
  compose_up word-select-dashboard-web "$ROOT_DIR/word_select_dashboard/web-react/docker-compose.yml"
  wait_for_url "React 管理前端" "http://127.0.0.1:$DASHBOARD_WEB_PORT/" 90
}

start_java_backend() {
  log "启动 Java 核心后端"
  compose_up rob-english-word-back "$ROOT_DIR/rob_english_word_back/docker-compose.yml"
  wait_for_port "Java 核心后端" "$ROB_WORD_HTTP_PORT" 180
}

start_main_frontend() {
  log "启动 Vue 主前端"
  compose_up rob-english-word-front "$ROOT_DIR/rob_english_word_front/docker-compose.yml"
  wait_for_url "Vue 主前端" "http://127.0.0.1:$ROB_WORD_FRONT_PORT/" 90
}

start_cloze_frontend() {
  log "启动 React 完形前端"
  compose_up rob-english-word-cloze-web "$ROOT_DIR/rob_english_word_cloze_web/docker-compose.yml"
  wait_for_url "React 完形前端" "http://127.0.0.1:$CLOZE_WEB_PORT/" 90
}

verify_containers() {
  local container
  for container in \
    word-select-dashboard \
    word-agent \
    word-select-dashboard-web-react \
    rob-english-word \
    rob-english-word-front-web \
    rob-english-word-cloze-web
  do
    wait_for_container "$container" 30
  done
}

start_project() {
  case "$TARGET_PROJECT" in
    word-select-dashboard-server)
      start_dashboard_server
      ;;
    word-agent)
      start_word_agent
      ;;
    rob-english-word-back)
      start_java_backend
      ;;
    word-select-dashboard-web)
      start_dashboard_web
      ;;
    rob-english-word-front)
      start_main_frontend
      ;;
    rob-english-word-cloze-web)
      start_cloze_frontend
      ;;
  esac
}

verify_project_container() {
  local container
  case "$TARGET_PROJECT" in
    word-select-dashboard-server) container="word-select-dashboard" ;;
    word-agent) container="word-agent" ;;
    rob-english-word-back) container="rob-english-word" ;;
    word-select-dashboard-web) container="word-select-dashboard-web-react" ;;
    rob-english-word-front) container="rob-english-word-front-web" ;;
    rob-english-word-cloze-web) container="rob-english-word-cloze-web" ;;
  esac
  wait_for_container "$container" 30
}

preflight
if (( CHECK_ONLY == 1 )); then
  START_COMPLETE=1
  exit 0
fi

if [[ -z "$TARGET_PROJECT" || "$TARGET_PROJECT" == "word-agent" ]]; then
  ensure_word_agent_runtime
fi
mkdir -p "$PROJECT_RUNTIME_DIR"
chmod 700 "$PROJECT_RUNTIME_DIR"
acquire_lifecycle_lock "$PROJECT_RUNTIME_DIR"

if [[ -n "$TARGET_PROJECT" ]]; then
  start_project
  verify_project_container
  START_COMPLETE=1
  log "项目 $TARGET_PROJECT 已启动并通过检查。"
  exit 0
fi

start_dashboard_server
start_cli_runner
start_word_agent
start_dashboard_web
start_java_backend
start_main_frontend
start_cloze_frontend
verify_containers

START_COMPLETE=1
log "六个 Docker 项目与 CLI Runner 已全部启动并通过检查。"
