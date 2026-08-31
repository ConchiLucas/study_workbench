import json
from dataclasses import dataclass
from pathlib import Path

from word_agent.cli_runner.config_client import CLIProviderConfig


class CLIRunnerError(RuntimeError):
    """Raised for a safe, user-facing local CLI execution failure."""


_CODEX_DISABLED_FEATURE_ARGS = tuple(
    argument
    for feature in (
        "shell_tool",
        "apps",
        "browser_use",
        "browser_use_external",
        "browser_use_full_cdp_access",
        "computer_use",
        "image_generation",
        "in_app_browser",
    )
    for argument in ("--disable", feature)
)


@dataclass(frozen=True)
class CLIInvocation:
    argv: tuple[str, ...]
    stdin_text: str | None
    final_output_path: Path | None


def build_codex_invocation(
    config: CLIProviderConfig,
    *,
    prompt: str,
    schema_path: Path,
    last_output_path: Path,
) -> CLIInvocation:
    return CLIInvocation(
        argv=(
            config.command_path,
            "exec",
            "--ignore-user-config",
            "--ignore-rules",
            *_CODEX_DISABLED_FEATURE_ARGS,
            "--model",
            config.model,
            "--config",
            f'model_reasoning_effort="{config.reasoning_effort}"',
            "--sandbox",
            "read-only",
            "--ephemeral",
            "--output-schema",
            str(schema_path),
            "--output-last-message",
            str(last_output_path),
            "-",
        ),
        stdin_text=prompt,
        final_output_path=last_output_path,
    )


def build_gemini_invocation(
    config: CLIProviderConfig,
    prompt: str,
    *,
    policy_path: Path,
) -> CLIInvocation:
    return CLIInvocation(
        argv=(
            config.command_path,
            "--model",
            config.model,
            "--prompt",
            "",
            "--output-format",
            "json",
            "--approval-mode",
            "plan",
            "--policy",
            str(policy_path),
        ),
        stdin_text=prompt,
        final_output_path=None,
    )


def extract_gemini_content(stdout: str) -> str:
    try:
        payload = json.loads(stdout)
    except (json.JSONDecodeError, TypeError) as exc:
        raise CLIRunnerError("Gemini CLI 返回内容不是合法 JSON") from exc
    if not isinstance(payload, dict):
        raise CLIRunnerError("Gemini CLI 返回格式错误")
    content = payload.get("response") or payload.get("content")
    if not isinstance(content, str) or not content.strip():
        raise CLIRunnerError("Gemini CLI 没有返回文本内容")
    return content.strip()
