import asyncio
import json
import re
from collections.abc import Mapping
from dataclasses import dataclass
from typing import Any

import httpx
import psycopg
import yaml
from psycopg.rows import dict_row

from word_agent.core.config import Settings


class LLMConfigError(RuntimeError):
    pass


class LLMRequestError(RuntimeError):
    pass


def build_sentence_prompt(words: list[str]) -> str:
    return (
        "Return one JSON object with exactly sentence, translation_zh, explanation_zh. "
        "Use every requested word in one natural English sentence. "
        "translation_zh must be a direct Chinese translation of the sentence. "
        "explanation_zh must be a concise Chinese explanation of the sentence meaning "
        "and word usage. Do not add Markdown, code fences, numbering, or extra keys. "
        "Words: "
        + ", ".join(words)
    )


@dataclass(frozen=True)
class AIProvider:
    id: str
    label: str
    type: str
    base_url: str
    api_key: str
    model: str
    max_tokens: int


@dataclass(frozen=True)
class SentenceGenerationResult:
    sentence: str
    translation_zh: str
    explanation_zh: str
    provider: AIProvider


@dataclass(frozen=True)
class WordCleanSentenceScorePromptItem:
    id: int
    word_clean_id: int
    word: str
    meaning: str
    model_name: str
    sentence: str
    sentence_translation: str


@dataclass(frozen=True)
class WordCleanSentenceScoreResult:
    id: int
    score: int
    score_reason: str


@dataclass(frozen=True)
class LLMClient:
    settings: Settings

    async def generate_sentence_from_words(self, *, words: list[str]) -> SentenceGenerationResult:
        from word_agent.services.sentence_executor import SentenceExecutorLoader

        loader = SentenceExecutorLoader(self.settings)
        target = await asyncio.to_thread(loader.load)
        prompt = build_sentence_prompt(words)
        if target.type == "api":
            provider = target.api_provider
            if provider is None:
                raise LLMConfigError(f"造句 API 配置不完整: {target.id}")
            if provider.type != "openai-compatible":
                raise LLMConfigError(f"暂不支持的模型接口类型: {provider.type}")
            content = await self._chat_completion(provider, prompt)
        else:
            provider = target.cli_provider
            if provider is None:
                raise LLMConfigError(f"造句 CLI 配置不完整: {target.id}")
            content = await self._cli_completion(target.id, prompt)
        sentence, translation_zh, explanation_zh = self._parse_sentence_payload(content)
        return SentenceGenerationResult(
            sentence=sentence,
            translation_zh=translation_zh,
            explanation_zh=explanation_zh,
            provider=provider,
        )

    def load_provider_by_model(self, model_name: str) -> AIProvider:
        cleaned_model_name = model_name.strip()
        if not cleaned_model_name:
            raise LLMConfigError("模型名称不能为空")

        for provider in self._load_providers().values():
            if provider.model == cleaned_model_name or provider.id == cleaned_model_name:
                return provider

        database_providers = self._load_providers(self._load_database_ai_config())
        for provider in database_providers.values():
            if provider.model == cleaned_model_name or provider.id == cleaned_model_name:
                return provider
        raise LLMConfigError(f"API 配置中找不到模型: {cleaned_model_name}")

    def load_active_provider(self) -> AIProvider:
        from word_agent.services.sentence_executor import SentenceExecutorLoader

        target = SentenceExecutorLoader(self.settings).load()
        if target.type != "api" or target.api_provider is None:
            raise LLMConfigError("当前造句执行器不是 API 模型")
        return target.api_provider

    async def score_word_clean_sentences(
        self,
        *,
        provider: AIProvider,
        items: list[WordCleanSentenceScorePromptItem],
    ) -> list[WordCleanSentenceScoreResult]:
        if provider.type != "openai-compatible":
            raise LLMConfigError(f"暂不支持的模型接口类型: {provider.type}")
        return await asyncio.to_thread(
            self._score_word_clean_sentences_sync,
            provider,
            items,
        )

    def _load_database_ai_config(self) -> dict[str, Any]:
        try:
            with psycopg.connect(self._resolve_select_db_dsn(), row_factory=dict_row) as conn:
                with conn.cursor() as cursor:
                    cursor.execute(
                        """
                        SELECT provider_id, label, type, base_url, api_key,
                               model, max_tokens
                        FROM ai_provider_configs
                        WHERE btrim(base_url) <> ''
                          AND btrim(api_key) <> ''
                          AND btrim(model) <> ''
                        ORDER BY provider_id ASC
                        """
                    )
                    rows = cursor.fetchall()
        except psycopg.Error as exc:
            raise LLMConfigError(f"读取数据库 AI provider 配置失败: {exc}") from exc

        providers: dict[str, dict[str, Any]] = {}
        for row in rows:
            provider_id = str(row["provider_id"] or "").strip()
            if not provider_id:
                continue
            providers[provider_id] = {
                "label": str(row["label"] or provider_id).strip(),
                "type": str(row["type"] or "openai-compatible").strip(),
                "base-url": str(row["base_url"] or "").strip(),
                "api-key": str(row["api_key"] or "").strip(),
                "model": str(row["model"] or "").strip(),
                "max-tokens": int(row["max_tokens"] or 128),
            }
        return {"providers": providers}

    def _resolve_select_db_dsn(self) -> str:
        if self.settings.select_db_dsn:
            return self.settings.select_db_dsn

        config_path = self.settings.go_config_path
        with config_path.open("r", encoding="utf-8") as file:
            data = yaml.safe_load(file) or {}
        pgsql = data.get("pgsql") or {}
        host = str(pgsql.get("path") or "127.0.0.1").strip()
        port = str(pgsql.get("port") or "5432").strip()
        dbname = str(pgsql.get("db-name") or "select_english_word").strip()
        user = str(pgsql.get("username") or "").strip()
        password = str(pgsql.get("password") or "").strip()
        if not user:
            raise LLMConfigError("Go 配置文件里缺少 pgsql.username")
        return " ".join(
            [
                f"host={host}",
                f"port={port}",
                f"dbname={dbname}",
                f"user={user}",
                f"password={password}",
            ]
        )

    def _load_providers(self, ai_config: dict[str, Any] | None = None) -> dict[str, AIProvider]:
        if ai_config is None:
            ai_config = self._load_ai_config()

        providers = ai_config.get("providers") or {}
        result: dict[str, AIProvider] = {}
        for provider_id, raw_provider in providers.items():
            raw_provider = raw_provider or {}
            provider = AIProvider(
                id=str(provider_id).strip(),
                label=str(raw_provider.get("label") or provider_id).strip(),
                type=str(raw_provider.get("type") or "openai-compatible").strip(),
                base_url=str(raw_provider.get("base-url") or "").strip().rstrip("/"),
                api_key=str(raw_provider.get("api-key") or "").strip(),
                model=str(raw_provider.get("model") or "").strip(),
                max_tokens=int(raw_provider.get("max-tokens") or 128),
            )
            if provider.base_url and provider.api_key and provider.model:
                result[provider.id] = provider
        return result

    def _load_ai_config(self) -> dict[str, Any]:
        config_path = self.settings.go_config_path
        if not config_path.exists():
            raise LLMConfigError(f"Go 配置文件不存在: {config_path}")

        with config_path.open("r", encoding="utf-8") as file:
            data = yaml.safe_load(file) or {}

        return data.get("ai") or {}

    async def _chat_completion(
        self,
        provider: AIProvider,
        prompt: str,
    ) -> str:
        return await asyncio.to_thread(self._chat_completion_sync, provider, prompt)

    def _chat_completion_sync(self, provider: AIProvider, prompt: str) -> str:
        url = f"{provider.base_url}/chat/completions"
        max_tokens = min(max(provider.max_tokens, 1), 512)
        payload: dict[str, Any] = {
            "model": provider.model,
            "messages": [
                {
                    "role": "user",
                    "content": prompt,
                },
            ],
            "temperature": 0.4,
            "max_tokens": max_tokens,
        }
        headers = {
            "Authorization": f"Bearer {provider.api_key}",
            "Content-Type": "application/json",
        }
        timeout = httpx.Timeout(self.settings.llm_timeout_seconds)

        try:
            with httpx.Client(
                timeout=timeout,
                verify=self.settings.llm_verify_ssl,
                trust_env=False,
            ) as client:
                response = client.post(url, json=payload, headers=headers)
                response.raise_for_status()
        except httpx.HTTPStatusError as exc:
            raise LLMRequestError(
                f"大模型请求失败: HTTP {exc.response.status_code}"
            ) from exc
        except httpx.HTTPError as exc:
            raise LLMRequestError(f"大模型请求失败: {exc}") from exc

        try:
            data = response.json()
        except ValueError as exc:
            raise LLMRequestError("大模型返回格式错误: 不是合法 JSON") from exc
        if not isinstance(data, Mapping):
            raise LLMRequestError("大模型返回格式错误: 顶层必须是 JSON 对象")
        choices = data.get("choices")
        if not isinstance(choices, list) or not choices:
            raise LLMRequestError("大模型返回格式错误: choices 必须是非空数组")
        choice = choices[0]
        if not isinstance(choice, Mapping):
            raise LLMRequestError("大模型返回格式错误: choice 必须是 JSON 对象")
        message = choice.get("message")
        if not isinstance(message, Mapping):
            raise LLMRequestError("大模型返回格式错误: message 必须是 JSON 对象")
        raw_content = message.get("content")
        if not isinstance(raw_content, str):
            raise LLMRequestError("大模型返回格式错误: content 必须是字符串")
        content = raw_content.strip()
        if not content:
            raise LLMRequestError("大模型没有返回句子内容")

        return content

    async def _cli_completion(self, executor_id: str, prompt: str) -> str:
        return await asyncio.to_thread(self._cli_completion_sync, executor_id, prompt)

    def _cli_completion_sync(self, executor_id: str, prompt: str) -> str:
        runner_url = self.settings.cli_runner_url.strip().rstrip("/")
        if not runner_url:
            raise LLMConfigError("CLI Runner URL 未配置")

        timeout = httpx.Timeout(self.settings.llm_timeout_seconds)
        try:
            with httpx.Client(
                timeout=timeout,
                verify=self.settings.llm_verify_ssl,
                trust_env=False,
            ) as client:
                response = client.post(
                    f"{runner_url}/v1/text/generate",
                    json={"executor_id": executor_id, "prompt": prompt},
                )
                response.raise_for_status()
        except httpx.HTTPStatusError as exc:
            raise LLMRequestError(
                f"CLI Runner 请求失败: HTTP {exc.response.status_code}"
            ) from exc
        except httpx.HTTPError as exc:
            raise LLMRequestError(f"CLI Runner 请求失败: {exc}") from exc

        try:
            data = response.json()
        except ValueError as exc:
            raise LLMRequestError("CLI Runner 返回内容不是合法 JSON") from exc
        if not isinstance(data, Mapping):
            raise LLMRequestError("CLI Runner 返回格式错误: 顶层必须是 JSON 对象")

        returned_executor_id = data.get("executor_id")
        if not isinstance(returned_executor_id, str):
            raise LLMRequestError("CLI Runner 返回格式错误: executor_id 必须是文本类型")
        if returned_executor_id.strip() != executor_id:
            raise LLMRequestError("CLI Runner 返回的执行器不一致")

        raw_content = data.get("content")
        if not isinstance(raw_content, str):
            raise LLMRequestError("CLI Runner 返回格式错误: content 必须是文本类型")
        content = raw_content.strip()
        if not content:
            raise LLMRequestError("CLI Runner 没有返回句子内容")
        return content

    def _parse_sentence_payload(self, content: str) -> tuple[str, str, str]:
        json_text = self._strip_json_fence(content)
        try:
            payload = json.loads(json_text)
        except json.JSONDecodeError as exc:
            raise LLMRequestError("大模型返回内容不是合法 JSON") from exc
        if not isinstance(payload, Mapping):
            raise LLMRequestError("大模型返回内容必须是 JSON 对象")

        sentence = self._require_sentence_text(payload, "sentence")
        translation_zh = self._require_sentence_text(
            payload,
            "translation_zh",
            alias="translationZh",
        )
        explanation_zh = self._require_sentence_text(
            payload,
            "explanation_zh",
            alias="explanationZh",
        )
        if not sentence:
            raise LLMRequestError("大模型返回 JSON 缺少 sentence")
        if not translation_zh:
            raise LLMRequestError("大模型返回 JSON 缺少 translation_zh")
        if not explanation_zh:
            raise LLMRequestError("大模型返回 JSON 缺少 explanation_zh")

        return sentence, translation_zh, explanation_zh

    @staticmethod
    def _require_sentence_text(
        payload: Mapping[str, Any],
        field: str,
        *,
        alias: str | None = None,
    ) -> str:
        lookup_key = field if field in payload or alias is None else alias
        value = payload.get(lookup_key)
        if not isinstance(value, str):
            raise LLMRequestError(f"大模型返回 JSON 字段 {field} 必须是字符串")
        return value.strip()

    def _score_word_clean_sentences_sync(
        self,
        provider: AIProvider,
        items: list[WordCleanSentenceScorePromptItem],
    ) -> list[WordCleanSentenceScoreResult]:
        if not items:
            return []

        url = f"{provider.base_url}/chat/completions"
        max_tokens = min(max(provider.max_tokens, 1), 4096)
        input_items = [
            {
                "id": item.id,
                "word_clean_id": item.word_clean_id,
                "word": item.word,
                "meaning_zh": item.meaning,
                "generated_by_model": item.model_name,
                "sentence": item.sentence,
                "sentence_translation": item.sentence_translation,
            }
            for item in items
        ]
        payload: dict[str, Any] = {
            "model": provider.model,
            "messages": [
                {
                    "role": "system",
                    "content": (
                        "你是严格但公平的英语单词例句评分老师。只返回 JSON，"
                        "顶层只有 scores 一个键。scores 是数组；每个输入必须返回"
                        "一项，字段只能是 id, score, score_reason。score 是 0 到 "
                        "100 的整数，分数越高越好。评分标准：目标词含义和用法 "
                        "40 分，英文句子自然度 25 分，中文翻译准确度 20 分，"
                        "学习者可理解性和清晰度 15 分。优秀且无明显问题通常应为 "
                        "85-100 分；基本可用但有小问题为 70-84 分；有明显问题为 "
                        "50-69 分；只有未使用目标词、句子完全不可读、或含义/翻译"
                        "严重错误到无法学习时，才给 0-49 分。若目标词已出现且英文"
                        "语法通顺，不要给 0 分。输出前必须检查 score 和 "
                        "score_reason 是否一致：如果原因写“准确、自然、清晰、优秀”，"
                        "分数通常应在 80 分以上；如果只有小问题，分数通常应在 "
                        "70 分以上；0 分只能用于未使用目标词或完全不可用。"
                        "score_reason 用简洁中文说明主要优缺点。不要返回 Markdown、"
                        "代码块、编号或额外字段。"
                    ),
                },
                {
                    "role": "user",
                    "content": json.dumps(input_items, ensure_ascii=False),
                },
            ],
            "temperature": 0,
            "max_tokens": max_tokens,
        }
        headers = {
            "Authorization": f"Bearer {provider.api_key}",
            "Content-Type": "application/json",
        }
        timeout = httpx.Timeout(self.settings.llm_timeout_seconds)

        try:
            with httpx.Client(
                timeout=timeout,
                verify=self.settings.llm_verify_ssl,
                trust_env=False,
            ) as client:
                response = client.post(url, json=payload, headers=headers)
                response.raise_for_status()
        except httpx.HTTPStatusError as exc:
            raise LLMRequestError(
                f"大模型评分请求失败: HTTP {exc.response.status_code}"
            ) from exc
        except httpx.HTTPError as exc:
            raise LLMRequestError(f"大模型评分请求失败: {exc}") from exc

        data = response.json()
        content = data.get("choices", [{}])[0].get("message", {}).get("content", "").strip()
        if not content:
            raise LLMRequestError("大模型没有返回评分内容")

        return self._parse_word_clean_sentence_score_payload(content)

    def _parse_word_clean_sentence_score_payload(
        self,
        content: str,
    ) -> list[WordCleanSentenceScoreResult]:
        json_text = self._strip_json_fence(content)
        try:
            payload = json.loads(json_text)
        except json.JSONDecodeError as exc:
            raise LLMRequestError("大模型评分返回内容不是合法 JSON") from exc

        raw_items = payload.get("scores") if isinstance(payload, dict) else payload
        if not isinstance(raw_items, list):
            raise LLMRequestError("大模型评分返回 JSON 缺少 scores 数组")

        results: list[WordCleanSentenceScoreResult] = []
        seen_ids: set[int] = set()
        for raw_item in raw_items:
            if not isinstance(raw_item, dict):
                continue
            try:
                item_id = int(raw_item.get("id"))
                score = int(raw_item.get("score"))
            except (TypeError, ValueError):
                continue
            score_reason = str(raw_item.get("score_reason") or raw_item.get("scoreReason") or "")
            score_reason = score_reason.strip()
            if item_id <= 0 or item_id in seen_ids:
                continue
            seen_ids.add(item_id)
            results.append(
                WordCleanSentenceScoreResult(
                    id=item_id,
                    score=min(max(score, 0), 100),
                    score_reason=score_reason or "大模型未返回明确评分原因",
                )
            )

        if not results:
            raise LLMRequestError("大模型评分返回 JSON 没有有效结果")
        return results

    @staticmethod
    def _strip_json_fence(content: str) -> str:
        text = content.strip()
        match = re.fullmatch(r"```(?:json)?\s*(.*?)\s*```", text, flags=re.DOTALL)
        if match:
            return match.group(1).strip()
        return text
