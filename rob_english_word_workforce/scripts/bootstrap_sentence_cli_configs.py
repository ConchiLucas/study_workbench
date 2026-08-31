#!/usr/bin/env python3
"""Seed the two local sentence CLI providers without selecting either one."""

from __future__ import annotations

import argparse
import os
import re
import shlex
import sys
import tempfile
from collections.abc import Mapping
from pathlib import Path
from typing import Protocol

import psycopg
import yaml

BOOTSTRAP_ADVISORY_LOCK_ID = 0x45584543434647
CODEX_COMMAND_PATH = "/Applications/ChatGPT.app/Contents/Resources/codex"
GEMINI_COMMAND_PATH = "/Users/conchi/.npm-global/bin/gemini"
_SSLMODE_PATTERN = re.compile(r"(?:^|\s)sslmode=([^\s]+)")


class BootstrapConfigError(RuntimeError):
    """Raised when a safe database/bootstrap configuration cannot be resolved."""


class Cursor(Protocol):
    def __enter__(self) -> Cursor: ...

    def __exit__(self, exc_type, exc, traceback) -> None: ...

    def execute(self, query: str, params: tuple[object, ...]) -> None: ...


class Connection(Protocol):
    def cursor(self) -> Cursor: ...


def resolve_database_dsns(config_path: Path) -> tuple[str, str]:
    host_environment_dsn = os.environ.get("WORD_AGENT_CLI_RUNNER_DB_DSN", "").strip()
    container_environment_dsn = os.environ.get("WORD_AGENT_SELECT_DB_DSN", "").strip()
    if host_environment_dsn or container_environment_dsn:
        if not host_environment_dsn or not container_environment_dsn:
            raise BootstrapConfigError("统一运行配置中的数据库 DSN 不完整")
        return host_environment_dsn, container_environment_dsn

    try:
        document = yaml.safe_load(config_path.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, yaml.YAMLError) as exc:
        raise BootstrapConfigError("无法安全读取 server/config.yaml 数据库配置") from exc

    if not isinstance(document, dict) or not isinstance(document.get("pgsql"), dict):
        raise BootstrapConfigError("server/config.yaml 缺少 pgsql 配置")
    pgsql = document["pgsql"]
    host = _required_config_text(pgsql, "path")
    shared_fields = {
        "port": _required_config_text(pgsql, "port"),
        "dbname": _required_config_text(pgsql, "db-name"),
        "user": _required_config_text(pgsql, "username"),
        "password": _required_config_text(pgsql, "password"),
    }
    raw_options = str(pgsql.get("config") or "")
    sslmode_match = _SSLMODE_PATTERN.search(raw_options)
    if sslmode_match:
        shared_fields["sslmode"] = sslmode_match.group(1)
    try:
        return (
            psycopg.conninfo.make_conninfo(host=host, **shared_fields),
            psycopg.conninfo.make_conninfo(host="host.docker.internal", **shared_fields),
        )
    except psycopg.Error as exc:
        raise BootstrapConfigError("server/config.yaml 的 pgsql 配置无效") from exc


def _required_config_text(config: Mapping[str, object], key: str) -> str:
    value = str(config.get(key) or "").strip()
    if not value or "\x00" in value or "\n" in value or "\r" in value:
        raise BootstrapConfigError(f"server/config.yaml 的 pgsql.{key} 无效")
    return value


def insert_missing_cli_configs(
    connection: Connection,
    *,
    project_root: Path,
) -> tuple[str, str]:
    working_directory = str(project_root.resolve())
    codex_command_path = os.environ.get(
        "CODEX_COMMAND_PATH", CODEX_COMMAND_PATH
    ).strip() or CODEX_COMMAND_PATH
    gemini_command_path = os.environ.get(
        "GEMINI_COMMAND_PATH", GEMINI_COMMAND_PATH
    ).strip() or GEMINI_COMMAND_PATH
    providers = (
        (
            "codex",
            "Codex CLI",
            "codex",
            codex_command_path,
            "gpt-5.6-sol",
            "high",
            working_directory,
            300,
            True,
        ),
        (
            "gemini",
            "Gemini CLI",
            "gemini",
            gemini_command_path,
            "auto",
            "",
            working_directory,
            300,
            True,
        ),
    )
    with connection.cursor() as cursor:
        cursor.execute(
            "SELECT pg_advisory_xact_lock(%s)",
            (BOOTSTRAP_ADVISORY_LOCK_ID,),
        )
        for provider in providers:
            cursor.execute(
                """
                INSERT INTO cli_provider_configs
                    (provider_id, label, driver, command_path, model,
                     reasoning_effort, working_directory, timeout_seconds, enabled,
                     created_at, updated_at)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, NOW(), NOW())
                ON CONFLICT (provider_id) DO NOTHING
                """,
                provider,
            )
    return ("codex", "gemini")


def bootstrap_cli_configs(database_dsn: str, *, project_root: Path) -> tuple[str, str]:
    try:
        with psycopg.connect(
            database_dsn,
            connect_timeout=5,
            options="-c statement_timeout=10000 -c lock_timeout=5000",
        ) as connection:
            return insert_missing_cli_configs(connection, project_root=project_root)
    except psycopg.Error as exc:
        raise BootstrapConfigError("初始化本地 CLI 配置失败") from exc


def write_runner_environment(
    env_path: Path,
    *,
    host_database_dsn: str,
    container_database_dsn: str,
) -> None:
    clean_host_dsn = host_database_dsn.strip()
    clean_container_dsn = container_database_dsn.strip()
    values = (clean_host_dsn, clean_container_dsn)
    if any(not value for value in values):
        raise BootstrapConfigError("Runner 数据库 DSN 为空")
    if any(character in "".join(values) for character in ("\x00", "\n", "\r")):
        raise BootstrapConfigError("Runner 环境配置包含非法换行或空字节")

    env_path.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
    env_path.parent.chmod(0o700)
    lines = (
        f"WORD_AGENT_CLI_RUNNER_DB_DSN={shlex.quote(clean_host_dsn)}\n"
        f"WORD_AGENT_SELECT_DB_DSN={shlex.quote(clean_container_dsn)}\n"
    )
    temporary_path: Path | None = None
    try:
        with tempfile.NamedTemporaryFile(
            mode="w",
            encoding="utf-8",
            dir=env_path.parent,
            prefix=f".{env_path.name}.",
            delete=False,
        ) as temporary:
            temporary_path = Path(temporary.name)
            os.fchmod(temporary.fileno(), 0o600)
            temporary.write(lines)
        os.replace(temporary_path, env_path)
        env_path.chmod(0o600)
    finally:
        if temporary_path is not None and temporary_path.exists():
            temporary_path.unlink()


def _parse_args() -> argparse.Namespace:
    default_project_root = Path(__file__).resolve().parents[1]
    parser = argparse.ArgumentParser(
        description="Insert missing Codex/Gemini CLI configs without changing the active executor."
    )
    parser.add_argument("--project-root", type=Path, default=default_project_root)
    parser.add_argument(
        "--config",
        type=Path,
        default=default_project_root / "word_select_dashboard" / "server" / "config.yaml",
    )
    parser.add_argument("--runner-env-file", type=Path)
    return parser.parse_args()


def main() -> int:
    args = _parse_args()
    try:
        host_database_dsn, container_database_dsn = resolve_database_dsns(args.config)
        if args.runner_env_file is not None:
            write_runner_environment(
                args.runner_env_file,
                host_database_dsn=host_database_dsn,
                container_database_dsn=container_database_dsn,
            )
        bootstrap_cli_configs(host_database_dsn, project_root=args.project_root)
    except BootstrapConfigError as exc:
        print(str(exc), file=sys.stderr)
        return 1
    print("本地 CLI 配置已完成幂等初始化；当前造句执行器未改变。")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
