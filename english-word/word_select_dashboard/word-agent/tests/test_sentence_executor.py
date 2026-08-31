from types import SimpleNamespace

import pytest

from word_agent.core.config import Settings
from word_agent.services import sentence_executor
from word_agent.services.llm_client import LLMConfigError
from word_agent.services.sentence_executor import SentenceExecutorLoader


class FakeCursor:
    def __init__(self, rows: list[dict[str, object] | None]) -> None:
        self.rows = iter(rows)
        self.queries: list[tuple[str, tuple[object, ...]]] = []

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc, traceback) -> None:
        return None

    def execute(self, query: str, params: tuple[object, ...]) -> None:
        self.queries.append((query, params))

    def fetchone(self):
        return next(self.rows)


class FakeConnection:
    def __init__(self, cursor: FakeCursor) -> None:
        self.fake_cursor = cursor

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc, traceback) -> None:
        return None

    def cursor(self) -> FakeCursor:
        return self.fake_cursor


def install_fake_database(
    monkeypatch: pytest.MonkeyPatch,
    rows: list[dict[str, object] | None],
) -> FakeCursor:
    cursor = FakeCursor(rows)
    fake_dict_row = object()

    def fake_connect(dsn: str, **kwargs):
        assert dsn == "postgresql://word-agent"
        assert kwargs["row_factory"] is fake_dict_row
        return FakeConnection(cursor)

    monkeypatch.setattr(
        sentence_executor,
        "psycopg",
        SimpleNamespace(connect=fake_connect, Error=Exception),
    )
    monkeypatch.setattr(sentence_executor, "dict_row", fake_dict_row)
    return cursor


def test_loader_returns_exact_singleton_target(monkeypatch: pytest.MonkeyPatch) -> None:
    cursor = install_fake_database(
        monkeypatch,
        [
            {"executor_type": "cli", "executor_id": "codex"},
            {
                "provider_id": "codex",
                "label": "Codex CLI",
                "driver": "codex",
                "command_path": "/Applications/ChatGPT.app/Contents/Resources/codex",
                "model": "gpt-5.6-sol",
                "working_directory": "/workspace",
                "timeout_seconds": 300,
                "enabled": True,
            },
        ],
    )

    target = SentenceExecutorLoader(
        Settings(select_db_dsn="postgresql://word-agent")
    ).load()

    assert (target.type, target.id) == ("cli", "codex")
    assert cursor.queries[0][1] == ("default",)
    assert cursor.queries[1][1] == ("codex",)


def test_loader_does_not_fallback_when_target_is_missing(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    install_fake_database(
        monkeypatch,
        [
            {"executor_type": "cli", "executor_id": "missing"},
            None,
        ],
    )

    with pytest.raises(LLMConfigError, match="不存在"):
        SentenceExecutorLoader(
            Settings(select_db_dsn="postgresql://word-agent")
        ).load()


def test_loader_rejects_disabled_cli_target(monkeypatch: pytest.MonkeyPatch) -> None:
    install_fake_database(
        monkeypatch,
        [
            {"executor_type": "cli", "executor_id": "codex"},
            {
                "provider_id": "codex",
                "label": "Codex CLI",
                "driver": "codex",
                "command_path": "/usr/local/bin/codex",
                "model": "gpt-5.6-sol",
                "working_directory": "/workspace",
                "timeout_seconds": 300,
                "enabled": False,
            },
        ],
    )

    with pytest.raises(LLMConfigError, match="已停用"):
        SentenceExecutorLoader(
            Settings(select_db_dsn="postgresql://word-agent")
        ).load()


def test_loader_rejects_incomplete_cli_target(monkeypatch: pytest.MonkeyPatch) -> None:
    install_fake_database(
        monkeypatch,
        [
            {"executor_type": "cli", "executor_id": "codex"},
            {
                "provider_id": "codex",
                "label": "Codex CLI",
                "driver": "codex",
                "command_path": "",
                "model": "gpt-5.6-sol",
                "working_directory": "/workspace",
                "timeout_seconds": 300,
                "enabled": True,
            },
        ],
    )

    with pytest.raises(LLMConfigError, match="不完整"):
        SentenceExecutorLoader(
            Settings(select_db_dsn="postgresql://word-agent")
        ).load()


def test_loader_rejects_incomplete_api_target(monkeypatch: pytest.MonkeyPatch) -> None:
    install_fake_database(
        monkeypatch,
        [
            {"executor_type": "api", "executor_id": "aliyun"},
            {
                "provider_id": "aliyun",
                "label": "Aliyun",
                "type": "openai-compatible",
                "base_url": "https://example.com/v1",
                "api_key": "",
                "model": "qwen3.6-flash",
                "max_tokens": 4096,
            },
        ],
    )

    with pytest.raises(LLMConfigError, match="不完整"):
        SentenceExecutorLoader(
            Settings(select_db_dsn="postgresql://word-agent")
        ).load()
