#!/bin/zsh

LIFECYCLE_LOCK_DIR=""
LIFECYCLE_LOCK_HELD=0
typeset -ga COMPOSE_UP_BUILD_ARGS
typeset -g LAUNCHED_RUNNER_PID=""

resolve_compose_up_build_args() {
  local skip_build="$1"
  case "$skip_build" in
    0)
      COMPOSE_UP_BUILD_ARGS=(--build)
      ;;
    1)
      COMPOSE_UP_BUILD_ARGS=(--no-build)
      ;;
    *)
      print -u2 -- "WORD_SELECT_DASHBOARD_SKIP_BUILD 只能是 0 或 1。"
      return 2
      ;;
  esac
}

on_start_exit() {
  local exit_code="$1"
  local start_complete="${START_COMPLETE:-0}"
  release_lifecycle_lock
  if (( exit_code != 0 && start_complete == 0 )); then
    print -u2 -- "启动未完成；部分服务可能已启动，使用 stop 脚本安全停止当前项目。"
  fi
  return "$exit_code"
}

release_lifecycle_lock_preserving_exit_code() {
  local exit_code="$1"
  release_lifecycle_lock
  return "$exit_code"
}

project_runner_marker() {
  local absolute_project_root="${1:A}"
  print -r -- "word-select-dashboard:$absolute_project_root"
}

project_pid_exists() {
  local pid="$1"
  kill -0 "$pid" 2>/dev/null
}

project_process_command() {
  local pid="$1"
  ps -p "$pid" -o command= 2>/dev/null
}

project_runner_command_matches() {
  local command="$1"
  local python_path="$2"
  local marker="$3"
  local expected="$python_path -m word_agent.cli_runner.main --runner-marker=$marker"
  local resolved_python_path="${python_path:A}"
  local resolved_expected="$resolved_python_path -m word_agent.cli_runner.main --runner-marker=$marker"
  [[ "$command" == "$expected" || "$command" == "$resolved_expected" ]]
}

project_runner_process_is_valid() {
  local pid="$1"
  local python_path="$2"
  local marker="$3"
  [[ "$pid" == <-> ]] || return 1
  project_pid_exists "$pid" || return 1
  local command
  command="$(project_process_command "$pid" || true)"
  project_runner_command_matches "$command" "$python_path" "$marker"
}

atomic_write_runner_pid() {
  local pid_file="$1"
  local pid="$2"
  local temporary_pid_file="${pid_file}.tmp.$$"
  (umask 077; print -r -- "$pid" > "$temporary_pid_file")
  chmod 600 "$temporary_pid_file"
  mv "$temporary_pid_file" "$pid_file"
}

launch_project_runner() {
  local python_path="$1"
  local word_agent_directory="$2"
  local marker="$3"
  local log_file="$4"
  local pid_file="$5"
  local environment_file="${6:-}"
  local launch_label="${7:-}"
  touch "$log_file"
  chmod 600 "$log_file"
  if [[ "$(uname -s)" == "Darwin" ]]; then
    if [[ -z "$environment_file" || -z "$launch_label" || ! -f "$environment_file" || -L "$environment_file" || ! -O "$environment_file" ]]; then
      print -u2 -- "launchd Runner 环境文件或任务标签不可信。"
      return 2
    fi
    launchctl submit -l "$launch_label" -o "$log_file" -e "$log_file" -- \
      /bin/zsh -c 'set -a; source "$1"; set +a; cd "$2"; export PYTHONPATH="$2/src"; exec "$3" -m word_agent.cli_runner.main --runner-marker="$4"' \
      zsh "$environment_file" "$word_agent_directory" "$python_path" "$marker"
    local attempt launchd_pid=""
    for attempt in {1..50}; do
      launchd_pid="$(project_launchd_pid "$launch_label" || true)"
      [[ "$launchd_pid" == <-> ]] && break
      sleep 0.1
    done
    if [[ "$launchd_pid" != <-> ]]; then
      launchctl remove "$launch_label" >/dev/null 2>&1 || true
      print -u2 -- "无法从 launchd 获取 CLI Runner PID。"
      return 3
    fi
    LAUNCHED_RUNNER_PID="$launchd_pid"
    atomic_write_runner_pid "$pid_file" "$LAUNCHED_RUNNER_PID"
    return 0
  fi
  (
    cd "$word_agent_directory"
    export PYTHONPATH="$word_agent_directory/src"
    exec nohup "$python_path" -m word_agent.cli_runner.main --runner-marker="$marker"
  ) </dev/null >>"$log_file" 2>&1 &
  LAUNCHED_RUNNER_PID=$!
  disown
  atomic_write_runner_pid "$pid_file" "$LAUNCHED_RUNNER_PID"
}

project_launchd_pid() {
  local launch_label="$1"
  local domain="gui/$(id -u)/$launch_label"
  local job_state
  job_state="$(launchctl print "$domain" 2>/dev/null)" || return 1
  print -r -- "$job_state" | awk '$1 == "pid" && $2 == "=" { print $3; exit }'
}

project_launchd_job_exists() {
  local launch_label="$1"
  launchctl print "gui/$(id -u)/$launch_label" >/dev/null 2>&1
}

stop_project_launchd_runner() {
  local launch_label="$1"
  local pid_file="$2"
  local python_path="$3"
  local marker="$4"
  local launchd_pid=""
  launchd_pid="$(project_launchd_pid "$launch_label" || true)"
  if [[ -n "$launchd_pid" ]] && ! project_runner_process_is_valid "$launchd_pid" "$python_path" "$marker"; then
    print -u2 -- "launchd 任务 PID 未匹配当前项目 CLI Runner。"
    return 3
  fi
  if ! launchctl remove "$launch_label"; then
    print -u2 -- "无法移除 launchd CLI Runner。"
    return 5
  fi
  if [[ "$launchd_pid" == <-> ]]; then
    local attempt
    for attempt in {1..50}; do
      project_pid_exists "$launchd_pid" || break
      sleep 0.2
    done
    if project_pid_exists "$launchd_pid"; then
      print -u2 -- "launchd CLI Runner 未在移除任务后退出。"
      return 4
    fi
  fi
  rm -f "$pid_file"
}

stop_project_runner() {
  local pid_file="$1"
  local python_path="$2"
  local marker="$3"
  local launch_label="${4:-}"
  if [[ "$(uname -s)" == "Darwin" && -n "$launch_label" ]] && project_launchd_job_exists "$launch_label"; then
    stop_project_launchd_runner "$launch_label" "$pid_file" "$python_path" "$marker"
    return $?
  fi
  stop_project_runner_from_pid_file "$pid_file" "$python_path" "$marker"
}

stop_project_runner_from_pid_file() {
  local pid_file="$1"
  local python_path="$2"
  local marker="$3"
  if [[ ! -s "$pid_file" || -L "$pid_file" || ! -O "$pid_file" ]]; then
    print -u2 -- "Runner PID 文件不存在或不可信。"
    return 2
  fi
  local pid="$(<"$pid_file")"
  if ! project_runner_process_is_valid "$pid" "$python_path" "$marker"; then
    print -u2 -- "PID 未同时匹配绝对 Python、模块和当前项目副本 marker。"
    return 3
  fi
  kill -TERM "$pid"
  local attempt
  for attempt in {1..50}; do
    project_pid_exists "$pid" || break
    sleep 0.2
  done
  if project_pid_exists "$pid"; then
    print -u2 -- "CLI Runner 未在 TERM 后退出。"
    return 4
  fi
  rm -f "$pid_file"
}

acquire_lifecycle_lock() {
  local runtime_dir="$1"
  if [[ -L "$runtime_dir" ]]; then
    print -u2 -- "运行目录不能是符号链接。"
    return 1
  fi
  mkdir -p "$runtime_dir"
  if [[ ! -d "$runtime_dir" || ! -O "$runtime_dir" ]]; then
    print -u2 -- "运行目录不属于当前用户。"
    return 1
  fi
  chmod 700 "$runtime_dir"

  LIFECYCLE_LOCK_DIR="$runtime_dir/word-select-dashboard.lifecycle.lock"
  local attempt owner owner_file stale_dir owner_temp missing_attempt
  for attempt in 1 2 3; do
    if mkdir "$LIFECYCLE_LOCK_DIR" 2>/dev/null; then
      chmod 700 "$LIFECYCLE_LOCK_DIR"
      owner_temp="$LIFECYCLE_LOCK_DIR/.owner.$$"
      print -r -- "$$" > "$owner_temp"
      chmod 600 "$owner_temp"
      mv "$owner_temp" "$LIFECYCLE_LOCK_DIR/owner.pid"
      LIFECYCLE_LOCK_HELD=1
      return 0
    fi

    if [[ ! -d "$LIFECYCLE_LOCK_DIR" || -L "$LIFECYCLE_LOCK_DIR" || ! -O "$LIFECYCLE_LOCK_DIR" ]]; then
      print -u2 -- "生命周期锁路径不可信，拒绝继续。"
      return 1
    fi

    owner_file="$LIFECYCLE_LOCK_DIR/owner.pid"
    owner=""
    for missing_attempt in 1 2 3; do
      if [[ -f "$owner_file" && ! -L "$owner_file" && -O "$owner_file" ]]; then
        owner="$(<"$owner_file")"
        break
      fi
      sleep 0.05
    done
    if [[ "$owner" == <-> ]] && project_pid_exists "$owner"; then
      print -u2 -- "另一个启动或停止操作仍在运行（PID $owner）。"
      return 1
    fi

    stale_dir="$runtime_dir/.word-select-dashboard.lifecycle.stale.$$.$attempt"
    if mv "$LIFECYCLE_LOCK_DIR" "$stale_dir" 2>/dev/null; then
      rm -f "$stale_dir/owner.pid"
      if ! rmdir "$stale_dir" 2>/dev/null; then
        print -u2 -- "已隔离残留生命周期锁，目录保留供人工检查：$stale_dir"
      fi
    fi
  done
  print -u2 -- "无法获取项目生命周期锁。"
  return 1
}

release_lifecycle_lock() {
  if [[ "$LIFECYCLE_LOCK_HELD" != 1 || -z "$LIFECYCLE_LOCK_DIR" ]]; then
    return 0
  fi
  local owner_file="$LIFECYCLE_LOCK_DIR/owner.pid"
  local owner=""
  if [[ -f "$owner_file" && ! -L "$owner_file" && -O "$owner_file" ]]; then
    owner="$(<"$owner_file")"
  fi
  if [[ "$owner" == "$$" ]]; then
    rm -f "$owner_file"
    rmdir "$LIFECYCLE_LOCK_DIR" 2>/dev/null || true
  fi
  LIFECYCLE_LOCK_HELD=0
}
