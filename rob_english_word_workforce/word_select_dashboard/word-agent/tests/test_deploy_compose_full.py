from __future__ import annotations

import os
import stat
import subprocess
from pathlib import Path

PROJECT_ROOT = Path(__file__).resolve().parents[3]
DEPLOY_SCRIPT = PROJECT_ROOT / "deploy-compose-full.sh"


def valid_env_text() -> str:
    return """
SELECT_DB_HOST=127.0.0.1
SELECT_DB_PORT=5432
SELECT_DB_NAME=select_english_word
SELECT_DB_USER=select_user
SELECT_DB_PASSWORD='select secret'
SELECT_DB_SSLMODE=disable
ROB_WORD_DB_HOST=127.0.0.1
ROB_WORD_DB_PORT=5432
ROB_WORD_DB_NAME=rob_english_word
ROB_WORD_DB_USER=rob_user
ROB_WORD_DB_PASSWORD='rob secret'
ROB_WORD_DB_SSLMODE=disable
REDIS_HOST=127.0.0.1
REDIS_PORT=6379
REDIS_PASSWORD=
MINIO_HOST=127.0.0.1
MINIO_PORT=19100
MINIO_ACCESS_KEY=minio-user
MINIO_SECRET_KEY='minio secret'
MINIO_BUCKET=ai-file-navigation
MINIO_USE_SSL=false
ROB_WORD_HTTP_PORT=10111
ROB_WORD_WEBSOCKET_PORT=10112
ROB_WORD_FRONT_PORT=6111
CLOZE_WEB_PORT=6014
DASHBOARD_SERVER_PORT=6015
DASHBOARD_WEB_PORT=6016
WORD_AGENT_PORT=6017
CLI_RUNNER_PORT=6018
""".strip()


def write_fake_docker(directory: Path, log_path: Path) -> None:
    executable = directory / "docker"
    executable.write_text(
        "#!/bin/sh\n"
        "printf '%s\\n' \"$*\" >> \"$DOCKER_LOG\"\n"
        "if [ \"${1:-}\" = inspect ]; then printf 'true false\\n'; fi\n"
        "exit 0\n",
        encoding="utf-8",
    )
    executable.chmod(executable.stat().st_mode | stat.S_IXUSR)


def write_fake_curl(directory: Path) -> None:
    executable = directory / "curl"
    executable.write_text("#!/bin/sh\nexit 0\n", encoding="utf-8")
    executable.chmod(executable.stat().st_mode | stat.S_IXUSR)


def run_script(
    tmp_path: Path,
    env_text: str,
    *script_args: str,
    skip_dependencies: bool = True,
) -> tuple[subprocess.CompletedProcess[str], Path]:
    config = tmp_path / ".env.local"
    config.write_text(env_text + "\n", encoding="utf-8")
    fake_bin = tmp_path / "bin"
    fake_bin.mkdir()
    docker_log = tmp_path / "docker.log"
    write_fake_docker(fake_bin, docker_log)
    write_fake_curl(fake_bin)
    environment = {
        **os.environ,
        "PATH": f"{fake_bin}:{os.environ['PATH']}",
        "DOCKER_LOG": str(docker_log),
        "WORKSPACE_ENV_FILE": str(config),
    }
    if skip_dependencies:
        environment["WORKSPACE_SKIP_DEPENDENCY_CHECK"] = "1"
    else:
        environment.pop("WORKSPACE_SKIP_DEPENDENCY_CHECK", None)
    result = subprocess.run(
        [str(DEPLOY_SCRIPT), *script_args],
        cwd=tmp_path,
        env=environment,
        capture_output=True,
        text=True,
        timeout=30,
    )
    return result, docker_log


def run_check(
    tmp_path: Path,
    env_text: str,
    *,
    skip_dependencies: bool = True,
) -> tuple[subprocess.CompletedProcess[str], Path]:
    return run_script(
        tmp_path,
        env_text,
        "--check",
        skip_dependencies=skip_dependencies,
    )


def test_check_works_from_outside_repository_and_validates_six_compose_files(
    tmp_path: Path,
) -> None:
    result, docker_log = run_check(tmp_path, valid_env_text())

    assert result.returncode == 0, result.stderr
    calls = docker_log.read_text(encoding="utf-8").splitlines()
    compose_config_calls = [
        call for call in calls if " compose " in f" {call} " and " config -q" in call
    ]
    assert len(compose_config_calls) == 6
    assert not any(" up " in f" {call} " or " build " in f" {call} " for call in calls)
    assert "统一运行配置与六个 Compose 校验通过" in result.stdout


def test_missing_config_is_aggregated_before_docker_is_called(tmp_path: Path) -> None:
    result, docker_log = run_check(tmp_path, "SELECT_DB_HOST=127.0.0.1")

    assert result.returncode != 0
    assert "SELECT_DB_PORT" in result.stderr
    assert "REDIS_HOST" in result.stderr
    assert "MINIO_HOST" in result.stderr
    assert not docker_log.exists()
    assert "select secret" not in result.stdout + result.stderr


def test_script_uses_script_directory_and_starts_cli_before_agent() -> None:
    script = DEPLOY_SCRIPT.read_text(encoding="utf-8")

    assert '$(dirname -- "$0")' in script
    assert "ROOT_DIR=$(pwd)" not in script
    assert "--check" in script
    assert script.index("start_cli_runner") < script.index("start_word_agent")
    assert "docker compose" in script
    assert "wait_for_container" in script
    assert "wait_for_url" in script


def test_project_mode_starts_only_requested_project_from_unified_config(
    tmp_path: Path,
) -> None:
    result, docker_log = run_script(
        tmp_path,
        valid_env_text(),
        "--project",
        "rob-english-word-front",
    )

    assert result.returncode == 0, result.stderr
    calls = docker_log.read_text(encoding="utf-8").splitlines()
    compose_config_calls = [call for call in calls if " config -q" in call]
    compose_up_calls = [call for call in calls if " up --build -d" in call]
    assert len(compose_config_calls) == 6
    assert len(compose_up_calls) == 1
    assert "rob_english_word_front/docker-compose.yml" in compose_up_calls[0]
    assert "项目 rob-english-word-front 已启动并通过检查" in result.stdout


def test_unreachable_dependency_reports_its_name(tmp_path: Path) -> None:
    config = valid_env_text().replace("SELECT_DB_PORT=5432", "SELECT_DB_PORT=1")

    result, _docker_log = run_check(
        tmp_path,
        config,
        skip_dependencies=False,
    )

    assert result.returncode != 0
    assert "select_english_word PostgreSQL 不可达" in result.stderr
