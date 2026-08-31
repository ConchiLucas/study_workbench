import asyncio
import logging

from word_agent.services.wrong_word_strategy import WrongWordStrategyService

logger = logging.getLogger(__name__)


class WrongWordEventProcessor:
    def __init__(self, strategy: WrongWordStrategyService) -> None:
        self._strategy = strategy
        self._wake_event = asyncio.Event()
        self._stopping = False
        self._task: asyncio.Task[None] | None = None
        self._startup_batch_keys: set[str] = set()

    async def start(self) -> None:
        if self._task is not None:
            return
        await asyncio.to_thread(self._strategy.ensure_schema)
        await asyncio.to_thread(self._strategy.recover_stale_processing)
        self._startup_batch_keys = await asyncio.to_thread(
            self._strategy.snapshot_startup_retry_batch_keys
        )
        self._stopping = False
        self._task = asyncio.create_task(self._run(), name="wrong-word-event-processor")
        self._wake_event.set()

    def notify(self) -> None:
        self._wake_event.set()

    async def stop(self) -> None:
        self._stopping = True
        self._wake_event.set()
        if self._task is not None:
            await self._task
            self._task = None

    async def _run(self) -> None:
        while True:
            await self._wake_event.wait()
            self._wake_event.clear()
            if self._stopping:
                return
            try:
                result = await asyncio.to_thread(
                    self._strategy.process_available,
                    startup_batch_keys=self._startup_batch_keys,
                )
            except Exception:
                logger.exception("错词事件处理器本轮执行失败，等待下一次新错题触发")
                continue
            self._startup_batch_keys = set(result.remaining_startup_batch_keys)
            if result.has_more or self._startup_batch_keys:
                self._wake_event.set()
