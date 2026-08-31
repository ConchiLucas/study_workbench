from collections.abc import Callable
from typing import Any

import psycopg
import pytest

from word_agent.core.config import Settings
from word_agent.services.tts_config import TTSConfigError, TTSConfigLoader


def valid_row(**overrides: Any) -> dict[str, Any]:
    row = {
        "provider_id": "xiaomi-mimo-tts",
        "type": "mimo-tts",
        "base_url": "https://api.xiaomimimo.com/v1/",
        "api_key": "stored-secret",
        "model": "mimo-v2.5-tts",
        "voice": "Chloe",
    }
    row.update(overrides)
    return row


class FakeCursor:
    def __init__(self, rows: list[dict[str, Any]], executed: list[str]) -> None:
        self._rows = rows
        self._executed = executed

    def __enter__(self) -> "FakeCursor":
        return self

    def __exit__(self, exc_type, exc, traceback) -> None:
        return None

    def execute(self, query: str) -> None:
        self._executed.append(query)

    def fetchall(self) -> list[dict[str, Any]]:
        return self._rows


class FakeConnection:
    def __init__(self, rows: list[dict[str, Any]], executed: list[str]) -> None:
        self._rows = rows
        self._executed = executed

    def __enter__(self) -> "FakeConnection":
        return self

    def __exit__(self, exc_type, exc, traceback) -> None:
        return None

    def cursor(self) -> FakeCursor:
        return FakeCursor(self._rows, self._executed)


def loader_for_rows(
    rows: list[dict[str, Any]],
    executed: list[str] | None = None,
) -> TTSConfigLoader:
    captured_queries = executed if executed is not None else []

    def connect(dsn: str, **kwargs: Any) -> FakeConnection:
        assert dsn == "dbname=test"
        assert "row_factory" in kwargs
        assert 0 < kwargs["connect_timeout"] <= 10
        return FakeConnection(rows, captured_queries)

    return TTSConfigLoader(Settings(select_db_dsn="dbname=test"), connect=connect)


def test_load_active_mimo_config_returns_the_unique_enabled_active_row() -> None:
    executed: list[str] = []

    config = loader_for_rows([valid_row()], executed).load_active_mimo_config()

    assert config.provider_id == "xiaomi-mimo-tts"
    assert config.base_url == "https://api.xiaomimimo.com/v1"
    assert config.api_key == "stored-secret"
    assert config.model == "mimo-v2.5-tts"
    assert config.voice == "Chloe"
    assert len(executed) == 1
    normalized_query = " ".join(executed[0].split())
    assert "FROM tts_provider_configs" in normalized_query
    assert "type = 'mimo-tts'" in normalized_query
    assert "enabled = TRUE" in normalized_query
    assert "active = TRUE" in normalized_query
    assert "LIMIT 2" in normalized_query


@pytest.mark.parametrize(
    ("rows", "message"),
    [
        ([], "没有启用的默认"),
        ([valid_row(), valid_row(provider_id="second")], "存在多个"),
        ([valid_row(model="  ")], "字段不完整"),
        ([valid_row(api_key="  ")], "字段不完整"),
        ([valid_row(base_url="https://attacker.example/v1")], "官方"),
    ],
)
def test_load_active_mimo_config_rejects_invalid_database_state(
    rows: list[dict[str, Any]],
    message: str,
) -> None:
    with pytest.raises(TTSConfigError, match=message) as raised:
        loader_for_rows(rows).load_active_mimo_config()

    assert "stored-secret" not in str(raised.value)


def test_load_active_mimo_config_wraps_database_errors_without_secret() -> None:
    def broken_connect(dsn: str, **kwargs: Any) -> Callable[..., Any]:
        _ = (dsn, kwargs)
        raise psycopg.OperationalError("database unavailable; stored-secret")

    loader = TTSConfigLoader(
        Settings(select_db_dsn="dbname=test"),
        connect=broken_connect,
    )

    with pytest.raises(TTSConfigError, match="读取数据库 TTS 配置失败") as raised:
        loader.load_active_mimo_config()

    assert "stored-secret" not in str(raised.value)
