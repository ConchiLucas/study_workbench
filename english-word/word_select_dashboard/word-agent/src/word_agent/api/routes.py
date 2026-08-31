import asyncio
from typing import Annotated

from fastapi import APIRouter, BackgroundTasks, Depends, Request, status
from fastapi.exceptions import HTTPException
from fastapi.responses import FileResponse

from word_agent import __version__
from word_agent.core.config import Settings, get_settings
from word_agent.domain.schemas import (
    HealthResponse,
    RunExecutionRequest,
    RunExecutionResponse,
    SentenceGenerationRequest,
    SentenceGenerationResponse,
    StepEvent,
    StepStatus,
    TTSGenerationRequest,
    TTSGenerationResponse,
    WordCleanSentenceScoreRequest,
    WordCleanSentenceScoreResponse,
    WrongWordEventRequest,
    WrongWordEventResponse,
)
from word_agent.services.callbacks import CallbackClient
from word_agent.services.executor import WordRunExecutor
from word_agent.services.llm_client import LLMClient, LLMConfigError, LLMRequestError
from word_agent.services.mimo_tts import (
    MiMoTTSService,
    TTSRequestError,
)
from word_agent.services.minio_storage import MinIOStorage, MinIOStorageError
from word_agent.services.tts_config import TTSConfigError
from word_agent.services.word_clean_sentence_score import (
    WordCleanSentenceScoreError,
    WordCleanSentenceScoreService,
    WordCleanSentenceScoreValidationError,
)
from word_agent.services.wrong_word_strategy import (
    WrongWordPersistenceError,
    WrongWordStrategyError,
)

router = APIRouter()


@router.get("/health", response_model=HealthResponse)
async def health() -> HealthResponse:
    return HealthResponse(service="word-agent", status="ok", version=__version__)


@router.post(
    "/v1/runs/execute",
    response_model=RunExecutionResponse,
    status_code=status.HTTP_202_ACCEPTED,
)
async def execute_run(
    request: RunExecutionRequest,
    background_tasks: BackgroundTasks,
    settings: Annotated[Settings, Depends(get_settings)],
) -> RunExecutionResponse:
    callback_client = CallbackClient(settings)
    callback_url = str(request.callback_url) if request.callback_url else None

    async def emit_event(event: StepEvent) -> None:
        await callback_client.post_event(callback_url, event)

    executor = WordRunExecutor(emit_event=emit_event)
    background_tasks.add_task(executor.execute, request)

    return RunExecutionResponse(
        run_id=request.run_id,
        status=StepStatus.PENDING,
        message="Python agent 已接收任务，执行过程将通过 callbackUrl 回传给 Go",
    )


@router.post("/v1/sentences/generate", response_model=SentenceGenerationResponse)
async def generate_sentence(
    request: SentenceGenerationRequest,
    settings: Annotated[Settings, Depends(get_settings)],
) -> SentenceGenerationResponse:
    llm_client = LLMClient(settings)
    try:
        result = await llm_client.generate_sentence_from_words(words=request.words)
    except LLMConfigError as exc:
        raise HTTPException(status_code=500, detail=str(exc)) from exc
    except LLMRequestError as exc:
        raise HTTPException(status_code=502, detail=str(exc)) from exc

    tts_service = MiMoTTSService(settings)
    try:
        tts_result = await tts_service.generate(TTSGenerationRequest(text=result.sentence))
    except TTSConfigError as exc:
        raise HTTPException(status_code=500, detail=str(exc)) from exc
    except TTSRequestError as exc:
        raise HTTPException(status_code=502, detail=str(exc)) from exc

    try:
        storage = MinIOStorage(settings)
        upload = storage.upload_audio(
            tts_result.file_path,
            content_type=tts_result.content_type,
        )
    except MinIOStorageError as exc:
        raise HTTPException(status_code=502, detail=str(exc)) from exc
    finally:
        tts_result.file_path.unlink(missing_ok=True)

    return SentenceGenerationResponse(
        sentence=result.sentence,
        translation_zh=result.translation_zh,
        explanation_zh=result.explanation_zh,
        words=request.words,
        provider_id=result.provider.id,
        provider_label=result.provider.label,
        model=result.provider.model,
        sentence_audio_url=upload.object_url,
        sentence_audio_bucket=upload.bucket,
        sentence_audio_object_key=upload.object_key,
        sentence_audio_byte_size=upload.byte_size,
        sentence_audio_content_type=upload.content_type,
        tts_provider="xiaomi-mimo",
        tts_model=tts_result.model,
        tts_voice=tts_result.voice,
        tts_format=tts_result.audio_format,
    )


@router.post("/v1/tts/generate", response_model=TTSGenerationResponse)
async def generate_tts(
    request: TTSGenerationRequest,
    settings: Annotated[Settings, Depends(get_settings)],
) -> TTSGenerationResponse:
    tts_service = MiMoTTSService(settings)
    try:
        result = await tts_service.generate(request)
    except TTSConfigError as exc:
        raise HTTPException(status_code=500, detail=str(exc)) from exc
    except TTSRequestError as exc:
        raise HTTPException(status_code=502, detail=str(exc)) from exc

    return TTSGenerationResponse(
        text=result.text,
        voice=result.voice,
        model=result.model,
        audio_format=result.audio_format,
        file_name=result.file_name,
        file_path=str(result.file_path),
        download_url=f"/v1/tts/files/{result.file_name}",
        byte_size=result.byte_size,
        content_type=result.content_type,
    )


@router.get("/v1/tts/files/{file_name}")
async def get_tts_file(
    file_name: str,
    settings: Annotated[Settings, Depends(get_settings)],
) -> FileResponse:
    tts_service = MiMoTTSService(settings)
    try:
        file_path = tts_service.get_saved_file(file_name)
    except FileNotFoundError as exc:
        raise HTTPException(status_code=404, detail="TTS 音频文件不存在") from exc
    except TTSRequestError as exc:
        raise HTTPException(status_code=400, detail=str(exc)) from exc

    media_type = MiMoTTSService.content_type_for(file_path.suffix.lstrip(".").lower())
    return FileResponse(path=file_path, media_type=media_type, filename=file_path.name)


@router.post("/v1/wrong-words/events", response_model=WrongWordEventResponse)
async def receive_wrong_word_event(
    request_body: WrongWordEventRequest,
    request: Request,
) -> WrongWordEventResponse:
    try:
        response = await asyncio.to_thread(
            request.app.state.wrong_word_strategy.enqueue_event,
            request_body,
        )
    except WrongWordPersistenceError as exc:
        raise HTTPException(status_code=500, detail=str(exc)) from exc
    except WrongWordStrategyError as exc:
        raise HTTPException(status_code=502, detail=str(exc)) from exc
    request.app.state.wrong_word_processor.notify()
    return response


@router.post("/v1/word-clean-sentences/score", response_model=WordCleanSentenceScoreResponse)
async def score_word_clean_sentences(
    request: WordCleanSentenceScoreRequest,
    settings: Annotated[Settings, Depends(get_settings)],
) -> WordCleanSentenceScoreResponse:
    score_service = WordCleanSentenceScoreService(settings)
    try:
        return await score_service.score(request)
    except WordCleanSentenceScoreValidationError as exc:
        raise HTTPException(status_code=400, detail=str(exc)) from exc
    except WordCleanSentenceScoreError as exc:
        raise HTTPException(status_code=502, detail=str(exc)) from exc
