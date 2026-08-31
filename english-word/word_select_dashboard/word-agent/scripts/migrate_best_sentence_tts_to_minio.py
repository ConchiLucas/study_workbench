#!/usr/bin/env python3
"""Migrate successful best-sentence TTS files from local disks to MinIO."""

from __future__ import annotations

import argparse
import hashlib
from concurrent.futures import ThreadPoolExecutor, as_completed
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import psycopg
import yaml
from minio import Minio

from word_agent.core.config import get_settings


DEFAULT_TASK_CENTER_DIR = Path(
    "/Users/conchi/workforce/python_workforce/ai-task-center/data/tts_audio"
)


@dataclass(frozen=True)
class MigrationItem:
    best_id: int
    word: str
    source_path: Path
    old_url: str
    file_size: int
    object_key: str
    object_url: str


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--limit", type=int)
    parser.add_argument("--workers", type=int, default=12)
    parser.add_argument("--batch-size", type=int, default=250)
    parser.add_argument("--task-center-dir", type=Path, default=DEFAULT_TASK_CENTER_DIR)
    return parser.parse_args()


def load_minio_config(path: Path) -> dict[str, Any]:
    with path.open("r", encoding="utf-8") as stream:
        config = yaml.safe_load(stream) or {}
    minio_config = config.get("minio") or {}
    required = ["endpoint", "access-key-id", "secret-access-key", "bucket-name"]
    missing = [key for key in required if not minio_config.get(key)]
    if missing:
        raise RuntimeError(f"MinIO configuration is missing: {', '.join(missing)}")
    return minio_config


def rob_word_dsn() -> str:
    settings = get_settings()
    return settings.rob_word_db_dsn or (
        "host=127.0.0.1 port=5432 dbname=rob_english_word "
        "user=conchi password=conchi123456"
    )


def source_path_for(
    *,
    file_name: str,
    object_url: str,
    task_center_dir: Path,
    word_agent_dir: Path,
) -> Path:
    if ":19186/" in object_url:
        return task_center_dir / file_name
    if ":8010/" in object_url:
        return word_agent_dir / file_name
    raise RuntimeError(f"Unsupported local TTS URL: {object_url}")


def fetch_items(
    conn: psycopg.Connection,
    *,
    bucket: str,
    task_center_dir: Path,
    word_agent_dir: Path,
    limit: int | None,
) -> list[MigrationItem]:
    rows = conn.execute(
        """
        SELECT id, word, tts_object_key, tts_object_url, tts_file_size
        FROM public.word_clean_best_sentence
        WHERE tts_status = 'success'
          AND NOT (tts_bucket <> '' AND tts_object_key <> '')
        ORDER BY id
        """
    ).fetchall()
    if limit is not None:
        rows = rows[:limit]

    items: list[MigrationItem] = []
    names: set[str] = set()
    for best_id, word, current_key, old_url, db_size in rows:
        file_name = Path(current_key).name
        if not file_name or file_name in names:
            raise RuntimeError(f"Invalid or duplicate TTS file name: {file_name}")
        names.add(file_name)
        source_path = source_path_for(
            file_name=file_name,
            object_url=old_url,
            task_center_dir=task_center_dir,
            word_agent_dir=word_agent_dir,
        )
        if not source_path.is_file():
            raise RuntimeError(f"TTS source file does not exist: {source_path}")
        file_size = source_path.stat().st_size
        if db_size is not None and file_size != db_size:
            raise RuntimeError(
                f"TTS source size mismatch for best sentence {best_id}: "
                f"database={db_size}, disk={file_size}"
            )
        with source_path.open("rb") as stream:
            header = stream.read(12)
        if not (header.startswith(b"RIFF") and header[8:12] == b"WAVE"):
            raise RuntimeError(f"TTS source is not a WAV file: {source_path}")
        object_key = f"word_clean_tts/{file_name}"
        items.append(
            MigrationItem(
                best_id=int(best_id),
                word=str(word),
                source_path=source_path,
                old_url=str(old_url),
                file_size=file_size,
                object_key=object_key,
                object_url=f"/{bucket}/{object_key}",
            )
        )
    return items


def file_md5(path: Path) -> str:
    digest = hashlib.md5(usedforsecurity=False)
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def upload_and_verify(client: Minio, bucket: str, item: MigrationItem) -> MigrationItem:
    local_md5 = file_md5(item.source_path)
    try:
        existing = client.stat_object(bucket, item.object_key)
    except Exception:
        existing = None
    if existing is None or existing.size != item.file_size or existing.etag != local_md5:
        client.fput_object(
            bucket,
            item.object_key,
            str(item.source_path),
            content_type="audio/wav",
        )
    verified = client.stat_object(bucket, item.object_key)
    if verified.size != item.file_size or verified.etag != local_md5:
        raise RuntimeError(
            f"MinIO verification failed for {item.object_key}: "
            f"expected size={item.file_size}, etag={local_md5}; "
            f"actual size={verified.size}, etag={verified.etag}"
        )
    return item


def update_database(
    conn: psycopg.Connection,
    *,
    bucket: str,
    items: list[MigrationItem],
) -> None:
    with conn.transaction():
        for item in items:
            updated = conn.execute(
                """
                UPDATE public.word_clean_best_sentence
                SET tts_bucket = %(bucket)s,
                    tts_object_key = %(object_key)s,
                    tts_object_url = %(object_url)s,
                    updated_at = now()
                WHERE id = %(best_id)s
                  AND tts_status = 'success'
                  AND tts_object_url = %(old_url)s
                  AND tts_file_size = %(file_size)s
                """,
                {
                    "bucket": bucket,
                    "object_key": item.object_key,
                    "object_url": item.object_url,
                    "best_id": item.best_id,
                    "old_url": item.old_url,
                    "file_size": item.file_size,
                },
            ).rowcount
            if updated != 1:
                raise RuntimeError(
                    f"Database guard failed for best sentence {item.best_id}: updated={updated}"
                )
            conn.execute(
                """
                UPDATE public.word_clean_sentence_tts_job
                SET tts_bucket = %(bucket)s,
                    tts_object_key = %(object_key)s,
                    tts_object_url = %(object_url)s,
                    updated_at = now()
                WHERE best_sentence_id = %(best_id)s
                  AND status = 'success'
                """,
                {
                    "bucket": bucket,
                    "object_key": item.object_key,
                    "object_url": item.object_url,
                    "best_id": item.best_id,
                },
            )


def main() -> int:
    args = parse_args()
    settings = get_settings()
    minio_config = load_minio_config(settings.go_config_path)
    bucket = str(minio_config["bucket-name"])
    client = Minio(
        str(minio_config["endpoint"]),
        access_key=str(minio_config["access-key-id"]),
        secret_key=str(minio_config["secret-access-key"]),
        secure=bool(minio_config.get("use-ssl", False)),
    )
    if not client.bucket_exists(bucket):
        raise RuntimeError(f"MinIO bucket does not exist: {bucket}")

    with psycopg.connect(rob_word_dsn()) as conn:
        items = fetch_items(
            conn,
            bucket=bucket,
            task_center_dir=args.task_center_dir.resolve(),
            word_agent_dir=settings.tts_output_dir.resolve(),
            limit=args.limit,
        )
        total_bytes = sum(item.file_size for item in items)
        print(f"migration_items={len(items)} bytes={total_bytes}", flush=True)
        if args.dry_run or not items:
            return 0

        completed = 0
        pending_updates: list[MigrationItem] = []
        with ThreadPoolExecutor(max_workers=args.workers) as executor:
            futures = {
                executor.submit(upload_and_verify, client, bucket, item): item
                for item in items
            }
            for future in as_completed(futures):
                migrated = future.result()
                pending_updates.append(migrated)
                if len(pending_updates) >= args.batch_size:
                    update_database(conn, bucket=bucket, items=pending_updates)
                    completed += len(pending_updates)
                    pending_updates.clear()
                    print(f"migrated={completed}/{len(items)}", flush=True)
            if pending_updates:
                update_database(conn, bucket=bucket, items=pending_updates)
                completed += len(pending_updates)
                print(f"migrated={completed}/{len(items)}", flush=True)
        return 0


if __name__ == "__main__":
    raise SystemExit(main())
