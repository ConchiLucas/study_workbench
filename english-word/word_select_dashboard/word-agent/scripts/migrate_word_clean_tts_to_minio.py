#!/usr/bin/env python3
"""Migrate successful base-word TTS files from local disk to MinIO."""

from __future__ import annotations

import argparse
import hashlib
from concurrent.futures import ThreadPoolExecutor, as_completed
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import httpx
import psycopg
from minio import Minio

from word_agent.core.config import get_settings

DEFAULT_TASK_CENTER_DIR = Path(
    "/Users/conchi/workforce/python_workforce/ai-task-center/data/tts_audio"
)


@dataclass(frozen=True)
class MigrationItem:
    tts_id: int
    word_clean_id: int
    word: str
    source_path: Path
    old_key: str
    old_url: str
    file_size: int
    object_key: str
    object_url: str


@dataclass(frozen=True)
class ValidationSummary:
    expected: int
    verified: int
    failures: tuple[str, ...]


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--limit", type=int)
    parser.add_argument("--workers", type=int, default=12)
    parser.add_argument("--batch-size", type=int, default=250)
    parser.add_argument("--task-center-dir", type=Path, default=DEFAULT_TASK_CENTER_DIR)
    parser.add_argument("--verify-proxy-base-url", default="http://127.0.0.1:7001")
    parser.add_argument("--sample-size", type=int, default=20)
    parser.add_argument("--delete-local", action="store_true")
    return parser.parse_args()


def rob_word_dsn() -> str:
    settings = get_settings()
    return settings.rob_word_db_dsn or (
        "host=127.0.0.1 port=5432 dbname=rob_english_word "
        "user=conchi password=conchi123456"
    )


def build_item(
    *,
    row: tuple[int, int, str, str, str, int | None],
    task_center_dir: Path,
    bucket: str,
) -> MigrationItem:
    tts_id, word_clean_id, word, current_key, old_url, db_size = row
    file_name = Path(current_key).name
    if not file_name or file_name != current_key:
        raise RuntimeError(f"TTS object key must be a basename before migration: {current_key}")
    if ":19186/" not in old_url:
        raise RuntimeError(f"Unsupported local TTS URL: {old_url}")

    source_path = task_center_dir / file_name
    if not source_path.is_file():
        raise RuntimeError(f"TTS source file does not exist: {source_path}")
    file_size = source_path.stat().st_size
    if db_size is not None and file_size != db_size:
        raise RuntimeError(
            f"TTS source size mismatch for row {tts_id}: database={db_size}, disk={file_size}"
        )
    with source_path.open("rb") as stream:
        header = stream.read(12)
    if not (header.startswith(b"RIFF") and header[8:12] == b"WAVE"):
        raise RuntimeError(f"TTS source is not a WAV file: {source_path}")

    object_key = f"word_clean_tts/{file_name}"
    return MigrationItem(
        tts_id=int(tts_id),
        word_clean_id=int(word_clean_id),
        word=str(word),
        source_path=source_path,
        old_key=current_key,
        old_url=old_url,
        file_size=file_size,
        object_key=object_key,
        object_url=f"/{bucket}/{object_key}",
    )


def fetch_migration_rows(
    conn: Any,
    *,
    limit: int | None,
) -> list[tuple[int, int, str, str, str, int | None]]:
    query = """
        SELECT id, word_clean_id, word, tts_object_key, tts_object_url, file_size
        FROM public.word_clean_tts
        WHERE status = 'success'
          AND tts_object_url LIKE '%%:19186/%%'
        ORDER BY id
    """
    params = None
    if limit is not None:
        query += " LIMIT %(limit)s"
        params = {"limit": limit}
    return conn.execute(query, params).fetchall()


def fetch_status_counts(conn: Any) -> dict[str, int]:
    return {
        str(status): int(count)
        for status, count in conn.execute(
            """
            SELECT status, count(*)
            FROM public.word_clean_tts
            GROUP BY status
            ORDER BY status
            """
        ).fetchall()
    }


def fetch_success_rows(
    conn: Any,
) -> list[tuple[int, int, str, str, str, str, int | None]]:
    return conn.execute(
        """
        SELECT id, word_clean_id, word, tts_bucket, tts_object_key,
               tts_object_url, file_size
        FROM public.word_clean_tts
        WHERE status = 'success'
        ORDER BY id
        """
    ).fetchall()


def file_md5(path: Path) -> str:
    digest = hashlib.md5(usedforsecurity=False)
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def upload_and_verify(client: Any, bucket: str, item: MigrationItem) -> MigrationItem:
    local_md5 = file_md5(item.source_path)
    try:
        existing = client.stat_object(bucket, item.object_key)
    except Exception:
        existing = None
    existing_etag = str(getattr(existing, "etag", "")).strip('"') if existing else ""
    if existing is None or existing.size != item.file_size or existing_etag != local_md5:
        client.fput_object(
            bucket,
            item.object_key,
            str(item.source_path),
            content_type="audio/wav",
        )
    verified = client.stat_object(bucket, item.object_key)
    verified_etag = str(getattr(verified, "etag", "")).strip('"')
    if verified.size != item.file_size or verified_etag != local_md5:
        raise RuntimeError(
            f"MinIO verification failed for {item.object_key}: "
            f"expected size={item.file_size}, etag={local_md5}; "
            f"actual size={verified.size}, etag={verified_etag}"
        )
    return item


def update_database(
    conn: Any,
    *,
    bucket: str,
    items: list[MigrationItem],
) -> None:
    with conn.transaction():
        for item in items:
            updated = conn.execute(
                """
                UPDATE public.word_clean_tts
                SET tts_bucket = %(bucket)s,
                    tts_object_key = %(object_key)s,
                    tts_object_url = %(object_url)s,
                    updated_at = now()
                WHERE id = %(tts_id)s
                  AND status = 'success'
                  AND tts_object_key = %(old_key)s
                  AND tts_object_url = %(old_url)s
                  AND file_size = %(file_size)s
                """,
                {
                    "bucket": bucket,
                    "object_key": item.object_key,
                    "object_url": item.object_url,
                    "tts_id": item.tts_id,
                    "old_key": item.old_key,
                    "old_url": item.old_url,
                    "file_size": item.file_size,
                },
            ).rowcount
            if updated != 1:
                raise RuntimeError(
                    f"Database guard failed for word TTS {item.tts_id}: updated={updated}"
                )


def migrate_items(
    *,
    conn: Any,
    client: Any,
    bucket: str,
    items: list[MigrationItem],
    workers: int,
    batch_size: int,
) -> int:
    if workers < 1 or batch_size < 1:
        raise ValueError("workers and batch-size must be positive")
    completed = 0
    pending_updates: list[MigrationItem] = []
    with ThreadPoolExecutor(max_workers=workers) as executor:
        futures = {
            executor.submit(upload_and_verify, client, bucket, item): item for item in items
        }
        for future in as_completed(futures):
            pending_updates.append(future.result())
            if len(pending_updates) >= batch_size:
                update_database(conn, bucket=bucket, items=pending_updates)
                completed += len(pending_updates)
                pending_updates.clear()
                print(f"migrated={completed}/{len(items)}", flush=True)
        if pending_updates:
            update_database(conn, bucket=bucket, items=pending_updates)
            completed += len(pending_updates)
            print(f"migrated={completed}/{len(items)}", flush=True)
    return completed


def _sample_indexes(total: int, sample_size: int) -> list[int]:
    if total <= 0 or sample_size <= 0:
        return []
    if total <= sample_size:
        return list(range(total))
    if sample_size == 1:
        return [total // 2]
    return sorted({round(index * (total - 1) / (sample_size - 1)) for index in range(sample_size)})


def validate_success_rows(
    *,
    rows: list[tuple[int, int, str, str, str, str, int | None]],
    client: Any,
    bucket: str,
    task_center_dir: Path,
    proxy_base_url: str,
    sample_size: int,
    proxy_get: Any,
) -> tuple[list[MigrationItem], ValidationSummary]:
    failures: list[str] = []
    failed_ids: set[int] = set()
    items: list[MigrationItem] = []
    valid_for_proxy: list[MigrationItem] = []
    root = task_center_dir.resolve()

    for tts_id, word_clean_id, word, db_bucket, object_key, object_url, db_size in rows:
        tts_id = int(tts_id)
        file_name = Path(object_key).name
        expected_key = f"word_clean_tts/{file_name}" if file_name else ""
        expected_url = f"/{bucket}/{expected_key}" if expected_key else ""
        source_path = root / file_name if file_name else root
        size = int(db_size) if db_size is not None else -1
        item = MigrationItem(
            tts_id=tts_id,
            word_clean_id=int(word_clean_id),
            word=str(word),
            source_path=source_path,
            old_key=file_name,
            old_url=str(object_url),
            file_size=size,
            object_key=str(object_key),
            object_url=str(object_url),
        )
        items.append(item)

        if (
            not file_name
            or db_bucket != bucket
            or object_key != expected_key
            or object_url != expected_url
            or ":19186/" in object_url
            or size < 0
        ):
            failures.append(f"word TTS {tts_id}: database fields do not match MinIO target")
            failed_ids.add(tts_id)
            continue
        try:
            remote = client.stat_object(bucket, object_key)
        except Exception as exc:
            failures.append(f"word TTS {tts_id}: MinIO object missing: {exc}")
            failed_ids.add(tts_id)
            continue
        if remote.size != size:
            failures.append(
                f"word TTS {tts_id}: MinIO size mismatch database={size}, minio={remote.size}"
            )
            failed_ids.add(tts_id)
            continue
        if not source_path.is_file():
            failures.append(f"word TTS {tts_id}: local source missing: {source_path}")
            failed_ids.add(tts_id)
            continue
        if source_path.stat().st_size != size:
            failures.append(
                f"word TTS {tts_id}: local size mismatch database={size}, "
                f"disk={source_path.stat().st_size}"
            )
            failed_ids.add(tts_id)
            continue
        valid_for_proxy.append(item)

    for index in _sample_indexes(len(valid_for_proxy), sample_size):
        item = valid_for_proxy[index]
        url = f"{proxy_base_url.rstrip('/')}{item.object_url}"
        try:
            response = proxy_get(url)
        except Exception as exc:
            failures.append(f"word TTS {item.tts_id}: proxy request failed: {exc}")
            failed_ids.add(item.tts_id)
            continue
        if response.status_code != 200:
            failures.append(
                f"word TTS {item.tts_id}: proxy status is {response.status_code}, expected 200"
            )
            failed_ids.add(item.tts_id)
            continue
        content_type = response.headers.get("content-type", "").lower()
        if not content_type.startswith("audio/"):
            failures.append(
                f"word TTS {item.tts_id}: proxy content-type is {content_type or 'missing'}"
            )
            failed_ids.add(item.tts_id)
            continue
        header = response.content[:12]
        if not (header.startswith(b"RIFF") and header[8:12] == b"WAVE"):
            failures.append(f"word TTS {item.tts_id}: proxy WAV header is invalid")
            failed_ids.add(item.tts_id)

    return items, ValidationSummary(
        expected=len(rows),
        verified=len(rows) - len(failed_ids),
        failures=tuple(failures),
    )


def delete_verified_sources(
    items: list[MigrationItem],
    summary: ValidationSummary,
    *,
    task_center_dir: Path,
) -> int:
    if summary.failures or summary.verified != summary.expected or len(items) != summary.expected:
        raise RuntimeError(
            "Full validation did not pass; refusing local deletion: "
            f"expected={summary.expected}, verified={summary.verified}, "
            f"failures={len(summary.failures)}"
        )

    root = task_center_dir.resolve()
    resolved_paths: list[Path] = []
    for item in items:
        resolved = item.source_path.resolve()
        if resolved.parent != root:
            raise RuntimeError(f"TTS source is outside task-center directory: {resolved}")
        if not resolved.is_file():
            raise RuntimeError(f"Verified TTS source disappeared before deletion: {resolved}")
        resolved_paths.append(resolved)

    for source_path in resolved_paths:
        source_path.unlink()
    return len(resolved_paths)


def build_migration_items(
    rows: list[tuple[int, int, str, str, str, int | None]],
    *,
    task_center_dir: Path,
    bucket: str,
) -> list[MigrationItem]:
    items = [
        build_item(
            row=row,
            task_center_dir=task_center_dir,
            bucket=bucket,
        )
        for row in rows
    ]
    names = [item.source_path.name for item in items]
    if len(names) != len(set(names)):
        raise RuntimeError("Duplicate local TTS filenames exist in the migration set")
    return items


def main() -> int:
    args = parse_args()
    if args.limit is not None and args.limit < 1:
        raise RuntimeError("--limit must be positive")
    if args.sample_size < 1:
        raise RuntimeError("--sample-size must be positive")
    if args.delete_local and args.limit is not None:
        raise RuntimeError("--delete-local cannot be combined with --limit")

    settings = get_settings()
    bucket = settings.minio_bucket_name
    if not settings.minio_access_key_id or not settings.minio_secret_access_key:
        raise RuntimeError("MinIO credentials are missing from word-agent settings")
    client = Minio(
        settings.minio_endpoint,
        access_key=settings.minio_access_key_id,
        secret_key=settings.minio_secret_access_key,
        secure=settings.minio_use_ssl,
    )
    if not client.bucket_exists(bucket):
        raise RuntimeError(f"MinIO bucket does not exist: {bucket}")

    task_center_dir = args.task_center_dir.resolve()
    with psycopg.connect(rob_word_dsn(), autocommit=True) as conn:
        counts = fetch_status_counts(conn)
        legacy_rows = fetch_migration_rows(conn, limit=args.limit)
        migration_items = build_migration_items(
            legacy_rows,
            task_center_dir=task_center_dir,
            bucket=bucket,
        )
        total_bytes = sum(item.file_size for item in migration_items)
        print(
            f"success={counts.get('success', 0)} pending={counts.get('pending', 0)} "
            f"migration_items={len(migration_items)} bytes={total_bytes}",
            flush=True,
        )
        if args.dry_run:
            print("dry_run=true mutations=0 deletions=0", flush=True)
            return 0

        migrated = migrate_items(
            conn=conn,
            client=client,
            bucket=bucket,
            items=migration_items,
            workers=args.workers,
            batch_size=args.batch_size,
        )
        if args.limit is not None:
            print(f"limited_run=true migrated={migrated} deletions=0", flush=True)
            return 0

        success_rows = fetch_success_rows(conn)
        with httpx.Client(timeout=30.0, follow_redirects=True) as http_client:
            verified_items, summary = validate_success_rows(
                rows=success_rows,
                client=client,
                bucket=bucket,
                task_center_dir=task_center_dir,
                proxy_base_url=args.verify_proxy_base_url,
                sample_size=args.sample_size,
                proxy_get=http_client.get,
            )
        print(
            f"verified={summary.verified}/{summary.expected} "
            f"validation_failures={len(summary.failures)}",
            flush=True,
        )
        if summary.failures:
            for failure in summary.failures[:20]:
                print(f"validation_failure={failure}", flush=True)
            raise RuntimeError("Full MinIO/database/local/proxy validation failed")
        if summary.expected != counts.get("success", 0):
            raise RuntimeError(
                "Success count changed during migration; refusing local deletion: "
                f"before={counts.get('success', 0)}, after={summary.expected}"
            )

        deleted = 0
        if args.delete_local:
            deleted = delete_verified_sources(
                verified_items,
                summary,
                task_center_dir=task_center_dir,
            )
            remaining = sum(item.source_path.exists() for item in verified_items)
            if remaining:
                raise RuntimeError(f"Local deletion verification failed: remaining={remaining}")
        print(
            f"migration_complete=true migrated={migrated} verified={summary.verified} "
            f"deleted={deleted} pending={counts.get('pending', 0)}",
            flush=True,
        )
        return 0


if __name__ == "__main__":
    raise SystemExit(main())
