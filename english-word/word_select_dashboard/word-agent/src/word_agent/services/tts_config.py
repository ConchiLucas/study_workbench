from collections.abc import Callable
from dataclasses import dataclass
from typing import Any
from urllib.parse import urlparse

import psycopg
import yaml
from psycopg.rows import dict_row

from word_agent.core.config import Settings


OFFICIAL_MIMO_API_HOST = "api.xiaomimimo.com"
POSTGRES_CONNECT_TIMEOUT_SECONDS = 5


class TTSConfigError(RuntimeError):
    pass


@dataclass(frozen=True, slots=True)
class ActiveTTSConfig:
    provider_id: str
    base_url: str
    api_key: str
    model: str
    voice: str


class TTSConfigLoader:
    def __init__(
        self,
        settings: Settings,
        *,
        connect: Callable[..., Any] | None = None,
    ) -> None:
        self.settings = settings
        self._connect = connect or psycopg.connect

    def load_active_mimo_config(self) -> ActiveTTSConfig:
        dsn = self._resolve_select_db_dsn()
        try:
            with self._connect(
                dsn,
                row_factory=dict_row,
                connect_timeout=POSTGRES_CONNECT_TIMEOUT_SECONDS,
            ) as conn:
                with conn.cursor() as cursor:
                    cursor.execute(
                        """
                        SELECT provider_id, type, base_url, api_key, model, voice
                        FROM tts_provider_configs
                        WHERE type = 'mimo-tts'
                          AND enabled = TRUE
                          AND active = TRUE
                        ORDER BY id
                        LIMIT 2
                        """
                    )
                    rows = cursor.fetchall()
        except psycopg.Error as exc:
            raise TTSConfigError("读取数据库 TTS 配置失败") from exc

        if not rows:
            raise TTSConfigError("数据库里没有启用的默认 MiMo TTS 配置")
        if len(rows) > 1:
            raise TTSConfigError("数据库里存在多个启用的默认 MiMo TTS 配置")

        row = rows[0]
        provider_id = str(row.get("provider_id") or "").strip()
        base_url = str(row.get("base_url") or "").strip().rstrip("/")
        api_key = str(row.get("api_key") or "").strip()
        model = str(row.get("model") or "").strip()
        voice = str(row.get("voice") or "").strip()
        parsed_url = urlparse(base_url)
        if (
            not provider_id
            or not base_url
            or not api_key
            or not model
            or not voice
        ):
            raise TTSConfigError("默认 MiMo TTS 配置字段不完整")

        try:
            port = parsed_url.port
        except ValueError as exc:
            raise TTSConfigError("默认 MiMo TTS 配置必须使用官方 HTTPS 地址") from exc
        if (
            parsed_url.scheme != "https"
            or parsed_url.hostname != OFFICIAL_MIMO_API_HOST
            or port not in {None, 443}
            or parsed_url.username is not None
            or parsed_url.password is not None
            or parsed_url.query
            or parsed_url.fragment
        ):
            raise TTSConfigError("默认 MiMo TTS 配置必须使用官方 HTTPS 地址")

        return ActiveTTSConfig(
            provider_id=provider_id,
            base_url=base_url,
            api_key=api_key,
            model=model,
            voice=voice,
        )

    def _resolve_select_db_dsn(self) -> str:
        if self.settings.select_db_dsn:
            return self.settings.select_db_dsn

        config_path = self.settings.go_config_path
        try:
            with config_path.open("r", encoding="utf-8") as file:
                data = yaml.safe_load(file) or {}
        except (OSError, yaml.YAMLError) as exc:
            raise TTSConfigError("无法读取 Go 数据库配置") from exc

        pgsql = data.get("pgsql") or {}
        host = str(pgsql.get("path") or "127.0.0.1").strip()
        port = str(pgsql.get("port") or "5432").strip()
        dbname = str(pgsql.get("db-name") or "select_english_word").strip()
        user = str(pgsql.get("username") or "").strip()
        password = str(pgsql.get("password") or "").strip()
        if not user:
            raise TTSConfigError("Go 配置文件里缺少 pgsql.username")
        return " ".join(
            [
                f"host={host}",
                f"port={port}",
                f"dbname={dbname}",
                f"user={user}",
                f"password={password}",
            ]
        )
