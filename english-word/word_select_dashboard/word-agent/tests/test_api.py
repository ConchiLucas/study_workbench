from fastapi.testclient import TestClient

from word_agent import main as app_main
from word_agent.api import routes
from word_agent.domain.schemas import WrongWordEventResponse
from word_agent.main import create_app
from word_agent.services.llm_client import AIProvider, SentenceGenerationResult
from word_agent.services.mimo_tts import TTSGenerationResult, TTSRequestError
from word_agent.services.minio_storage import (
    MinIOStorageError,
    MinIOUploadResult,
)
from word_agent.services.tts_config import TTSConfigError


def test_health() -> None:
    client = TestClient(create_app())

    response = client.get("/health")

    assert response.status_code == 200
    assert response.json()["status"] == "ok"


def test_app_lifespan_starts_and_stops_wrong_word_processor(monkeypatch) -> None:
    lifecycle: list[str] = []

    class FakeStrategy:
        def __init__(self, settings) -> None:
            self.settings = settings

    class FakeProcessor:
        def __init__(self, strategy) -> None:
            self.strategy = strategy

        async def start(self) -> None:
            lifecycle.append("started")

        async def stop(self) -> None:
            lifecycle.append("stopped")

    monkeypatch.setattr(app_main, "WrongWordStrategyService", FakeStrategy)
    monkeypatch.setattr(app_main, "WrongWordEventProcessor", FakeProcessor)

    with TestClient(create_app()) as client:
        assert client.get("/health").status_code == 200
        assert lifecycle == ["started"]

    assert lifecycle == ["started", "stopped"]


def test_execute_run_accepts_go_payload_without_callback() -> None:
    client = TestClient(create_app())

    response = client.post(
        "/v1/runs/execute",
        json={
            "runId": "run-test-001",
            "word": "brisk",
            "meaning": "quick and energetic",
            "metadata": {"source": "test"},
        },
    )

    assert response.status_code == 202
    assert response.json()["runId"] == "run-test-001"
    assert response.json()["status"] == "pending"


def test_generate_sentence_uses_llm_tts_and_minio(monkeypatch, tmp_path) -> None:
    client = TestClient(create_app())
    audio_path = tmp_path / "generated.wav"

    async def fake_generate_sentence_from_words(
        self, *, words: list[str]
    ) -> SentenceGenerationResult:
        _ = self
        return SentenceGenerationResult(
            sentence="A brisk anchor can harbor a quiet plan.",
            translation_zh="一个敏捷的锚可以承载一个安静的计划。",
            explanation_zh=(
                "这个句子表示一个可靠的支撑可以承载一个安静的计划。"
            ),
            provider=AIProvider(
                id="test-provider",
                label="Test Provider",
                type="openai-compatible",
                base_url="https://example.com/v1",
                api_key="secret",
                model="test-model",
                max_tokens=128,
            ),
        )

    monkeypatch.setattr(
        routes.LLMClient,
        "generate_sentence_from_words",
        fake_generate_sentence_from_words,
    )

    class FakeMiMoTTSService:
        def __init__(self, settings) -> None:
            self.settings = settings

        async def generate(self, request) -> TTSGenerationResult:
            assert request.text == "A brisk anchor can harbor a quiet plan."
            audio_path.write_bytes(b"RIFF")
            return TTSGenerationResult(
                text=request.text,
                voice="Chloe",
                model="mimo-v2.5-tts",
                audio_format="wav",
                file_name=audio_path.name,
                file_path=audio_path,
                byte_size=4,
                content_type="audio/wav",
            )

    class FakeMinIOStorage:
        def __init__(self, settings) -> None:
            self.settings = settings

        def upload_audio(self, file_path, *, content_type) -> MinIOUploadResult:
            assert file_path == audio_path
            assert content_type == "audio/wav"
            return MinIOUploadResult(
                bucket="ai-file-navigation",
                object_key="sentence_cloze_tts/example.wav",
                object_url="/ai-file-navigation/sentence_cloze_tts/example.wav",
                byte_size=4,
                content_type="audio/wav",
            )

    monkeypatch.setattr(routes, "MiMoTTSService", FakeMiMoTTSService)
    monkeypatch.setattr(routes, "MinIOStorage", FakeMinIOStorage)

    response = client.post(
        "/v1/sentences/generate",
        json={"words": ["brisk", "anchor", "harbor"]},
    )

    assert response.status_code == 200
    assert response.json() == {
        "sentence": "A brisk anchor can harbor a quiet plan.",
        "translationZh": "一个敏捷的锚可以承载一个安静的计划。",
        "explanationZh": (
            "这个句子表示一个可靠的支撑可以承载一个安静的计划。"
        ),
        "words": ["brisk", "anchor", "harbor"],
        "providerId": "test-provider",
        "providerLabel": "Test Provider",
        "model": "test-model",
        "sentenceAudioUrl": "/ai-file-navigation/sentence_cloze_tts/example.wav",
        "sentenceAudioBucket": "ai-file-navigation",
        "sentenceAudioObjectKey": "sentence_cloze_tts/example.wav",
        "sentenceAudioByteSize": 4,
        "sentenceAudioContentType": "audio/wav",
        "ttsProvider": "xiaomi-mimo",
        "ttsModel": "mimo-v2.5-tts",
        "ttsVoice": "Chloe",
        "ttsFormat": "wav",
    }
    assert not audio_path.exists()


def test_generate_sentence_stops_before_minio_when_tts_fails(monkeypatch) -> None:
    client = TestClient(create_app())
    minio_created = False

    async def fake_generate_sentence_from_words(
        self, *, words: list[str]
    ) -> SentenceGenerationResult:
        _ = self, words
        return SentenceGenerationResult(
            sentence="A brisk anchor can harbor a quiet plan.",
            translation_zh="译文",
            explanation_zh="解释",
            provider=AIProvider(
                id="provider",
                label="Provider",
                type="openai-compatible",
                base_url="https://example.com/v1",
                api_key="secret",
                model="model",
                max_tokens=128,
            ),
        )

    class FailingMiMoTTSService:
        def __init__(self, settings) -> None:
            self.settings = settings

        async def generate(self, request) -> TTSGenerationResult:
            raise TTSRequestError("tts unavailable")

    class UnexpectedMinIOStorage:
        def __init__(self, settings) -> None:
            nonlocal minio_created
            minio_created = True

    monkeypatch.setattr(
        routes.LLMClient,
        "generate_sentence_from_words",
        fake_generate_sentence_from_words,
    )
    monkeypatch.setattr(routes, "MiMoTTSService", FailingMiMoTTSService)
    monkeypatch.setattr(routes, "MinIOStorage", UnexpectedMinIOStorage)

    response = client.post(
        "/v1/sentences/generate",
        json={"words": ["brisk", "anchor", "harbor"]},
    )

    assert response.status_code == 502
    assert "tts unavailable" in response.json()["detail"]
    assert minio_created is False


def test_generate_sentence_maps_tts_config_error_to_500(monkeypatch) -> None:
    client = TestClient(create_app())

    async def fake_generate_sentence_from_words(
        self, *, words: list[str]
    ) -> SentenceGenerationResult:
        _ = self, words
        return SentenceGenerationResult(
            sentence="A brisk anchor can harbor a quiet plan.",
            translation_zh="译文",
            explanation_zh="解释",
            provider=AIProvider(
                id="provider",
                label="Provider",
                type="openai-compatible",
                base_url="https://example.com/v1",
                api_key="secret",
                model="model",
                max_tokens=128,
            ),
        )

    class MissingConfigMiMoTTSService:
        def __init__(self, settings) -> None:
            self.settings = settings

        async def generate(self, request) -> TTSGenerationResult:
            _ = request
            raise TTSConfigError("数据库里没有启用的默认 MiMo TTS 配置")

    monkeypatch.setattr(
        routes.LLMClient,
        "generate_sentence_from_words",
        fake_generate_sentence_from_words,
    )
    monkeypatch.setattr(routes, "MiMoTTSService", MissingConfigMiMoTTSService)

    response = client.post(
        "/v1/sentences/generate",
        json={"words": ["brisk", "anchor", "harbor"]},
    )

    assert response.status_code == 500
    assert "没有启用的默认" in response.json()["detail"]


def test_generate_sentence_cleans_temp_file_when_minio_fails(
    monkeypatch,
    tmp_path,
) -> None:
    client = TestClient(create_app())
    audio_path = tmp_path / "failed-upload.wav"

    async def fake_generate_sentence_from_words(
        self, *, words: list[str]
    ) -> SentenceGenerationResult:
        _ = self, words
        return SentenceGenerationResult(
            sentence="A brisk anchor can harbor a quiet plan.",
            translation_zh="译文",
            explanation_zh="解释",
            provider=AIProvider(
                id="provider",
                label="Provider",
                type="openai-compatible",
                base_url="https://example.com/v1",
                api_key="secret",
                model="model",
                max_tokens=128,
            ),
        )

    class FakeMiMoTTSService:
        def __init__(self, settings) -> None:
            self.settings = settings

        async def generate(self, request) -> TTSGenerationResult:
            audio_path.write_bytes(b"RIFF")
            return TTSGenerationResult(
                text=request.text,
                voice="Chloe",
                model="mimo-v2.5-tts",
                audio_format="wav",
                file_name=audio_path.name,
                file_path=audio_path,
                byte_size=4,
                content_type="audio/wav",
            )

    class FailingMinIOStorage:
        def __init__(self, settings) -> None:
            self.settings = settings

        def upload_audio(self, file_path, *, content_type) -> MinIOUploadResult:
            raise MinIOStorageError("minio unavailable")

    monkeypatch.setattr(
        routes.LLMClient,
        "generate_sentence_from_words",
        fake_generate_sentence_from_words,
    )
    monkeypatch.setattr(routes, "MiMoTTSService", FakeMiMoTTSService)
    monkeypatch.setattr(routes, "MinIOStorage", FailingMinIOStorage)

    response = client.post(
        "/v1/sentences/generate",
        json={"words": ["brisk", "anchor", "harbor"]},
    )

    assert response.status_code == 502
    assert "minio unavailable" in response.json()["detail"]
    assert not audio_path.exists()


def test_receive_wrong_word_event_uses_strategy_service(monkeypatch) -> None:
    _ = monkeypatch
    app = create_app()

    class FakeWrongWordStrategyService:
        def enqueue_event(self, request) -> WrongWordEventResponse:
            assert request.word == "brisk"
            assert request.answer_detail_id == 123
            return WrongWordEventResponse(
                event_id=10,
                pending_count=1,
                generated=False,
            )

    class FakeWrongWordProcessor:
        def __init__(self) -> None:
            self.notifications = 0

        def notify(self) -> None:
            self.notifications += 1

    processor = FakeWrongWordProcessor()
    app.state.wrong_word_strategy = FakeWrongWordStrategyService()
    app.state.wrong_word_processor = processor
    client = TestClient(app)

    response = client.post(
        "/v1/wrong-words/events",
        json={
            "answerDetailId": 123,
            "recordId": 9,
            "userId": 1,
            "wordId": 77,
            "word": "brisk",
            "correctMeaning": "轻快的",
            "selectedMeaning": "港口",
        },
    )

    assert response.status_code == 200
    assert response.json() == {
        "eventId": 10,
        "pendingCount": 1,
        "generated": False,
        "batchWords": [],
        "clozeItemId": None,
    }
    assert processor.notifications == 1


def test_generate_tts_uses_mimo_service(monkeypatch, tmp_path) -> None:
    client = TestClient(create_app())

    class FakeMiMoTTSService:
        def __init__(self, settings) -> None:
            self.settings = settings

        async def generate(self, request) -> TTSGenerationResult:
            assert request.text == "apple"
            assert not hasattr(request, "voice")
            assert not hasattr(request, "model")
            file_path = tmp_path / "apple.wav"
            file_path.write_bytes(b"RIFF")
            return TTSGenerationResult(
                text=request.text,
                voice="Chloe",
                model="mimo-v2.5-tts",
                audio_format=request.audio_format,
                file_name=file_path.name,
                file_path=file_path,
                byte_size=4,
                content_type="audio/wav",
            )

    monkeypatch.setattr(routes, "MiMoTTSService", FakeMiMoTTSService)

    response = client.post(
        "/v1/tts/generate",
        json={
            "text": "apple",
            "voice": "RequestVoice",
            "model": "request-model",
            "format": "wav",
        },
    )

    assert response.status_code == 200
    assert response.json() == {
        "text": "apple",
        "voice": "Chloe",
        "model": "mimo-v2.5-tts",
        "format": "wav",
        "fileName": "apple.wav",
        "filePath": str(tmp_path / "apple.wav"),
        "downloadUrl": "/v1/tts/files/apple.wav",
        "byteSize": 4,
        "contentType": "audio/wav",
    }
