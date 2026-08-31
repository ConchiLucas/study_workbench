#!/usr/bin/env python3
"""Add and backfill cloze fields for every best sentence."""

from __future__ import annotations

import argparse
import csv
from collections import Counter
from dataclasses import dataclass
from pathlib import Path

import psycopg
from psycopg.rows import dict_row

from word_agent.core.config import get_settings
from word_agent.services.best_sentence_cloze import BLANK, build_best_sentence_cloze

REPO_ROOT = Path(__file__).resolve().parents[3]
DEFAULT_REPORT = REPO_ROOT / "outputs" / "word_clean_best_sentence_cloze_unmatched.csv"


@dataclass(frozen=True)
class BackfillItem:
    best_id: int
    word: str
    sentence: str
    cloze_sentence: str
    cloze_answer: str
    match_kind: str
    occurrences: int


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--batch-size", type=int, default=500)
    parser.add_argument("--report", type=Path, default=DEFAULT_REPORT)
    return parser.parse_args()


def rob_word_dsn() -> str:
    settings = get_settings()
    return settings.rob_word_db_dsn or (
        "host=127.0.0.1 port=5432 dbname=rob_english_word "
        "user=conchi password=conchi123456"
    )


def fetch_rows(conn: psycopg.Connection) -> list[dict[str, object]]:
    return conn.execute(
        """
        SELECT id, word, sentence
        FROM public.word_clean_best_sentence
        ORDER BY id
        """
    ).fetchall()


def build_items(
    rows: list[dict[str, object]],
) -> tuple[list[BackfillItem], list[dict[str, object]]]:
    items: list[BackfillItem] = []
    unmatched: list[dict[str, object]] = []
    for row in rows:
        best_id = int(row["id"])
        word = str(row["word"])
        sentence = str(row["sentence"])
        result = build_best_sentence_cloze(word, sentence)
        if result is None:
            unmatched.append({"id": best_id, "word": word, "sentence": sentence})
            continue
        restored = result.cloze_sentence.replace(BLANK, result.answer)
        if restored.casefold() != sentence.casefold():
            unmatched.append({"id": best_id, "word": word, "sentence": sentence})
            continue
        items.append(
            BackfillItem(
                best_id=best_id,
                word=word,
                sentence=sentence,
                cloze_sentence=result.cloze_sentence,
                cloze_answer=result.answer,
                match_kind=result.match_kind,
                occurrences=result.occurrences,
            )
        )
    return items, unmatched


def write_unmatched_report(path: Path, rows: list[dict[str, object]]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8-sig", newline="") as stream:
        writer = csv.DictWriter(stream, fieldnames=["id", "word", "sentence"])
        writer.writeheader()
        writer.writerows(rows)


def ensure_columns(conn: psycopg.Connection) -> None:
    conn.execute(
        """
        ALTER TABLE public.word_clean_best_sentence
        ADD COLUMN IF NOT EXISTS cloze_sentence text NOT NULL DEFAULT ''
        """
    )
    conn.execute(
        """
        ALTER TABLE public.word_clean_best_sentence
        ADD COLUMN IF NOT EXISTS cloze_answer text NOT NULL DEFAULT ''
        """
    )
    conn.execute(
        """
        COMMENT ON COLUMN public.word_clean_best_sentence.cloze_sentence
        IS '将例句中的目标词形替换为 ____ 后的挖空句'
        """
    )
    conn.execute(
        """
        COMMENT ON COLUMN public.word_clean_best_sentence.cloze_answer
        IS '例句中实际被挖掉的词形，保留大小写和屈折变化'
        """
    )


def update_batch(conn: psycopg.Connection, items: list[BackfillItem]) -> int:
    values_sql = ", ".join(["(%s, %s, %s, %s, %s)"] * len(items))
    params: list[object] = []
    for item in items:
        params.extend(
            [
                item.best_id,
                item.word,
                item.sentence,
                item.cloze_sentence,
                item.cloze_answer,
            ]
        )
    return conn.execute(
        f"""
        UPDATE public.word_clean_best_sentence AS target
        SET cloze_sentence = source.cloze_sentence,
            cloze_answer = source.cloze_answer,
            updated_at = now()
        FROM (VALUES {values_sql})
             AS source(id, word, sentence, cloze_sentence, cloze_answer)
        WHERE target.id = source.id
          AND target.word = source.word
          AND target.sentence = source.sentence
        """,
        params,
    ).rowcount


def verify_database(conn: psycopg.Connection, expected_count: int) -> dict[str, int]:
    row = conn.execute(
        """
        SELECT count(*)::bigint AS total,
               count(*) FILTER (
                   WHERE cloze_sentence = '' OR cloze_answer = ''
               )::bigint AS empty_fields,
               count(*) FILTER (
                   WHERE position('____' in cloze_sentence) = 0
               )::bigint AS missing_blank
        FROM public.word_clean_best_sentence
        """
    ).fetchone()
    result = {key: int(value) for key, value in row.items()}
    if result != {"total": expected_count, "empty_fields": 0, "missing_blank": 0}:
        raise RuntimeError(f"Database verification failed: {result}")
    return result


def main() -> int:
    args = parse_args()
    if args.batch_size <= 0:
        raise ValueError("--batch-size must be positive")

    with psycopg.connect(rob_word_dsn(), row_factory=dict_row) as conn:
        rows = fetch_rows(conn)
        items, unmatched = build_items(rows)
        kinds = Counter(item.match_kind for item in items)
        multiple = sum(item.occurrences > 1 for item in items)
        print(
            f"total={len(rows)} matched={len(items)} unmatched={len(unmatched)} "
            f"exact={kinds['exact']} inflected={kinds['inflected']} "
            f"phrase_anchor={kinds['phrase_anchor']} multiple={multiple}",
            flush=True,
        )
        if unmatched:
            write_unmatched_report(args.report.resolve(), unmatched)
            print(f"unmatched_report={args.report.resolve()}", flush=True)
            return 1
        if args.dry_run:
            return 0

        with conn.transaction():
            ensure_columns(conn)
            updated = 0
            for offset in range(0, len(items), args.batch_size):
                updated += update_batch(conn, items[offset : offset + args.batch_size])
            if updated != len(items):
                raise RuntimeError(
                    f"Concurrent data change detected: expected={len(items)} updated={updated}"
                )
            verification = verify_database(conn, len(items))
        print(f"updated={updated} verification={verification}", flush=True)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
