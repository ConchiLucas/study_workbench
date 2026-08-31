import logging
from contextlib import asynccontextmanager

import uvicorn
from fastapi import FastAPI

from word_agent.api.routes import router
from word_agent.core.config import get_settings
from word_agent.services.wrong_word_processor import WrongWordEventProcessor
from word_agent.services.wrong_word_strategy import WrongWordStrategyService


@asynccontextmanager
async def lifespan(app: FastAPI):
    strategy = WrongWordStrategyService(get_settings())
    processor = WrongWordEventProcessor(strategy)
    app.state.wrong_word_strategy = strategy
    app.state.wrong_word_processor = processor
    await processor.start()
    try:
        yield
    finally:
        await processor.stop()


def create_app() -> FastAPI:
    app = FastAPI(
        title="word-agent",
        version="0.1.0",
        description="Python execution agent called by the Go tracking server.",
        lifespan=lifespan,
    )
    app.include_router(router)
    return app


app = create_app()


def main() -> None:
    settings = get_settings()
    logging.basicConfig(level=settings.log_level)
    uvicorn.run(
        "word_agent.main:app",
        host=settings.host,
        port=settings.port,
        log_level=settings.log_level.lower(),
        reload=False,
    )


if __name__ == "__main__":
    main()
