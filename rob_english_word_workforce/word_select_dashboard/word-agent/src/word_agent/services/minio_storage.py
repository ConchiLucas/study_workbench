import re
import uuid
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from minio import Minio

from word_agent.core.config import Settings


class MinIOStorageError(RuntimeError):
    pass


@dataclass(frozen=True)
class MinIOUploadResult:
    bucket: str
    object_key: str
    object_url: str
    byte_size: int
    content_type: str


class MinIOStorage:
    def __init__(self, settings: Settings, *, client: Any | None = None) -> None:
        self.settings = settings
        self.bucket = settings.minio_bucket_name.strip()
        if not self.bucket:
            raise MinIOStorageError("MinIO bucket 不能为空")

        if client is not None:
            self.client = client
            return

        endpoint = settings.minio_endpoint.strip()
        access_key = settings.minio_access_key_id.strip()
        secret_key = settings.minio_secret_access_key.strip()
        if not endpoint or not access_key or not secret_key:
            raise MinIOStorageError("MinIO endpoint、access key 和 secret key 必须完整配置")
        self.client = Minio(
            endpoint,
            access_key=access_key,
            secret_key=secret_key,
            secure=settings.minio_use_ssl,
        )

    def upload_audio(self, file_path: Path, *, content_type: str) -> MinIOUploadResult:
        resolved_path = file_path.expanduser().resolve()
        if not resolved_path.is_file():
            raise MinIOStorageError(f"待上传音频不存在: {resolved_path}")

        object_key = self._unique_object_key(resolved_path)
        local_size = resolved_path.stat().st_size
        try:
            if not self.client.bucket_exists(self.bucket):
                raise MinIOStorageError(f"MinIO bucket 不存在: {self.bucket}")
            self.client.fput_object(
                self.bucket,
                object_key,
                str(resolved_path),
                content_type=content_type,
            )
            remote_size = int(self.client.stat_object(self.bucket, object_key).size)
        except MinIOStorageError:
            raise
        except Exception as exc:
            raise MinIOStorageError(f"上传 TTS 音频到 MinIO 失败: {exc}") from exc

        if remote_size != local_size:
            raise MinIOStorageError(
                f"MinIO object size mismatch: local={local_size}, remote={remote_size}"
            )
        return MinIOUploadResult(
            bucket=self.bucket,
            object_key=object_key,
            object_url=f"/{self.bucket}/{object_key}",
            byte_size=remote_size,
            content_type=content_type,
        )

    def _unique_object_key(self, file_path: Path) -> str:
        safe_stem = re.sub(r"[^A-Za-z0-9._-]+", "_", file_path.stem).strip("._-")
        safe_stem = safe_stem or "sentence"
        suffix = file_path.suffix.lower() or ".wav"
        file_name = f"{safe_stem}_{uuid.uuid4().hex}{suffix}"
        path_parts = [
            self.settings.minio_base_path.strip("/"),
            self.settings.cloze_tts_object_prefix.strip("/"),
            file_name,
        ]
        return "/".join(part for part in path_parts if part)
