import importlib.util
from pathlib import Path
from types import ModuleType
from typing import Any

import pytest

SCRIPT_PATH = Path(__file__).resolve().parents[1] / "scripts" / "migrate_mimo_tts_config.py"
PROJECT_ROOT = Path(__file__).resolve().parents[1]


def load_script() -> ModuleType:
    spec = importlib.util.spec_from_file_location("migrate_mimo_tts_config", SCRIPT_PATH)
    if spec is None or spec.loader is None:
        raise RuntimeError("failed to load migration script")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def legacy_env() -> dict[str, str]:
    return {
        "WORD_AGENT_SELECT_DB_DSN": "dbname=test",
        "WORD_AGENT_MIMO_API_KEY": "migration-secret",
        "WORD_AGENT_MIMO_TTS_BASE_URL": "https://api.xiaomimimo.com/v1",
        "WORD_AGENT_MIMO_TTS_DEFAULT_MODEL": "mimo-v2.5-tts",
        "WORD_AGENT_MIMO_TTS_DEFAULT_VOICE": "Chloe",
    }


class FakeCursor:
    def __init__(self, connection: "FakeConnection") -> None:
        self.connection = connection

    def __enter__(self) -> "FakeCursor":
        return self

    def __exit__(self, exc_type, exc, traceback) -> None:
        return None

    def execute(self, query: str, params: tuple[Any, ...] | None = None) -> None:
        self.connection.executed.append((query, params))

    def fetchone(self) -> dict[str, Any]:
        return self.connection.verification_row


class FakeConnection:
    def __init__(self, verification_row: dict[str, Any]) -> None:
        self.verification_row = verification_row
        self.executed: list[tuple[str, tuple[Any, ...] | None]] = []
        self.committed = False
        self.rolled_back = False

    def __enter__(self) -> "FakeConnection":
        return self

    def __exit__(self, exc_type, exc, traceback) -> None:
        if exc_type is None:
            self.committed = True
        else:
            self.rolled_back = True

    def cursor(self) -> FakeCursor:
        return FakeCursor(self)


def valid_verification_row() -> dict[str, Any]:
    return {
        "provider_id": "xiaomi-mimo-tts",
        "type": "mimo-tts",
        "base_url": "https://api.xiaomimimo.com/v1",
        "model": "mimo-v2.5-tts",
        "voice": "Chloe",
        "enabled": True,
        "active": True,
        "api_key_configured": True,
    }


def test_migration_upserts_one_active_provider_without_exposing_secret() -> None:
    module = load_script()
    connection = FakeConnection(valid_verification_row())
    output: list[str] = []

    def connect(dsn: str, **kwargs: Any) -> FakeConnection:
        assert dsn == "dbname=test"
        assert "row_factory" in kwargs
        return connection

    result = module.migrate_mimo_tts_config(
        legacy_env(),
        connect=connect,
        output=output.append,
    )

    assert result.provider_id == "xiaomi-mimo-tts"
    assert result.api_key_configured is True
    assert connection.committed is True
    assert connection.rolled_back is False
    sql = "\n".join(query for query, _ in connection.executed)
    assert "INSERT INTO tts_provider_configs" in sql
    assert "ON CONFLICT (provider_id) DO UPDATE" in sql
    assert "ai_provider_configs" not in sql
    assert "migration-secret" not in sql
    assert "migration-secret" not in "\n".join(output)
    assert any(
        params is not None and "migration-secret" in params
        for _, params in connection.executed
    )


def test_migration_requires_legacy_key_before_connecting() -> None:
    module = load_script()
    env = legacy_env()
    env["WORD_AGENT_MIMO_API_KEY"] = ""
    connected = False

    def connect(dsn: str, **kwargs: Any) -> FakeConnection:
        nonlocal connected
        _ = (dsn, kwargs)
        connected = True
        return FakeConnection(valid_verification_row())

    with pytest.raises(module.MigrationError, match="MiMo API Key"):
        module.migrate_mimo_tts_config(env, connect=connect, output=lambda _: None)

    assert connected is False


def test_migration_rolls_back_when_verification_fails() -> None:
    module = load_script()
    invalid_row = valid_verification_row()
    invalid_row["api_key_configured"] = False
    connection = FakeConnection(invalid_row)

    with pytest.raises(module.MigrationError, match="校验失败"):
        module.migrate_mimo_tts_config(
            legacy_env(),
            connect=lambda *args, **kwargs: connection,
            output=lambda _: None,
        )

    assert connection.committed is False
    assert connection.rolled_back is True


def test_normal_runtime_config_does_not_declare_legacy_mimo_provider_fields() -> None:
    runtime_config = "\n".join(
        [
            (PROJECT_ROOT / ".env.example").read_text(encoding="utf-8"),
            (PROJECT_ROOT / "docker-compose.yml").read_text(encoding="utf-8"),
        ]
    )

    for variable in (
        "WORD_AGENT_MIMO_API_KEY",
        "WORD_AGENT_MIMO_TTS_BASE_URL",
        "WORD_AGENT_MIMO_TTS_DEFAULT_MODEL",
        "WORD_AGENT_MIMO_TTS_DEFAULT_VOICE",
    ):
        assert variable not in runtime_config
    assert "WORD_AGENT_TTS_TIMEOUT_SECONDS" in runtime_config
    assert "WORD_AGENT_TTS_OUTPUT_DIR" in runtime_config
