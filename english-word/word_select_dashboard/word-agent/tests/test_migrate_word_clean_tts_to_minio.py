import hashlib
import importlib.util
import sys
from contextlib import nullcontext
from pathlib import Path
from types import SimpleNamespace

import pytest

SCRIPT_PATH = (
    Path(__file__).resolve().parents[1]
    / "scripts"
    / "migrate_word_clean_tts_to_minio.py"
)


def test_base_word_tts_migration_script_exists() -> None:
    assert SCRIPT_PATH.is_file()


@pytest.fixture(scope="module")
def migration():
    spec = importlib.util.spec_from_file_location("base_word_tts_migration", SCRIPT_PATH)
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


def write_wav(path: Path) -> None:
    path.write_bytes(b"RIFF\x04\x00\x00\x00WAVEdata")


def make_row(*, key: str = "word_clean_1_tts_1.wav", size: int = 16):
    return (
        1,
        9,
        "word",
        key,
        f"http://127.0.0.1:19186/api/tts/files/{Path(key).name}",
        size,
    )


def test_build_item_accepts_valid_wav(migration, tmp_path: Path) -> None:
    source = tmp_path / "word_clean_1_tts_1.wav"
    write_wav(source)

    item = migration.build_item(
        row=make_row(),
        task_center_dir=tmp_path,
        bucket="ai-file-navigation",
    )

    assert item.tts_id == 1
    assert item.word_clean_id == 9
    assert item.word == "word"
    assert item.source_path == source
    assert item.file_size == 16
    assert item.object_key == "word_clean_tts/word_clean_1_tts_1.wav"
    assert item.object_url == "/ai-file-navigation/word_clean_tts/word_clean_1_tts_1.wav"


@pytest.mark.parametrize(
    ("row", "message"),
    [
        (make_row(key="../escape.wav"), "basename"),
        (make_row(size=99), "size mismatch"),
    ],
)
def test_build_item_rejects_invalid_metadata(
    migration,
    tmp_path: Path,
    row,
    message: str,
) -> None:
    write_wav(tmp_path / Path(row[3]).name)

    with pytest.raises(RuntimeError, match=message):
        migration.build_item(
            row=row,
            task_center_dir=tmp_path,
            bucket="ai-file-navigation",
        )


def test_build_item_rejects_missing_file(migration, tmp_path: Path) -> None:
    with pytest.raises(RuntimeError, match="does not exist"):
        migration.build_item(
            row=make_row(),
            task_center_dir=tmp_path,
            bucket="ai-file-navigation",
        )


def test_build_item_rejects_non_wav(migration, tmp_path: Path) -> None:
    source = tmp_path / "word_clean_1_tts_1.wav"
    source.write_bytes(b"not a wave file")

    with pytest.raises(RuntimeError, match="not a WAV"):
        migration.build_item(
            row=make_row(size=15),
            task_center_dir=tmp_path,
            bucket="ai-file-navigation",
        )


class FakeMinio:
    def __init__(self, objects: dict[str, bytes] | None = None) -> None:
        self.objects = dict(objects or {})
        self.uploads: list[str] = []

    def stat_object(self, bucket: str, key: str):
        assert bucket == "ai-file-navigation"
        if key not in self.objects:
            raise RuntimeError("not found")
        data = self.objects[key]
        return SimpleNamespace(
            size=len(data),
            etag=hashlib.md5(data, usedforsecurity=False).hexdigest(),
            content_type="audio/wav",
        )

    def fput_object(self, bucket: str, key: str, file_path: str, *, content_type: str):
        assert bucket == "ai-file-navigation"
        assert content_type == "audio/wav"
        self.objects[key] = Path(file_path).read_bytes()
        self.uploads.append(key)


@pytest.fixture
def valid_item(migration, tmp_path: Path):
    source = tmp_path / "word_clean_1_tts_1.wav"
    write_wav(source)
    return migration.build_item(
        row=make_row(),
        task_center_dir=tmp_path,
        bucket="ai-file-navigation",
    )


def test_upload_reuses_matching_minio_object(migration, valid_item) -> None:
    client = FakeMinio({valid_item.object_key: valid_item.source_path.read_bytes()})

    migration.upload_and_verify(client, "ai-file-navigation", valid_item)

    assert client.uploads == []


def test_upload_replaces_mismatched_minio_object(migration, valid_item) -> None:
    client = FakeMinio({valid_item.object_key: b"wrong"})

    migration.upload_and_verify(client, "ai-file-navigation", valid_item)

    assert client.uploads == [valid_item.object_key]
    assert client.objects[valid_item.object_key] == valid_item.source_path.read_bytes()


class FakeResult:
    def __init__(self, rowcount: int) -> None:
        self.rowcount = rowcount


class FakeConnection:
    def __init__(self, rowcount: int) -> None:
        self.rowcount = rowcount
        self.calls: list[tuple[str, dict]] = []

    def transaction(self):
        return nullcontext()

    def execute(self, sql: str, params: dict):
        self.calls.append((sql, params))
        return FakeResult(self.rowcount)


class FakeRowsResult:
    def __init__(self, rows: list[tuple]) -> None:
        self.rows = rows

    def fetchall(self):
        return self.rows


class FakeReadConnection:
    def __init__(self, rows: list[tuple]) -> None:
        self.rows = rows
        self.calls: list[tuple[str, dict | None]] = []

    def execute(self, sql: str, params: dict | None = None):
        self.calls.append((sql, params))
        return FakeRowsResult(self.rows)


def test_update_database_uses_guards(migration, valid_item) -> None:
    conn = FakeConnection(rowcount=1)

    migration.update_database(conn, bucket="ai-file-navigation", items=[valid_item])

    sql, params = conn.calls[0]
    assert "status = 'success'" in sql
    assert "tts_object_url = %(old_url)s" in sql
    assert "tts_object_key = %(old_key)s" in sql
    assert "file_size = %(file_size)s" in sql
    assert params["tts_id"] == valid_item.tts_id


def test_update_database_requires_one_guarded_row(migration, valid_item) -> None:
    conn = FakeConnection(rowcount=0)

    with pytest.raises(RuntimeError, match="Database guard failed"):
        migration.update_database(conn, bucket="ai-file-navigation", items=[valid_item])


def test_fetch_migration_rows_selects_only_legacy_success_rows(migration) -> None:
    conn = FakeReadConnection(rows=[make_row()])

    rows = migration.fetch_migration_rows(conn, limit=20)

    sql, params = conn.calls[0]
    assert rows == [make_row()]
    assert "status = 'success'" in sql
    assert "tts_object_url LIKE '%%:19186/%%'" in sql
    assert "LIMIT %(limit)s" in sql
    assert params == {"limit": 20}


def test_migrate_items_uploads_and_updates_database(migration, valid_item) -> None:
    conn = FakeConnection(rowcount=1)
    client = FakeMinio()

    completed = migration.migrate_items(
        conn=conn,
        client=client,
        bucket="ai-file-navigation",
        items=[valid_item],
        workers=1,
        batch_size=1,
    )

    assert completed == 1
    assert client.uploads == [valid_item.object_key]
    assert len(conn.calls) == 1


def test_delete_local_requires_complete_validation(migration, valid_item) -> None:
    summary = migration.ValidationSummary(expected=1, verified=0, failures=("missing",))

    with pytest.raises(RuntimeError, match="refusing local deletion"):
        migration.delete_verified_sources(
            [valid_item],
            summary,
            task_center_dir=valid_item.source_path.parent,
        )

    assert valid_item.source_path.exists()


def test_delete_local_unlinks_only_verified_items(migration, valid_item) -> None:
    unrelated = valid_item.source_path.parent / "unrelated.wav"
    write_wav(unrelated)
    summary = migration.ValidationSummary(expected=1, verified=1, failures=())

    deleted = migration.delete_verified_sources(
        [valid_item],
        summary,
        task_center_dir=valid_item.source_path.parent,
    )

    assert deleted == 1
    assert not valid_item.source_path.exists()
    assert unrelated.exists()


def test_delete_local_rejects_source_outside_root(migration, valid_item, tmp_path: Path) -> None:
    wrong_root = tmp_path / "wrong-root"
    wrong_root.mkdir()
    summary = migration.ValidationSummary(expected=1, verified=1, failures=())

    with pytest.raises(RuntimeError, match="outside task-center directory"):
        migration.delete_verified_sources(
            [valid_item],
            summary,
            task_center_dir=wrong_root,
        )

    assert valid_item.source_path.exists()


class FakeResponse:
    def __init__(
        self,
        *,
        status_code: int = 200,
        content_type: str = "audio/wav",
        content: bytes = b"RIFF\x04\x00\x00\x00WAVEdata",
    ) -> None:
        self.status_code = status_code
        self.headers = {"content-type": content_type}
        self.content = content


def make_success_row(valid_item, *, bucket: str = "ai-file-navigation"):
    return (
        valid_item.tts_id,
        valid_item.word_clean_id,
        valid_item.word,
        bucket,
        valid_item.object_key,
        valid_item.object_url,
        valid_item.file_size,
    )


def test_validate_success_rows_checks_minio_local_and_proxy(
    migration,
    valid_item,
) -> None:
    client = FakeMinio({valid_item.object_key: valid_item.source_path.read_bytes()})
    requested: list[str] = []

    def proxy_get(url: str):
        requested.append(url)
        return FakeResponse()

    items, summary = migration.validate_success_rows(
        rows=[make_success_row(valid_item)],
        client=client,
        bucket="ai-file-navigation",
        task_center_dir=valid_item.source_path.parent,
        proxy_base_url="http://127.0.0.1:7001",
        sample_size=20,
        proxy_get=proxy_get,
    )

    assert len(items) == 1
    assert items[0].tts_id == valid_item.tts_id
    assert items[0].source_path == valid_item.source_path
    assert items[0].object_key == valid_item.object_key
    assert summary == migration.ValidationSummary(expected=1, verified=1, failures=())
    assert requested == [f"http://127.0.0.1:7001{valid_item.object_url}"]


@pytest.mark.parametrize(
    ("row_change", "response", "message"),
    [
        ({"bucket": ""}, FakeResponse(), "database fields"),
        ({}, FakeResponse(status_code=404), "proxy status"),
        ({}, FakeResponse(content_type="text/html"), "proxy content-type"),
        ({}, FakeResponse(content=b"not-wave-data"), "proxy WAV"),
    ],
)
def test_validate_success_rows_reports_failures(
    migration,
    valid_item,
    row_change: dict,
    response: FakeResponse,
    message: str,
) -> None:
    row = list(make_success_row(valid_item))
    if "bucket" in row_change:
        row[3] = row_change["bucket"]
    client = FakeMinio({valid_item.object_key: valid_item.source_path.read_bytes()})

    _, summary = migration.validate_success_rows(
        rows=[tuple(row)],
        client=client,
        bucket="ai-file-navigation",
        task_center_dir=valid_item.source_path.parent,
        proxy_base_url="http://127.0.0.1:7001",
        sample_size=20,
        proxy_get=lambda _url: response,
    )

    assert summary.verified == 0
    assert any(message in failure for failure in summary.failures)
