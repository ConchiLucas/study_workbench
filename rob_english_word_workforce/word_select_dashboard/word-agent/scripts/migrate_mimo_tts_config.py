#!/usr/bin/env python3
"""Migrate legacy MiMo environment settings into the dedicated TTS table."""

from __future__ import annotations

import os
import sys
from collections.abc import Callable, Mapping
from typing import Any, NamedTuple
from urllib.parse import urlparse

import psycopg
from psycopg.rows import dict_row

PROVIDER_ID = "xiaomi-mimo-tts"
PROVIDER_LABEL = "Xiaomi MiMo TTS"
PROVIDER_TYPE = "mimo-tts"
DEFAULT_BASE_URL = "https://api.xiaomimimo.com/v1"
DEFAULT_MODEL = "mimo-v2.5-tts"
DEFAULT_VOICE = "Chloe"


class MigrationError(RuntimeError):
    pass


class MigrationResult(NamedTuple):
    provider_id: str
    model: str
    voice: str
    api_key_configured: bool


def migrate_mimo_tts_config(
    env: Mapping[str, str],
    *,
    connect: Callable[..., Any] = psycopg.connect,
    output: Callable[[str], None] = print,
) -> MigrationResult:
    dsn = _first_nonblank(env, "WORD_AGENT_SELECT_DB_DSN")
    if not dsn:
        raise MigrationError("缺少 WORD_AGENT_SELECT_DB_DSN")

    api_key = _first_nonblank(env, "WORD_AGENT_MIMO_API_KEY", "MIMO_API_KEY")
    if not api_key:
        raise MigrationError("缺少旧 MiMo API Key，无法执行迁移")

    base_url = (
        _first_nonblank(env, "WORD_AGENT_MIMO_TTS_BASE_URL", "MIMO_TTS_BASE_URL")
        or DEFAULT_BASE_URL
    ).rstrip("/")
    model = (
        _first_nonblank(
            env,
            "WORD_AGENT_MIMO_TTS_DEFAULT_MODEL",
            "MIMO_TTS_DEFAULT_MODEL",
        )
        or DEFAULT_MODEL
    )
    voice = (
        _first_nonblank(
            env,
            "WORD_AGENT_MIMO_TTS_DEFAULT_VOICE",
            "MIMO_TTS_DEFAULT_VOICE",
        )
        or DEFAULT_VOICE
    )
    parsed_url = urlparse(base_url)
    if parsed_url.scheme not in {"http", "https"} or not parsed_url.netloc:
        raise MigrationError("旧 MiMo Base URL 无效")

    try:
        with connect(dsn, row_factory=dict_row) as conn:
            with conn.cursor() as cursor:
                cursor.execute(
                    """
                    UPDATE tts_provider_configs
                    SET active = FALSE, updated_at = NOW()
                    WHERE provider_id <> %s
                    """,
                    (PROVIDER_ID,),
                )
                cursor.execute(
                    """
                    INSERT INTO tts_provider_configs
                        (provider_id, label, type, base_url, api_key, model, voice,
                         enabled, active, created_at, updated_at)
                    VALUES
                        (%s, %s, %s, %s, %s, %s, %s,
                         TRUE, TRUE, NOW(), NOW())
                    ON CONFLICT (provider_id) DO UPDATE SET
                        label = EXCLUDED.label,
                        type = EXCLUDED.type,
                        base_url = EXCLUDED.base_url,
                        api_key = EXCLUDED.api_key,
                        model = EXCLUDED.model,
                        voice = EXCLUDED.voice,
                        enabled = TRUE,
                        active = TRUE,
                        updated_at = NOW()
                    """,
                    (
                        PROVIDER_ID,
                        PROVIDER_LABEL,
                        PROVIDER_TYPE,
                        base_url,
                        api_key,
                        model,
                        voice,
                    ),
                )
                cursor.execute(
                    """
                    SELECT provider_id, type, base_url, model, voice, enabled, active,
                           btrim(api_key) <> '' AS api_key_configured
                    FROM tts_provider_configs
                    WHERE provider_id = %s
                    """,
                    (PROVIDER_ID,),
                )
                row = cursor.fetchone()
                if not _verification_matches(
                    row,
                    base_url=base_url,
                    model=model,
                    voice=voice,
                ):
                    raise MigrationError("MiMo TTS 配置迁移后校验失败")
    except MigrationError:
        raise
    except psycopg.Error as exc:
        raise MigrationError("MiMo TTS 配置数据库迁移失败") from exc

    result = MigrationResult(
        provider_id=PROVIDER_ID,
        model=model,
        voice=voice,
        api_key_configured=True,
    )
    output(
        "Migration completed: "
        f"provider_id={result.provider_id}, "
        f"model={result.model}, "
        f"voice={result.voice}, "
        "api_key_configured=true"
    )
    return result


def _first_nonblank(env: Mapping[str, str], *names: str) -> str:
    for name in names:
        value = str(env.get(name) or "").strip()
        if value:
            return value
    return ""


def _verification_matches(
    row: Mapping[str, Any] | None,
    *,
    base_url: str,
    model: str,
    voice: str,
) -> bool:
    if row is None:
        return False
    return (
        str(row.get("provider_id") or "").strip() == PROVIDER_ID
        and str(row.get("type") or "").strip() == PROVIDER_TYPE
        and str(row.get("base_url") or "").strip().rstrip("/") == base_url
        and str(row.get("model") or "").strip() == model
        and str(row.get("voice") or "").strip() == voice
        and row.get("enabled") is True
        and row.get("active") is True
        and row.get("api_key_configured") is True
    )


def main() -> int:
    try:
        migrate_mimo_tts_config(os.environ)
    except MigrationError as exc:
        print(str(exc), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
