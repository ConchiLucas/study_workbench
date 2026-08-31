from functools import lru_cache
from pathlib import Path

from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


def _default_go_config_path() -> Path:
    return Path(__file__).resolve().parents[4] / "server" / "config.yaml"


def _default_tts_output_dir() -> Path:
    return Path(__file__).resolve().parents[3] / "tts_audio"


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env",
        env_prefix="WORD_AGENT_",
        extra="ignore",
    )

    host: str = "0.0.0.0"
    port: int = 6017
    log_level: str = "INFO"
    callback_timeout_seconds: float = 10.0
    llm_timeout_seconds: float = 180.0
    llm_verify_ssl: bool = False
    cli_runner_url: str = "http://127.0.0.1:6018"
    cloze_batch_size: int = 3
    cloze_request_timeout_seconds: float = 300.0
    wrong_word_max_batches_per_wake: int = 10
    wrong_word_max_retries: int = 3
    wrong_word_processing_timeout_seconds: float = 600.0
    cloze_generate_url: str = "http://127.0.0.1:6012/api/external/sentence-cloze/generate"
    select_db_dsn: str | None = None
    rob_word_db_dsn: str | None = None
    go_config_path: Path = Field(default_factory=_default_go_config_path)
    default_model: str = "gpt-4.1-mini"
    word_clean_score_default_model: str = "qwen3.6-flash"
    openai_api_key: str | None = Field(default=None, alias="OPENAI_API_KEY")
    tts_output_dir: Path = Field(default_factory=_default_tts_output_dir)
    tts_timeout_seconds: float = 60.0
    tts_verify_ssl: bool = True
    minio_endpoint: str = "127.0.0.1:19100"
    minio_access_key_id: str = ""
    minio_secret_access_key: str = ""
    minio_use_ssl: bool = False
    minio_bucket_name: str = "ai-file-navigation"
    minio_base_path: str = ""
    cloze_tts_object_prefix: str = "sentence_cloze_tts"


@lru_cache
def get_settings() -> Settings:
    return Settings()
