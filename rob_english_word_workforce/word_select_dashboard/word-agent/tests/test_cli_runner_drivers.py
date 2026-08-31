import json
from pathlib import Path

import pytest

from word_agent.cli_runner.config_client import CLIProviderConfig
from word_agent.cli_runner.drivers import (
    CLIRunnerError,
    build_codex_invocation,
    build_gemini_invocation,
    extract_gemini_content,
)


def cli_config(tmp_path: Path, *, driver: str = "codex") -> CLIProviderConfig:
    command = tmp_path / driver
    command.write_text("#!/bin/sh\n", encoding="utf-8")
    command.chmod(0o700)
    return CLIProviderConfig(
        provider_id=driver,
        label=f"{driver.title()} CLI",
        driver=driver,
        command_path=str(command),
        model="gpt-5.6-sol" if driver == "codex" else "pro",
        reasoning_effort="high" if driver == "codex" else "",
        working_directory=str(tmp_path),
        timeout_seconds=30,
        enabled=True,
    )


def adjacent_pair(argv: tuple[str, ...], option: str) -> tuple[str, str]:
    index = argv.index(option)
    return argv[index], argv[index + 1]


def test_codex_command_is_fixed_and_reads_prompt_from_stdin(tmp_path: Path) -> None:
    config = cli_config(tmp_path)
    invocation = build_codex_invocation(
        config,
        prompt="private prompt",
        schema_path=tmp_path / "schema.json",
        last_output_path=tmp_path / "last.txt",
    )

    assert invocation.argv[:2] == (config.command_path, "exec")
    assert adjacent_pair(invocation.argv, "--model") == ("--model", "gpt-5.6-sol")
    assert adjacent_pair(invocation.argv, "--sandbox") == ("--sandbox", "read-only")
    assert adjacent_pair(invocation.argv, "--config") == (
        "--config",
        'model_reasoning_effort="high"',
    )
    assert "--ephemeral" in invocation.argv
    assert "--ignore-user-config" in invocation.argv
    assert "--ignore-rules" in invocation.argv
    assert adjacent_pair(invocation.argv, "--disable") == ("--disable", "shell_tool")
    disabled_features = {
        invocation.argv[index + 1]
        for index, argument in enumerate(invocation.argv[:-1])
        if argument == "--disable"
    }
    assert disabled_features == {
        "shell_tool",
        "apps",
        "browser_use",
        "browser_use_external",
        "browser_use_full_cdp_access",
        "computer_use",
        "image_generation",
        "in_app_browser",
    }
    assert invocation.argv[-1] == "-"
    assert "private prompt" not in invocation.argv
    assert invocation.stdin_text == "private prompt"
    assert invocation.final_output_path == tmp_path / "last.txt"


def test_gemini_command_uses_json_plan_mode(tmp_path: Path) -> None:
    config = cli_config(tmp_path, driver="gemini")
    policy_path = tmp_path / "deny-all.toml"
    invocation = build_gemini_invocation(config, "hello", policy_path=policy_path)

    assert invocation.argv == (
        config.command_path,
        "--model",
        "pro",
        "--prompt",
        "",
        "--output-format",
        "json",
        "--approval-mode",
        "plan",
        "--policy",
        str(policy_path),
    )
    assert "hello" not in invocation.argv
    assert invocation.stdin_text == "hello"
    assert invocation.final_output_path is None


@pytest.mark.parametrize("field", ["response", "content"])
def test_gemini_json_content_is_extracted(field: str) -> None:
    assert extract_gemini_content(json.dumps({field: "  generated text  "})) == "generated text"


@pytest.mark.parametrize(
    "stdout",
    ["not json", "[]", "{}", '{"response": 3}', '{"response": "  "}'],
)
def test_gemini_invalid_output_is_rejected_without_echoing_output(stdout: str) -> None:
    with pytest.raises(CLIRunnerError, match="Gemini CLI") as error:
        extract_gemini_content(stdout)

    assert stdout not in str(error.value)
