from __future__ import annotations

from contextlib import nullcontext
from typing import Any

from word_agent.core.config import Settings
from word_agent.domain.schemas import WrongWordEventRequest
from word_agent.services.wrong_word_strategy import PendingWrongWord, WrongWordStrategyService


class RecordingResult:
    def __init__(self, rows: list[dict[str, Any]] | None = None) -> None:
        self.rows = rows or []

    def fetchone(self) -> dict[str, Any] | None:
        return self.rows[0] if self.rows else None

    def fetchall(self) -> list[dict[str, Any]]:
        return list(self.rows)


class RecordingConnection:
    def __init__(self, *, max_event_id: int = 53) -> None:
        self.max_event_id = max_event_id
        self.executions: list[tuple[str, dict[str, Any]]] = []

    def execute(
        self,
        sql: str,
        params: dict[str, Any] | None = None,
    ) -> RecordingResult:
        normalized = " ".join(sql.split())
        captured_params = dict(params or {})
        self.executions.append((normalized, captured_params))
        if "SELECT MAX(id) AS max_id" in normalized:
            return RecordingResult([{"max_id": self.max_event_id}])
        return RecordingResult()

    def transaction(self):
        return nullcontext()


def settings() -> Settings:
    return Settings(select_db_dsn="dbname=test")


def sample_batch(*, retry_count: int = 0) -> list[PendingWrongWord]:
    return [
        PendingWrongWord(
            id=event_id,
            user_id=2,
            user_name="conchi",
            source_answer_detail_id=100 + event_id,
            record_id=10,
            word_id=200 + event_id,
            word=word,
            retry_count=retry_count,
            batch_key="wrong-word-events:47-48-49",
        )
        for event_id, word in zip(
            [47, 48, 49],
            ["my", "adolescence", "outpost"],
            strict=True,
        )
    ]


def test_enqueue_event_only_persists_and_never_generates() -> None:
    class EnqueueOnlyService(WrongWordStrategyService):
        def _connect(self):
            return nullcontext(RecordingConnection())

        def _ensure_table(self, conn) -> None:
            _ = conn

        def _insert_event(self, conn, event) -> int:
            _ = conn, event
            return 54

        def _pending_count(self, conn, user_id: int) -> int:
            _ = conn, user_id
            return 4

        def _generate_cloze_item(self, batch, batch_key):
            raise AssertionError("enqueue must not generate a cloze item")

    service = EnqueueOnlyService(settings())
    event = WrongWordEventRequest(
        answerDetailId=154,
        userId=2,
        word="clearly",
    )

    response = service.enqueue_event(event)

    assert response.event_id == 54
    assert response.pending_count == 4
    assert response.generated is False


def test_schema_adds_retry_watermark_and_lock_columns() -> None:
    service = WrongWordStrategyService(settings())
    conn = RecordingConnection()

    service._ensure_table(conn)

    sql = "\n".join(statement for statement, _ in conn.executions)
    assert "retry_count" in sql
    assert "retry_after_event_id" in sql
    assert "locked_at" in sql
    assert "last_attempt_at" in sql
    assert "idx_wrong_word_events_user_retry" in sql
    assert "idx_wrong_word_events_status_locked" in sql


def test_batch_key_is_stable_for_the_same_event_ids() -> None:
    service = WrongWordStrategyService(settings())

    assert service._batch_key(sample_batch()) == "wrong-word-events:47-48-49"


def test_failed_batch_waits_for_a_newer_event() -> None:
    service = WrongWordStrategyService(settings())
    conn = RecordingConnection(max_event_id=53)

    service._mark_batch_failed(conn, sample_batch(), "HTTP 400")

    update_sql, params = conn.executions[-1]
    assert "status = 'retry_wait'" in update_sql
    assert params["retry_count"] == 1
    assert params["retry_after_event_id"] == 53
    assert params["batch_key"] == "wrong-word-events:47-48-49"


def test_third_failure_is_isolated() -> None:
    service = WrongWordStrategyService(settings())
    conn = RecordingConnection(max_event_id=53)

    service._mark_batch_failed(conn, sample_batch(retry_count=2), "HTTP 400")

    update_sql, params = conn.executions[-1]
    assert "status = 'failed'" in update_sql
    assert params["retry_count"] == 3


def test_retry_claim_requires_same_user_newer_event() -> None:
    service = WrongWordStrategyService(settings())
    conn = RecordingConnection()

    assert service._reserve_retry_batch(conn) == []

    sql = conn.executions[-1][0]
    assert "newer.user_id = candidate.user_id" in sql
    assert "newer.id > candidate.retry_after_event_id" in sql
    assert "FOR UPDATE" in sql
    assert "SKIP LOCKED" in sql


def test_fresh_claim_reads_only_pending_rows() -> None:
    service = WrongWordStrategyService(settings())
    conn = RecordingConnection()

    assert service._reserve_pending_batch(conn) == []

    sql = conn.executions[-1][0]
    assert "status = 'pending'" in sql
    assert "retry_wait" not in sql
