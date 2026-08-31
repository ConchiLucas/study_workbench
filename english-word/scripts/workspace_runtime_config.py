#!/usr/bin/env python3
"""Validate one workspace config and derive safe host/container runtime variables."""

from __future__ import annotations

import argparse
import os
import shlex
import stat
import tempfile
from collections.abc import Mapping
from pathlib import Path
from urllib.parse import quote


class ConfigError(RuntimeError):
    """Raised when the workspace runtime configuration is incomplete or invalid."""


_REQUIRED_FIELDS = (
    "SELECT_DB_HOST",
    "SELECT_DB_PORT",
    "SELECT_DB_NAME",
    "SELECT_DB_USER",
    "SELECT_DB_PASSWORD",
    "SELECT_DB_SSLMODE",
    "ROB_WORD_DB_HOST",
    "ROB_WORD_DB_PORT",
    "ROB_WORD_DB_NAME",
    "ROB_WORD_DB_USER",
    "ROB_WORD_DB_PASSWORD",
    "ROB_WORD_DB_SSLMODE",
    "REDIS_HOST",
    "REDIS_PORT",
    "MINIO_HOST",
    "MINIO_PORT",
    "MINIO_ACCESS_KEY",
    "MINIO_SECRET_KEY",
    "MINIO_BUCKET",
    "MINIO_USE_SSL",
)

_PORT_DEFAULTS = {
    "ROB_WORD_HTTP_PORT": "10111",
    "ROB_WORD_WEBSOCKET_PORT": "10112",
    "ROB_WORD_FRONT_PORT": "6111",
    "CLOZE_WEB_PORT": "6014",
    "DASHBOARD_SERVER_PORT": "6015",
    "DASHBOARD_WEB_PORT": "6016",
    "WORD_AGENT_PORT": "6017",
    "CLI_RUNNER_PORT": "6018",
}


def container_host(host: str) -> str:
    normalized = host.strip()
    if normalized.lower() in {"127.0.0.1", "localhost", "::1"}:
        return "host.docker.internal"
    return normalized


def _require_values(values: Mapping[str, str]) -> dict[str, str]:
    normalized = {key: str(values.get(key, "")).strip() for key in _REQUIRED_FIELDS}
    missing = [key for key, value in normalized.items() if not value]
    if missing:
        raise ConfigError("缺少必填配置: " + ", ".join(missing))
    return normalized


def _port(value: str, field: str) -> str:
    try:
        port = int(value)
    except ValueError as exc:
        raise ConfigError(f"{field} 必须是 1-65535 的端口") from exc
    if port < 1 or port > 65535:
        raise ConfigError(f"{field} 必须是 1-65535 的端口")
    return str(port)


def _boolean(value: str, field: str) -> str:
    normalized = value.strip().lower()
    if normalized not in {"true", "false"}:
        raise ConfigError(f"{field} 必须是 true 或 false")
    return normalized


def _uri_host(host: str) -> str:
    return f"[{host}]" if ":" in host and not host.startswith("[") else host


def _postgres_uri(
    *,
    host: str,
    port: str,
    name: str,
    user: str,
    password: str,
    sslmode: str,
) -> str:
    return (
        "postgresql://"
        f"{quote(user, safe='')}:{quote(password, safe='')}"
        f"@{_uri_host(host)}:{port}/{quote(name, safe='')}"
        f"?sslmode={quote(sslmode, safe='')}"
    )


def derive_runtime(values: Mapping[str, str]) -> dict[str, str]:
    config = _require_values(values)
    for field in ("SELECT_DB_PORT", "ROB_WORD_DB_PORT", "REDIS_PORT", "MINIO_PORT"):
        config[field] = _port(config[field], field)
    config["MINIO_USE_SSL"] = _boolean(config["MINIO_USE_SSL"], "MINIO_USE_SSL")

    ports = {
        field: _port(str(values.get(field, default)).strip() or default, field)
        for field, default in _PORT_DEFAULTS.items()
    }
    select_host = config["SELECT_DB_HOST"]
    select_container_host = container_host(select_host)
    rob_host = config["ROB_WORD_DB_HOST"]
    rob_container_host = container_host(rob_host)
    redis_host = config["REDIS_HOST"]
    redis_container_host = container_host(redis_host)
    minio_host = config["MINIO_HOST"]
    minio_container_host = container_host(minio_host)

    select_host_dsn = _postgres_uri(
        host=select_host,
        port=config["SELECT_DB_PORT"],
        name=config["SELECT_DB_NAME"],
        user=config["SELECT_DB_USER"],
        password=config["SELECT_DB_PASSWORD"],
        sslmode=config["SELECT_DB_SSLMODE"],
    )
    select_container_dsn = _postgres_uri(
        host=select_container_host,
        port=config["SELECT_DB_PORT"],
        name=config["SELECT_DB_NAME"],
        user=config["SELECT_DB_USER"],
        password=config["SELECT_DB_PASSWORD"],
        sslmode=config["SELECT_DB_SSLMODE"],
    )
    rob_container_dsn = _postgres_uri(
        host=rob_container_host,
        port=config["ROB_WORD_DB_PORT"],
        name=config["ROB_WORD_DB_NAME"],
        user=config["ROB_WORD_DB_USER"],
        password=config["ROB_WORD_DB_PASSWORD"],
        sslmode=config["ROB_WORD_DB_SSLMODE"],
    )
    redis_password = str(values.get("REDIS_PASSWORD", ""))
    minio_endpoint = f"{minio_container_host}:{config['MINIO_PORT']}"

    runtime = {
        "SELECT_DB_HOST_RUNTIME": select_host,
        "SELECT_DB_CONTAINER_HOST": select_container_host,
        "SELECT_DB_PORT": config["SELECT_DB_PORT"],
        "SELECT_DB_NAME": config["SELECT_DB_NAME"],
        "SELECT_DB_USER": config["SELECT_DB_USER"],
        "SELECT_DB_PASSWORD": config["SELECT_DB_PASSWORD"],
        "SELECT_DB_SSLMODE": config["SELECT_DB_SSLMODE"],
        "SELECT_DB_CONFIG": f"sslmode={config['SELECT_DB_SSLMODE']} TimeZone=Asia/Shanghai",
        "ROB_WORD_DB_HOST_RUNTIME": rob_host,
        "ROB_WORD_DB_CONTAINER_HOST": rob_container_host,
        "ROB_WORD_DB_PORT": config["ROB_WORD_DB_PORT"],
        "ROB_WORD_DB_NAME": config["ROB_WORD_DB_NAME"],
        "ROB_WORD_DB_USER": config["ROB_WORD_DB_USER"],
        "ROB_WORD_DB_PASSWORD": config["ROB_WORD_DB_PASSWORD"],
        "ROB_WORD_DB_SSLMODE": config["ROB_WORD_DB_SSLMODE"],
        "REDIS_HOST_RUNTIME": redis_host,
        "REDIS_CONTAINER_HOST": redis_container_host,
        "REDIS_PORT": config["REDIS_PORT"],
        "REDIS_PASSWORD": redis_password,
        "REDIS_CONTAINER_ADDR": f"{redis_container_host}:{config['REDIS_PORT']}",
        "MINIO_HOST_RUNTIME": minio_host,
        "MINIO_CONTAINER_HOST": minio_container_host,
        "MINIO_PORT": config["MINIO_PORT"],
        "MINIO_CONTAINER_ENDPOINT": minio_endpoint,
        "MINIO_ACCESS_KEY": config["MINIO_ACCESS_KEY"],
        "MINIO_SECRET_KEY": config["MINIO_SECRET_KEY"],
        "MINIO_BUCKET": config["MINIO_BUCKET"],
        "MINIO_USE_SSL": config["MINIO_USE_SSL"],
        "WORD_AGENT_SELECT_DB_DSN": select_container_dsn,
        "WORD_AGENT_CLI_RUNNER_DB_DSN": select_host_dsn,
        "WORD_AGENT_ROB_WORD_DB_DSN": rob_container_dsn,
        "SPRING_DATASOURCE_URL": (
            f"jdbc:postgresql://{rob_container_host}:{config['ROB_WORD_DB_PORT']}/"
            f"{config['ROB_WORD_DB_NAME']}?sslmode={config['ROB_WORD_DB_SSLMODE']}"
        ),
        "SPRING_DATASOURCE_USERNAME": config["ROB_WORD_DB_USER"],
        "SPRING_DATASOURCE_PASSWORD": config["ROB_WORD_DB_PASSWORD"],
        "SPRING_DATA_REDIS_HOST": redis_container_host,
        "SPRING_DATA_REDIS_PORT": config["REDIS_PORT"],
        "SPRING_DATA_REDIS_PASSWORD": redis_password,
        "WORD_AGENT_MINIO_ENDPOINT": minio_endpoint,
        "WORD_AGENT_MINIO_ACCESS_KEY_ID": config["MINIO_ACCESS_KEY"],
        "WORD_AGENT_MINIO_SECRET_ACCESS_KEY": config["MINIO_SECRET_KEY"],
        "WORD_AGENT_MINIO_BUCKET_NAME": config["MINIO_BUCKET"],
        "WORD_AGENT_MINIO_USE_SSL": config["MINIO_USE_SSL"],
    }
    runtime.update(ports)
    runtime.update(
        {
            "WORD_AGENT_HOST_URL": f"http://127.0.0.1:{ports['WORD_AGENT_PORT']}",
            "WORD_AGENT_CONTAINER_URL": (
                f"http://host.docker.internal:{ports['WORD_AGENT_PORT']}"
            ),
            "CLI_RUNNER_HOST_URL": f"http://127.0.0.1:{ports['CLI_RUNNER_PORT']}",
            "WORD_AGENT_CONTAINER_CLI_RUNNER_URL": (
                f"http://host.docker.internal:{ports['CLI_RUNNER_PORT']}"
            ),
            "DASHBOARD_HOST_URL": (
                f"http://127.0.0.1:{ports['DASHBOARD_SERVER_PORT']}"
            ),
            "DASHBOARD_CONTAINER_URL": (
                f"http://host.docker.internal:{ports['DASHBOARD_SERVER_PORT']}"
            ),
            "ROB_WORD_CONTAINER_URL": (
                f"http://host.docker.internal:{ports['ROB_WORD_HTTP_PORT']}"
            ),
            "ROB_WORD_WEBSOCKET_CONTAINER_URL": (
                f"http://host.docker.internal:{ports['ROB_WORD_WEBSOCKET_PORT']}"
            ),
        }
    )
    return runtime


def parse_shell_value(value: str) -> str:
    parts = shlex.split(value, posix=True)
    if len(parts) != 1:
        raise ConfigError("运行配置值无法安全解析")
    return parts[0]


def write_runtime_environment(path: Path, values: Mapping[str, str]) -> None:
    parent_existed = path.parent.exists()
    path.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
    if not parent_existed:
        path.parent.chmod(0o700)
    temporary_path: Path | None = None
    try:
        with tempfile.NamedTemporaryFile(
            mode="w",
            encoding="utf-8",
            dir=path.parent,
            prefix=f".{path.name}.",
            delete=False,
        ) as temporary:
            temporary_path = Path(temporary.name)
            os.fchmod(temporary.fileno(), stat.S_IRUSR | stat.S_IWUSR)
            for key in sorted(values):
                temporary.write(f"{key}={shlex.quote(str(values[key]))}\n")
        os.replace(temporary_path, path)
        path.chmod(0o600)
    finally:
        if temporary_path is not None and temporary_path.exists():
            temporary_path.unlink()


def _parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Validate one workspace config and write derived runtime values."
    )
    parser.add_argument("--output", required=True, type=Path)
    return parser.parse_args()


def main() -> int:
    args = _parse_args()
    try:
        runtime = derive_runtime(os.environ)
        write_runtime_environment(args.output, runtime)
    except ConfigError as exc:
        print(str(exc), file=os.sys.stderr)
        return 1
    print("统一运行配置校验通过。")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
