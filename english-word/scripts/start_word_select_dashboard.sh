#!/bin/zsh
set -euo pipefail
unsetopt BG_NICE

SCRIPT_DIR="${0:A:h}"
PROJECT_ROOT="${SCRIPT_DIR:h}"
source "$SCRIPT_DIR/word_select_dashboard_runtime.sh"
RUNTIME_DIR="$PROJECT_ROOT/.runtime"
ENV_FILE="$RUNTIME_DIR/cli-runner.env"
PID_FILE="$RUNTIME_DIR/cli-runner.pid"
LOG_FILE="$RUNTIME_DIR/cli-runner.log"
SERVER_COMPOSE="$PROJECT_ROOT/word_select_dashboard/server/docker-compose.yml"
WEB_COMPOSE="$PROJECT_ROOT/word_select_dashboard/web-react/docker-compose.yml"
AGENT_COMPOSE="$PROJECT_ROOT/word_select_dashboard/word-agent/docker-compose.yml"
SERVER_COMPOSE_PROJECT="word-select-dashboard-server"
WEB_COMPOSE_PROJECT="word-select-dashboard-web"
AGENT_COMPOSE_PROJECT="word-agent"
WORD_AGENT_DIR="$PROJECT_ROOT/word_select_dashboard/word-agent"
PYTHON="$WORD_AGENT_DIR/.venv/bin/python"
RUNNER_MARKER="$(project_runner_marker "$PROJECT_ROOT")"
RUNNER_LAUNCH_LABEL="com.rob-english-word-workforce.word-select-dashboard-cli-runner"
resolve_compose_up_build_args "${WORD_SELECT_DASHBOARD_SKIP_BUILD:-0}"
START_COMPLETE=0

trap 'on_start_exit $?' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM HUP

wait_for_url() {
  local url="$1"
  local service_name="$2"
  local attempts="${3:-90}"
  local attempt
  for attempt in {1..$attempts}; do
    if curl --fail --silent --show-error --max-time 2 "$url" >/dev/null 2>&1; then
      print -- "$service_name 已就绪。"
      return 0
    fi
    sleep 1
  done
  print -u2 -- "等待 $service_name 超时。"
  return 1
}

runner_process_is_valid() {
  local pid="$1"
  project_runner_process_is_valid "$pid" "$PYTHON" "$RUNNER_MARKER"
}

stop_existing_project_runner() {
  local runner_pid=""
  if [[ "$(uname -s)" == "Darwin" ]] && project_launchd_job_exists "$RUNNER_LAUNCH_LABEL"; then
    stop_project_launchd_runner "$RUNNER_LAUNCH_LABEL" "$PID_FILE" "$PYTHON" "$RUNNER_MARKER"
    return $?
  fi
  if [[ -s "$PID_FILE" && ! -L "$PID_FILE" && -O "$PID_FILE" ]]; then
    runner_pid="$(<"$PID_FILE")"
  fi
  if [[ -z "$runner_pid" ]]; then
    return 0
  fi
  if ! runner_process_is_valid "$runner_pid"; then
    print -u2 -- "旧 PID 文件未指向带本项目 marker 的 Runner，未终止该进程。"
    return 0
  fi
  stop_project_runner_from_pid_file "$PID_FILE" "$PYTHON" "$RUNNER_MARKER"
}

command -v docker >/dev/null || { print -u2 -- "未找到 docker。"; exit 1; }
command -v curl >/dev/null || { print -u2 -- "未找到 curl。"; exit 1; }
if [[ "$(uname -s)" == "Darwin" ]]; then
  command -v launchctl >/dev/null || { print -u2 -- "未找到 launchctl。"; exit 1; }
else
  command -v nohup >/dev/null || { print -u2 -- "未找到 nohup。"; exit 1; }
fi
[[ -x "$PYTHON" ]] || { print -u2 -- "缺少 Word Agent Python 环境: $PYTHON"; exit 1; }

acquire_lifecycle_lock "$RUNTIME_DIR"
chmod 700 "$RUNTIME_DIR"
umask 077
touch "$LOG_FILE"
chmod 600 "$LOG_FILE"

docker compose --project-name "$SERVER_COMPOSE_PROJECT" -f "$SERVER_COMPOSE" up -d "${COMPOSE_UP_BUILD_ARGS[@]}"
wait_for_url "http://127.0.0.1:6015/health" "Go 服务"

"$PYTHON" "$PROJECT_ROOT/scripts/bootstrap_sentence_cli_configs.py" \
  --project-root "$PROJECT_ROOT" \
  --config "$PROJECT_ROOT/word_select_dashboard/server/config.yaml" \
  --runner-env-file "$ENV_FILE"
chmod 600 "$ENV_FILE"

set -a
source "$ENV_FILE"
set +a

stop_existing_project_runner
launch_project_runner "$PYTHON" "$WORD_AGENT_DIR" "$RUNNER_MARKER" "$LOG_FILE" "$PID_FILE" "$ENV_FILE" "$RUNNER_LAUNCH_LABEL"
runner_pid="$LAUNCHED_RUNNER_PID"
wait_for_url "http://127.0.0.1:6018/health" "CLI Runner" 30
runner_process_is_valid "$runner_pid" || { print -u2 -- "CLI Runner 进程未保持运行。"; exit 1; }

docker compose --project-name "$WEB_COMPOSE_PROJECT" -f "$WEB_COMPOSE" up -d "${COMPOSE_UP_BUILD_ARGS[@]}"
docker compose --project-name "$AGENT_COMPOSE_PROJECT" -f "$AGENT_COMPOSE" up -d "${COMPOSE_UP_BUILD_ARGS[@]}"
wait_for_url "http://127.0.0.1:6016/" "React 管理后台"
wait_for_url "http://127.0.0.1:6017/health" "Word Agent"

START_COMPLETE=1
print -- "Word Select Dashboard 已启动：6015 / 6016 / 6017 / 6018。"
