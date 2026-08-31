from datetime import UTC, datetime
from enum import StrEnum
from typing import Any, Literal

from pydantic import BaseModel, ConfigDict, Field, HttpUrl, field_validator


class StepStatus(StrEnum):
    PENDING = "pending"
    RUNNING = "running"
    SUCCESS = "success"
    FAILED = "failed"
    SKIPPED = "skipped"
    RETRYING = "retrying"


class RunExecutionRequest(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    run_id: str = Field(alias="runId", min_length=1)
    word: str = Field(min_length=1)
    meaning: str = Field(min_length=1)
    callback_url: HttpUrl | None = Field(default=None, alias="callbackUrl")
    metadata: dict[str, Any] = Field(default_factory=dict)


class RunExecutionResponse(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    run_id: str = Field(alias="runId")
    status: StepStatus
    message: str


class StepEvent(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    run_id: str = Field(alias="runId")
    step_id: str = Field(alias="stepId")
    status: StepStatus
    message: str
    input_data: dict[str, Any] = Field(default_factory=dict, alias="input")
    output_data: dict[str, Any] = Field(default_factory=dict, alias="output")
    error: str | None = None
    created_at: datetime = Field(default_factory=lambda: datetime.now(UTC), alias="createdAt")


class HealthResponse(BaseModel):
    service: str
    status: str
    version: str


class SentenceGenerationRequest(BaseModel):
    words: list[str] = Field(min_length=1, max_length=12)

    @field_validator("words")
    @classmethod
    def clean_words(cls, words: list[str]) -> list[str]:
        cleaned_words = [word.strip() for word in words if word.strip()]
        if not cleaned_words:
            raise ValueError("words 至少需要包含一个非空单词")
        return cleaned_words


class SentenceGenerationResponse(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    sentence: str
    translation_zh: str = Field(alias="translationZh")
    explanation_zh: str = Field(alias="explanationZh")
    words: list[str]
    provider_id: str = Field(alias="providerId")
    provider_label: str = Field(alias="providerLabel")
    model: str
    sentence_audio_url: str = Field(alias="sentenceAudioUrl")
    sentence_audio_bucket: str = Field(alias="sentenceAudioBucket")
    sentence_audio_object_key: str = Field(alias="sentenceAudioObjectKey")
    sentence_audio_byte_size: int = Field(alias="sentenceAudioByteSize")
    sentence_audio_content_type: str = Field(alias="sentenceAudioContentType")
    tts_provider: str = Field(alias="ttsProvider")
    tts_model: str = Field(alias="ttsModel")
    tts_voice: str = Field(alias="ttsVoice")
    tts_format: str = Field(alias="ttsFormat")


class WrongWordEventRequest(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    source: str = "rob_english_word_back"
    answer_detail_id: int = Field(alias="answerDetailId")
    record_id: int | None = Field(default=None, alias="recordId")
    user_id: int = Field(alias="userId")
    user_name: str | None = Field(default=None, alias="userName")
    word_id: int | None = Field(default=None, alias="wordId")
    word: str = Field(min_length=1)
    word_difficulty: int | None = Field(default=None, alias="wordDifficulty")
    options: list[str | None] = Field(default_factory=list)
    correct_answer_index: int | None = Field(default=None, alias="correctAnswerIndex")
    selected_answer_index: int | None = Field(default=None, alias="selectedAnswerIndex")
    correct_meaning: str | None = Field(default=None, alias="correctMeaning")
    selected_meaning: str | None = Field(default=None, alias="selectedMeaning")

    @field_validator("word")
    @classmethod
    def clean_word(cls, word: str) -> str:
        cleaned_word = word.strip()
        if not cleaned_word:
            raise ValueError("word 不能为空")
        return cleaned_word


class WrongWordEventResponse(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    event_id: int | None = Field(alias="eventId")
    pending_count: int = Field(alias="pendingCount")
    generated: bool
    batch_words: list[str] = Field(default_factory=list, alias="batchWords")
    cloze_item_id: int | None = Field(default=None, alias="clozeItemId")


class WordCleanSentenceScoreRequest(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    ids: list[int] | None = None
    word_clean_ids: list[int] | None = Field(default=None, alias="wordCleanIds")
    model_names: list[str] | None = Field(default=None, alias="modelNames")
    judge_model: str | None = Field(default=None, alias="judgeModel")
    limit: int = Field(default=10, ge=1, le=50)
    overwrite: bool = False

    @field_validator("ids", "word_clean_ids")
    @classmethod
    def clean_ids(cls, values: list[int] | None) -> list[int] | None:
        if values is None:
            return None
        cleaned_values = [int(value) for value in values if int(value) > 0]
        return cleaned_values or None

    @field_validator("model_names")
    @classmethod
    def clean_model_names(cls, values: list[str] | None) -> list[str] | None:
        if values is None:
            return None
        cleaned_values = [value.strip() for value in values if value.strip()]
        return cleaned_values or None

    @field_validator("judge_model")
    @classmethod
    def clean_judge_model(cls, value: str | None) -> str | None:
        if value is None:
            return None
        cleaned_value = value.strip()
        return cleaned_value or None


class WordCleanSentenceScoreItem(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    id: int
    word_clean_id: int = Field(alias="wordCleanId")
    word: str
    model_name: str = Field(alias="modelName")
    score: int
    score_reason: str = Field(alias="scoreReason")


class WordCleanBestSentenceItem(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    id: int
    word_clean_id: int = Field(alias="wordCleanId")
    word: str
    meaning: str
    source_sentence_id: int = Field(alias="sourceSentenceId")
    source_model_name: str = Field(alias="sourceModelName")
    sentence: str
    sentence_translation: str = Field(alias="sentenceTranslation")
    score: int
    score_reason: str = Field(alias="scoreReason")
    score_model_name: str = Field(alias="scoreModelName")
    scored_at: datetime | None = Field(default=None, alias="scoredAt")
    tts_status: str = Field(alias="ttsStatus")
    tts_bucket: str = Field(alias="ttsBucket")
    tts_object_key: str = Field(alias="ttsObjectKey")
    tts_object_url: str = Field(alias="ttsObjectUrl")


class WordCleanSentenceScoreResponse(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    status: StepStatus
    message: str
    judge_model: str = Field(alias="judgeModel")
    processed_count: int = Field(alias="processedCount")
    scored_count: int = Field(alias="scoredCount")
    failed_count: int = Field(alias="failedCount")
    items: list[WordCleanSentenceScoreItem] = Field(default_factory=list)
    best_items: list[WordCleanBestSentenceItem] = Field(default_factory=list, alias="bestItems")


class TTSGenerationRequest(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    text: str = Field(min_length=1, max_length=2000)
    audio_format: Literal["wav"] = Field(default="wav", alias="format")
    style: str | None = (
        "Clear English pronunciation for vocabulary learning. "
        "Speak naturally and clearly."
    )
    file_name: str | None = Field(default=None, alias="fileName")
    overwrite: bool = False

    @field_validator("text")
    @classmethod
    def clean_text(cls, text: str) -> str:
        cleaned_text = text.strip()
        if not cleaned_text:
            raise ValueError("text 不能为空")
        return cleaned_text

    @field_validator("style", "file_name")
    @classmethod
    def clean_optional_text(cls, value: str | None) -> str | None:
        if value is None:
            return None
        cleaned_value = value.strip()
        return cleaned_value or None


class TTSGenerationResponse(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    text: str
    voice: str
    model: str
    audio_format: str = Field(alias="format")
    file_name: str = Field(alias="fileName")
    file_path: str = Field(alias="filePath")
    download_url: str = Field(alias="downloadUrl")
    byte_size: int = Field(alias="byteSize")
    content_type: str = Field(alias="contentType")
