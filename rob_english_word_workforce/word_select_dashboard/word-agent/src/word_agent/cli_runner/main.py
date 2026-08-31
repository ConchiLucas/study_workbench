import argparse
import logging
import os
import sys
from collections.abc import Sequence

import uvicorn

from word_agent.cli_runner.service import (
    CLIRunnerSettings,
    CLIRunnerSettingsError,
    create_app,
)

app = create_app()


def main(argv: Sequence[str] = ()) -> None:
    parser = argparse.ArgumentParser(add_help=False)
    parser.add_argument("--runner-marker", default="")
    parser.parse_args(argv)
    settings = CLIRunnerSettings.from_environment()
    try:
        settings.require_ready()
    except CLIRunnerSettingsError as exc:
        raise SystemExit(str(exc)) from None
    logging.basicConfig(
        level=os.environ.get("WORD_AGENT_CLI_RUNNER_LOG_LEVEL", "INFO"),
        format="%(asctime)s %(levelname)s %(name)s %(message)s",
    )
    uvicorn.run(
        create_app(settings=settings),
        host=os.environ.get("WORD_AGENT_CLI_RUNNER_HOST", "0.0.0.0"),
        port=int(os.environ.get("WORD_AGENT_CLI_RUNNER_PORT", "6018")),
        log_level=os.environ.get("WORD_AGENT_CLI_RUNNER_LOG_LEVEL", "info").lower(),
    )


if __name__ == "__main__":
    main(sys.argv[1:])
