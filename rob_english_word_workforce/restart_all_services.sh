#!/usr/bin/env bash
# Canonical workspace startup entrypoint. For plain startup requests, run:
#   ./restart_all_services.sh restart
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RUNTIME_DIR="$ROOT_DIR/.service-runtime"
PID_DIR="$RUNTIME_DIR/pids"
LOG_DIR="$RUNTIME_DIR/logs"
LAUNCHD_DIR="$RUNTIME_DIR/launchd"

AUTO_INSTALL="${AUTO_INSTALL:-1}"
KILL_PORTS="${KILL_PORTS:-1}"
WAIT_FOR_PORTS="${WAIT_FOR_PORTS:-1}"
START_DOCKER_DEPS="${START_DOCKER_DEPS:-0}"
STOP_DOCKER_STACKS="${STOP_DOCKER_STACKS:-0}"
USE_LAUNCHCTL="${USE_LAUNCHCTL:-auto}"

LOCAL_PORTS=(6011 6012 6013 6014 6015 6016 6017 6018)
PROCESS_SERVICES=(
  cli_runner
  word_agent
  word_select_dashboard_server
  rob_english_word_back
  word_select_dashboard_web
  rob_english_word_front
  rob_english_word_cloze_web
)

mkdir -p "$PID_DIR" "$LOG_DIR" "$LAUNCHD_DIR"
chmod 700 "$RUNTIME_DIR"

log() {
  printf '[%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*"
}

die() {
  log "ERROR: $*"
  exit 1
}

have_cmd() {
  command -v "$1" >/dev/null 2>&1
}

need_cmd() {
  have_cmd "$1" || die "Missing command: $1"
}

can_use_launchctl() {
  [ "$USE_LAUNCHCTL" != "0" ] || return 1
  [ "$(uname -s)" = "Darwin" ] || return 1
  have_cmd launchctl || return 1
  return 0
}

launchctl_domain() {
  printf 'gui/%s' "$(id -u)"
}

launchctl_label() {
  printf 'local.rob_english_word_workforce.%s' "$1"
}

xml_escape() {
  local value="$1"
  value="${value//&/&amp;}"
  value="${value//</&lt;}"
  value="${value//>/&gt;}"
  value="${value//\"/&quot;}"
  value="${value//\'/&apos;}"
  printf '%s' "$value"
}

launchctl_service_pid() {
  local label="$1"
  launchctl print "$(launchctl_domain)/$label" 2>/dev/null | \
    awk -F'= ' '/pid = / {print $2; exit}'
}

stop_launchd_services() {
  local name label plist domain

  can_use_launchctl || return 0
  domain="$(launchctl_domain)"

  for name in "${PROCESS_SERVICES[@]}"; do
    label="$(launchctl_label "$name")"
    plist="$LAUNCHD_DIR/$name.plist"
    if launchctl print "$domain/$label" >/dev/null 2>&1; then
      log "Stopping launchd service: $name"
      launchctl bootout "$domain/$label" >/dev/null 2>&1 || \
        launchctl bootout "$domain" "$plist" >/dev/null 2>&1 || \
        launchctl remove "$label" >/dev/null 2>&1 || true
    fi
  done
}

compose_down() {
  local dir="$1"
  local label="$2"

  if ! have_cmd docker; then
    return 0
  fi

  if [ -f "$dir/docker-compose.yml" ] || [ -f "$dir/docker-compose.yaml" ] || \
     [ -f "$dir/compose.yml" ] || [ -f "$dir/compose.yaml" ]; then
    log "Stopping compose stack: $label"
    (cd "$dir" && docker compose down --remove-orphans) >/dev/null 2>&1 || \
      log "WARN: Failed to stop compose stack: $label"
  fi
}

stop_pid_files() {
  local pid_file pid

  shopt -s nullglob
  for pid_file in "$PID_DIR"/*.pid; do
    pid="$(cat "$pid_file" 2>/dev/null || true)"
    if [ -n "$pid" ] && kill -0 "$pid" >/dev/null 2>&1; then
      log "Stopping pid $pid from $(basename "$pid_file")"
      kill "$pid" >/dev/null 2>&1 || true
    fi
  done

  sleep 2

  for pid_file in "$PID_DIR"/*.pid; do
    pid="$(cat "$pid_file" 2>/dev/null || true)"
    if [ -n "$pid" ] && kill -0 "$pid" >/dev/null 2>&1; then
      log "Force stopping pid $pid from $(basename "$pid_file")"
      kill -9 "$pid" >/dev/null 2>&1 || true
    fi
    rm -f "$pid_file"
  done
  shopt -u nullglob
}

kill_listeners_on_ports() {
  local port pids pid

  [ "$KILL_PORTS" = "1" ] || return 0

  if ! have_cmd lsof; then
    log "WARN: lsof not found; skipping port cleanup"
    return 0
  fi

  for port in "${LOCAL_PORTS[@]}"; do
    pids="$(lsof -nP -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)"
    [ -n "$pids" ] || continue

    for pid in $pids; do
      [ "$pid" != "$$" ] || continue
      log "Stopping listener on port $port: pid $pid"
      kill "$pid" >/dev/null 2>&1 || true
    done
  done

  sleep 1

  for port in "${LOCAL_PORTS[@]}"; do
    pids="$(lsof -nP -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)"
    [ -n "$pids" ] || continue

    for pid in $pids; do
      [ "$pid" != "$$" ] || continue
      log "Force stopping listener on port $port: pid $pid"
      kill -9 "$pid" >/dev/null 2>&1 || true
    done
  done
}

stop_compose_stacks() {
  compose_down "$ROOT_DIR" "root db/redis/frontend/backend"
  compose_down "$ROOT_DIR/rob_english_word_front" "rob_english_word_front"
  compose_down "$ROOT_DIR/rob_english_word_back" "rob_english_word_back"
  compose_down "$ROOT_DIR/word_select_dashboard/server" "word_select_dashboard/server"
}

stop_all() {
  log "Stopping all known services"
  stop_launchd_services
  stop_pid_files
  kill_listeners_on_ports
  if [ "$STOP_DOCKER_STACKS" = "1" ]; then
    stop_compose_stacks
  fi
  log "Stop phase complete"
}

start_docker_deps() {
  [ "$START_DOCKER_DEPS" = "1" ] || return 0

  if ! have_cmd docker; then
    log "WARN: docker not found; skipping db/redis compose dependencies"
    return 0
  fi

  log "Starting db and redis with root docker compose"
  (cd "$ROOT_DIR" && docker compose up -d db redis) || \
    log "WARN: Could not start db/redis. If you already run them locally, this is okay."
}

ensure_node_modules() {
  local dir="$1"

  [ -d "$dir/node_modules" ] && return 0

  [ "$AUTO_INSTALL" = "1" ] || \
    die "Missing node_modules in $dir. Run npm install there, or rerun with AUTO_INSTALL=1."

  need_cmd npm
  log "Installing npm dependencies in $dir"
  (cd "$dir" && npm install)
}

start_process() {
  local name="$1"
  local dir="$2"
  local log_file="$LOG_DIR/$name.log"
  local pid_file="$PID_DIR/$name.pid"
  shift 2

  if can_use_launchctl; then
    start_launchd_process "$name" "$dir" "$log_file" "$pid_file" "$@"
    return $?
  fi

  log "Starting $name"
  (
    cd "$dir"
    nohup "$@" >"$log_file" 2>&1 &
    printf '%s' "$!" >"$pid_file"
  )

  sleep 1

  local pid
  pid="$(cat "$pid_file")"
  if ! kill -0 "$pid" >/dev/null 2>&1; then
    log "ERROR: $name exited early. Last log lines:"
    tail -n 40 "$log_file" 2>/dev/null || true
    return 1
  fi

  log "$name started as pid $pid; log: $log_file"
}

start_launchd_process() {
  local name="$1"
  local dir="$2"
  local log_file="$3"
  local pid_file="$4"
  shift 4

  local label plist domain arg executable pid attempt
  label="$(launchctl_label "$name")"
  plist="$LAUNCHD_DIR/$name.plist"
  domain="$(launchctl_domain)"

  if [[ "$1" != */* ]] && executable="$(command -v "$1" 2>/dev/null)"; then
    set -- "$executable" "${@:2}"
  fi

  log "Starting $name with launchctl"
  launchctl bootout "$domain/$label" >/dev/null 2>&1 || \
    launchctl bootout "$domain" "$plist" >/dev/null 2>&1 || \
    launchctl remove "$label" >/dev/null 2>&1 || true

  {
    printf '%s\n' '<?xml version="1.0" encoding="UTF-8"?>'
    printf '%s\n' '<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">'
    printf '%s\n' '<plist version="1.0">'
    printf '%s\n' '<dict>'
    printf '  <key>Label</key><string>%s</string>\n' "$(xml_escape "$label")"
    printf '  <key>WorkingDirectory</key><string>%s</string>\n' "$(xml_escape "$dir")"
    printf '  <key>RunAtLoad</key><true/>\n'
    printf '  <key>StandardOutPath</key><string>%s</string>\n' "$(xml_escape "$log_file")"
    printf '  <key>StandardErrorPath</key><string>%s</string>\n' "$(xml_escape "$log_file")"
    printf '  <key>EnvironmentVariables</key>\n'
    printf '  <dict>\n'
    printf '    <key>PATH</key><string>%s</string>\n' "$(xml_escape "$PATH")"
    printf '  </dict>\n'
    printf '  <key>ProgramArguments</key>\n'
    printf '  <array>\n'
    for arg in "$@"; do
      printf '    <string>%s</string>\n' "$(xml_escape "$arg")"
    done
    printf '  </array>\n'
    printf '%s\n' '</dict>'
    printf '%s\n' '</plist>'
  } > "$plist"

  for attempt in 1 2 3; do
    if launchctl bootstrap "$domain" "$plist"; then
      break
    fi
    if [ "$attempt" = "3" ]; then
      die "launchctl bootstrap failed for $name after 3 attempts"
    fi
    log "Retrying launchctl bootstrap for $name after transient failure"
    sleep 1
  done
  launchctl kickstart -k "$domain/$label" >/dev/null 2>&1 || true

  sleep 1
  pid="$(launchctl_service_pid "$label" || true)"
  if [ -n "$pid" ]; then
    printf '%s' "$pid" > "$pid_file"
    log "$name started as pid $pid; log: $log_file"
    return 0
  fi

  if launchctl print "$domain/$label" >/dev/null 2>&1; then
    rm -f "$pid_file"
    log "$name submitted to launchctl; log: $log_file"
    return 0
  fi

  log "ERROR: $name failed to load. Last log lines:"
  tail -n 40 "$log_file" 2>/dev/null || true
  return 1
}

ensure_word_agent_runtime() {
  local dir="$ROOT_DIR/word_select_dashboard/word-agent"

  if [ -x "$dir/.venv/bin/python" ] && [ -x "$dir/.venv/bin/word-agent" ]; then
    return 0
  fi

  [ "$AUTO_INSTALL" = "1" ] || \
    die "Word Agent .venv is not ready. Install it or rerun with AUTO_INSTALL=1."

  need_cmd python3
  log "Creating Python venv for word-agent"
  (cd "$dir" && python3 -m venv .venv && .venv/bin/python -m pip install -e .)
}

start_cli_runner() {
  local dir="$ROOT_DIR/word_select_dashboard/word-agent"
  local python="$dir/.venv/bin/python"
  local env_file="$RUNTIME_DIR/cli-runner.env"
  local runner_marker="rob-english-word-workforce:$ROOT_DIR"

  ensure_word_agent_runtime
  "$python" "$ROOT_DIR/scripts/bootstrap_sentence_cli_configs.py" \
    --project-root "$ROOT_DIR" \
    --config "$ROOT_DIR/word_select_dashboard/server/config.yaml" \
    --runner-env-file "$env_file"
  chmod 600 "$env_file"

  start_process cli_runner "$dir" \
    /bin/zsh -c \
    'set -a; source "$1"; set +a; exec "$2" -m word_agent.cli_runner.main "$3"' \
    zsh "$env_file" "$python" "--runner-marker=$runner_marker"
  wait_for_port cli_runner 6018 30
}

start_word_agent() {
  local dir="$ROOT_DIR/word_select_dashboard/word-agent"

  ensure_word_agent_runtime
  start_process word_agent "$dir" \
    env \
      WORD_AGENT_CLI_RUNNER_URL=http://127.0.0.1:6018 \
      "$dir/.venv/bin/word-agent"
}

start_backends() {
  need_cmd mvn
  need_cmd go

  start_cli_runner
  start_word_agent

  start_process word_select_dashboard_server \
    "$ROOT_DIR/word_select_dashboard/server" \
    go run .

  start_process rob_english_word_back \
    "$ROOT_DIR/rob_english_word_back" \
    env \
      SPRING_DATASOURCE_URL="${SPRING_DATASOURCE_URL:-jdbc:postgresql://127.0.0.1:5432/rob_english_word}" \
      SPRING_DATASOURCE_USERNAME="${SPRING_DATASOURCE_USERNAME:-${POSTGRES_USER:-conchi}}" \
      SPRING_DATASOURCE_PASSWORD="${SPRING_DATASOURCE_PASSWORD:-${POSTGRES_PASSWORD:-conchi123456}}" \
      SPRING_REDIS_HOST="${SPRING_REDIS_HOST:-127.0.0.1}" \
      SPRING_DATA_REDIS_HOST="${SPRING_DATA_REDIS_HOST:-127.0.0.1}" \
      SPRING_DATA_REDIS_PASSWORD="${SPRING_DATA_REDIS_PASSWORD:-${REDIS_PASSWORD:-conchi123456}}" \
      WORD_AGENT_BASE_URL=${WORD_AGENT_BASE_URL:-http://127.0.0.1:6017} \
      mvn spring-boot:run
}

start_frontends() {
  need_cmd npm

  ensure_node_modules "$ROOT_DIR/word_select_dashboard/web-react"
  ensure_node_modules "$ROOT_DIR/rob_english_word_front"
  ensure_node_modules "$ROOT_DIR/rob_english_word_cloze_web"

  start_process word_select_dashboard_web \
    "$ROOT_DIR/word_select_dashboard/web-react" \
    npm run dev -- --host 0.0.0.0

  start_process rob_english_word_front \
    "$ROOT_DIR/rob_english_word_front" \
    npm run dev -- --host 0.0.0.0

  start_process rob_english_word_cloze_web \
    "$ROOT_DIR/rob_english_word_cloze_web" \
    npm run dev -- --host 0.0.0.0
}

wait_for_port() {
  local name="$1"
  local port="$2"
  local timeout="${3:-60}"
  local elapsed=0

  [ "$WAIT_FOR_PORTS" = "1" ] || return 0

  if ! have_cmd nc; then
    log "WARN: nc not found; skipping readiness wait for $name"
    return 0
  fi

  while [ "$elapsed" -lt "$timeout" ]; do
    if nc -z 127.0.0.1 "$port" >/dev/null 2>&1; then
      log "Ready: $name on port $port"
      return 0
    fi
    sleep 1
    elapsed=$((elapsed + 1))
  done

  log "WARN: $name did not open port $port within ${timeout}s. Check logs in $LOG_DIR."
}

wait_for_services() {
  wait_for_port cli_runner 6018 30
  wait_for_port word_agent 6017 60
  wait_for_port word_select_dashboard_server 6015 90
  wait_for_port rob_english_word_back 6012 120
  wait_for_port rob_english_word_back_websocket 6013 120
  wait_for_port word_select_dashboard_web 6016 60
  wait_for_port rob_english_word_front 6011 60
  wait_for_port rob_english_word_cloze_web 6014 60
}

start_all() {
  log "Starting all services"
  start_docker_deps
  start_backends
  start_frontends
  wait_for_services
  print_urls
}

status_all() {
  local name pid_file pid label domain

  for name in "${PROCESS_SERVICES[@]}"; do
    if can_use_launchctl; then
      label="$(launchctl_label "$name")"
      domain="$(launchctl_domain)"
      if launchctl print "$domain/$label" >/dev/null 2>&1; then
        pid="$(launchctl_service_pid "$label" || true)"
        if [ -n "$pid" ]; then
          log "RUNNING: $name pid=$pid"
        else
          log "LOADED: $name has no current pid"
        fi
        continue
      fi
    fi

    pid_file="$PID_DIR/$name.pid"
    if [ -f "$pid_file" ]; then
      pid="$(cat "$pid_file" 2>/dev/null || true)"
      if [ -n "$pid" ] && kill -0 "$pid" >/dev/null 2>&1; then
        log "RUNNING: $name pid=$pid"
      else
        log "STOPPED: $name"
      fi
    else
      log "UNKNOWN: $name has no pid file"
    fi
  done
}

show_logs() {
  local log_file

  shopt -s nullglob
  for log_file in "$LOG_DIR"/*.log; do
    printf '\n===== %s =====\n' "$(basename "$log_file")"
    tail -n 80 "$log_file"
  done
  shopt -u nullglob
}

print_urls() {
  cat <<EOF

Services:
  rob_english_word_front      http://127.0.0.1:6011
  rob_english_word_api        http://127.0.0.1:6012
  rob_english_word_ws         ws://127.0.0.1:6013
  rob_english_word_cloze_web  http://127.0.0.1:6014
  word_select_dashboard_api   http://127.0.0.1:6015
  word_select_dashboard_web   http://127.0.0.1:6016
  word_agent                  http://127.0.0.1:6017
  cli_runner                  http://127.0.0.1:6018

Logs: $LOG_DIR
Pids: $PID_DIR
EOF
}

usage() {
  cat <<EOF
Usage: $(basename "$0") [restart|start|stop|status|logs]

Default action is restart. For Codex or local development startup, use:
  ./restart_all_services.sh restart

Environment toggles:
  AUTO_INSTALL=0        Do not install missing npm/Python dependencies.
  KILL_PORTS=0          Do not kill listeners on local service ports.
  WAIT_FOR_PORTS=0      Do not wait for service ports after start.
  START_DOCKER_DEPS=1   Also start root db/redis compose services.
  STOP_DOCKER_STACKS=1  Also stop known docker compose stacks.
  USE_LAUNCHCTL=0       Do not use macOS launchctl for detached services.
EOF
}

main() {
  local action="${1:-restart}"

  case "$action" in
    restart)
      stop_all
      start_all
      ;;
    start)
      start_all
      ;;
    stop)
      stop_all
      ;;
    status)
      status_all
      ;;
    logs)
      show_logs
      ;;
    -h|--help|help)
      usage
      ;;
    *)
      usage
      exit 64
      ;;
  esac
}

main "$@"
