import asyncio
import base64
import binascii
import re
from dataclasses import dataclass
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

import httpx

from word_agent.core.config import Settings
from word_agent.domain.schemas import TTSGenerationRequest
from word_agent.services.tts_config import ActiveTTSConfig, TTSConfigLoader


class TTSRequestError(RuntimeError):
    pass


@dataclass(frozen=True)
class TTSGenerationResult:
    text: str
    voice: str
    model: str
    audio_format: str
    file_name: str
    file_path: Path
    byte_size: int
    content_type: str


class MiMoTTSService:
    def __init__(
        self,
        settings: Settings,
        *,
        config_loader: TTSConfigLoader | None = None,
    ) -> None:
        self.settings = settings
        self._config_loader = config_loader or TTSConfigLoader(settings)

    async def generate(self, request: TTSGenerationRequest) -> TTSGenerationResult:
        config = await asyncio.to_thread(self._config_loader.load_active_mimo_config)
        model = config.model
        voice = config.voice
        audio_format = request.audio_format
        payload = self._build_payload(request, model=model, voice=voice)
        audio_bytes = await self._request_audio(config=config, payload=payload)
        file_path = self._save_audio(
            audio_bytes=audio_bytes,
            text=request.text,
            file_name=request.file_name,
            audio_format=audio_format,
            overwrite=request.overwrite,
        )

        return TTSGenerationResult(
            text=request.text,
            voice=voice,
            model=model,
            audio_format=audio_format,
            file_name=file_path.name,
            file_path=file_path,
            byte_size=len(audio_bytes),
            content_type=self.content_type_for(audio_format),
        )

    def get_saved_file(self, file_name: str) -> Path:
        safe_name = self._safe_file_name(file_name, fallback="")
        if not safe_name:
            raise TTSRequestError("文件名不能为空")

        file_path = self._output_dir() / safe_name
        if not file_path.exists() or not file_path.is_file():
            raise FileNotFoundError(safe_name)
        return file_path

    def _build_payload(
        self,
        request: TTSGenerationRequest,
        *,
        model: str,
        voice: str,
    ) -> dict[str, Any]:
        messages: list[dict[str, str]] = []
        if request.style:
            messages.append({"role": "user", "content": request.style})
        messages.append({"role": "assistant", "content": request.text})

        return {
            "model": model,
            "messages": messages,
            "audio": {
                "format": request.audio_format,
                "voice": voice,
            },
        }

    async def _request_audio(
        self,
        *,
        config: ActiveTTSConfig,
        payload: dict[str, Any],
    ) -> bytes:
        url = f"{config.base_url}/chat/completions"
        headers = {
            "Content-Type": "application/json",
            "api-key": config.api_key,
        }
        timeout = httpx.Timeout(self.settings.tts_timeout_seconds)

        try:
            async with httpx.AsyncClient(
                timeout=timeout,
                verify=self.settings.tts_verify_ssl,
                trust_env=False,
            ) as client:
                response = await client.post(url, json=payload, headers=headers)
                response.raise_for_status()
        except httpx.HTTPStatusError as exc:
            detail = exc.response.text[:1000]
            raise TTSRequestError(
                f"MiMo TTS 请求失败: HTTP {exc.response.status_code}, {detail}"
            ) from exc
        except httpx.HTTPError as exc:
            raise TTSRequestError(f"MiMo TTS 请求失败: {exc}") from exc

        try:
            data = response.json()
        except ValueError as exc:
            raise TTSRequestError("MiMo TTS 返回内容不是合法 JSON") from exc

        audio_base64 = self._extract_audio_base64(data)
        try:
            return base64.b64decode(audio_base64, validate=True)
        except binascii.Error as exc:
            raise TTSRequestError("MiMo TTS 返回的音频不是合法 base64") from exc

    def _extract_audio_base64(self, data: dict[str, Any]) -> str:
        try:
            audio_data = data["choices"][0]["message"]["audio"]["data"]
        except (KeyError, IndexError, TypeError) as exc:
            raise TTSRequestError(
                "MiMo TTS 返回内容缺少 choices[0].message.audio.data"
            ) from exc

        if not isinstance(audio_data, str) or not audio_data:
            raise TTSRequestError("MiMo TTS 返回的音频内容为空")
        return audio_data

    def _save_audio(
        self,
        *,
        audio_bytes: bytes,
        text: str,
        file_name: str | None,
        audio_format: str,
        overwrite: bool,
    ) -> Path:
        output_dir = self._output_dir()
        output_dir.mkdir(parents=True, exist_ok=True)

        safe_name = (
            self._safe_file_name(file_name, fallback="")
            if file_name
            else self._make_default_file_name(text, audio_format)
        )
        if not safe_name:
            safe_name = self._make_default_file_name(text, audio_format)
        if not safe_name.lower().endswith(f".{audio_format}"):
            safe_name = f"{safe_name}.{audio_format}"

        output_path = output_dir / safe_name
        if not overwrite:
            output_path = self._next_available_path(output_path)

        output_path.write_bytes(audio_bytes)
        return output_path

    def _output_dir(self) -> Path:
        return self.settings.tts_output_dir.expanduser().resolve()

    def _make_default_file_name(self, text: str, audio_format: str) -> str:
        prefix = self._safe_file_name(text[:48], fallback="tts")
        timestamp = datetime.now(UTC).strftime("%Y%m%d%H%M%S%f")
        return f"{prefix}_{timestamp}.{audio_format}"

    def _safe_file_name(self, value: str | None, *, fallback: str) -> str:
        name = Path(value or "").name
        name = re.sub(r"[^A-Za-z0-9._-]+", "_", name).strip("._-")
        return name or fallback

    def _next_available_path(self, path: Path) -> Path:
        if not path.exists():
            return path

        stem = path.stem
        suffix = path.suffix
        for index in range(1, 10000):
            candidate = path.with_name(f"{stem}_{index}{suffix}")
            if not candidate.exists():
                return candidate
        raise TTSRequestError("无法生成不冲突的音频文件名")

    @staticmethod
    def content_type_for(audio_format: str) -> str:
        if audio_format == "wav":
            return "audio/wav"
        return "application/octet-stream"
