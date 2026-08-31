import json
import logging
from dataclasses import dataclass
from itertools import groupby
from typing import Any

import httpx
import psycopg
import yaml
from psycopg.rows import dict_row

from word_agent.core.config import Settings
from word_agent.domain.schemas import WrongWordEventRequest, WrongWordEventResponse

logger = logging.getLogger(__name__)


class WrongWordPersistenceError(RuntimeError):
    pass


class WrongWordStrategyError(RuntimeError):
    pass


@dataclass(frozen=True)
class PendingWrongWord:
    id: int
    user_id: int
    user_name: str | None
    source_answer_detail_id: int
    record_id: int | None
    word_id: int | None
    word: str
    retry_count: int = 0
    batch_key: str | None = None


@dataclass(frozen=True)
class DrainResult:
    processed_batches: int
    has_more: bool
    remaining_startup_batch_keys: frozenset[str] = frozenset()


class WrongWordStrategyService:
    def __init__(self, settings: Settings) -> None:
        self.settings = settings

    def enqueue_event(self, event: WrongWordEventRequest) -> WrongWordEventResponse:
        with self._connect() as conn:
            self._ensure_table(conn)
            event_id = self._insert_event(conn, event)
            pending_count = self._pending_count(conn, event.user_id)
        return WrongWordEventResponse(
            event_id=event_id,
            pending_count=pending_count,
            generated=False,
        )

    def handle_event(self, event: WrongWordEventRequest) -> WrongWordEventResponse:
        return self.enqueue_event(event)

    def ensure_schema(self) -> None:
        with self._connect() as conn:
            self._ensure_table(conn)

    def recover_stale_processing(self) -> None:
        with self._connect() as conn:
            self._ensure_table(conn)
            self._recover_stale_processing(conn)

    def snapshot_startup_retry_batch_keys(self) -> set[str]:
        with self._connect() as conn:
            self._ensure_table(conn)
            rows = conn.execute(
                """
                SELECT DISTINCT batch_key
                FROM public.wrong_word_events
                WHERE status = 'retry_wait'
                  AND retry_count < %(max_retries)s
                  AND batch_key IS NOT NULL
                """,
                {"max_retries": self.settings.wrong_word_max_retries},
            ).fetchall()
        return {str(row["batch_key"]) for row in rows}

    def process_available(
        self,
        *,
        startup_batch_keys: set[str] | None = None,
    ) -> DrainResult:
        remaining_startup_keys = set(startup_batch_keys or set())
        processed_batches = 0

        while processed_batches < self.settings.wrong_word_max_batches_per_wake:
            batch: list[PendingWrongWord] = []
            attempted_startup_key: str | None = None
            with self._connect() as conn:
                self._ensure_table(conn)
                if remaining_startup_keys:
                    attempted_startup_key = min(remaining_startup_keys)
                    batch = self._reserve_startup_batch(conn, attempted_startup_key)
                    remaining_startup_keys.discard(attempted_startup_key)
                if not batch:
                    batch = self._reserve_retry_batch(conn)
                if not batch:
                    batch = self._reserve_pending_batch(conn)

            if not batch:
                break

            self._process_reserved_batch(batch)
            processed_batches += 1

        with self._connect() as conn:
            has_more = bool(remaining_startup_keys) or self._has_eligible_work(conn)
        return DrainResult(
            processed_batches=processed_batches,
            has_more=has_more,
            remaining_startup_batch_keys=frozenset(remaining_startup_keys),
        )

    @property
    def _batch_size(self) -> int:
        return max(int(self.settings.cloze_batch_size), 1)

    def _connect(self) -> psycopg.Connection:
        try:
            return psycopg.connect(self._resolve_dsn(), autocommit=True, row_factory=dict_row)
        except psycopg.Error as exc:
            raise WrongWordPersistenceError(f"连接 select_english_word 数据库失败: {exc}") from exc

    def _resolve_dsn(self) -> str:
        if self.settings.select_db_dsn:
            return self.settings.select_db_dsn

        config_path = self.settings.go_config_path
        if not config_path.exists():
            raise WrongWordPersistenceError(f"Go 配置文件不存在: {config_path}")

        with config_path.open("r", encoding="utf-8") as file:
            data = yaml.safe_load(file) or {}

        pgsql = data.get("pgsql") or {}
        host = str(pgsql.get("path") or "127.0.0.1").strip()
        port = str(pgsql.get("port") or "5432").strip()
        dbname = str(pgsql.get("db-name") or "select_english_word").strip()
        user = str(pgsql.get("username") or "").strip()
        password = str(pgsql.get("password") or "").strip()
        if not user:
            raise WrongWordPersistenceError("Go 配置文件里缺少 pgsql.username")

        parts = [
            f"host={host}",
            f"port={port}",
            f"dbname={dbname}",
            f"user={user}",
        ]
        if password:
            parts.append(f"password={password}")
        return " ".join(parts)

    def _ensure_table(self, conn: psycopg.Connection) -> None:
        conn.execute(
            """
            CREATE TABLE IF NOT EXISTS public.wrong_word_events (
                id bigserial PRIMARY KEY,
                source varchar(64) NOT NULL,
                source_answer_detail_id bigint NOT NULL,
                record_id bigint NULL,
                user_id bigint NOT NULL,
                user_name varchar(100) NULL,
                word_id bigint NULL,
                word varchar(100) NOT NULL,
                word_difficulty int4 NULL,
                options_json text NULL,
                correct_answer_index int4 NULL,
                selected_answer_index int4 NULL,
                correct_meaning text NULL,
                selected_meaning text NULL,
                status varchar(32) NOT NULL DEFAULT 'pending',
                batch_key varchar(64) NULL,
                cloze_item_id bigint NULL,
                error text NULL,
                retry_count int NOT NULL DEFAULT 0,
                retry_after_event_id bigint NULL,
                locked_at timestamp NULL,
                last_attempt_at timestamp NULL,
                created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
                processed_at timestamp NULL,
                CONSTRAINT uk_wrong_word_events_source_detail
                    UNIQUE (source, source_answer_detail_id)
            )
            """
        )
        for statement in (
            "ALTER TABLE public.wrong_word_events "
            "ADD COLUMN IF NOT EXISTS retry_count int NOT NULL DEFAULT 0",
            "ALTER TABLE public.wrong_word_events "
            "ADD COLUMN IF NOT EXISTS retry_after_event_id bigint NULL",
            "ALTER TABLE public.wrong_word_events "
            "ADD COLUMN IF NOT EXISTS locked_at timestamp NULL",
            "ALTER TABLE public.wrong_word_events "
            "ADD COLUMN IF NOT EXISTS last_attempt_at timestamp NULL",
        ):
            conn.execute(statement)
        conn.execute(
            """
            CREATE INDEX IF NOT EXISTS idx_wrong_word_events_status_id
            ON public.wrong_word_events(status, id)
            """
        )
        conn.execute(
            """
            CREATE INDEX IF NOT EXISTS idx_wrong_word_events_user_status_id
            ON public.wrong_word_events(user_id, status, id)
            """
        )
        conn.execute(
            """
            CREATE INDEX IF NOT EXISTS idx_wrong_word_events_user_word
            ON public.wrong_word_events(user_id, word)
            """
        )
        conn.execute(
            """
            CREATE INDEX IF NOT EXISTS idx_wrong_word_events_user_retry
            ON public.wrong_word_events(user_id, status, retry_after_event_id, id)
            """
        )
        conn.execute(
            """
            CREATE INDEX IF NOT EXISTS idx_wrong_word_events_status_locked
            ON public.wrong_word_events(status, locked_at)
            """
        )
        self._migrate_legacy_failed_batches(conn)

    def _migrate_legacy_failed_batches(self, conn: psycopg.Connection) -> None:
        rows = conn.execute(
            """
            SELECT id, user_id
            FROM public.wrong_word_events
            WHERE status = 'pending'
              AND error IS NOT NULL
              AND batch_key IS NULL
            ORDER BY user_id, id
            """
        ).fetchall()
        for user_id, grouped_rows in groupby(rows, key=lambda row: int(row["user_id"])):
            user_rows = list(grouped_rows)
            max_row = conn.execute(
                """
                SELECT MAX(id) AS max_id
                FROM public.wrong_word_events
                WHERE user_id = %(user_id)s
                """,
                {"user_id": user_id},
            ).fetchone()
            max_event_id = int(max_row["max_id"]) if max_row and max_row["max_id"] else None
            for start in range(0, len(user_rows), self._batch_size):
                chunk = user_rows[start : start + self._batch_size]
                if len(chunk) != self._batch_size:
                    logger.warning(
                        "保留不完整的历史错误批次: user_id=%s event_ids=%s",
                        user_id,
                        [row["id"] for row in chunk],
                    )
                    continue
                ids = sorted(int(row["id"]) for row in chunk)
                batch_key = "wrong-word-events:" + "-".join(str(event_id) for event_id in ids)
                conn.execute(
                    """
                    UPDATE public.wrong_word_events
                    SET status = 'retry_wait',
                        retry_count = GREATEST(retry_count, 1),
                        retry_after_event_id = %(retry_after_event_id)s,
                        batch_key = %(batch_key)s
                    WHERE id = ANY(%(ids)s)
                    """,
                    {
                        "ids": ids,
                        "retry_after_event_id": max_event_id,
                        "batch_key": batch_key,
                    },
                )

    def _insert_event(self, conn: psycopg.Connection, event: WrongWordEventRequest) -> int | None:
        row = conn.execute(
            """
            INSERT INTO public.wrong_word_events (
                source,
                source_answer_detail_id,
                record_id,
                user_id,
                user_name,
                word_id,
                word,
                word_difficulty,
                options_json,
                correct_answer_index,
                selected_answer_index,
                correct_meaning,
                selected_meaning
            )
            VALUES (
                %(source)s,
                %(answer_detail_id)s,
                %(record_id)s,
                %(user_id)s,
                %(user_name)s,
                %(word_id)s,
                %(word)s,
                %(word_difficulty)s,
                %(options_json)s,
                %(correct_answer_index)s,
                %(selected_answer_index)s,
                %(correct_meaning)s,
                %(selected_meaning)s
            )
            ON CONFLICT (source, source_answer_detail_id) DO NOTHING
            RETURNING id
            """,
            {
                "source": event.source,
                "answer_detail_id": event.answer_detail_id,
                "record_id": event.record_id,
                "user_id": event.user_id,
                "user_name": event.user_name,
                "word_id": event.word_id,
                "word": event.word,
                "word_difficulty": event.word_difficulty,
                "options_json": json.dumps(event.options, ensure_ascii=False),
                "correct_answer_index": event.correct_answer_index,
                "selected_answer_index": event.selected_answer_index,
                "correct_meaning": event.correct_meaning,
                "selected_meaning": event.selected_meaning,
            },
        ).fetchone()
        if row:
            return int(row["id"])

        existing = conn.execute(
            """
            SELECT id
            FROM public.wrong_word_events
            WHERE source = %(source)s AND source_answer_detail_id = %(answer_detail_id)s
            """,
            {"source": event.source, "answer_detail_id": event.answer_detail_id},
        ).fetchone()
        return int(existing["id"]) if existing else None

    def _reserve_pending_batch(
        self,
        conn: psycopg.Connection,
    ) -> list[PendingWrongWord]:
        with conn.transaction():
            rows = conn.execute(
                """
                WITH candidate_user AS (
                    SELECT user_id, MIN(id) AS first_id
                    FROM public.wrong_word_events
                    WHERE status = 'pending'
                    GROUP BY user_id
                    HAVING COUNT(*) >= %(limit)s
                    ORDER BY first_id
                    LIMIT 1
                )
                SELECT event.id,
                       event.user_id,
                       event.user_name,
                       event.source_answer_detail_id,
                       event.record_id,
                       event.word_id,
                       event.word,
                       event.retry_count,
                       event.batch_key
                FROM public.wrong_word_events event
                JOIN candidate_user ON candidate_user.user_id = event.user_id
                WHERE event.status = 'pending'
                ORDER BY event.id
                LIMIT %(limit)s
                FOR UPDATE OF event SKIP LOCKED
                """,
                {"limit": self._batch_size},
            ).fetchall()
            if len(rows) < self._batch_size:
                return []
            batch = self._rows_to_batch(rows)
            batch_key = self._batch_key(batch)
            self._mark_batch_processing(conn, batch, batch_key)
            return [
                PendingWrongWord(
                    **{**item.__dict__, "batch_key": batch_key},
                )
                for item in batch
            ]

    def _reserve_retry_batch(self, conn: psycopg.Connection) -> list[PendingWrongWord]:
        with conn.transaction():
            rows = conn.execute(
                """
                WITH candidate AS (
                    SELECT candidate.batch_key,
                           candidate.user_id,
                           MIN(candidate.id) AS first_id
                    FROM public.wrong_word_events candidate
                    WHERE candidate.status = 'retry_wait'
                      AND candidate.retry_count < %(max_retries)s
                      AND candidate.batch_key IS NOT NULL
                      AND EXISTS (
                          SELECT 1
                          FROM public.wrong_word_events newer
                          WHERE newer.user_id = candidate.user_id
                            AND newer.id > candidate.retry_after_event_id
                      )
                    GROUP BY candidate.batch_key, candidate.user_id
                    HAVING COUNT(*) = %(limit)s
                    ORDER BY first_id
                    LIMIT 1
                )
                SELECT event.id,
                       event.user_id,
                       event.user_name,
                       event.source_answer_detail_id,
                       event.record_id,
                       event.word_id,
                       event.word,
                       event.retry_count,
                       event.batch_key
                FROM public.wrong_word_events event
                JOIN candidate ON candidate.batch_key = event.batch_key
                WHERE event.status = 'retry_wait'
                ORDER BY event.id
                FOR UPDATE OF event SKIP LOCKED
                """,
                {
                    "limit": self._batch_size,
                    "max_retries": self.settings.wrong_word_max_retries,
                },
            ).fetchall()
            if len(rows) != self._batch_size:
                return []
            batch = self._rows_to_batch(rows)
            self._mark_batch_processing(conn, batch, str(batch[0].batch_key))
            return batch

    def _reserve_startup_batch(
        self,
        conn: psycopg.Connection,
        batch_key: str,
    ) -> list[PendingWrongWord]:
        with conn.transaction():
            rows = conn.execute(
                """
                SELECT id,
                       user_id,
                       user_name,
                       source_answer_detail_id,
                       record_id,
                       word_id,
                       word,
                       retry_count,
                       batch_key
                FROM public.wrong_word_events
                WHERE status = 'retry_wait'
                  AND retry_count < %(max_retries)s
                  AND batch_key = %(batch_key)s
                ORDER BY id
                FOR UPDATE SKIP LOCKED
                """,
                {
                    "batch_key": batch_key,
                    "max_retries": self.settings.wrong_word_max_retries,
                },
            ).fetchall()
            if len(rows) != self._batch_size:
                return []
            batch = self._rows_to_batch(rows)
            self._mark_batch_processing(conn, batch, batch_key)
            return batch

    def _rows_to_batch(self, rows: list[dict[str, Any]]) -> list[PendingWrongWord]:
        return [
            PendingWrongWord(
                id=int(row["id"]),
                user_id=int(row["user_id"]),
                user_name=str(row["user_name"]) if row["user_name"] is not None else None,
                source_answer_detail_id=int(row["source_answer_detail_id"]),
                record_id=int(row["record_id"]) if row["record_id"] is not None else None,
                word_id=int(row["word_id"]) if row["word_id"] is not None else None,
                word=str(row["word"]),
                retry_count=int(row.get("retry_count") or 0),
                batch_key=str(row["batch_key"]) if row.get("batch_key") else None,
            )
            for row in rows
        ]

    def _mark_batch_processing(
        self,
        conn: psycopg.Connection,
        batch: list[PendingWrongWord],
        batch_key: str,
    ) -> None:
        conn.execute(
            """
            UPDATE public.wrong_word_events
            SET status = 'processing',
                batch_key = %(batch_key)s,
                locked_at = CURRENT_TIMESTAMP,
                last_attempt_at = CURRENT_TIMESTAMP,
                error = NULL
            WHERE id = ANY(%(ids)s)
            """,
            {"ids": [item.id for item in batch], "batch_key": batch_key},
        )

    def _batch_key(self, batch: list[PendingWrongWord]) -> str:
        ordered_batch = sorted(batch, key=lambda item: item.id)
        return "wrong-word-events:" + "-".join(str(item.id) for item in ordered_batch)

    def _process_reserved_batch(self, batch: list[PendingWrongWord]) -> None:
        batch_key = batch[0].batch_key or self._batch_key(batch)
        try:
            cloze_item_id = self._generate_cloze_item(batch, batch_key)
        except Exception as exc:
            error = self._format_error(exc, batch)
            logger.exception("错词批次生成失败: batch_key=%s error=%s", batch_key, error)
            with self._connect() as conn:
                self._mark_batch_failed(conn, batch, error)
            return

        with self._connect() as conn:
            self._mark_batch_processed(
                conn,
                [item.id for item in batch],
                batch_key,
                cloze_item_id,
            )

    def _generate_cloze_item(
        self,
        batch: list[PendingWrongWord],
        batch_key: str,
    ) -> int | None:
        words = [item.word for item in batch]
        user_name = next((item.user_name for item in batch if item.user_name), None)
        payload = {
            "generationKey": batch_key,
            "userId": batch[0].user_id,
            "userName": user_name,
            "words": words,
            "sourceEventIds": [item.id for item in batch],
            "sourceAnswerDetailIds": [item.source_answer_detail_id for item in batch],
            "sourceRecordIds": self._compact_ints(item.record_id for item in batch),
            "sourceWordIds": self._compact_ints(item.word_id for item in batch),
        }
        timeout_seconds = min(
            self.settings.cloze_request_timeout_seconds,
            self.settings.wrong_word_processing_timeout_seconds,
        )
        with httpx.Client(timeout=timeout_seconds, trust_env=False) as client:
            response = client.post(self.settings.cloze_generate_url, json=payload)
            response.raise_for_status()
        data: dict[str, Any] = response.json()
        cloze_item_id = data.get("id")
        return int(cloze_item_id) if cloze_item_id is not None else None

    def _format_error(
        self,
        exc: Exception,
        batch: list[PendingWrongWord],
    ) -> str:
        event_ids = [item.id for item in batch]
        words = [item.word for item in batch]
        if isinstance(exc, httpx.HTTPStatusError):
            response_text = exc.response.text[:1000]
            return (
                f"url={exc.request.url} status={exc.response.status_code} "
                f"events={event_ids} words={words} response={response_text}"
            )[:2000]
        return f"events={event_ids} words={words} error={exc}"[:2000]

    def _compact_ints(self, values: Any) -> list[int]:
        seen: set[int] = set()
        compacted: list[int] = []
        for value in values:
            if value is None:
                continue
            numeric_value = int(value)
            if numeric_value > 0 and numeric_value not in seen:
                seen.add(numeric_value)
                compacted.append(numeric_value)
        return compacted

    def _mark_batch_processed(
        self,
        conn: psycopg.Connection,
        batch_ids: list[int],
        batch_key: str,
        cloze_item_id: int | None,
    ) -> None:
        conn.execute(
            """
            UPDATE public.wrong_word_events
            SET status = 'processed',
                batch_key = %(batch_key)s,
                cloze_item_id = %(cloze_item_id)s,
                processed_at = CURRENT_TIMESTAMP,
                locked_at = NULL,
                error = NULL
            WHERE id = ANY(%(ids)s)
            """,
            {"ids": batch_ids, "batch_key": batch_key, "cloze_item_id": cloze_item_id},
        )

    def _mark_batch_failed(
        self,
        conn: psycopg.Connection,
        batch: list[PendingWrongWord],
        error: str,
    ) -> None:
        retry_count = max(item.retry_count for item in batch) + 1
        batch_key = batch[0].batch_key or self._batch_key(batch)
        params: dict[str, Any] = {
            "ids": [item.id for item in batch],
            "retry_count": retry_count,
            "batch_key": batch_key,
            "error": error[:2000],
        }
        if retry_count >= self.settings.wrong_word_max_retries:
            conn.execute(
                """
                UPDATE public.wrong_word_events
                SET status = 'failed',
                    retry_count = %(retry_count)s,
                    batch_key = %(batch_key)s,
                    locked_at = NULL,
                    error = %(error)s
                WHERE id = ANY(%(ids)s)
                """,
                params,
            )
            return

        max_row = conn.execute(
            """
            SELECT MAX(id) AS max_id
            FROM public.wrong_word_events
            WHERE user_id = %(user_id)s
            """,
            {"user_id": batch[0].user_id},
        ).fetchone()
        params["retry_after_event_id"] = (
            int(max_row["max_id"])
            if max_row and max_row["max_id"] is not None
            else max(item.id for item in batch)
        )
        conn.execute(
            """
            UPDATE public.wrong_word_events
            SET status = 'retry_wait',
                retry_count = %(retry_count)s,
                retry_after_event_id = %(retry_after_event_id)s,
                batch_key = %(batch_key)s,
                locked_at = NULL,
                error = %(error)s
            WHERE id = ANY(%(ids)s)
            """,
            params,
        )

    def _recover_stale_processing(self, conn: psycopg.Connection) -> None:
        conn.execute(
            """
            UPDATE public.wrong_word_events event
            SET retry_count = event.retry_count + 1,
                status = CASE
                    WHEN event.retry_count + 1 >= %(max_retries)s THEN 'failed'
                    ELSE 'retry_wait'
                END,
                retry_after_event_id = (
                    SELECT MAX(newer.id)
                    FROM public.wrong_word_events newer
                    WHERE newer.user_id = event.user_id
                ),
                locked_at = NULL,
                error = '处理超时或处理器异常退出'
            WHERE event.status = 'processing'
              AND event.locked_at < CURRENT_TIMESTAMP
                  - (%(timeout_seconds)s * INTERVAL '1 second')
            """,
            {
                "max_retries": self.settings.wrong_word_max_retries,
                "timeout_seconds": self.settings.wrong_word_processing_timeout_seconds,
            },
        )

    def _has_eligible_work(self, conn: psycopg.Connection) -> bool:
        row = conn.execute(
            """
            SELECT (
                EXISTS (
                    SELECT 1
                    FROM public.wrong_word_events pending
                    WHERE pending.status = 'pending'
                    GROUP BY pending.user_id
                    HAVING COUNT(*) >= %(limit)s
                )
                OR EXISTS (
                    SELECT 1
                    FROM public.wrong_word_events candidate
                    WHERE candidate.status = 'retry_wait'
                      AND candidate.retry_count < %(max_retries)s
                      AND EXISTS (
                          SELECT 1
                          FROM public.wrong_word_events newer
                          WHERE newer.user_id = candidate.user_id
                            AND newer.id > candidate.retry_after_event_id
                      )
                )
            ) AS has_work
            """,
            {
                "limit": self._batch_size,
                "max_retries": self.settings.wrong_word_max_retries,
            },
        ).fetchone()
        return bool(row and row["has_work"])

    def _pending_count(self, conn: psycopg.Connection, user_id: int) -> int:
        row = conn.execute(
            """
            SELECT COUNT(*) AS count
            FROM public.wrong_word_events
            WHERE status IN ('pending', 'processing', 'retry_wait')
              AND user_id = %(user_id)s
            """,
            {"user_id": user_id},
        ).fetchone()
        return int(row["count"]) if row else 0
