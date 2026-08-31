from dataclasses import dataclass
from pathlib import Path

import psycopg
from psycopg.rows import dict_row


class CLIProviderConfigError(RuntimeError):
    """Raised when the requested CLI is not the complete active sentence executor."""


@dataclass(frozen=True)
class CLIProviderConfig:
    provider_id: str
    label: str
    driver: str
    command_path: str
    model: str
    reasoning_effort: str
    working_directory: str
    timeout_seconds: int
    enabled: bool


_CODEX_MODELS = {
    "gpt-5.6-sol",
    "gpt-5.6-terra",
    "gpt-5.6-luna",
    "gpt-5.5",
    "gpt-5.4",
    "gpt-5.4-mini",
    "gpt-5.3-codex-spark",
}
_CODEX_REASONING = {"low", "medium", "high", "xhigh"}
_GEMINI_MODELS = {"auto", "pro", "flash", "flash-lite"}


@dataclass(frozen=True)
class CLIProviderConfigClient:
    database_dsn: str

    def load_active(self, executor_id: str) -> CLIProviderConfig:
        requested_id = executor_id.strip()
        if not requested_id:
            raise CLIProviderConfigError("请求的 CLI 配置 ID 不能为空")
        if not self.database_dsn.strip():
            raise CLIProviderConfigError("CLI Runner 数据库 DSN 未配置")

        try:
            with psycopg.connect(
                self.database_dsn,
                row_factory=dict_row,
                connect_timeout=5,
                options=(
                    "-c default_transaction_read_only=on "
                    "-c statement_timeout=5000"
                ),
            ) as connection:
                with connection.cursor() as cursor:
                    cursor.execute(
                        """
                        WITH active AS (
                            SELECT executor_type, executor_id
                            FROM sentence_executor_config
                            WHERE singleton_key = %s
                        )
                        SELECT active.executor_type, active.executor_id,
                               cli.provider_id, cli.label, cli.driver,
                               cli.command_path, cli.model, cli.reasoning_effort,
                               cli.working_directory, cli.timeout_seconds, cli.enabled
                        FROM active
                        LEFT JOIN cli_provider_configs AS cli
                          ON cli.provider_id = %s
                        """,
                        ("default", requested_id),
                    )
                    row = cursor.fetchone()
        except psycopg.Error as exc:
            raise CLIProviderConfigError("读取 CLI 执行器配置失败") from exc

        return self._validate_row(row, requested_id)

    @staticmethod
    def _validate_row(row: dict[str, object] | None, requested_id: str) -> CLIProviderConfig:
        if row is None:
            raise CLIProviderConfigError("尚未选择造句执行器")

        executor_type = _clean_text(row.get("executor_type"))
        executor_id = _clean_text(row.get("executor_id"))
        if executor_type != "cli":
            raise CLIProviderConfigError("当前造句执行器不是 CLI")
        if executor_id != requested_id:
            raise CLIProviderConfigError("请求的 CLI 不是当前造句执行器")
        if not _clean_text(row.get("provider_id")):
            raise CLIProviderConfigError(f"造句 CLI 配置不存在: {requested_id}")
        if row.get("enabled") is not True:
            raise CLIProviderConfigError(f"造句 CLI 配置已停用: {requested_id}")

        config = CLIProviderConfig(
            provider_id=_clean_text(row.get("provider_id")),
            label=_clean_text(row.get("label")) or requested_id,
            driver=_clean_text(row.get("driver")),
            command_path=_clean_text(row.get("command_path")),
            model=_clean_text(row.get("model")),
            reasoning_effort=_clean_text(row.get("reasoning_effort")),
            working_directory=_clean_text(row.get("working_directory")),
            timeout_seconds=_positive_int(row.get("timeout_seconds")),
            enabled=True,
        )
        required_text = (
            config.provider_id,
            config.driver,
            config.command_path,
            config.model,
            config.working_directory,
        )
        if (
            any(not value for value in required_text)
            or config.timeout_seconds <= 0
            or not Path(config.command_path).is_absolute()
            or not Path(config.working_directory).is_absolute()
        ):
            raise CLIProviderConfigError(f"造句 CLI 配置不完整: {requested_id}")
        _validate_driver_options(config)
        return config


def _clean_text(value: object) -> str:
    return str(value or "").strip()


def _positive_int(value: object) -> int:
    try:
        return int(value or 0)
    except (TypeError, ValueError, OverflowError):
        return 0


def _validate_driver_options(config: CLIProviderConfig) -> None:
    if config.driver == "codex":
        if config.model not in _CODEX_MODELS or config.reasoning_effort not in _CODEX_REASONING:
            raise CLIProviderConfigError(f"造句 CLI 配置不完整: {config.provider_id}")
        return
    if config.driver == "gemini":
        if config.model not in _GEMINI_MODELS or config.reasoning_effort:
            raise CLIProviderConfigError(f"造句 CLI 配置不完整: {config.provider_id}")
        return
    raise CLIProviderConfigError(f"不支持的 CLI driver: {config.driver}")
