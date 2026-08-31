from contextlib import nullcontext

import pytest

from word_agent.core.config import Settings
from word_agent.domain.schemas import WordCleanSentenceScoreRequest
from word_agent.services.llm_client import (
    AIProvider,
    LLMClient,
    WordCleanSentenceScoreResult,
)
from word_agent.services.sentence_executor import (
    SentenceExecutorLoader,
    SentenceExecutorTarget,
)
from word_agent.services.word_clean_sentence_score import (
    WordCleanSentenceForScore,
    WordCleanSentenceScoreError,
    WordCleanSentenceScoreService,
)


@pytest.mark.anyio
async def test_score_uses_independent_api_model_when_sentence_executor_is_cli(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    selected_models: list[str] = []
    scored_providers: list[AIProvider] = []
    sentence_selector_calls = 0
    runner_calls = 0
    api_provider = AIProvider(
        id="qwen",
        label="Qwen",
        type="openai-compatible",
        base_url="https://example.com/v1",
        api_key="secret",
        model="qwen3.6-flash",
        max_tokens=4096,
    )

    def selected_cli(_self: SentenceExecutorLoader) -> SentenceExecutorTarget:
        nonlocal sentence_selector_calls
        sentence_selector_calls += 1
        return SentenceExecutorTarget(type="cli", id="codex")

    def load_provider(_self: LLMClient, model_name: str) -> AIProvider:
        selected_models.append(model_name)
        return api_provider

    async def score_sentences(
        _self: LLMClient,
        *,
        provider: AIProvider,
        items,
    ) -> list[WordCleanSentenceScoreResult]:
        scored_providers.append(provider)
        assert len(items) == 1
        return [WordCleanSentenceScoreResult(id=1, score=95, score_reason="准确自然")]

    def call_runner(*args, **kwargs):
        nonlocal runner_calls
        runner_calls += 1
        raise AssertionError("sentence scoring must not call the CLI Runner")

    monkeypatch.setattr(SentenceExecutorLoader, "load", selected_cli)
    monkeypatch.setattr(LLMClient, "load_provider_by_model", load_provider)
    monkeypatch.setattr(LLMClient, "_cli_completion_sync", call_runner)
    monkeypatch.setattr(LLMClient, "score_word_clean_sentences", score_sentences)

    service = WordCleanSentenceScoreService(Settings())
    fake_connection = object()
    monkeypatch.setattr(service, "_connect", lambda: nullcontext(fake_connection))
    monkeypatch.setattr(service, "_ensure_score_columns", lambda conn: None)
    monkeypatch.setattr(service, "_ensure_best_sentence_table", lambda conn: None)
    monkeypatch.setattr(service, "_resolve_requested_word_clean_ids", lambda conn, req: [7])
    monkeypatch.setattr(
        service,
        "_fetch_items",
        lambda conn, req: [
            WordCleanSentenceForScore(
                id=1,
                word_clean_id=7,
                word="salute",
                meaning="致敬",
                model_name="sentence-model",
                sentence="We salute her courage.",
                sentence_translation="我们向她的勇气致敬。",
            )
        ],
    )
    monkeypatch.setattr(service, "_save_scores", lambda conn, **kwargs: [])
    monkeypatch.setattr(service, "_upsert_best_sentences", lambda conn, ids: [])

    await service.score(WordCleanSentenceScoreRequest(ids=[1]))

    assert selected_models == ["qwen3.6-flash"]
    assert scored_providers == [api_provider]
    assert sentence_selector_calls == 0
    assert runner_calls == 0


@pytest.mark.anyio
async def test_score_rejects_empty_scoring_model_without_loading_sentence_executor(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    sentence_selector_calls = 0
    provider_calls = 0

    def selected_cli(_self: SentenceExecutorLoader) -> SentenceExecutorTarget:
        nonlocal sentence_selector_calls
        sentence_selector_calls += 1
        return SentenceExecutorTarget(type="cli", id="codex")

    def load_provider(_self: LLMClient, model_name: str) -> AIProvider:
        nonlocal provider_calls
        provider_calls += 1
        raise AssertionError("an empty scoring model must fail before provider loading")

    monkeypatch.setattr(SentenceExecutorLoader, "load", selected_cli)
    monkeypatch.setattr(LLMClient, "load_provider_by_model", load_provider)
    service = WordCleanSentenceScoreService(Settings(word_clean_score_default_model=""))

    with pytest.raises(WordCleanSentenceScoreError, match="评分模型未配置"):
        await service.score(WordCleanSentenceScoreRequest(ids=[1]))

    assert provider_calls == 0
    assert sentence_selector_calls == 0
