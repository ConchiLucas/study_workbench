#!/bin/zsh
set -euo pipefail

SCRIPT_DIR="${0:A:h}"
PROJECT_ROOT="${SCRIPT_DIR:h}"
source "$SCRIPT_DIR/word_select_dashboard_runtime.sh"
RUNTIME_DIR="$PROJECT_ROOT/.runtime"
PID_FILE="$RUNTIME_DIR/cli-runner.pid"
SERVER_COMPOSE="$PROJECT_ROOT/word_select_dashboard/server/docker-compose.yml"
WEB_COMPOSE="$PROJECT_ROOT/word_select_dashboard/web-react/docker-compose.yml"
AGENT_COMPOSE="$PROJECT_ROOT/word_select_dashboard/word-agent/docker-compose.yml"
SERVER_COMPOSE_PROJECT="word-select-dashboard-server"
WEB_COMPOSE_PROJECT="word-select-dashboard-web"
AGENT_COMPOSE_PROJECT="word-agent"
PYTHON="$PROJECT_ROOT/word_select_dashboard/word-agent/.venv/bin/python"
RUNNER_MARKER="$(project_runner_marker "$PROJECT_ROOT")"
RUNNER_LAUNCH_LABEL="com.rob-english-word-workforce.word-select-dashboard-cli-runner"

trap 'release_lifecycle_lock_preserving_exit_code $?' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM HUP

acquire_lifecycle_lock "$RUNTIME_DIR"

export WORD_AGENT_SELECT_DB_DSN="host=stop-only-placeholder"

runner_stop_failed=0
if [[ -s "$PID_FILE" ]] || { [[ "$(uname -s)" == "Darwin" ]] && project_launchd_job_exists "$RUNNER_LAUNCH_LABEL"; }; then
  if ! stop_project_runner "$PID_FILE" "$PYTHON" "$RUNNER_MARKER" "$RUNNER_LAUNCH_LABEL"; then
    print -u2 -- "未终止不匹配或未退出的进程，继续停止当前项目容器。"
    runner_stop_failed=1
  fi
fi

docker compose --project-name "$AGENT_COMPOSE_PROJECT" -f "$AGENT_COMPOSE" stop
docker compose --project-name "$WEB_COMPOSE_PROJECT" -f "$WEB_COMPOSE" stop
docker compose --project-name "$SERVER_COMPOSE_PROJECT" -f "$SERVER_COMPOSE" stop

if (( runner_stop_failed )); then
  print -u2 -- "项目容器已停止，但 CLI Runner 未停止。"
  exit 1
fi
print -- "Word Select Dashboard 当前项目服务已停止。"
