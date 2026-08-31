import asyncio
import base64
import threading
import time
from pathlib import Path
from typing import Any

from word_agent.core.config import Settings
from word_agent.domain.schemas import TTSGenerationRequest
from word_agent.services import mimo_tts
from word_agent.services.mimo_tts import MiMoTTSService
from word_agent.services.tts_config import ActiveTTSConfig


class SequencedConfigLoader:
    def __init__(self, configs: list[ActiveTTSConfig]) -> None:
        self.configs = configs
        self.calls = 0

    def load_active_mimo_config(self) -> ActiveTTSConfig:
        config = self.configs[self.calls]
        self.calls += 1
        return config


class BlockingConfigLoader:
    def __init__(
        self,
        config: ActiveTTSConfig,
        started: threading.Event,
        release: threading.Event,
    ) -> None:
        self.config = config
        self.started = started
        self.release = release

    def load_active_mimo_config(self) -> ActiveTTSConfig:
        self.started.set()
        self.release.wait(timeout=0.4)
        return self.config


class FakeResponse:
    def raise_for_status(self) -> None:
        return None

    def json(self) -> dict[str, Any]:
        return {
            "choices": [
                {
                    "message": {
                        "audio": {"data": base64.b64encode(b"RIFF").decode("ascii")}
                    }
                }
            ]
        }


class FakeAsyncClient:
    def __init__(self, calls: list[dict[str, Any]], **kwargs: Any) -> None:
        self.calls = calls
        self.kwargs = kwargs

    async def __aenter__(self) -> "FakeAsyncClient":
        return self

    async def __aexit__(self, exc_type, exc, traceback) -> None:
        return None

    async def post(
        self,
        url: str,
        *,
        json: dict[str, Any],
        headers: dict[str, str],
    ) -> FakeResponse:
        self.calls.append({"url": url, "json": json, "headers": headers, "client": self.kwargs})
        return FakeResponse()


def test_generate_reloads_database_configuration_and_ignores_legacy_overrides(
    monkeypatch,
    tmp_path: Path,
) -> None:
    first = ActiveTTSConfig(
        provider_id="first",
        base_url="https://first.example.com/v1",
        api_key="first-secret",
        model="first-model",
        voice="FirstVoice",
    )
    second = ActiveTTSConfig(
        provider_id="second",
        base_url="https://second.example.com/v1",
        api_key="second-secret",
        model="second-model",
        voice="SecondVoice",
    )
    loader = SequencedConfigLoader([first, second])
    calls: list[dict[str, Any]] = []
    monkeypatch.setattr(
        mimo_tts.httpx,
        "AsyncClient",
        lambda **kwargs: FakeAsyncClient(calls, **kwargs),
    )
    settings = Settings(
        MIMO_API_KEY="environment-secret",
        mimo_tts_base_url="https://environment.example.com/v1",
        mimo_tts_default_model="environment-model",
        mimo_tts_default_voice="EnvironmentVoice",
        tts_output_dir=tmp_path,
    )
    service = MiMoTTSService(settings, config_loader=loader)

    first_result = asyncio.run(
        service.generate(
            TTSGenerationRequest.model_validate(
                {
                    "text": "hello",
                    "model": "request-model",
                    "voice": "RequestVoice",
                    "fileName": "first.wav",
                }
            )
        )
    )
    second_result = asyncio.run(
        service.generate(TTSGenerationRequest(text="world", file_name="second.wav"))
    )

    assert loader.calls == 2
    assert first_result.model == "first-model"
    assert first_result.voice == "FirstVoice"
    assert second_result.model == "second-model"
    assert second_result.voice == "SecondVoice"
    assert [call["url"] for call in calls] == [
        "https://first.example.com/v1/chat/completions",
        "https://second.example.com/v1/chat/completions",
    ]
    assert calls[0]["headers"]["api-key"] == "first-secret"
    assert calls[1]["headers"]["api-key"] == "second-secret"
    assert calls[0]["json"]["model"] == "first-model"
    assert calls[0]["json"]["audio"]["voice"] == "FirstVoice"
    assert first_result.file_path.read_bytes() == b"RIFF"
    assert second_result.file_path.read_bytes() == b"RIFF"


def test_provider_settings_and_request_overrides_are_not_runtime_fields() -> None:
    settings = Settings(
        MIMO_API_KEY="environment-secret",
        mimo_tts_base_url="https://environment.example.com/v1",
        mimo_tts_default_model="environment-model",
        mimo_tts_default_voice="EnvironmentVoice",
    )
    request = TTSGenerationRequest.model_validate(
        {"text": "apple", "model": "request-model", "voice": "RequestVoice"}
    )

    for field in (
        "mimo_api_key",
        "mimo_tts_base_url",
        "mimo_tts_default_model",
        "mimo_tts_default_voice",
    ):
        assert not hasattr(settings, field)
    assert not hasattr(request, "model")
    assert not hasattr(request, "voice")


def test_generate_loads_database_config_without_blocking_event_loop(
    monkeypatch,
    tmp_path: Path,
) -> None:
    config = ActiveTTSConfig(
        provider_id="xiaomi-mimo-tts",
        base_url="https://api.xiaomimimo.com/v1",
        api_key="stored-secret",
        model="mimo-v2.5-tts",
        voice="Chloe",
    )
    started = threading.Event()
    release = threading.Event()
    calls: list[dict[str, Any]] = []
    monkeypatch.setattr(
        mimo_tts.httpx,
        "AsyncClient",
        lambda **kwargs: FakeAsyncClient(calls, **kwargs),
    )
    service = MiMoTTSService(
        Settings(tts_output_dir=tmp_path),
        config_loader=BlockingConfigLoader(config, started, release),
    )

    async def exercise() -> float:
        before = time.monotonic()
        task = asyncio.create_task(
            service.generate(TTSGenerationRequest(text="hello", file_name="event-loop.wav"))
        )
        await asyncio.sleep(0.02)
        event_loop_delay = time.monotonic() - before
        release.set()
        await task
        return event_loop_delay

    delay = asyncio.run(exercise())

    assert started.is_set()
    assert delay < 0.15
