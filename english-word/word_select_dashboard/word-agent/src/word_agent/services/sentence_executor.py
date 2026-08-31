from dataclasses import dataclass
from typing import Literal

import psycopg
import yaml
from psycopg.rows import dict_row

from word_agent.core.config import Settings
from word_agent.services.llm_client import AIProvider, LLMConfigError


@dataclass(frozen=True)
class SentenceExecutorTarget:
    type: Literal["api", "cli"]
    id: str
    api_provider: AIProvider | None = None
    cli_provider: AIProvider | None = None


@dataclass(frozen=True)
class SentenceExecutorLoader:
    settings: Settings

    def load(self) -> SentenceExecutorTarget:
        try:
            with psycopg.connect(self._resolve_select_db_dsn(), row_factory=dict_row) as conn:
                with conn.cursor() as cursor:
                    cursor.execute(
                        """
                        SELECT executor_type, executor_id
                        FROM sentence_executor_config
                        WHERE singleton_key = %s
                        """,
                        ("default",),
                    )
                    selected = cursor.fetchone()
                    if selected is None:
                        raise LLMConfigError("尚未选择造句执行器")

                    executor_type = str(selected["executor_type"] or "").strip()
                    executor_id = str(selected["executor_id"] or "").strip()
                    if executor_type == "cli":
                        return self._load_cli(cursor, executor_id)
                    if executor_type == "api":
                        return self._load_api(cursor, executor_id)
                    raise LLMConfigError(f"不支持的造句执行器类型: {executor_type}")
        except LLMConfigError:
            raise
        except psycopg.Error as exc:
            raise LLMConfigError(f"读取数据库造句执行器配置失败: {exc}") from exc

    def _load_cli(self, cursor, executor_id: str) -> SentenceExecutorTarget:
        if not executor_id:
            raise LLMConfigError("造句 CLI 配置 ID 不能为空")
        cursor.execute(
            """
            SELECT provider_id, label, driver, command_path, model, working_directory,
                   timeout_seconds, enabled
            FROM cli_provider_configs
            WHERE provider_id = %s
            """,
            (executor_id,),
        )
        row = cursor.fetchone()
        if row is None:
            raise LLMConfigError(f"造句 CLI 配置不存在: {executor_id}")
        if not row["enabled"]:
            raise LLMConfigError(f"造句 CLI 配置已停用: {executor_id}")
        required_text = (
            row["provider_id"],
            row["driver"],
            row["command_path"],
            row["model"],
            row["working_directory"],
        )
        if any(not str(value or "").strip() for value in required_text):
            raise LLMConfigError(f"造句 CLI 配置不完整: {executor_id}")
        if int(row["timeout_seconds"] or 0) <= 0:
            raise LLMConfigError(f"造句 CLI 配置不完整: {executor_id}")
        provider = AIProvider(
            id=str(row["provider_id"] or "").strip(),
            label=str(row["label"] or executor_id).strip(),
            type="cli",
            base_url="",
            api_key="",
            model=str(row["model"] or "").strip(),
            max_tokens=0,
        )
        return SentenceExecutorTarget(type="cli", id=executor_id, cli_provider=provider)

    def _load_api(self, cursor, executor_id: str) -> SentenceExecutorTarget:
        if not executor_id:
            raise LLMConfigError("造句 API 配置 ID 不能为空")
        cursor.execute(
            """
            SELECT provider_id, label, type, base_url, api_key, model, max_tokens
            FROM ai_provider_configs
            WHERE provider_id = %s
            """,
            (executor_id,),
        )
        row = cursor.fetchone()
        if row is None:
            raise LLMConfigError(f"造句 API 配置不存在: {executor_id}")
        provider = AIProvider(
            id=str(row["provider_id"] or "").strip(),
            label=str(row["label"] or executor_id).strip(),
            type=str(row["type"] or "").strip(),
            base_url=str(row["base_url"] or "").strip().rstrip("/"),
            api_key=str(row["api_key"] or "").strip(),
            model=str(row["model"] or "").strip(),
            max_tokens=int(row["max_tokens"] or 0),
        )
        if not all(
            (
                provider.id,
                provider.type,
                provider.base_url,
                provider.api_key,
                provider.model,
            )
        ) or provider.max_tokens <= 0:
            raise LLMConfigError(f"造句 API 配置不完整: {executor_id}")
        return SentenceExecutorTarget(type="api", id=executor_id, api_provider=provider)

    def _resolve_select_db_dsn(self) -> str:
        if self.settings.select_db_dsn:
            return self.settings.select_db_dsn

        with self.settings.go_config_path.open("r", encoding="utf-8") as file:
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
