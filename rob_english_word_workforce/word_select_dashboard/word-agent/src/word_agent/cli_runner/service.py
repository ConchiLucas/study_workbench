import asyncio
import json
import logging
import os
import signal
import tempfile
import time
from collections.abc import Awaitable, Callable
from dataclasses import dataclass
from pathlib import Path
from typing import Protocol

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, ConfigDict, Field, field_validator

from word_agent.cli_runner.config_client import (
    CLIProviderConfig,
    CLIProviderConfigClient,
    CLIProviderConfigError,
)
from word_agent.cli_runner.drivers import (
    CLIRunnerError,
    build_codex_invocation,
    build_gemini_invocation,
    extract_gemini_content,
)

logger = logging.getLogger(__name__)

_CODEX_OUTPUT_SCHEMA = {
    "type": "object",
    "additionalProperties": False,
    "required": ["sentence", "translation_zh", "explanation_zh"],
    "properties": {
        "sentence": {"type": "string"},
        "translation_zh": {"type": "string"},
        "explanation_zh": {"type": "string"},
    },
}
_GEMINI_DENY_ALL_POLICY = """[[rule]]
toolName = "*"
decision = "deny"
priority = 999
interactive = false
"""
_CLI_ENVIRONMENT_KEYS = {
    "HOME",
    "PATH",
    "TMPDIR",
    "TMP",
    "TEMP",
    "LANG",
    "LC_ALL",
    "LC_CTYPE",
    "LC_MESSAGES",
    "LC_COLLATE",
    "LC_MONETARY",
    "LC_NUMERIC",
    "LC_TIME",
    "USER",
    "LOGNAME",
    "SHELL",
    "SSL_CERT_FILE",
    "SSL_CERT_DIR",
    "XDG_CONFIG_HOME",
    "XDG_CACHE_HOME",
}


class ActiveConfigClient(Protocol):
    def load_active(self, executor_id: str) -> CLIProviderConfig: ...


class TextRunner(Protocol):
    async def generate(self, executor_id: str, prompt: str) -> "CLIExecutionResult": ...


ProcessFactory = Callable[..., Awaitable[asyncio.subprocess.Process]]
KillProcessGroup = Callable[[int, signal.Signals], None]


@dataclass(frozen=True)
class CLIRunnerSettings:
    database_dsn: str

    @property
    def ready(self) -> bool:
        return bool(self.database_dsn.strip())

    def require_ready(self) -> None:
        if not self.ready:
            raise CLIRunnerSettingsError("CLI Runner 配置不完整")

    @classmethod
    def from_environment(cls) -> "CLIRunnerSettings":
        database_dsn = os.environ.get("WORD_AGENT_CLI_RUNNER_DB_DSN", "").strip()
        if not database_dsn:
            database_dsn = os.environ.get("WORD_AGENT_SELECT_DB_DSN", "").strip()
        return cls(database_dsn=database_dsn)


class CLIRunnerSettingsError(RuntimeError):
    pass


@dataclass(frozen=True)
class CLIExecutionResult:
    content: str
    driver: str
    model: str
    duration_ms: int


class TextGenerationRequest(BaseModel):
    model_config = ConfigDict(extra="forbid", str_strip_whitespace=True)

    executor_id: str = Field(min_length=1, max_length=120)
    prompt: str = Field(min_length=1, max_length=100_000)

    @field_validator("executor_id", "prompt")
    @classmethod
    def reject_blank_text(cls, value: str) -> str:
        if not value.strip():
            raise ValueError("must not be blank")
        return value


class TextGenerationResponse(BaseModel):
    executor_id: str
    content: str
    driver: str
    model: str
    duration_ms: int


class CLIRunner:
    def __init__(
        self,
        config_client: ActiveConfigClient,
        *,
        process_factory: ProcessFactory = asyncio.create_subprocess_exec,
        kill_process_group: KillProcessGroup = os.killpg,
    ) -> None:
        self._config_client = config_client
        self._process_factory = process_factory
        self._kill_process_group = kill_process_group
        self._provider_semaphores: dict[str, asyncio.Semaphore] = {}

    async def generate(self, executor_id: str, prompt: str) -> CLIExecutionResult:
        semaphore = self._provider_semaphores.setdefault(executor_id, asyncio.Semaphore(1))
        async with semaphore:
            config = await asyncio.to_thread(self._config_client.load_active, executor_id)
            return await self._generate_locked(config, prompt)

    async def _generate_locked(
        self,
        config: CLIProviderConfig,
        prompt: str,
    ) -> CLIExecutionResult:
        self._validate_host_paths(config)
        started_at = time.monotonic()
        exit_code: int | None = None
        try:
            with tempfile.TemporaryDirectory(prefix="word-agent-cli-") as temporary_directory:
                temp_path = Path(temporary_directory)
                if config.driver == "codex":
                    schema_path = temp_path / "schema.json"
                    last_output_path = temp_path / "last-message.txt"
                    schema_path.write_text(
                        json.dumps(_CODEX_OUTPUT_SCHEMA, ensure_ascii=False),
                        encoding="utf-8",
                    )
                    invocation = build_codex_invocation(
                        config,
                        prompt=prompt,
                        schema_path=schema_path,
                        last_output_path=last_output_path,
                    )
                elif config.driver == "gemini":
                    policy_path = temp_path / "deny-all-policy.toml"
                    policy_path.write_text(_GEMINI_DENY_ALL_POLICY, encoding="utf-8")
                    invocation = build_gemini_invocation(
                        config,
                        prompt,
                        policy_path=policy_path,
                    )
                else:
                    raise CLIRunnerError(f"不支持的 CLI driver: {config.driver}")

                environment = _build_cli_environment()
                stdin_target = (
                    asyncio.subprocess.PIPE
                    if invocation.stdin_text is not None
                    else asyncio.subprocess.DEVNULL
                )
                try:
                    process = await self._process_factory(
                        *invocation.argv,
                        stdin=stdin_target,
                        stdout=asyncio.subprocess.PIPE,
                        stderr=asyncio.subprocess.PIPE,
                        cwd=config.working_directory,
                        env=environment,
                        start_new_session=True,
                    )
                except OSError as exc:
                    raise CLIRunnerError("无法启动 CLI 进程") from exc
                try:
                    stdout, _stderr = await asyncio.wait_for(
                        process.communicate(
                            input=(
                                invocation.stdin_text.encode("utf-8")
                                if invocation.stdin_text is not None
                                else None
                            )
                        ),
                        timeout=config.timeout_seconds,
                    )
                except TimeoutError as exc:
                    await self._stop_process_group(process)
                    raise CLIRunnerError("CLI 执行超时") from exc
                except asyncio.CancelledError:
                    await asyncio.shield(self._stop_process_group(process))
                    raise

                exit_code = process.returncode
                if exit_code != 0:
                    raise CLIRunnerError(f"CLI 执行失败，退出码: {exit_code}")
                if invocation.final_output_path is not None:
                    try:
                        content = invocation.final_output_path.read_text(encoding="utf-8").strip()
                    except (OSError, UnicodeError) as exc:
                        raise CLIRunnerError("Codex CLI 未生成有效文本内容") from exc
                    if not content:
                        raise CLIRunnerError("Codex CLI 未生成有效文本内容")
                else:
                    try:
                        decoded_stdout = stdout.decode("utf-8")
                    except UnicodeDecodeError as exc:
                        raise CLIRunnerError("Gemini CLI 返回内容编码无效") from exc
                    content = extract_gemini_content(decoded_stdout)
                return CLIExecutionResult(
                    content=content,
                    driver=config.driver,
                    model=config.model,
                    duration_ms=int((time.monotonic() - started_at) * 1000),
                )
        finally:
            logger.info(
                "CLI execution completed executor_id=%s driver=%s model=%s "
                "elapsed_ms=%d exit_code=%s",
                config.provider_id,
                config.driver,
                config.model,
                int((time.monotonic() - started_at) * 1000),
                exit_code,
            )

    async def _stop_process_group(self, process: asyncio.subprocess.Process) -> None:
        self._signal_process_group(process.pid, signal.SIGTERM)
        try:
            await asyncio.wait_for(process.wait(), timeout=2.0)
            return
        except TimeoutError:
            pass
        self._signal_process_group(process.pid, signal.SIGKILL)
        try:
            await asyncio.wait_for(process.wait(), timeout=2.0)
        except TimeoutError as exc:
            raise CLIRunnerError("无法终止 CLI 进程组") from exc

    def _signal_process_group(self, pid: int, sent_signal: signal.Signals) -> None:
        try:
            self._kill_process_group(pid, sent_signal)
        except ProcessLookupError:
            return
        except (OSError, PermissionError) as exc:
            raise CLIRunnerError("无法终止 CLI 进程组") from exc

    @staticmethod
    def _validate_host_paths(config: CLIProviderConfig) -> None:
        command = Path(config.command_path)
        working_directory = Path(config.working_directory)
        if not command.is_file() or not os.access(command, os.X_OK):
            raise CLIRunnerError(f"CLI 命令不可执行: {config.provider_id}")
        if not working_directory.is_dir():
            raise CLIRunnerError(f"CLI 工作目录不存在: {config.provider_id}")


def create_app(
    *,
    settings: CLIRunnerSettings | None = None,
    runner: TextRunner | None = None,
) -> FastAPI:
    resolved_settings = settings or CLIRunnerSettings.from_environment()
    resolved_runner = runner or CLIRunner(
        CLIProviderConfigClient(resolved_settings.database_dsn)
    )
    app = FastAPI(title="Word Agent CLI Runner", docs_url=None, redoc_url=None)

    @app.get("/health")
    async def health() -> dict[str, str]:
        _require_ready_settings(resolved_settings)
        return {"service": "word-agent-cli-runner", "status": "ok"}

    @app.post("/v1/text/generate", response_model=TextGenerationResponse)
    async def generate(request: TextGenerationRequest) -> TextGenerationResponse:
        _require_ready_settings(resolved_settings)
        try:
            result = await resolved_runner.generate(request.executor_id, request.prompt)
        except CLIProviderConfigError as exc:
            raise HTTPException(status_code=409, detail=str(exc)) from exc
        except CLIRunnerError as exc:
            raise HTTPException(status_code=502, detail=str(exc)) from exc
        return TextGenerationResponse(
            executor_id=request.executor_id,
            content=result.content,
            driver=result.driver,
            model=result.model,
            duration_ms=result.duration_ms,
        )

    return app


def _build_cli_environment() -> dict[str, str]:
    environment = {
        key: value
        for key, value in os.environ.items()
        if key in _CLI_ENVIRONMENT_KEYS
    }
    environment.update({"CODEX_CI": "1", "TERM": "dumb"})
    return environment


def _require_ready_settings(settings: CLIRunnerSettings) -> None:
    try:
        settings.require_ready()
    except CLIRunnerSettingsError as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc
