import asyncio
import json
import logging
import signal
import threading
from pathlib import Path
from types import SimpleNamespace

import pytest
from fastapi.testclient import TestClient

from word_agent.cli_runner import main as cli_runner_main
from word_agent.cli_runner.config_client import (
    CLIProviderConfig,
    CLIProviderConfigClient,
    CLIProviderConfigError,
)
from word_agent.cli_runner.drivers import CLIRunnerError
from word_agent.cli_runner.service import (
    CLIExecutionResult,
    CLIRunner,
    CLIRunnerSettings,
    create_app,
)


class FakeCursor:
    def __init__(self, row: dict[str, object] | None) -> None:
        self.row = row
        self.query = ""
        self.params: tuple[object, ...] = ()

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc, traceback) -> None:
        return None

    def execute(self, query: str, params: tuple[object, ...]) -> None:
        self.query = query
        self.params = params

    def fetchone(self):
        return self.row


class FakeConnection:
    def __init__(self, cursor: FakeCursor) -> None:
        self._cursor = cursor

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc, traceback) -> None:
        return None

    def cursor(self) -> FakeCursor:
        return self._cursor


def database_row(tmp_path: Path, **patch: object) -> dict[str, object]:
    command = tmp_path / "codex"
    command.write_text("#!/bin/sh\n", encoding="utf-8")
    command.chmod(0o700)
    row: dict[str, object] = {
        "executor_type": "cli",
        "executor_id": "codex",
        "provider_id": "codex",
        "label": "Codex CLI",
        "driver": "codex",
        "command_path": str(command),
        "model": "gpt-5.6-sol",
        "reasoning_effort": "high",
        "working_directory": str(tmp_path),
        "timeout_seconds": 30,
        "enabled": True,
    }
    row.update(patch)
    return row


def config_from_row(row: dict[str, object]) -> CLIProviderConfig:
    return CLIProviderConfig(
        provider_id=str(row["provider_id"]),
        label=str(row["label"]),
        driver=str(row["driver"]),
        command_path=str(row["command_path"]),
        model=str(row["model"]),
        reasoning_effort=str(row["reasoning_effort"]),
        working_directory=str(row["working_directory"]),
        timeout_seconds=int(row["timeout_seconds"]),
        enabled=bool(row["enabled"]),
    )


def install_fake_database(monkeypatch: pytest.MonkeyPatch, row: dict[str, object] | None):
    cursor = FakeCursor(row)
    fake_dict_row = object()

    def connect(dsn: str, **kwargs):
        assert dsn == "postgresql://runner"
        assert kwargs["row_factory"] is fake_dict_row
        assert kwargs["connect_timeout"] == 5
        assert "default_transaction_read_only=on" in kwargs["options"]
        assert "statement_timeout=5000" in kwargs["options"]
        return FakeConnection(cursor)

    from word_agent.cli_runner import config_client

    monkeypatch.setattr(
        config_client,
        "psycopg",
        SimpleNamespace(connect=connect, Error=Exception),
    )
    monkeypatch.setattr(config_client, "dict_row", fake_dict_row)
    return cursor


def test_config_client_reads_only_exact_active_cli(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    cursor = install_fake_database(monkeypatch, database_row(tmp_path))

    config = CLIProviderConfigClient("postgresql://runner").load_active("codex")

    assert config.provider_id == "codex"
    assert cursor.params == ("default", "codex")
    assert "sentence_executor_config" in cursor.query
    assert "cli_provider_configs" in cursor.query


@pytest.mark.parametrize(
    ("patch", "message"),
    [
        ({"executor_type": "api"}, "不是 CLI"),
        ({"executor_id": "gemini"}, "不是当前"),
        ({"provider_id": None}, "不存在"),
        ({"enabled": False}, "已停用"),
        ({"command_path": ""}, "不完整"),
        ({"timeout_seconds": 0}, "不完整"),
    ],
)
def test_config_client_rejects_non_active_or_incomplete_config(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
    patch: dict[str, object],
    message: str,
) -> None:
    install_fake_database(monkeypatch, database_row(tmp_path, **patch))

    with pytest.raises(CLIProviderConfigError, match=message):
        CLIProviderConfigClient("postgresql://runner").load_active("codex")


class StaticConfigClient:
    def __init__(self, config: CLIProviderConfig) -> None:
        self.config = config

    def load_active(self, executor_id: str) -> CLIProviderConfig:
        if executor_id != self.config.provider_id:
            raise CLIProviderConfigError("请求的 CLI 不是当前造句执行器")
        return self.config


class FakeRunner:
    async def generate(self, executor_id: str, prompt: str) -> CLIExecutionResult:
        return CLIExecutionResult(
            content=f"{executor_id}:{prompt}",
            driver="codex",
            model="gpt-5.6-sol",
            duration_ms=12,
        )


def test_health_is_public_and_does_not_expose_settings() -> None:
    app = create_app(
        settings=CLIRunnerSettings(database_dsn="postgresql://runner"),
        runner=FakeRunner(),
    )

    response = TestClient(app).get("/health")

    assert response.status_code == 200
    assert response.json() == {"service": "word-agent-cli-runner", "status": "ok"}
    assert "postgresql" not in response.text


def test_health_reports_unready_when_database_dsn_is_missing() -> None:
    missing_dsn = create_app(
        settings=CLIRunnerSettings(database_dsn=""),
        runner=FakeRunner(),
    )

    response = TestClient(missing_dsn).get("/health")
    assert response.status_code == 503
    assert response.json() == {"detail": "CLI Runner 配置不完整"}


def test_main_binds_all_interfaces_by_default(monkeypatch: pytest.MonkeyPatch) -> None:
    invocation: dict[str, object] = {}

    def fake_run(app, **kwargs) -> None:
        invocation.update(kwargs)

    monkeypatch.delenv("WORD_AGENT_CLI_RUNNER_HOST", raising=False)
    monkeypatch.setenv("WORD_AGENT_CLI_RUNNER_DB_DSN", "postgresql://runner")
    monkeypatch.setattr(cli_runner_main.uvicorn, "run", fake_run)

    cli_runner_main.main()

    assert invocation["host"] == "0.0.0.0"


def test_main_accepts_project_runner_marker(monkeypatch: pytest.MonkeyPatch) -> None:
    invocation: dict[str, object] = {}

    def fake_run(app, **kwargs) -> None:
        invocation.update(kwargs)

    monkeypatch.setenv("WORD_AGENT_CLI_RUNNER_DB_DSN", "postgresql://runner")
    monkeypatch.setattr(cli_runner_main.uvicorn, "run", fake_run)

    cli_runner_main.main(["--runner-marker=rob-english-word-workforce-word-select-dashboard"])

    assert invocation["port"] == 6018


def test_main_fails_fast_without_required_configuration(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    monkeypatch.delenv("WORD_AGENT_CLI_RUNNER_DB_DSN", raising=False)
    monkeypatch.delenv("WORD_AGENT_SELECT_DB_DSN", raising=False)

    def unexpected_run(*args, **kwargs) -> None:
        raise AssertionError("uvicorn must not start")

    monkeypatch.setattr(cli_runner_main.uvicorn, "run", unexpected_run)

    with pytest.raises(SystemExit, match="配置不完整") as error:
        cli_runner_main.main()

    assert "postgresql" not in str(error.value)


def test_generate_does_not_require_authentication() -> None:
    app = create_app(
        settings=CLIRunnerSettings(database_dsn="postgresql://runner"),
        runner=FakeRunner(),
    )
    client = TestClient(app)

    response = client.post(
        "/v1/text/generate",
        json={"executor_id": "codex", "prompt": "hello"},
    )
    assert response.status_code == 200
    assert response.json() == {
        "executor_id": "codex",
        "content": "codex:hello",
        "driver": "codex",
        "model": "gpt-5.6-sol",
        "duration_ms": 12,
    }


def test_generate_maps_non_active_cli_to_conflict(tmp_path: Path) -> None:
    config = config_from_row(database_row(tmp_path, provider_id="gemini"))
    app = create_app(
        settings=CLIRunnerSettings(database_dsn="postgresql://runner"),
        runner=CLIRunner(StaticConfigClient(config)),
    )

    response = TestClient(app).post(
        "/v1/text/generate",
        json={"executor_id": "codex", "prompt": "hello"},
    )

    assert response.status_code == 409
    assert response.json() == {"detail": "请求的 CLI 不是当前造句执行器"}


class FakeProcess:
    def __init__(
        self,
        *,
        returncode: int = 0,
        stdout: bytes = b"",
        stderr: bytes = b"",
        communicate_error: BaseException | None = None,
        wait_errors: list[BaseException] | None = None,
    ) -> None:
        self.pid = 43210
        self.returncode = returncode
        self.stdout = stdout
        self.stderr = stderr
        self.communicate_error = communicate_error
        self.wait_errors = iter(wait_errors or [])
        self.input: bytes | None = None
        self.wait_calls = 0

    async def communicate(self, input: bytes | None = None):
        self.input = input
        if self.communicate_error is not None:
            raise self.communicate_error
        return self.stdout, self.stderr

    async def wait(self):
        self.wait_calls += 1
        try:
            raise next(self.wait_errors)
        except StopIteration:
            return self.returncode


def test_codex_process_uses_safe_environment_and_last_message(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
    caplog: pytest.LogCaptureFixture,
) -> None:
    async def exercise() -> None:
        config = config_from_row(database_row(tmp_path))
        process = FakeProcess()
        invocation: dict[str, object] = {}

        async def create_process(*argv: str, **kwargs):
            invocation["argv"] = argv
            invocation["kwargs"] = kwargs
            schema_path = Path(argv[argv.index("--output-schema") + 1])
            schema = json.loads(schema_path.read_text(encoding="utf-8"))
            assert schema["type"] == "object"
            assert schema["additionalProperties"] is False
            assert schema["required"] == ["sentence", "translation_zh", "explanation_zh"]
            assert set(schema["properties"]) == {
                "sentence",
                "translation_zh",
                "explanation_zh",
            }
            last_path = Path(argv[argv.index("--output-last-message") + 1])
            last_path.write_text('{"sentence":"ok"}', encoding="utf-8")
            return process

        monkeypatch.setenv("CODEX_HOME", "/secret/codex")
        monkeypatch.setenv("CODEX_TOKEN", "must-not-leak")
        monkeypatch.setenv("WORD_AGENT_SELECT_DB_DSN", "postgresql://private")
        monkeypatch.setenv("MINIO_SECRET_ACCESS_KEY", "minio-private-secret")
        monkeypatch.setenv("OPENAI_API_KEY", "openai-private-secret")
        monkeypatch.setenv("HOME", str(tmp_path))
        monkeypatch.setenv("PATH", "/usr/bin:/bin")
        monkeypatch.setenv("LC_CTYPE", "UTF-8")
        monkeypatch.setenv("LC_SECRET", "locale-private-secret")
        runner = CLIRunner(StaticConfigClient(config), process_factory=create_process)

        with caplog.at_level(logging.INFO, logger="word_agent.cli_runner.service"):
            result = await runner.generate("codex", "private prompt")

        assert result.content == '{"sentence":"ok"}'
        assert result.driver == "codex"
        assert result.model == "gpt-5.6-sol"
        assert result.duration_ms >= 0
        assert process.input == b"private prompt"
        kwargs = invocation["kwargs"]
        assert isinstance(kwargs, dict)
        assert kwargs["start_new_session"] is True
        assert kwargs["cwd"] == str(tmp_path)
        assert kwargs["env"]["CODEX_CI"] == "1"
        assert kwargs["env"]["TERM"] == "dumb"
        assert "CODEX_HOME" not in kwargs["env"]
        assert "CODEX_TOKEN" not in kwargs["env"]
        assert "WORD_AGENT_SELECT_DB_DSN" not in kwargs["env"]
        assert "MINIO_SECRET_ACCESS_KEY" not in kwargs["env"]
        assert "OPENAI_API_KEY" not in kwargs["env"]
        assert kwargs["env"]["HOME"] == str(tmp_path)
        assert kwargs["env"]["PATH"] == "/usr/bin:/bin"
        assert kwargs["env"]["LC_CTYPE"] == "UTF-8"
        assert "LC_SECRET" not in kwargs["env"]
        allowed_environment_keys = {
            "HOME",
            "PATH",
            "TMPDIR",
            "TMP",
            "TEMP",
            "LANG",
            "LC_CTYPE",
            "LC_ALL",
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
            "CODEX_CI",
            "TERM",
        }
        assert set(kwargs["env"]) <= allowed_environment_keys
        assert "shell" not in kwargs
        assert "private prompt" not in caplog.text
        assert '{"sentence":"ok"}' not in caplog.text

    asyncio.run(exercise())


def test_process_timeout_terminates_then_kills_process_group(
    tmp_path: Path,
) -> None:
    async def exercise() -> None:
        config = config_from_row(database_row(tmp_path))
        process = FakeProcess(
            communicate_error=TimeoutError(),
            wait_errors=[TimeoutError()],
        )
        signals: list[tuple[int, signal.Signals]] = []

        async def create_process(*argv: str, **kwargs):
            return process

        def kill_process_group(pid: int, sent_signal: signal.Signals) -> None:
            signals.append((pid, sent_signal))

        runner = CLIRunner(
            StaticConfigClient(config),
            process_factory=create_process,
            kill_process_group=kill_process_group,
        )

        with pytest.raises(CLIRunnerError, match="超时"):
            await runner.generate("codex", "hello")

        assert signals == [
            (process.pid, signal.SIGTERM),
            (process.pid, signal.SIGKILL),
        ]
        assert process.wait_calls == 2

    asyncio.run(exercise())


def test_process_spawn_oserror_is_mapped_to_safe_runner_error(tmp_path: Path) -> None:
    async def exercise() -> None:
        config = config_from_row(database_row(tmp_path))

        async def create_process(*argv: str, **kwargs):
            raise OSError("private filesystem detail")

        runner = CLIRunner(StaticConfigClient(config), process_factory=create_process)

        with pytest.raises(CLIRunnerError, match="无法启动 CLI") as error:
            await runner.generate("codex", "hello")
        assert "private filesystem detail" not in str(error.value)

    asyncio.run(exercise())


def test_nonzero_exit_never_returns_or_logs_stderr(
    tmp_path: Path,
    caplog: pytest.LogCaptureFixture,
) -> None:
    async def exercise() -> None:
        config = config_from_row(database_row(tmp_path))
        prompt = "private full prompt"
        private_token = "sk-private-token-value"
        stderr = (
            "\x1b[31mFAILED\x1b[0m\x00\n"
            f"prompt={prompt}\n"
            f"Authorization: Bearer {private_token}\n"
            f"API_KEY={private_token}\n"
            f"token: {private_token}\n"
            f"secret={private_token}\n"
            f"details={('x' * 700)}tail-secret-should-not-leak"
        ).encode() + b"\xff"
        process = FakeProcess(returncode=7, stderr=stderr)

        async def create_process(*argv: str, **kwargs):
            return process

        runner = CLIRunner(StaticConfigClient(config), process_factory=create_process)
        with caplog.at_level(logging.INFO, logger="word_agent.cli_runner.service"):
            with pytest.raises(CLIRunnerError) as error:
                await runner.generate("codex", prompt)

        assert str(error.value) == "CLI 执行失败，退出码: 7"
        assert private_token not in caplog.text
        assert prompt not in caplog.text
        assert "FAILED" not in caplog.text
        assert "tail-secret-should-not-leak" not in caplog.text

    asyncio.run(exercise())


def test_gemini_process_uses_deny_all_policy_and_prompt_stdin(tmp_path: Path) -> None:
    async def exercise() -> None:
        prompt = "untrusted sentence prompt"
        config = config_from_row(
            database_row(tmp_path, driver="gemini", model="pro", reasoning_effort="")
        )
        process = FakeProcess(stdout=json.dumps({"response": "safe result"}).encode())

        async def create_process(*argv: str, **kwargs):
            assert prompt not in argv
            assert argv[argv.index("--prompt") + 1] == ""
            assert argv[argv.index("--approval-mode") + 1] == "plan"
            policy_path = Path(argv[argv.index("--policy") + 1])
            assert policy_path.name == "deny-all-policy.toml"
            assert policy_path.read_text(encoding="utf-8") == (
                '[[rule]]\n'
                'toolName = "*"\n'
                'decision = "deny"\n'
                'priority = 999\n'
                'interactive = false\n'
            )
            return process

        runner = CLIRunner(StaticConfigClient(config), process_factory=create_process)
        result = await runner.generate("codex", prompt)

        assert result.content == "safe result"
        assert process.input == prompt.encode("utf-8")

    asyncio.run(exercise())


def test_nonzero_exit_with_empty_stderr_only_returns_exit_code(tmp_path: Path) -> None:
    async def exercise() -> None:
        config = config_from_row(database_row(tmp_path))
        process = FakeProcess(returncode=9, stderr=b"\x00\n\t")

        async def create_process(*argv: str, **kwargs):
            return process

        runner = CLIRunner(StaticConfigClient(config), process_factory=create_process)
        with pytest.raises(CLIRunnerError) as error:
            await runner.generate("codex", "hello")

        assert str(error.value) == "CLI 执行失败，退出码: 9"

    asyncio.run(exercise())


def test_same_provider_requests_are_serialized(tmp_path: Path) -> None:
    async def exercise() -> None:
        config = config_from_row(
            database_row(tmp_path, driver="gemini", model="pro", reasoning_effort="")
        )
        active = 0
        maximum_active = 0

        class BlockingProcess(FakeProcess):
            async def communicate(self, input: bytes | None = None):
                nonlocal active, maximum_active
                active += 1
                maximum_active = max(maximum_active, active)
                await asyncio.sleep(0.01)
                active -= 1
                return json.dumps({"response": "ok"}).encode(), b""

        async def create_process(*argv: str, **kwargs):
            return BlockingProcess()

        runner = CLIRunner(StaticConfigClient(config), process_factory=create_process)

        results = await asyncio.gather(
            runner.generate("codex", "one"),
            runner.generate("codex", "two"),
        )
        assert [result.content for result in results] == ["ok", "ok"]
        assert maximum_active == 1

    asyncio.run(exercise())


def test_queued_request_reloads_active_config_before_spawning(tmp_path: Path) -> None:
    async def exercise() -> None:
        config = config_from_row(database_row(tmp_path))
        first_started = asyncio.Event()
        release_first = asyncio.Event()
        second_load_seen = threading.Event()
        spawn_count = 0

        class SwitchingConfigClient:
            def __init__(self) -> None:
                self.calls = 0
                self.active = True

            def load_active(self, executor_id: str) -> CLIProviderConfig:
                self.calls += 1
                if self.calls == 2:
                    second_load_seen.set()
                if not self.active:
                    raise CLIProviderConfigError("请求的 CLI 不是当前造句执行器")
                return config

        class QueueingProcess(FakeProcess):
            def __init__(self, *, wait_for_release: bool) -> None:
                super().__init__()
                self.wait_for_release = wait_for_release

            async def communicate(self, input: bytes | None = None):
                if self.wait_for_release:
                    first_started.set()
                    await release_first.wait()
                return b"", b""

        async def create_process(*argv: str, **kwargs):
            nonlocal spawn_count
            spawn_count += 1
            last_path = Path(argv[argv.index("--output-last-message") + 1])
            last_path.write_text('{"sentence":"ok"}', encoding="utf-8")
            return QueueingProcess(wait_for_release=spawn_count == 1)

        config_client = SwitchingConfigClient()
        runner = CLIRunner(config_client, process_factory=create_process)
        first = asyncio.create_task(runner.generate("codex", "one"))
        await first_started.wait()
        second = asyncio.create_task(runner.generate("codex", "two"))

        await asyncio.to_thread(second_load_seen.wait, 0.1)
        config_client.active = False
        release_first.set()

        assert (await first).content == '{"sentence":"ok"}'
        with pytest.raises(CLIProviderConfigError, match="不是当前"):
            await second
        assert spawn_count == 1

    asyncio.run(exercise())
