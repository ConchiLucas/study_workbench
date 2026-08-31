#!/usr/bin/env bash

if [ -z "${ROOT_DIR:-}" ]; then
  ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fi

RUNTIME_DIR="$ROOT_DIR/.service-runtime"
PID_DIR="$RUNTIME_DIR/pids"
LOG_DIR="$RUNTIME_DIR/logs"
LAUNCHD_DIR="$RUNTIME_DIR/launchd"

AUTO_INSTALL="${AUTO_INSTALL:-1}"
KILL_PORTS="${KILL_PORTS:-1}"
WAIT_FOR_PORTS="${WAIT_FOR_PORTS:-1}"
USE_LAUNCHCTL="${USE_LAUNCHCTL:-auto}"

mkdir -p "$PID_DIR" "$LOG_DIR" "$LAUNCHD_DIR"

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

stop_launchd_service() {
  local name="$1"
  local label plist domain

  can_use_launchctl || return 0

  label="$(launchctl_label "$name")"
  plist="$LAUNCHD_DIR/$name.plist"
  domain="$(launchctl_domain)"

  if launchctl print "$domain/$label" >/dev/null 2>&1; then
    log "Stopping launchd service: $name"
    launchctl bootout "$domain/$label" >/dev/null 2>&1 || \
      launchctl bootout "$domain" "$plist" >/dev/null 2>&1 || \
      launchctl remove "$label" >/dev/null 2>&1 || true
  fi
}

stop_pid_file() {
  local name="$1"
  local pid_file="$PID_DIR/$name.pid"
  local pid

  [ -f "$pid_file" ] || return 0

  pid="$(cat "$pid_file" 2>/dev/null || true)"
  if [ -n "$pid" ] && kill -0 "$pid" >/dev/null 2>&1; then
    log "Stopping $name pid $pid"
    kill "$pid" >/dev/null 2>&1 || true
    sleep 2
  fi

  if [ -n "${pid:-}" ] && kill -0 "$pid" >/dev/null 2>&1; then
    log "Force stopping $name pid $pid"
    kill -9 "$pid" >/dev/null 2>&1 || true
  fi

  rm -f "$pid_file"
}

kill_listeners_on_ports() {
  local port pids pid

  [ "$KILL_PORTS" = "1" ] || return 0

  if ! have_cmd lsof; then
    log "WARN: lsof not found; skipping port cleanup"
    return 0
  fi

  for port in "$@"; do
    pids="$(lsof -nP -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)"
    [ -n "$pids" ] || continue

    for pid in $pids; do
      [ "$pid" != "$$" ] || continue
      log "Stopping listener on port $port: pid $pid"
      kill "$pid" >/dev/null 2>&1 || true
    done
  done

  sleep 1

  for port in "$@"; do
    pids="$(lsof -nP -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)"
    [ -n "$pids" ] || continue

    for pid in $pids; do
      [ "$pid" != "$$" ] || continue
      log "Force stopping listener on port $port: pid $pid"
      kill -9 "$pid" >/dev/null 2>&1 || true
    done
  done
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

  local label plist domain executable pid
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

  launchctl bootstrap "$domain" "$plist"
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

restart_service() {
  local name="$1"
  shift

  log "Restarting $name"
  stop_launchd_service "$name"
  stop_pid_file "$name"
  kill_listeners_on_ports "$@"
}
