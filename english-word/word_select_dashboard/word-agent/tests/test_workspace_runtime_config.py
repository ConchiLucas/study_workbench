from __future__ import annotations

import importlib.util
import os
from pathlib import Path
from types import ModuleType

import pytest

PROJECT_ROOT = Path(__file__).resolve().parents[3]
RESOLVER_PATH = PROJECT_ROOT / "scripts" / "workspace_runtime_config.py"


def load_resolver() -> ModuleType:
    spec = importlib.util.spec_from_file_location("workspace_runtime_config", RESOLVER_PATH)
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def valid_values(**overrides: str) -> dict[str, str]:
    values = {
        "SELECT_DB_HOST": "127.0.0.1",
        "SELECT_DB_PORT": "5432",
        "SELECT_DB_NAME": "select_english_word",
        "SELECT_DB_USER": "select_user",
        "SELECT_DB_PASSWORD": "select password/@",
        "SELECT_DB_SSLMODE": "disable",
        "ROB_WORD_DB_HOST": "localhost",
        "ROB_WORD_DB_PORT": "5432",
        "ROB_WORD_DB_NAME": "rob_english_word",
        "ROB_WORD_DB_USER": "rob_user",
        "ROB_WORD_DB_PASSWORD": "rob password/@",
        "ROB_WORD_DB_SSLMODE": "disable",
        "REDIS_HOST": "127.0.0.1",
        "REDIS_PORT": "6379",
        "REDIS_PASSWORD": "",
        "MINIO_HOST": "localhost",
        "MINIO_PORT": "19100",
        "MINIO_ACCESS_KEY": "minio-user",
        "MINIO_SECRET_KEY": "minio secret/@",
        "MINIO_BUCKET": "ai-file-navigation",
        "MINIO_USE_SSL": "false",
    }
    values.update(overrides)
    return values


def test_derive_runtime_converts_loopback_only_for_containers() -> None:
    resolver = load_resolver()

    runtime = resolver.derive_runtime(valid_values())

    assert runtime["SELECT_DB_HOST_RUNTIME"] == "127.0.0.1"
    assert runtime["SELECT_DB_CONTAINER_HOST"] == "host.docker.internal"
    assert runtime["ROB_WORD_DB_HOST_RUNTIME"] == "localhost"
    assert runtime["ROB_WORD_DB_CONTAINER_HOST"] == "host.docker.internal"
    assert runtime["REDIS_CONTAINER_HOST"] == "host.docker.internal"
    assert runtime["MINIO_CONTAINER_HOST"] == "host.docker.internal"
    assert "host.docker.internal" in runtime["WORD_AGENT_SELECT_DB_DSN"]
    assert "127.0.0.1" in runtime["WORD_AGENT_CLI_RUNNER_DB_DSN"]


def test_derive_runtime_preserves_remote_hosts() -> None:
    resolver = load_resolver()

    runtime = resolver.derive_runtime(
        valid_values(
            SELECT_DB_HOST="db.example.internal",
            ROB_WORD_DB_HOST="10.0.0.8",
            REDIS_HOST="redis.example.internal",
            MINIO_HOST="10.0.0.9",
        )
    )

    assert runtime["SELECT_DB_CONTAINER_HOST"] == "db.example.internal"
    assert runtime["ROB_WORD_DB_CONTAINER_HOST"] == "10.0.0.8"
    assert runtime["REDIS_CONTAINER_HOST"] == "redis.example.internal"
    assert runtime["MINIO_CONTAINER_HOST"] == "10.0.0.9"


def test_derive_runtime_encodes_postgres_credentials() -> None:
    resolver = load_resolver()

    runtime = resolver.derive_runtime(valid_values())

    select_dsn = runtime["WORD_AGENT_SELECT_DB_DSN"]
    rob_dsn = runtime["WORD_AGENT_ROB_WORD_DB_DSN"]
    assert "select%20password%2F%40" in select_dsn
    assert "rob%20password%2F%40" in rob_dsn
    assert "select password/@" not in select_dsn
    assert "rob password/@" not in rob_dsn


def test_missing_fields_are_reported_together_without_values() -> None:
    resolver = load_resolver()
    values = valid_values()
    values["SELECT_DB_HOST"] = ""
    values["MINIO_SECRET_KEY"] = ""

    with pytest.raises(resolver.ConfigError) as exc_info:
        resolver.derive_runtime(values)

    message = str(exc_info.value)
    assert "SELECT_DB_HOST" in message
    assert "MINIO_SECRET_KEY" in message
    assert "select password/@" not in message
    assert "minio secret/@" not in message


def test_write_runtime_environment_is_private_and_shell_loadable(
    tmp_path: Path,
) -> None:
    resolver = load_resolver()
    output = tmp_path / "runtime.env"
    runtime = resolver.derive_runtime(valid_values())

    resolver.write_runtime_environment(output, runtime)

    assert output.stat().st_mode & 0o777 == 0o600
    contents = output.read_text(encoding="utf-8")
    assert "WORD_AGENT_SELECT_DB_DSN=" in contents
    loaded: dict[str, str] = {}
    for line in contents.splitlines():
        key, value = line.split("=", 1)
        loaded[key] = resolver.parse_shell_value(value)
    assert loaded == runtime
    assert os.access(output, os.R_OK)


def test_write_runtime_environment_preserves_existing_parent_permissions(
    tmp_path: Path,
) -> None:
    resolver = load_resolver()
    parent = tmp_path / "shared"
    parent.mkdir(mode=0o755)
    parent.chmod(0o755)

    resolver.write_runtime_environment(
        parent / "runtime.env",
        resolver.derive_runtime(valid_values()),
    )

    assert parent.stat().st_mode & 0o777 == 0o755
