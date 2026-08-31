import importlib.util
import json
from pathlib import Path
from types import ModuleType
from typing import Any

ROOT = Path(__file__).resolve().parents[3]
SCRIPT_PATH = ROOT / "scripts" / "mimo_tts_word.py"


def load_script() -> ModuleType:
    spec = importlib.util.spec_from_file_location("mimo_tts_word_script", SCRIPT_PATH)
    if spec is None or spec.loader is None:
        raise RuntimeError("failed to load MiMo TTS example script")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class FakeResponse:
    def __init__(self, body: bytes) -> None:
        self._body = body

    def __enter__(self) -> "FakeResponse":
        return self

    def __exit__(self, exc_type, exc, traceback) -> None:
        return None

    def read(self) -> bytes:
        return self._body


def test_script_posts_to_word_agent_and_downloads_returned_audio(monkeypatch) -> None:
    module = load_script()
    calls: list[dict[str, Any]] = []

    def fake_urlopen(request, timeout: int) -> FakeResponse:
        calls.append(
            {
                "url": request.full_url,
                "method": request.get_method(),
                "headers": dict(request.header_items()),
                "data": request.data,
                "timeout": timeout,
            }
        )
        if len(calls) == 1:
            return FakeResponse(
                json.dumps({"downloadUrl": "/v1/tts/files/apple.wav"}).encode("utf-8")
            )
        return FakeResponse(b"RIFF")

    monkeypatch.setattr(module.urllib.request, "urlopen", fake_urlopen)

    audio = module.generate_tts_audio(
        word="apple",
        word_agent_url="http://127.0.0.1:6017",
        style="Speak clearly.",
        file_name="apple.wav",
        timeout=30,
    )

    assert audio == b"RIFF"
    assert calls[0]["url"] == "http://127.0.0.1:6017/v1/tts/generate"
    assert calls[0]["method"] == "POST"
    payload = json.loads(calls[0]["data"].decode("utf-8"))
    assert payload == {
        "text": "apple",
        "style": "Speak clearly.",
        "fileName": "apple.wav",
        "format": "wav",
    }
    assert calls[1]["url"] == "http://127.0.0.1:6017/v1/tts/files/apple.wav"
    assert calls[1]["method"] == "GET"


def test_script_source_contains_no_xiaomi_credentials_or_provider_overrides() -> None:
    source = SCRIPT_PATH.read_text(encoding="utf-8")

    assert "MIMO_API_KEY" not in source
    assert "xiaomimimo.com" not in source
    assert '"model"' not in source
    assert '"voice"' not in source
    assert "--model" not in source
    assert "--voice" not in source
