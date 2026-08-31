from pathlib import Path
from types import SimpleNamespace

import pytest

from word_agent.core.config import Settings
from word_agent.services.minio_storage import MinIOStorage, MinIOStorageError


class FakeMinIOClient:
    def __init__(self, *, remote_size: int | None = None) -> None:
        self.remote_size = remote_size
        self.uploads: list[tuple[str, str, str, str]] = []

    def bucket_exists(self, bucket: str) -> bool:
        return bucket == "ai-file-navigation"

    def fput_object(
        self,
        bucket: str,
        object_key: str,
        file_path: str,
        *,
        content_type: str,
    ) -> None:
        self.uploads.append((bucket, object_key, file_path, content_type))

    def stat_object(self, bucket: str, object_key: str) -> SimpleNamespace:
        assert bucket == "ai-file-navigation"
        assert object_key.startswith("sentence_cloze_tts/")
        local_size = Path(self.uploads[-1][2]).stat().st_size
        remote_size = self.remote_size if self.remote_size is not None else local_size
        return SimpleNamespace(size=remote_size)


def make_settings() -> Settings:
    return Settings(
        minio_endpoint="127.0.0.1:19100",
        minio_access_key_id="access",
        minio_secret_access_key="secret",
        minio_use_ssl=False,
        minio_bucket_name="ai-file-navigation",
        minio_base_path="",
        cloze_tts_object_prefix="sentence_cloze_tts",
    )


def test_cloze_request_timeout_covers_sentence_and_tts_generation() -> None:
    settings = Settings(_env_file=None)

    assert settings.cloze_request_timeout_seconds >= (
        settings.llm_timeout_seconds + settings.tts_timeout_seconds + 30
    )


def test_upload_audio_uses_cloze_prefix_and_returns_proxy_url(tmp_path) -> None:
    audio_path = tmp_path / "generated sentence.wav"
    audio_path.write_bytes(b"RIFF")
    client = FakeMinIOClient()

    result = MinIOStorage(make_settings(), client=client).upload_audio(
        audio_path,
        content_type="audio/wav",
    )

    assert result.bucket == "ai-file-navigation"
    assert result.object_key.startswith("sentence_cloze_tts/generated_sentence_")
    assert result.object_key.endswith(".wav")
    assert result.object_url == f"/ai-file-navigation/{result.object_key}"
    assert result.byte_size == 4
    assert result.content_type == "audio/wav"
    assert client.uploads == [
        (
            "ai-file-navigation",
            result.object_key,
            str(audio_path),
            "audio/wav",
        )
    ]


def test_upload_audio_rejects_remote_size_mismatch(tmp_path) -> None:
    audio_path = tmp_path / "sentence.wav"
    audio_path.write_bytes(b"RIFF")

    with pytest.raises(MinIOStorageError, match="size mismatch"):
        MinIOStorage(
            make_settings(),
            client=FakeMinIOClient(remote_size=3),
        ).upload_audio(audio_path, content_type="audio/wav")
