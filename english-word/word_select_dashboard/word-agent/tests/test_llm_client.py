import asyncio
import json
import threading
import time
from pathlib import Path
from types import SimpleNamespace

import pytest

from word_agent.core.config import Settings
from word_agent.services import llm_client
from word_agent.services.llm_client import (
    AIProvider,
    LLMClient,
    LLMRequestError,
    build_sentence_prompt,
)
from word_agent.services.sentence_executor import (
    SentenceExecutorLoader,
    SentenceExecutorTarget,
)


def test_unused_batch_sentence_generation_feature_is_removed() -> None:
    assert not hasattr(LLMClient, "generate_word_clean_sentences")
    assert not hasattr(LLMClient, "_word_clean_chat_completion_sync")
    assert not hasattr(LLMClient, "_parse_word_clean_sentence_payload")
    assert not hasattr(llm_client, "WordCleanPromptItem")
    assert not hasattr(llm_client, "WordCleanSentenceResult")


def test_unimplemented_sentence_guidance_feature_is_removed() -> None:
    assert not hasattr(LLMClient, "generate_sentence_guidance")


def test_scoring_provider_loads_database_model_when_sentence_executor_is_cli(
    monkeypatch,
    tmp_path: Path,
) -> None:
    config_path = tmp_path / "config.yaml"
    config_path.write_text(
        """
ai:
  active: default
  providers:
    default:
      type: openai-compatible
      base-url: ""
      api-key: ""
      model: ""
pgsql:
  path: 127.0.0.1
  port: "5432"
  db-name: select_english_word
  username: conchi
  password: secret
""".strip(),
        encoding="utf-8",
    )

    rows = [
        {
            "provider_id": "configured-provider",
            "label": "Configured Provider",
            "type": "openai-compatible",
            "base_url": "https://example.com/v1",
            "api_key": "database-secret",
            "model": "configured-model",
            "max_tokens": 4096,
            "active": True,
        }
    ]

    class FakeCursor:
        def __enter__(self):
            return self

        def __exit__(self, exc_type, exc, traceback) -> None:
            return None

        def execute(self, query: str) -> None:
            assert "FROM ai_provider_configs" in query

        def fetchall(self):
            return rows

    class FakeConnection:
        def __enter__(self):
            return self

        def __exit__(self, exc_type, exc, traceback) -> None:
            return None

        def cursor(self) -> FakeCursor:
            return FakeCursor()

    fake_dict_row = object()

    def fake_connect(dsn: str, **kwargs):
        assert "dbname=select_english_word" in dsn
        assert kwargs["row_factory"] is fake_dict_row
        return FakeConnection()

    monkeypatch.setattr(
        llm_client,
        "psycopg",
        SimpleNamespace(connect=fake_connect, Error=Exception),
        raising=False,
    )
    monkeypatch.setattr(llm_client, "dict_row", fake_dict_row, raising=False)

    selector_calls = 0

    def selected_cli(_self):
        nonlocal selector_calls
        selector_calls += 1
        return cli_target()

    monkeypatch.setattr(SentenceExecutorLoader, "load", selected_cli)

    provider = LLMClient(Settings(go_config_path=config_path)).load_provider_by_model(
        "configured-model"
    )

    assert provider.id == "configured-provider"
    assert provider.model == "configured-model"
    assert provider.api_key == "database-secret"
    assert selector_calls == 0


def cli_target() -> SentenceExecutorTarget:
    return SentenceExecutorTarget(
        type="cli",
        id="codex",
        cli_provider=AIProvider(
            id="codex",
            label="Codex CLI",
            type="cli",
            base_url="",
            api_key="",
            model="gpt-5.6-sol",
            max_tokens=0,
        ),
    )


def api_target() -> SentenceExecutorTarget:
    return SentenceExecutorTarget(
        type="api",
        id="aliyun",
        api_provider=AIProvider(
            id="aliyun",
            label="Aliyun",
            type="openai-compatible",
            base_url="https://example.com/v1",
            api_key="api-secret",
            model="qwen3.6-flash",
            max_tokens=4096,
        ),
    )


def valid_sentence_json() -> str:
    return json.dumps(
        {
            "sentence": "We salute her courage.",
            "translation_zh": "我们向她的勇气致敬。",
            "explanation_zh": "salute 表示致敬。",
        },
        ensure_ascii=False,
    )


class FakeHTTPResponse:
    def __init__(self, *, status_code: int, payload: object) -> None:
        self.status_code = status_code
        self._payload = payload

    def raise_for_status(self) -> None:
        if self.status_code >= 400:
            request = llm_client.httpx.Request("POST", "http://runner/v1/text/generate")
            response = llm_client.httpx.Response(self.status_code, request=request)
            raise llm_client.httpx.HTTPStatusError(
                "runner failed",
                request=request,
                response=response,
            )

    def json(self) -> object:
        return self._payload


class FakeHTTPClient:
    def __init__(
        self,
        response: FakeHTTPResponse,
        calls: list[dict[str, object]],
        **kwargs,
    ) -> None:
        self.response = response
        self.calls = calls
        self.kwargs = kwargs

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc, traceback) -> None:
        return None

    def post(self, url: str, **kwargs) -> FakeHTTPResponse:
        self.calls.append({"url": url, **kwargs})
        return self.response


@pytest.mark.anyio
async def test_api_and_cli_generation_share_the_sentence_prompt_and_parser(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    calls: list[tuple[AIProvider, str]] = []

    async def fake_chat_completion(
        _self: LLMClient,
        provider: AIProvider,
        prompt: str,
    ) -> str:
        calls.append((provider, prompt))
        return valid_sentence_json()

    monkeypatch.setattr(SentenceExecutorLoader, "load", lambda _self: api_target())
    monkeypatch.setattr(LLMClient, "_chat_completion", fake_chat_completion)

    result = await LLMClient(Settings()).generate_sentence_from_words(words=["salute"])

    assert result.sentence == "We salute her courage."
    assert calls == [(api_target().api_provider, build_sentence_prompt(["salute"]))]


@pytest.mark.anyio
async def test_sentence_executor_loading_does_not_block_the_event_loop(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    release_loader = threading.Event()
    loop_progressed = asyncio.Event()

    def blocking_load(_self: SentenceExecutorLoader) -> SentenceExecutorTarget:
        release_loader.wait(timeout=0.5)
        return api_target()

    async def fake_chat_completion(
        _self: LLMClient,
        provider: AIProvider,
        prompt: str,
    ) -> str:
        return valid_sentence_json()

    monkeypatch.setattr(SentenceExecutorLoader, "load", blocking_load)
    monkeypatch.setattr(LLMClient, "_chat_completion", fake_chat_completion)

    timer = threading.Timer(0.5, release_loader.set)
    timer.start()
    started_at = time.monotonic()
    try:
        task = asyncio.create_task(
            LLMClient(Settings()).generate_sentence_from_words(words=["salute"])
        )
        asyncio.get_running_loop().call_soon(loop_progressed.set)
        await asyncio.wait_for(loop_progressed.wait(), timeout=1)
        scheduling_delay = time.monotonic() - started_at
        result = await task
    finally:
        release_loader.set()
        timer.cancel()

    assert scheduling_delay < 0.25
    assert result.sentence == "We salute her courage."


@pytest.mark.anyio
@pytest.mark.parametrize(
    ("runner_url", "expected_url"),
    [
        (
            "http://127.0.0.1:6018/",
            "http://127.0.0.1:6018/v1/text/generate",
        ),
        (
            "http://host.docker.internal:6018",
            "http://host.docker.internal:6018/v1/text/generate",
        ),
    ],
)
async def test_generate_sentence_uses_cli_runner_for_cli_target(
    monkeypatch: pytest.MonkeyPatch,
    runner_url: str,
    expected_url: str,
) -> None:
    calls: list[dict[str, object]] = []
    response = FakeHTTPResponse(
        status_code=200,
        payload={
            "content": valid_sentence_json(),
            "executor_id": "codex",
            "driver": "codex",
            "model": "gpt-5.6-sol",
            "duration_ms": 5,
        },
    )

    monkeypatch.setattr(SentenceExecutorLoader, "load", lambda _self: cli_target())
    monkeypatch.setattr(
        llm_client.httpx,
        "Client",
        lambda **kwargs: FakeHTTPClient(response, calls, **kwargs),
    )
    client = LLMClient(
        Settings(
            cli_runner_url=runner_url,
            select_db_dsn=None,
        )
    )

    result = await client.generate_sentence_from_words(words=["salute"])

    assert result.provider.id == "codex"
    assert result.provider.model == "gpt-5.6-sol"
    assert calls == [
        {
            "url": expected_url,
            "json": {
                "executor_id": "codex",
                "prompt": build_sentence_prompt(["salute"]),
            },
        }
    ]


@pytest.mark.anyio
async def test_cli_failure_does_not_call_api(monkeypatch: pytest.MonkeyPatch) -> None:
    calls: list[dict[str, object]] = []
    response = FakeHTTPResponse(status_code=502, payload={"detail": "CLI failed"})
    api_calls: list[object] = []

    monkeypatch.setattr(SentenceExecutorLoader, "load", lambda _self: cli_target())
    monkeypatch.setattr(
        llm_client.httpx,
        "Client",
        lambda **kwargs: FakeHTTPClient(response, calls, **kwargs),
    )
    monkeypatch.setattr(
        LLMClient,
        "_chat_completion",
        lambda *args, **kwargs: api_calls.append((args, kwargs)),
    )
    client = LLMClient(Settings())

    with pytest.raises(LLMRequestError, match="CLI"):
        await client.generate_sentence_from_words(words=["salute"])

    assert len(calls) == 1
    assert api_calls == []


@pytest.mark.anyio
@pytest.mark.parametrize(
    ("payload", "error"),
    [
        ([], "顶层必须是 JSON 对象"),
        (123, "顶层必须是 JSON 对象"),
        (True, "顶层必须是 JSON 对象"),
        ({"executor_id": 123, "content": valid_sentence_json()}, "executor_id"),
        (
            {"executor_id": "gemini", "content": valid_sentence_json()},
            "执行器不一致",
        ),
        ({"executor_id": "codex", "content": "  "}, "没有返回句子内容"),
        (
            {"content": {"sentence": "wrong container"}, "executor_id": "codex"},
            "文本类型",
        ),
    ],
)
async def test_cli_runner_rejects_malformed_response_contract(
    monkeypatch: pytest.MonkeyPatch,
    payload: object,
    error: str,
) -> None:
    calls: list[dict[str, object]] = []
    response = FakeHTTPResponse(status_code=200, payload=payload)
    monkeypatch.setattr(SentenceExecutorLoader, "load", lambda _self: cli_target())
    monkeypatch.setattr(
        llm_client.httpx,
        "Client",
        lambda **kwargs: FakeHTTPClient(response, calls, **kwargs),
    )

    with pytest.raises(LLMRequestError, match=error):
        await LLMClient(Settings()).generate_sentence_from_words(words=["salute"])


@pytest.mark.parametrize(
    "payload",
    [
        [],
        123,
        True,
        {"choices": {}},
        {"choices": [123]},
        {"choices": [{"message": []}]},
        {"choices": [{"message": {"content": {"sentence": "wrong"}}}]},
    ],
)
def test_api_completion_rejects_malformed_response_contract(
    monkeypatch: pytest.MonkeyPatch,
    payload: object,
) -> None:
    calls: list[dict[str, object]] = []
    response = FakeHTTPResponse(status_code=200, payload=payload)
    monkeypatch.setattr(
        llm_client.httpx,
        "Client",
        lambda **kwargs: FakeHTTPClient(response, calls, **kwargs),
    )

    with pytest.raises(LLMRequestError, match="返回格式"):
        LLMClient(Settings())._chat_completion_sync(
            api_target().api_provider,
            build_sentence_prompt(["salute"]),
        )


@pytest.mark.parametrize("payload", ["[]", "123", "true"])
def test_sentence_parser_rejects_non_object_top_level(payload: str) -> None:
    with pytest.raises(LLMRequestError, match="JSON 对象"):
        LLMClient(Settings())._parse_sentence_payload(payload)


@pytest.mark.parametrize("field", ["sentence", "translation_zh", "explanation_zh"])
@pytest.mark.parametrize("invalid_value", [123, True, [], {}])
def test_sentence_parser_rejects_non_string_fields(
    field: str,
    invalid_value: object,
) -> None:
    payload: dict[str, object] = {
        "sentence": "We salute her courage.",
        "translation_zh": "我们向她的勇气致敬。",
        "explanation_zh": "salute 表示致敬。",
    }
    payload[field] = invalid_value

    with pytest.raises(LLMRequestError, match=field):
        LLMClient(Settings())._parse_sentence_payload(json.dumps(payload, ensure_ascii=False))
