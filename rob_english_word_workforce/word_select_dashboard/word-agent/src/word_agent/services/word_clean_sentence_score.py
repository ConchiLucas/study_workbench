from dataclasses import dataclass
from typing import Any

import psycopg
from psycopg.rows import dict_row

from word_agent.core.config import Settings
from word_agent.domain.schemas import (
    StepStatus,
    WordCleanBestSentenceItem,
    WordCleanSentenceScoreItem,
    WordCleanSentenceScoreRequest,
    WordCleanSentenceScoreResponse,
)
from word_agent.services.best_sentence_cloze import build_best_sentence_cloze
from word_agent.services.llm_client import (
    LLMClient,
    WordCleanSentenceScorePromptItem,
    WordCleanSentenceScoreResult,
)


class WordCleanSentenceScoreError(RuntimeError):
    pass


class WordCleanSentenceScoreValidationError(WordCleanSentenceScoreError):
    pass


@dataclass(frozen=True)
class WordCleanSentenceForScore:
    id: int
    word_clean_id: int
    word: str
    meaning: str
    model_name: str
    sentence: str
    sentence_translation: str


class WordCleanSentenceScoreService:
    def __init__(self, settings: Settings) -> None:
        self.settings = settings
        self.llm_client = LLMClient(settings)

    async def score(self, request: WordCleanSentenceScoreRequest) -> WordCleanSentenceScoreResponse:
        if not request.ids and not request.word_clean_ids:
            raise WordCleanSentenceScoreValidationError(
                "必须传入 ids 或 wordCleanIds，避免误触发全表评分"
            )

        judge_model = (
            request.judge_model or self.settings.word_clean_score_default_model
        ).strip()
        if not judge_model:
            raise WordCleanSentenceScoreError("评分模型未配置")
        provider = self.llm_client.load_provider_by_model(judge_model)

        with self._connect() as conn:
            self._ensure_score_columns(conn)
            self._ensure_best_sentence_table(conn)
            requested_word_clean_ids = self._resolve_requested_word_clean_ids(conn, request)
            items = self._fetch_items(conn, request)

        if not items:
            best_items: list[WordCleanBestSentenceItem] = []
            if requested_word_clean_ids:
                with self._connect() as conn:
                    self._ensure_best_sentence_table(conn)
                    best_items = self._upsert_best_sentences(conn, requested_word_clean_ids)
            message = "没有需要评分的造句结果"
            if best_items:
                message += f"，已刷新 {len(best_items)} 条最佳句子"
            return WordCleanSentenceScoreResponse(
                status=StepStatus.SUCCESS,
                message=message,
                judge_model=provider.model,
                processed_count=0,
                scored_count=0,
                failed_count=0,
                items=[],
                best_items=best_items,
            )

        try:
            score_results = await self.llm_client.score_word_clean_sentences(
                provider=provider,
                items=[
                    WordCleanSentenceScorePromptItem(
                        id=item.id,
                        word_clean_id=item.word_clean_id,
                        word=item.word,
                        meaning=item.meaning,
                        model_name=item.model_name,
                        sentence=item.sentence,
                        sentence_translation=item.sentence_translation,
                    )
                    for item in items
                ],
            )
        except Exception as exc:
            raise WordCleanSentenceScoreError(f"大模型评分失败: {exc}") from exc

        with self._connect() as conn:
            self._ensure_best_sentence_table(conn)
            saved_items = self._save_scores(
                conn,
                items=items,
                results=score_results,
                judge_model=provider.model,
            )
            best_items = self._upsert_best_sentences(
                conn,
                sorted({item.word_clean_id for item in items}),
            )

        scored_count = len(saved_items)
        failed_count = len(items) - scored_count
        message = f"已评分 {scored_count} 条，失败 {failed_count} 条"
        if best_items:
            message += f"，已更新 {len(best_items)} 条最佳句子"
        return WordCleanSentenceScoreResponse(
            status=StepStatus.SUCCESS if failed_count == 0 else StepStatus.FAILED,
            message=message,
            judge_model=provider.model,
            processed_count=len(items),
            scored_count=scored_count,
            failed_count=failed_count,
            items=saved_items,
            best_items=best_items,
        )

    def _connect(self) -> psycopg.Connection:
        try:
            return psycopg.connect(self._resolve_dsn(), row_factory=dict_row)
        except psycopg.Error as exc:
            raise WordCleanSentenceScoreError(
                f"连接 rob_english_word 数据库失败: {exc}"
            ) from exc

    def _resolve_dsn(self) -> str:
        if self.settings.rob_word_db_dsn:
            return self.settings.rob_word_db_dsn
        return "host=127.0.0.1 port=5432 dbname=rob_english_word user=conchi password=conchi123456"

    def _ensure_score_columns(self, conn: psycopg.Connection) -> None:
        with conn.transaction():
            conn.execute(
                """
                ALTER TABLE public.word_clean_sentence
                ADD COLUMN IF NOT EXISTS score integer NULL
                """
            )
            conn.execute(
                """
                ALTER TABLE public.word_clean_sentence
                ADD COLUMN IF NOT EXISTS score_reason text NOT NULL DEFAULT ''
                """
            )
            conn.execute(
                """
                ALTER TABLE public.word_clean_sentence
                ADD COLUMN IF NOT EXISTS score_model_name varchar(128) NOT NULL DEFAULT ''
                """
            )
            conn.execute(
                """
                ALTER TABLE public.word_clean_sentence
                ADD COLUMN IF NOT EXISTS scored_at timestamptz NULL
                """
            )
            conn.execute(
                """
                CREATE INDEX IF NOT EXISTS idx_word_clean_sentence_score
                ON public.word_clean_sentence(score)
                """
            )

    def _ensure_best_sentence_table(self, conn: psycopg.Connection) -> None:
        statements = [
            """
            CREATE TABLE IF NOT EXISTS public.word_clean_best_sentence (
                id bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
                word_clean_id bigint NOT NULL,
                word varchar(100) NOT NULL,
                meaning text NOT NULL DEFAULT '',
                source_sentence_id bigint NOT NULL,
                source_model_name varchar(160) NOT NULL,
                sentence text NOT NULL,
                sentence_translation text NOT NULL DEFAULT '',
                cloze_sentence text NOT NULL DEFAULT '',
                cloze_answer text NOT NULL DEFAULT '',
                score integer NOT NULL,
                score_reason text NOT NULL DEFAULT '',
                score_model_name varchar(128) NOT NULL DEFAULT '',
                scored_at timestamptz NULL,
                tts_status varchar(32) NOT NULL DEFAULT 'pending',
                tts_provider varchar(64) NOT NULL DEFAULT '',
                tts_model varchar(128) NOT NULL DEFAULT '',
                tts_voice varchar(128) NOT NULL DEFAULT '',
                tts_audio_format varchar(32) NOT NULL DEFAULT '',
                tts_bucket varchar(128) NOT NULL DEFAULT '',
                tts_object_key text NOT NULL DEFAULT '',
                tts_object_url text NOT NULL DEFAULT '',
                tts_content_type varchar(128) NOT NULL DEFAULT '',
                tts_file_size bigint NULL,
                tts_duration_ms integer NULL,
                tts_generated_at timestamptz NULL,
                tts_error_message text NOT NULL DEFAULT '',
                created_at timestamptz NOT NULL DEFAULT now(),
                updated_at timestamptz NOT NULL DEFAULT now()
            )
            """,
            """
            ALTER TABLE public.word_clean_best_sentence
            ADD COLUMN IF NOT EXISTS cloze_sentence text NOT NULL DEFAULT ''
            """,
            """
            ALTER TABLE public.word_clean_best_sentence
            ADD COLUMN IF NOT EXISTS cloze_answer text NOT NULL DEFAULT ''
            """,
            """
            CREATE UNIQUE INDEX IF NOT EXISTS uk_word_clean_best_sentence_word_clean
            ON public.word_clean_best_sentence(word_clean_id)
            """,
            """
            CREATE INDEX IF NOT EXISTS idx_word_clean_best_sentence_source_sentence
            ON public.word_clean_best_sentence(source_sentence_id)
            """,
            """
            CREATE INDEX IF NOT EXISTS idx_word_clean_best_sentence_score
            ON public.word_clean_best_sentence(score)
            """,
            """
            CREATE INDEX IF NOT EXISTS idx_word_clean_best_sentence_tts_status
            ON public.word_clean_best_sentence(tts_status)
            """,
            """
            DO $$
            BEGIN
                IF NOT EXISTS (
                    SELECT 1
                    FROM pg_constraint
                    WHERE conname = 'fk_word_clean_best_sentence_word_clean'
                ) THEN
                    ALTER TABLE public.word_clean_best_sentence
                        ADD CONSTRAINT fk_word_clean_best_sentence_word_clean
                        FOREIGN KEY (word_clean_id)
                        REFERENCES public.word_clean(id)
                        ON DELETE CASCADE;
                END IF;

                IF NOT EXISTS (
                    SELECT 1
                    FROM pg_constraint
                    WHERE conname = 'fk_word_clean_best_sentence_source_sentence'
                ) THEN
                    ALTER TABLE public.word_clean_best_sentence
                        ADD CONSTRAINT fk_word_clean_best_sentence_source_sentence
                        FOREIGN KEY (source_sentence_id)
                        REFERENCES public.word_clean_sentence(id)
                        ON DELETE CASCADE;
                END IF;
            END $$
            """,
        ]
        with conn.transaction():
            for statement in statements:
                conn.execute(statement)

    def _resolve_requested_word_clean_ids(
        self,
        conn: psycopg.Connection,
        request: WordCleanSentenceScoreRequest,
    ) -> list[int]:
        word_clean_ids = set(request.word_clean_ids or [])
        if request.ids:
            rows = conn.execute(
                """
                SELECT DISTINCT word_clean_id
                FROM public.word_clean_sentence
                WHERE id = ANY(%(ids)s)
                """,
                {"ids": request.ids},
            ).fetchall()
            word_clean_ids.update(int(row["word_clean_id"]) for row in rows)
        return sorted(word_clean_ids)

    def _fetch_items(
        self,
        conn: psycopg.Connection,
        request: WordCleanSentenceScoreRequest,
    ) -> list[WordCleanSentenceForScore]:
        clauses = ["1=1"]
        params: dict[str, Any] = {"limit": request.limit}
        if request.ids:
            clauses.append("wcs.id = ANY(%(ids)s)")
            params["ids"] = request.ids
        if request.word_clean_ids:
            clauses.append("wcs.word_clean_id = ANY(%(word_clean_ids)s)")
            params["word_clean_ids"] = request.word_clean_ids
        if request.model_names:
            clauses.append("wcs.model_name = ANY(%(model_names)s)")
            params["model_names"] = request.model_names
        if not request.overwrite:
            clauses.append("wcs.score IS NULL")

        rows = conn.execute(
            f"""
            SELECT wcs.id,
                   wcs.word_clean_id,
                   wcs.word,
                   COALESCE(wc.meaning, '') AS meaning,
                   wcs.model_name,
                   wcs.sentence,
                   COALESCE(wcs.sentence_translation, '') AS sentence_translation
            FROM public.word_clean_sentence wcs
            JOIN public.word_clean wc ON wc.id = wcs.word_clean_id
            WHERE {" AND ".join(clauses)}
            ORDER BY wcs.id ASC
            LIMIT %(limit)s
            """,
            params,
        ).fetchall()

        return [
            WordCleanSentenceForScore(
                id=int(row["id"]),
                word_clean_id=int(row["word_clean_id"]),
                word=str(row["word"]),
                meaning=str(row["meaning"]),
                model_name=str(row["model_name"]),
                sentence=str(row["sentence"]),
                sentence_translation=str(row["sentence_translation"]),
            )
            for row in rows
        ]

    def _save_scores(
        self,
        conn: psycopg.Connection,
        *,
        items: list[WordCleanSentenceForScore],
        results: list[WordCleanSentenceScoreResult],
        judge_model: str,
    ) -> list[WordCleanSentenceScoreItem]:
        item_by_id = {item.id: item for item in items}
        saved_items: list[WordCleanSentenceScoreItem] = []

        with conn.transaction():
            for result in results:
                item = item_by_id.get(result.id)
                if item is None:
                    continue
                conn.execute(
                    """
                    UPDATE public.word_clean_sentence
                    SET score = %(score)s,
                        score_reason = %(score_reason)s,
                        score_model_name = %(score_model_name)s,
                        scored_at = now()
                    WHERE id = %(id)s
                    """,
                    {
                        "id": result.id,
                        "score": result.score,
                        "score_reason": result.score_reason,
                        "score_model_name": judge_model,
                    },
                )
                saved_items.append(
                    WordCleanSentenceScoreItem(
                        id=item.id,
                        word_clean_id=item.word_clean_id,
                        word=item.word,
                        model_name=item.model_name,
                        score=result.score,
                        score_reason=result.score_reason,
                    )
                )

        return saved_items

    def _upsert_best_sentences(
        self,
        conn: psycopg.Connection,
        word_clean_ids: list[int],
    ) -> list[WordCleanBestSentenceItem]:
        if not word_clean_ids:
            return []

        best_rows = conn.execute(
            """
            WITH ranked AS (
                SELECT wcs.id AS source_sentence_id,
                       wcs.word_clean_id,
                       wcs.word,
                       COALESCE(wc.meaning, '') AS meaning,
                       wcs.model_name AS source_model_name,
                       wcs.sentence,
                       COALESCE(wcs.sentence_translation, '') AS sentence_translation,
                       wcs.score,
                       COALESCE(wcs.score_reason, '') AS score_reason,
                       COALESCE(wcs.score_model_name, '') AS score_model_name,
                       wcs.scored_at,
                       row_number() OVER (
                           PARTITION BY wcs.word_clean_id
                           ORDER BY wcs.score DESC NULLS LAST, wcs.id ASC
                       ) AS row_rank
                FROM public.word_clean_sentence wcs
                JOIN public.word_clean wc ON wc.id = wcs.word_clean_id
                WHERE wcs.word_clean_id = ANY(%(word_clean_ids)s)
                  AND wcs.score IS NOT NULL
            )
            SELECT *
            FROM ranked
            WHERE row_rank = 1
            ORDER BY word_clean_id ASC
            """,
            {"word_clean_ids": word_clean_ids},
        ).fetchall()
        if not best_rows:
            return []

        best_items: list[WordCleanBestSentenceItem] = []
        with conn.transaction():
            for row in best_rows:
                cloze = build_best_sentence_cloze(str(row["word"]), str(row["sentence"]))
                upserted_row = conn.execute(
                    """
                    INSERT INTO public.word_clean_best_sentence (
                        word_clean_id,
                        word,
                        meaning,
                        source_sentence_id,
                        source_model_name,
                        sentence,
                        sentence_translation,
                        cloze_sentence,
                        cloze_answer,
                        score,
                        score_reason,
                        score_model_name,
                        scored_at
                    )
                    VALUES (
                        %(word_clean_id)s,
                        %(word)s,
                        %(meaning)s,
                        %(source_sentence_id)s,
                        %(source_model_name)s,
                        %(sentence)s,
                        %(sentence_translation)s,
                        %(cloze_sentence)s,
                        %(cloze_answer)s,
                        %(score)s,
                        %(score_reason)s,
                        %(score_model_name)s,
                        %(scored_at)s
                    )
                    ON CONFLICT (word_clean_id) DO UPDATE
                    SET word = EXCLUDED.word,
                        meaning = EXCLUDED.meaning,
                        source_sentence_id = EXCLUDED.source_sentence_id,
                        source_model_name = EXCLUDED.source_model_name,
                        sentence = EXCLUDED.sentence,
                        sentence_translation = EXCLUDED.sentence_translation,
                        cloze_sentence = EXCLUDED.cloze_sentence,
                        cloze_answer = EXCLUDED.cloze_answer,
                        score = EXCLUDED.score,
                        score_reason = EXCLUDED.score_reason,
                        score_model_name = EXCLUDED.score_model_name,
                        scored_at = EXCLUDED.scored_at,
                        tts_status = CASE
                            WHEN public.word_clean_best_sentence.source_sentence_id IS DISTINCT FROM EXCLUDED.source_sentence_id
                              OR public.word_clean_best_sentence.sentence IS DISTINCT FROM EXCLUDED.sentence
                              OR public.word_clean_best_sentence.sentence_translation IS DISTINCT FROM EXCLUDED.sentence_translation
                            THEN 'pending'
                            ELSE public.word_clean_best_sentence.tts_status
                        END,
                        tts_provider = CASE
                            WHEN public.word_clean_best_sentence.source_sentence_id IS DISTINCT FROM EXCLUDED.source_sentence_id
                              OR public.word_clean_best_sentence.sentence IS DISTINCT FROM EXCLUDED.sentence
                              OR public.word_clean_best_sentence.sentence_translation IS DISTINCT FROM EXCLUDED.sentence_translation
                            THEN ''
                            ELSE public.word_clean_best_sentence.tts_provider
                        END,
                        tts_model = CASE
                            WHEN public.word_clean_best_sentence.source_sentence_id IS DISTINCT FROM EXCLUDED.source_sentence_id
                              OR public.word_clean_best_sentence.sentence IS DISTINCT FROM EXCLUDED.sentence
                              OR public.word_clean_best_sentence.sentence_translation IS DISTINCT FROM EXCLUDED.sentence_translation
                            THEN ''
                            ELSE public.word_clean_best_sentence.tts_model
                        END,
                        tts_voice = CASE
                            WHEN public.word_clean_best_sentence.source_sentence_id IS DISTINCT FROM EXCLUDED.source_sentence_id
                              OR public.word_clean_best_sentence.sentence IS DISTINCT FROM EXCLUDED.sentence
                              OR public.word_clean_best_sentence.sentence_translation IS DISTINCT FROM EXCLUDED.sentence_translation
                            THEN ''
                            ELSE public.word_clean_best_sentence.tts_voice
                        END,
                        tts_audio_format = CASE
                            WHEN public.word_clean_best_sentence.source_sentence_id IS DISTINCT FROM EXCLUDED.source_sentence_id
                              OR public.word_clean_best_sentence.sentence IS DISTINCT FROM EXCLUDED.sentence
                              OR public.word_clean_best_sentence.sentence_translation IS DISTINCT FROM EXCLUDED.sentence_translation
                            THEN ''
                            ELSE public.word_clean_best_sentence.tts_audio_format
                        END,
                        tts_bucket = CASE
                            WHEN public.word_clean_best_sentence.source_sentence_id IS DISTINCT FROM EXCLUDED.source_sentence_id
                              OR public.word_clean_best_sentence.sentence IS DISTINCT FROM EXCLUDED.sentence
                              OR public.word_clean_best_sentence.sentence_translation IS DISTINCT FROM EXCLUDED.sentence_translation
                            THEN ''
                            ELSE public.word_clean_best_sentence.tts_bucket
                        END,
                        tts_object_key = CASE
                            WHEN public.word_clean_best_sentence.source_sentence_id IS DISTINCT FROM EXCLUDED.source_sentence_id
                              OR public.word_clean_best_sentence.sentence IS DISTINCT FROM EXCLUDED.sentence
                              OR public.word_clean_best_sentence.sentence_translation IS DISTINCT FROM EXCLUDED.sentence_translation
                            THEN ''
                            ELSE public.word_clean_best_sentence.tts_object_key
                        END,
                        tts_object_url = CASE
                            WHEN public.word_clean_best_sentence.source_sentence_id IS DISTINCT FROM EXCLUDED.source_sentence_id
                              OR public.word_clean_best_sentence.sentence IS DISTINCT FROM EXCLUDED.sentence
                              OR public.word_clean_best_sentence.sentence_translation IS DISTINCT FROM EXCLUDED.sentence_translation
                            THEN ''
                            ELSE public.word_clean_best_sentence.tts_object_url
                        END,
                        tts_content_type = CASE
                            WHEN public.word_clean_best_sentence.source_sentence_id IS DISTINCT FROM EXCLUDED.source_sentence_id
                              OR public.word_clean_best_sentence.sentence IS DISTINCT FROM EXCLUDED.sentence
                              OR public.word_clean_best_sentence.sentence_translation IS DISTINCT FROM EXCLUDED.sentence_translation
                            THEN ''
                            ELSE public.word_clean_best_sentence.tts_content_type
                        END,
                        tts_file_size = CASE
                            WHEN public.word_clean_best_sentence.source_sentence_id IS DISTINCT FROM EXCLUDED.source_sentence_id
                              OR public.word_clean_best_sentence.sentence IS DISTINCT FROM EXCLUDED.sentence
                              OR public.word_clean_best_sentence.sentence_translation IS DISTINCT FROM EXCLUDED.sentence_translation
                            THEN NULL
                            ELSE public.word_clean_best_sentence.tts_file_size
                        END,
                        tts_duration_ms = CASE
                            WHEN public.word_clean_best_sentence.source_sentence_id IS DISTINCT FROM EXCLUDED.source_sentence_id
                              OR public.word_clean_best_sentence.sentence IS DISTINCT FROM EXCLUDED.sentence
                              OR public.word_clean_best_sentence.sentence_translation IS DISTINCT FROM EXCLUDED.sentence_translation
                            THEN NULL
                            ELSE public.word_clean_best_sentence.tts_duration_ms
                        END,
                        tts_generated_at = CASE
                            WHEN public.word_clean_best_sentence.source_sentence_id IS DISTINCT FROM EXCLUDED.source_sentence_id
                              OR public.word_clean_best_sentence.sentence IS DISTINCT FROM EXCLUDED.sentence
                              OR public.word_clean_best_sentence.sentence_translation IS DISTINCT FROM EXCLUDED.sentence_translation
                            THEN NULL
                            ELSE public.word_clean_best_sentence.tts_generated_at
                        END,
                        tts_error_message = CASE
                            WHEN public.word_clean_best_sentence.source_sentence_id IS DISTINCT FROM EXCLUDED.source_sentence_id
                              OR public.word_clean_best_sentence.sentence IS DISTINCT FROM EXCLUDED.sentence
                              OR public.word_clean_best_sentence.sentence_translation IS DISTINCT FROM EXCLUDED.sentence_translation
                            THEN ''
                            ELSE public.word_clean_best_sentence.tts_error_message
                        END,
                        updated_at = now()
                    RETURNING id,
                              word_clean_id,
                              word,
                              meaning,
                              source_sentence_id,
                              source_model_name,
                              sentence,
                              sentence_translation,
                              score,
                              score_reason,
                              score_model_name,
                              scored_at,
                              tts_status,
                              tts_bucket,
                              tts_object_key,
                              tts_object_url
                    """,
                    {
                        "word_clean_id": row["word_clean_id"],
                        "word": row["word"],
                        "meaning": row["meaning"],
                        "source_sentence_id": row["source_sentence_id"],
                        "source_model_name": row["source_model_name"],
                        "sentence": row["sentence"],
                        "sentence_translation": row["sentence_translation"],
                        "cloze_sentence": cloze.cloze_sentence if cloze else "",
                        "cloze_answer": cloze.answer if cloze else "",
                        "score": row["score"],
                        "score_reason": row["score_reason"],
                        "score_model_name": row["score_model_name"],
                        "scored_at": row["scored_at"],
                    },
                ).fetchone()
                if upserted_row is not None:
                    best_items.append(self._best_sentence_item_from_row(upserted_row))

        return best_items

    def _best_sentence_item_from_row(self, row: dict[str, Any]) -> WordCleanBestSentenceItem:
        return WordCleanBestSentenceItem(
            id=int(row["id"]),
            word_clean_id=int(row["word_clean_id"]),
            word=str(row["word"]),
            meaning=str(row["meaning"]),
            source_sentence_id=int(row["source_sentence_id"]),
            source_model_name=str(row["source_model_name"]),
            sentence=str(row["sentence"]),
            sentence_translation=str(row["sentence_translation"]),
            score=int(row["score"]),
            score_reason=str(row["score_reason"]),
            score_model_name=str(row["score_model_name"]),
            scored_at=row["scored_at"],
            tts_status=str(row["tts_status"]),
            tts_bucket=str(row["tts_bucket"]),
            tts_object_key=str(row["tts_object_key"]),
            tts_object_url=str(row["tts_object_url"]),
        )
