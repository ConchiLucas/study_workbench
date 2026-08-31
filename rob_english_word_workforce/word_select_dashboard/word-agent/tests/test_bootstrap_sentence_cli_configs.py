from __future__ import annotations

import importlib.util
import os
import shutil
import signal
import stat
import subprocess
import sys
import time
from pathlib import Path
from types import ModuleType

import pytest

PROJECT_ROOT = Path(__file__).resolve().parents[3]
BOOTSTRAP_PATH = PROJECT_ROOT / "scripts" / "bootstrap_sentence_cli_configs.py"
START_PATH = PROJECT_ROOT / "scripts" / "start_word_select_dashboard.sh"
STOP_PATH = PROJECT_ROOT / "scripts" / "stop_word_select_dashboard.sh"
RUNTIME_HELPER_PATH = PROJECT_ROOT / "scripts" / "word_select_dashboard_runtime.sh"
COMPOSE_PATH = PROJECT_ROOT / "word_select_dashboard" / "word-agent" / "docker-compose.yml"
GO_COMPOSE_PATH = PROJECT_ROOT / "word_select_dashboard" / "server" / "docker-compose.yml"
JAVA_COMPOSE_PATH = PROJECT_ROOT / "rob_english_word_back" / "docker-compose.yml"
FRONTEND_PROJECTS = (
    PROJECT_ROOT / "rob_english_word_front",
    PROJECT_ROOT / "rob_english_word_cloze_web",
    PROJECT_ROOT / "word_select_dashboard" / "web-react",
)
ENV_EXAMPLE_PATH = PROJECT_ROOT / "word_select_dashboard" / "word-agent" / ".env.example"
ROOT_START_PATH = PROJECT_ROOT / "restart_all_services.sh"


def load_bootstrap() -> ModuleType:
    spec = importlib.util.spec_from_file_location("bootstrap_sentence_cli_configs", BOOTSTRAP_PATH)
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class FakeCursor:
    def __init__(self) -> None:
        self.calls: list[tuple[str, tuple[object, ...]]] = []

    def __enter__(self) -> FakeCursor:
        return self

    def __exit__(self, exc_type, exc, traceback) -> None:
        return None

    def execute(self, query: str, params: tuple[object, ...]) -> None:
        self.calls.append((query, params))


class FakeConnection:
    def __init__(self) -> None:
        self.cursor_instance = FakeCursor()
        self.active_target = ("api", "aliyun-deepseek")

    def __enter__(self) -> FakeConnection:
        return self

    def __exit__(self, exc_type, exc, traceback) -> None:
        return None

    def cursor(self) -> FakeCursor:
        return self.cursor_instance


def test_bootstrap_adds_fixed_clis_without_changing_active_target(tmp_path: Path) -> None:
    bootstrap = load_bootstrap()
    assert bootstrap.BOOTSTRAP_ADVISORY_LOCK_ID == 0x45584543434647
    connection = FakeConnection()
    active_before = connection.active_target

    inserted = bootstrap.insert_missing_cli_configs(
        connection,
        project_root=tmp_path,
    )

    assert inserted == ("codex", "gemini")
    assert connection.active_target == active_before
    assert len(connection.cursor_instance.calls) == 3
    lock_query, lock_params = connection.cursor_instance.calls[0]
    assert "select pg_advisory_xact_lock" in " ".join(lock_query.lower().split())
    assert lock_params == (bootstrap.BOOTSTRAP_ADVISORY_LOCK_ID,)

    insert_calls = connection.cursor_instance.calls[1:]
    assert [call[1][0] for call in insert_calls] == ["codex", "gemini"]
    for query, _params in insert_calls:
        normalized = " ".join(query.lower().split())
        assert normalized.startswith("insert into cli_provider_configs")
        assert "on conflict (provider_id) do nothing" in normalized
        assert "created_at" in normalized
        assert "updated_at" in normalized
        assert normalized.count("now()") == 2
        assert "sentence_executor_config" not in normalized

    codex_params = insert_calls[0][1]
    gemini_params = insert_calls[1][1]
    assert codex_params[2:6] == (
        "codex",
        "/Applications/ChatGPT.app/Contents/Resources/codex",
        "gpt-5.6-sol",
        "high",
    )
    assert gemini_params[2:6] == (
        "gemini",
        "/Users/conchi/.npm-global/bin/gemini",
        "auto",
        "",
    )
    assert codex_params[6] == str(tmp_path.resolve())
    assert gemini_params[6] == str(tmp_path.resolve())


def test_cli_provider_paths_use_runtime_environment(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    bootstrap = load_bootstrap()
    monkeypatch.setenv("CODEX_COMMAND_PATH", "/opt/bin/codex")
    monkeypatch.setenv("GEMINI_COMMAND_PATH", "/opt/bin/gemini")
    connection = FakeConnection()

    bootstrap.insert_missing_cli_configs(connection, project_root=tmp_path)

    assert connection.cursor_instance.calls[1][1][3] == "/opt/bin/codex"
    assert connection.cursor_instance.calls[2][1][3] == "/opt/bin/gemini"


def test_database_dsns_prefer_runtime_environment(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    bootstrap = load_bootstrap()
    monkeypatch.setenv("WORD_AGENT_CLI_RUNNER_DB_DSN", "postgresql://external-runner")
    monkeypatch.setenv("WORD_AGENT_SELECT_DB_DSN", "postgresql://external-select")

    assert bootstrap.resolve_database_dsns(tmp_path / "missing.yaml") == (
        "postgresql://external-runner",
        "postgresql://external-select",
    )


def test_database_dsn_is_safely_built_from_server_config(tmp_path: Path) -> None:
    bootstrap = load_bootstrap()
    config_path = tmp_path / "config.yaml"
    config_path.write_text(
        """
pgsql:
  path: 127.0.0.1
  port: "5432"
  db-name: select_english_word
  username: local-user
  password: "space password"
  config: sslmode=disable TimeZone=Asia/Shanghai
""".strip(),
        encoding="utf-8",
    )

    dsn, container_dsn = bootstrap.resolve_database_dsns(config_path)

    parsed = bootstrap.psycopg.conninfo.conninfo_to_dict(dsn)
    assert parsed == {
        "user": "local-user",
        "password": "space password",
        "dbname": "select_english_word",
        "host": "127.0.0.1",
        "port": "5432",
        "sslmode": "disable",
    }
    assert bootstrap.psycopg.conninfo.conninfo_to_dict(container_dsn)["host"] == (
        "host.docker.internal"
    )


def test_runner_environment_is_atomically_written_with_private_permissions(
    tmp_path: Path,
) -> None:
    bootstrap = load_bootstrap()
    runtime_directory = tmp_path / ".runtime"
    env_path = runtime_directory / "cli-runner.env"

    bootstrap.write_runner_environment(
        env_path,
        host_database_dsn="host=127.0.0.1 password='quoted value'",
        container_database_dsn="host=host.docker.internal password='quoted value'",
    )

    assert stat.S_IMODE(runtime_directory.stat().st_mode) == 0o700
    assert stat.S_IMODE(env_path.stat().st_mode) == 0o600
    contents = env_path.read_text(encoding="utf-8")
    assert contents.count("\n") == 2
    assert contents.startswith("WORD_AGENT_CLI_RUNNER_DB_DSN=")
    assert "\nWORD_AGENT_SELECT_DB_DSN=" in contents
    assert "WORD_AGENT_CLI_RUNNER_TOKEN" not in contents


def test_runner_environment_rejects_line_injection(tmp_path: Path) -> None:
    bootstrap = load_bootstrap()

    with pytest.raises(bootstrap.BootstrapConfigError):
        bootstrap.write_runner_environment(
            tmp_path / "runner.env",
            host_database_dsn="postgresql://local\nINJECTED=value",
            container_database_dsn="postgresql://container",
        )


def test_runner_environment_can_be_sourced_without_executing_values(tmp_path: Path) -> None:
    bootstrap = load_bootstrap()
    env_path = tmp_path / ".runtime" / "cli-runner.env"
    injection_sentinel = tmp_path / "injected"
    suspicious = f"value $(touch {injection_sentinel}) ' quoted"
    bootstrap.write_runner_environment(
        env_path,
        host_database_dsn=f"host=127.0.0.1 password={suspicious}",
        container_database_dsn=f"host=host.docker.internal password={suspicious}",
    )

    result = subprocess.run(
        [
            "zsh",
            "-c",
            'source "$1"; print -r -- "$WORD_AGENT_CLI_RUNNER_DB_DSN"; '
            'print -r -- "$WORD_AGENT_SELECT_DB_DSN"',
            "zsh",
            str(env_path),
        ],
        check=True,
        capture_output=True,
        text=True,
    )

    assert result.stdout.splitlines() == [
        f"host=127.0.0.1 password={suspicious}",
        f"host=host.docker.internal password={suspicious}",
    ]
    assert not injection_sentinel.exists()


def test_lifecycle_lock_rejects_active_owner_and_recovers_dead_owner(tmp_path: Path) -> None:
    runtime_directory = tmp_path / ".runtime"
    holder = subprocess.Popen(
        [
            "zsh",
            "-c",
            'source "$1"; acquire_lifecycle_lock "$2"; print -r -- READY; sleep 30',
            "zsh",
            str(RUNTIME_HELPER_PATH),
            str(runtime_directory),
        ],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    try:
        assert holder.stdout is not None
        assert holder.stdout.readline().strip() == "READY"
        contender = subprocess.run(
            [
                "zsh",
                "-c",
                'source "$1"; acquire_lifecycle_lock "$2"',
                "zsh",
                str(RUNTIME_HELPER_PATH),
                str(runtime_directory),
            ],
            capture_output=True,
            text=True,
        )
        assert contender.returncode != 0
        assert "另一个启动或停止操作仍在运行" in contender.stderr
    finally:
        holder.terminate()
        holder.wait(timeout=3)

    recovered = subprocess.run(
        [
            "zsh",
            "-c",
            'source "$1"; acquire_lifecycle_lock "$2"; release_lifecycle_lock',
            "zsh",
            str(RUNTIME_HELPER_PATH),
            str(runtime_directory),
        ],
        capture_output=True,
        text=True,
    )
    assert recovered.returncode == 0, recovered.stderr
    assert not (runtime_directory / "word-select-dashboard.lifecycle.lock").exists()


def test_start_exit_trap_preserves_failure_releases_lock_and_warns(tmp_path: Path) -> None:
    runtime_directory = tmp_path / ".runtime"

    result = subprocess.run(
        [
            "zsh",
            "-c",
            'source "$1"; START_COMPLETE=0; acquire_lifecycle_lock "$2"; '
            "trap 'on_start_exit $?\' EXIT; false",
            "zsh",
            str(RUNTIME_HELPER_PATH),
            str(runtime_directory),
        ],
        capture_output=True,
        text=True,
    )

    assert result.returncode == 1
    assert "部分服务可能已启动，使用 stop 脚本" in result.stderr
    assert "read-only variable" not in result.stderr
    assert not (runtime_directory / "word-select-dashboard.lifecycle.lock").exists()


def test_stop_exit_trap_preserves_exit_code_and_releases_lock(tmp_path: Path) -> None:
    runtime_directory = tmp_path / ".runtime"

    result = subprocess.run(
        [
            "zsh",
            "-c",
            'source "$1"; acquire_lifecycle_lock "$2"; '
            "trap 'release_lifecycle_lock_preserving_exit_code $?\' EXIT; exit 23",
            "zsh",
            str(RUNTIME_HELPER_PATH),
            str(runtime_directory),
        ],
        capture_output=True,
        text=True,
    )

    assert result.returncode == 23
    assert not (runtime_directory / "word-select-dashboard.lifecycle.lock").exists()


@pytest.mark.parametrize(
    ("value", "expected"),
    [("0", "--build"), ("1", "--no-build")],
)
def test_compose_build_mode_resolves_default_and_skip(value: str, expected: str) -> None:
    result = subprocess.run(
        [
            "zsh",
            "-c",
            'source "$1"; resolve_compose_up_build_args "$2"; '
            'print -r -- "${(j: :)COMPOSE_UP_BUILD_ARGS}"',
            "zsh",
            str(RUNTIME_HELPER_PATH),
            value,
        ],
        capture_output=True,
        text=True,
    )

    assert result.returncode == 0, result.stderr
    assert result.stdout.strip() == expected


def test_compose_build_mode_rejects_invalid_value() -> None:
    result = subprocess.run(
        [
            "zsh",
            "-c",
            'source "$1"; resolve_compose_up_build_args invalid',
            "zsh",
            str(RUNTIME_HELPER_PATH),
        ],
        capture_output=True,
        text=True,
    )

    assert result.returncode != 0
    assert "WORD_SELECT_DASHBOARD_SKIP_BUILD 只能是 0 或 1" in result.stderr


def test_runner_marker_is_unique_to_absolute_project_copy(tmp_path: Path) -> None:
    first = tmp_path / "copy-one"
    second = tmp_path / "copy-two"
    first.mkdir()
    second.mkdir()

    result = subprocess.run(
        [
            "zsh",
            "-c",
            'source "$1"; project_runner_marker "$2"; project_runner_marker "$3"',
            "zsh",
            str(RUNTIME_HELPER_PATH),
            str(first),
            str(second),
        ],
        check=True,
        capture_output=True,
        text=True,
    )

    markers = result.stdout.splitlines()
    assert len(markers) == 2
    assert markers[0] != markers[1]
    assert markers[0] == f"word-select-dashboard:{first.resolve()}"


def test_runner_pid_helpers_require_exact_python_module_and_marker(tmp_path: Path) -> None:
    marker = f"word-select-dashboard:{tmp_path}"
    python = "/absolute/project/.venv/bin/python"
    pid_file = tmp_path / "runner.pid"
    expected_command = (
        f"{python} -m word_agent.cli_runner.main --runner-marker={marker}"
    )

    result = subprocess.run(
        [
            "zsh",
            "-c",
            'source "$1"; '
            'if project_runner_command_matches "$2" "$3" "$4"; then print MATCH; fi; '
            'if project_runner_command_matches "$2" "$3" wrong-marker; '
            'then print WRONG; else print REJECTED; fi; '
            'if project_runner_command_matches "$2-other-copy" "$3" "$4"; '
            'then print PREFIX_WRONG; else print PREFIX_REJECTED; fi; '
            'atomic_write_runner_pid "$5" 4242',
            "zsh",
            str(RUNTIME_HELPER_PATH),
            expected_command,
            python,
            marker,
            str(pid_file),
        ],
        check=True,
        capture_output=True,
        text=True,
    )

    assert result.stdout.splitlines() == ["MATCH", "REJECTED", "PREFIX_REJECTED"]
    assert pid_file.read_text(encoding="utf-8").strip() == "4242"
    assert stat.S_IMODE(pid_file.stat().st_mode) == 0o600


def test_runner_command_match_accepts_configured_python_symlink_realpath(tmp_path: Path) -> None:
    real_python = tmp_path / "runtime" / "python3"
    real_python.parent.mkdir()
    real_python.write_text("", encoding="utf-8")
    configured_python = tmp_path / ".venv" / "bin" / "python"
    configured_python.parent.mkdir(parents=True)
    configured_python.symlink_to(real_python)
    marker = f"word-select-dashboard:{tmp_path}"
    command = (
        f"{real_python} -m word_agent.cli_runner.main --runner-marker={marker}"
    )

    result = subprocess.run(
        [
            "zsh",
            "-c",
            'source "$1"; project_runner_command_matches "$2" "$3" "$4"',
            "zsh",
            str(RUNTIME_HELPER_PATH),
            command,
            str(configured_python),
            marker,
        ],
    )

    assert result.returncode == 0


def test_stop_runner_helper_terms_only_matching_marked_process(tmp_path: Path) -> None:
    marker = f"word-select-dashboard:{tmp_path}"
    python = "/absolute/project/.venv/bin/python"
    pid_file = tmp_path / "runner.pid"
    alive_file = tmp_path / "alive"
    alive_file.touch()
    process = subprocess.Popen(
        [
            "zsh",
            "-c",
            'trap \'rm -f "$1"; exit 0\' TERM; print READY; while true; do sleep 1; done',
            "zsh",
            str(alive_file),
        ],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    try:
        assert process.stdout is not None
        assert process.stdout.readline().strip() == "READY"
        expected_command = (
            f"{python} -m word_agent.cli_runner.main --runner-marker={marker}"
        )
        wrong_marker = subprocess.run(
            [
                "zsh",
                "-c",
                'source "$1"; EXPECTED_COMMAND="$5"; ALIVE_FILE="$6"; '
                'project_process_command() { print -r -- "$EXPECTED_COMMAND"; }; '
                'project_pid_exists() { [[ -e "$ALIVE_FILE" ]]; }; '
                'atomic_write_runner_pid "$2" "$3"; '
                'stop_project_runner_from_pid_file "$2" "$4" wrong-marker',
                "zsh",
                str(RUNTIME_HELPER_PATH),
                str(pid_file),
                str(process.pid),
                python,
                expected_command,
                str(alive_file),
            ],
            capture_output=True,
            text=True,
        )
        assert wrong_marker.returncode != 0
        assert process.poll() is None

        matched = subprocess.run(
            [
                "zsh",
                "-c",
                'source "$1"; EXPECTED_COMMAND="$5"; ALIVE_FILE="$6"; '
                'project_process_command() { print -r -- "$EXPECTED_COMMAND"; }; '
                'project_pid_exists() { [[ -e "$ALIVE_FILE" ]]; }; '
                'stop_project_runner_from_pid_file "$2" "$3" "$4"',
                "zsh",
                str(RUNTIME_HELPER_PATH),
                str(pid_file),
                python,
                marker,
                expected_command,
                str(alive_file),
            ],
            capture_output=True,
            text=True,
        )
        assert matched.returncode == 0, matched.stderr
        process.wait(timeout=3)
        assert not pid_file.exists()
    finally:
        if process.poll() is None:
            process.terminate()
            process.wait(timeout=3)


def test_launched_runner_survives_start_shell_exit_and_can_be_stopped(tmp_path: Path) -> None:
    word_agent_directory = tmp_path / "word-agent"
    module_directory = word_agent_directory / "src" / "word_agent" / "cli_runner"
    module_directory.mkdir(parents=True)
    (module_directory.parent / "__init__.py").write_text("", encoding="utf-8")
    (module_directory / "__init__.py").write_text("", encoding="utf-8")
    ready_file = tmp_path / "runner.ready"
    (module_directory / "main.py").write_text(
        """
import os
import signal
import time
from pathlib import Path

def stop(_signum, _frame):
    raise SystemExit(0)

signal.signal(signal.SIGTERM, stop)
Path(os.environ["TEST_RUNNER_READY_FILE"]).write_text(
    os.environ["WORD_AGENT_CLI_RUNNER_DB_DSN"],
    encoding="utf-8",
)
while True:
    time.sleep(0.1)
""".strip(),
        encoding="utf-8",
    )
    marker = f"word-select-dashboard:{tmp_path}"
    log_file = tmp_path / "runner.log"
    pid_file = tmp_path / "runner.pid"
    environment = {
        **os.environ,
        "TEST_RUNNER_READY_FILE": str(ready_file),
        "WORD_AGENT_CLI_RUNNER_DB_DSN": "postgresql://test-dsn-not-on-command-line",
    }
    runner_pid: int | None = None

    try:
        start_result = subprocess.run(
            [
                "zsh",
                "-c",
                'uname() { print Linux; }; source "$1"; '
                'launch_project_runner "$2" "$3" "$4" "$5" "$6"',
                "zsh",
                str(RUNTIME_HELPER_PATH),
                sys.executable,
                str(word_agent_directory),
                marker,
                str(log_file),
                str(pid_file),
            ],
            env=environment,
            capture_output=True,
            text=True,
        )
        assert start_result.returncode == 0, start_result.stderr
        runner_pid = int(pid_file.read_text(encoding="utf-8").strip())
        assert stat.S_IMODE(pid_file.stat().st_mode) == 0o600
        assert stat.S_IMODE(log_file.stat().st_mode) == 0o600

        for _attempt in range(40):
            if ready_file.exists():
                break
            time.sleep(0.05)
        assert ready_file.read_text(encoding="utf-8") == (
            "postgresql://test-dsn-not-on-command-line"
        )
        os.kill(runner_pid, 0)

        expected_command = (
            f"{sys.executable} -m word_agent.cli_runner.main --runner-marker={marker}"
        )
        stop_result = subprocess.run(
            [
                "zsh",
                "-c",
                'source "$1"; EXPECTED_COMMAND="$5"; '
                'project_process_command() { print -r -- "$EXPECTED_COMMAND"; }; '
                'stop_project_runner_from_pid_file "$2" "$3" "$4"',
                "zsh",
                str(RUNTIME_HELPER_PATH),
                str(pid_file),
                sys.executable,
                marker,
                expected_command,
            ],
            capture_output=True,
            text=True,
        )
        assert stop_result.returncode == 0, stop_result.stderr
        assert not pid_file.exists()
    finally:
        if runner_pid is not None:
            try:
                os.kill(runner_pid, signal.SIGTERM)
            except ProcessLookupError:
                pass


def test_darwin_runner_uses_launchd_without_secrets_in_arguments(tmp_path: Path) -> None:
    environment_file = tmp_path / "runner.env"
    environment_file.write_text(
        "WORD_AGENT_CLI_RUNNER_DB_DSN=secret-dsn\n",
        encoding="utf-8",
    )
    environment_file.chmod(0o600)
    command_file = tmp_path / "launchctl.commands"
    pid_file = tmp_path / "runner.pid"
    log_file = tmp_path / "runner.log"
    word_agent_directory = tmp_path / "word-agent"
    word_agent_directory.mkdir()
    marker = f"word-select-dashboard:{tmp_path}"
    label = "com.example.word-select-dashboard-runner"

    result = subprocess.run(
        [
            "zsh",
            "-c",
            'uname() { print Darwin; }; id() { print 501; }; '
            'launchctl() { print -r -- "$*" >> "$LAUNCHCTL_COMMAND_FILE"; '
            'if [[ "$1" == "print" ]]; then print "pid = 4242"; fi; }; '
            'source "$1"; launch_project_runner "$2" "$3" "$4" "$5" "$6" "$7" "$8"',
            "zsh",
            str(RUNTIME_HELPER_PATH),
            sys.executable,
            str(word_agent_directory),
            marker,
            str(log_file),
            str(pid_file),
            str(environment_file),
            label,
        ],
        env={**os.environ, "LAUNCHCTL_COMMAND_FILE": str(command_file)},
        capture_output=True,
        text=True,
    )

    assert result.returncode == 0, result.stderr
    assert pid_file.read_text(encoding="utf-8").strip() == "4242"
    assert stat.S_IMODE(pid_file.stat().st_mode) == 0o600
    command_text = command_file.read_text(encoding="utf-8")
    assert f"submit -l {label}" in command_text
    assert f"-o {log_file} -e {log_file}" in command_text
    assert str(environment_file) in command_text
    assert "secret-dsn" not in command_text


def test_darwin_runner_removes_submitted_job_when_pid_never_appears(tmp_path: Path) -> None:
    environment_file = tmp_path / "runner.env"
    environment_file.write_text(
        "WORD_AGENT_CLI_RUNNER_DB_DSN=test\n",
        encoding="utf-8",
    )
    environment_file.chmod(0o600)
    command_file = tmp_path / "launchctl.commands"

    result = subprocess.run(
        [
            "zsh",
            "-c",
            'uname() { print Darwin; }; id() { print 501; }; sleep() { :; }; '
            'launchctl() { print -r -- "$*" >> "$LAUNCHCTL_COMMAND_FILE"; }; '
            'source "$1"; launch_project_runner "$2" "$3" "$4" "$5" "$6" "$7" "$8"',
            "zsh",
            str(RUNTIME_HELPER_PATH),
            sys.executable,
            str(tmp_path),
            f"word-select-dashboard:{tmp_path}",
            str(tmp_path / "runner.log"),
            str(tmp_path / "runner.pid"),
            str(environment_file),
            "com.example.failed-runner",
        ],
        env={**os.environ, "LAUNCHCTL_COMMAND_FILE": str(command_file)},
        capture_output=True,
        text=True,
    )

    assert result.returncode == 3
    assert "remove com.example.failed-runner" in command_file.read_text(encoding="utf-8")


def test_launchd_stop_reports_remove_failure_and_keeps_pid_file(tmp_path: Path) -> None:
    pid_file = tmp_path / "runner.pid"
    pid_file.write_text("4242\n", encoding="utf-8")

    result = subprocess.run(
        [
            "zsh",
            "-c",
            'source "$1"; project_launchd_pid() { return 1; }; '
            'launchctl() { return 9; }; '
            'stop_project_launchd_runner test.label "$2" /test/python test-marker',
            "zsh",
            str(RUNTIME_HELPER_PATH),
            str(pid_file),
        ],
        capture_output=True,
        text=True,
    )

    assert result.returncode != 0
    assert "无法移除 launchd CLI Runner" in result.stderr
    assert pid_file.exists()


def test_stop_script_stops_containers_but_fails_when_launchd_runner_remains(
    tmp_path: Path,
) -> None:
    scripts_directory = tmp_path / "scripts"
    scripts_directory.mkdir()
    copied_stop = scripts_directory / STOP_PATH.name
    copied_helper = scripts_directory / RUNTIME_HELPER_PATH.name
    shutil.copy2(STOP_PATH, copied_stop)
    shutil.copy2(RUNTIME_HELPER_PATH, copied_helper)
    runtime_directory = tmp_path / ".runtime"
    runtime_directory.mkdir()
    (runtime_directory / "cli-runner.pid").write_text("4242\n", encoding="utf-8")
    fake_bin = tmp_path / "bin"
    fake_bin.mkdir()
    docker_log = tmp_path / "docker.log"
    (fake_bin / "docker").write_text(
        '#!/bin/sh\nprintf "%s\\n" "$*" >> "$DOCKER_LOG"\n', encoding="utf-8"
    )
    (fake_bin / "launchctl").write_text(
        '#!/bin/sh\nif [ "$1" = "print" ]; then exit 0; fi\n'
        'if [ "$1" = "remove" ]; then exit 9; fi\nexit 0\n',
        encoding="utf-8",
    )
    (fake_bin / "docker").chmod(0o755)
    (fake_bin / "launchctl").chmod(0o755)

    result = subprocess.run(
        ["zsh", str(copied_stop)],
        env={
            **os.environ,
            "PATH": f"{fake_bin}:/usr/bin:/bin:/usr/sbin:/sbin",
            "DOCKER_LOG": str(docker_log),
        },
        capture_output=True,
        text=True,
    )

    assert result.returncode != 0
    assert docker_log.read_text(encoding="utf-8").count(" stop") == 3
    assert "容器已停止，但 CLI Runner 未停止" in result.stderr
    assert "当前项目服务已停止" not in result.stdout


@pytest.mark.parametrize(
    "config_text",
    [
        "pgsql: []",
        "pgsql:\n  path: localhost",
        "!!python/object/apply:os.system ['echo unsafe']",
    ],
)
def test_invalid_server_database_config_is_rejected(tmp_path: Path, config_text: str) -> None:
    bootstrap = load_bootstrap()
    config_path = tmp_path / "config.yaml"
    config_path.write_text(config_text, encoding="utf-8")

    with pytest.raises(bootstrap.BootstrapConfigError):
        bootstrap.resolve_database_dsns(config_path)


def test_start_and_stop_scripts_are_project_scoped_and_secret_safe() -> None:
    start = START_PATH.read_text(encoding="utf-8")
    stop = STOP_PATH.read_text(encoding="utf-8")

    for relative_compose in (
        "word_select_dashboard/server/docker-compose.yml",
        "word_select_dashboard/web-react/docker-compose.yml",
        "word_select_dashboard/word-agent/docker-compose.yml",
    ):
        assert relative_compose in start
        assert relative_compose in stop

    project_bindings = (
        ("SERVER_COMPOSE_PROJECT", "word-select-dashboard-server", "SERVER_COMPOSE"),
        ("WEB_COMPOSE_PROJECT", "word-select-dashboard-web", "WEB_COMPOSE"),
        ("AGENT_COMPOSE_PROJECT", "word-agent", "AGENT_COMPOSE"),
    )
    for variable, project_name, compose_variable in project_bindings:
        assignment = f'{variable}="{project_name}"'
        assert assignment in start
        assert assignment in stop
        start_command = (
            f'docker compose --project-name "${variable}" -f "${compose_variable}" '
            'up -d "${COMPOSE_UP_BUILD_ARGS[@]}"'
        )
        stop_command = (
            f'docker compose --project-name "${variable}" -f "${compose_variable}" stop'
        )
        assert start_command in start
        assert stop_command in stop
    assert start.count("docker compose --project-name") == 3
    assert stop.count("docker compose --project-name") == 3

    assert 'chmod 700 "$RUNTIME_DIR"' in start
    assert 'chmod 600 "$ENV_FILE"' in start
    assert 'chmod 600 "$LOG_FILE"' in start
    assert "bootstrap_sentence_cli_configs.py" in start
    assert '"$PYTHON"' in start
    assert "word_select_dashboard_runtime.sh" in start
    assert "acquire_lifecycle_lock" in start
    assert "stop_existing_project_runner" in start
    assert start.index("stop_existing_project_runner") < start.index(
        "launch_project_runner"
    )
    assert "launch_project_runner" in start
    assert "on_start_exit $?" in start
    assert "resolve_compose_up_build_args" in start
    assert start.index("resolve_compose_up_build_args") < start.index("acquire_lifecycle_lock")
    assert start.count('up -d "${COMPOSE_UP_BUILD_ARGS[@]}"') == 3
    assert "WORD_AGENT_CLI_RUNNER_CONFIG_URL" not in start
    assert "WORD_AGENT_CLI_RUNNER_TOKEN" not in start
    assert "/ai/execution-config" not in start
    assert "TOKEN_FILE" not in start
    assert "openssl" not in start

    assert 'PYTHON="$PROJECT_ROOT/word_select_dashboard/word-agent/.venv/bin/python"' in stop
    assert "word_select_dashboard_runtime.sh" in stop
    assert "acquire_lifecycle_lock" in stop
    assert "release_lifecycle_lock" in stop
    helper = RUNTIME_HELPER_PATH.read_text(encoding="utf-8")
    assert "release_lifecycle_lock" in helper
    assert "部分服务可能已启动，使用 stop 脚本" in helper
    assert "project_runner_process_is_valid" in helper
    assert "word_agent.cli_runner.main" in helper
    assert "--runner-marker=" in helper
    assert "atomic_write_runner_pid" in helper
    assert "exec nohup" in helper
    assert "</dev/null" in helper
    assert "launchctl submit" in helper
    assert "launchctl remove" in helper
    assert "source \"$1\"" in helper
    assert "RUNNER_LAUNCH_LABEL" in start
    assert "RUNNER_LAUNCH_LABEL" in stop
    assert "cli-runner.env" in start
    assert "cli-runner.env" not in stop
    assert 'mv "$temporary_pid_file" "$pid_file"' in helper
    assert "cli-runner.env" not in stop
    assert "WORD_AGENT_CLI_RUNNER_TOKEN" not in stop
    assert "docker compose" in stop
    assert " stop" in stop
    assert " down" not in stop
    for forbidden in ("postgres", "redis", "minio", "docker stop", "docker kill"):
        assert forbidden not in stop.lower()


def test_word_agent_compose_uses_unauthenticated_host_runner() -> None:
    compose = COMPOSE_PATH.read_text(encoding="utf-8")

    assert "WORD_AGENT_CLI_RUNNER_URL:" in compose
    assert "${WORD_AGENT_CONTAINER_CLI_RUNNER_URL:?" in compose
    assert "WORD_AGENT_CLI_RUNNER_TOKEN" not in compose
    assert 'WORD_AGENT_SELECT_DB_DSN: "${WORD_AGENT_SELECT_DB_DSN:?' in compose
    assert "dbname=select_english_word user=conchi" not in compose


def test_full_compose_projects_use_root_runtime_contract() -> None:
    agent = COMPOSE_PATH.read_text(encoding="utf-8")
    go_backend = GO_COMPOSE_PATH.read_text(encoding="utf-8")
    java_backend = JAVA_COMPOSE_PATH.read_text(encoding="utf-8")

    assert '${WORD_AGENT_SELECT_DB_DSN:?' in agent
    assert '${WORD_AGENT_ROB_WORD_DB_DSN:?' in agent
    assert '${WORD_AGENT_CONTAINER_CLI_RUNNER_URL:?' in agent
    assert "../server/config.docker.yaml:/app/server-config/config.yaml:ro" in agent
    assert "../server/config.yaml:/app/server-config/config.yaml:ro" not in agent

    for variable in (
        "SELECT_DB_CONTAINER_HOST",
        "SELECT_DB_PORT",
        "SELECT_DB_NAME",
        "SELECT_DB_USER",
        "SELECT_DB_PASSWORD",
        "REDIS_CONTAINER_ADDR",
        "MINIO_CONTAINER_ENDPOINT",
        "WORD_AGENT_CONTAINER_URL",
    ):
        assert f"${{{variable}:?" in go_backend

    for variable in (
        "SPRING_DATASOURCE_URL",
        "SPRING_DATASOURCE_USERNAME",
        "SPRING_DATASOURCE_PASSWORD",
        "SPRING_DATA_REDIS_HOST",
        "WORD_AGENT_CONTAINER_URL",
    ):
        assert f"${{{variable}:?" in java_backend

    versioned_runtime = "\n".join((agent, go_backend, java_backend))
    assert "host.docker.internal:8010" not in versioned_runtime
    assert "conchi123456" not in versioned_runtime


def test_frontend_nginx_upstreams_are_runtime_templated() -> None:
    expected_variables = (
        ("rob_english_word_front", "ROB_WORD_HTTP_PORT", "DASHBOARD_SERVER_PORT"),
        ("rob_english_word_cloze_web", "ROB_WORD_HTTP_PORT", "DASHBOARD_SERVER_PORT"),
        ("web-react", "DASHBOARD_SERVER_PORT"),
    )
    for project, variables in zip(FRONTEND_PROJECTS, expected_variables, strict=True):
        dockerfile = (project / "Dockerfile").read_text(encoding="utf-8")
        nginx = (project / "nginx.conf").read_text(encoding="utf-8")
        compose = (project / "docker-compose.yml").read_text(encoding="utf-8")
        assert "/etc/nginx/templates/default.conf.template" in dockerfile
        for variable in variables[1:]:
            assert f"${{{variable}}}" in nginx
            assert f"{variable}:" in compose


def test_environment_example_uses_database_dsn_not_private_config_api() -> None:
    example = ENV_EXAMPLE_PATH.read_text(encoding="utf-8")

    assert "WORD_AGENT_CLI_RUNNER_URL=http://127.0.0.1:6018" in example
    assert "WORD_AGENT_CLI_RUNNER_TOKEN" not in example
    assert "WORD_AGENT_CLI_RUNNER_DB_DSN=" in example
    assert "WORD_AGENT_SELECT_DB_DSN=" in example
    assert "WORD_AGENT_CLI_RUNNER_PORT=6018" in example
    assert "WORD_AGENT_CLI_RUNNER_CONFIG_URL" not in example


def test_root_native_startup_manages_current_ports_and_cli_runner() -> None:
    start = ROOT_START_PATH.read_text(encoding="utf-8")

    assert "LOCAL_PORTS=(6011 6012 6013 6014 6015 6016 6017 6018)" in start
    assert "cli_runner" in start
    assert start.index("start_cli_runner") < start.index("start_word_agent")
    assert "bootstrap_sentence_cli_configs.py" in start
    assert "word_agent.cli_runner.main" in start
    assert "WORD_AGENT_CLI_RUNNER_URL=http://127.0.0.1:6018" in start
    assert "WORD_AGENT_BASE_URL=${WORD_AGENT_BASE_URL:-http://127.0.0.1:6017}" in start
    assert "for attempt in 1 2 3; do" in start
    assert 'if launchctl bootstrap "$domain" "$plist"; then' in start
    assert 'die "launchctl bootstrap failed for $name after 3 attempts"' in start

    for service, port in (
        ("rob_english_word_front", 6011),
        ("rob_english_word_back", 6012),
        ("rob_english_word_back_websocket", 6013),
        ("rob_english_word_cloze_web", 6014),
        ("word_select_dashboard_server", 6015),
        ("word_select_dashboard_web", 6016),
        ("word_agent", 6017),
        ("cli_runner", 6018),
    ):
        assert f"wait_for_port {service} {port} " in start
        assert f"http://127.0.0.1:{port}" in start or (
            service == "rob_english_word_back_websocket"
            and f"ws://127.0.0.1:{port}" in start
        )

    for stale_port in (7001, 7002, 7003, 8009, 8010, 8019, 9091):
        assert str(stale_port) not in start
