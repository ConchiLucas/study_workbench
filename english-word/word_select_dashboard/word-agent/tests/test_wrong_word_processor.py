import asyncio

from word_agent.services.wrong_word_processor import WrongWordEventProcessor
from word_agent.services.wrong_word_strategy import DrainResult


async def eventually(predicate) -> None:
    for _ in range(100):
        if predicate():
            return
        await asyncio.sleep(0.01)
    raise AssertionError("processor did not reach expected state")


class FakeStrategyService:
    def __init__(self, results: list[DrainResult] | None = None) -> None:
        self.ensure_calls = 0
        self.recover_calls = 0
        self.process_calls: list[set[str]] = []
        self.results = list(results or [DrainResult(processed_batches=0, has_more=False)])

    def ensure_schema(self) -> None:
        self.ensure_calls += 1

    def recover_stale_processing(self) -> None:
        self.recover_calls += 1

    def snapshot_startup_retry_batch_keys(self) -> set[str]:
        return {"wrong-word-events:47-48-49"}

    def process_available(self, *, startup_batch_keys: set[str]) -> DrainResult:
        self.process_calls.append(set(startup_batch_keys))
        return self.results.pop(0)


def test_processor_runs_once_on_start_and_does_not_poll() -> None:
    async def scenario() -> None:
        service = FakeStrategyService()
        processor = WrongWordEventProcessor(service)

        await processor.start()
        await eventually(lambda: len(service.process_calls) == 1)
        await asyncio.sleep(0.05)

        assert service.ensure_calls == 1
        assert service.recover_calls == 1
        assert service.process_calls == [{"wrong-word-events:47-48-49"}]
        await processor.stop()

    asyncio.run(scenario())


def test_new_event_wakes_processor_once() -> None:
    async def scenario() -> None:
        service = FakeStrategyService(
            [
                DrainResult(processed_batches=1, has_more=False),
                DrainResult(processed_batches=1, has_more=False),
            ]
        )
        processor = WrongWordEventProcessor(service)

        await processor.start()
        await eventually(lambda: len(service.process_calls) == 1)
        processor.notify()
        await eventually(lambda: len(service.process_calls) == 2)
        await asyncio.sleep(0.05)

        assert len(service.process_calls) == 2
        await processor.stop()

    asyncio.run(scenario())


def test_processor_reschedules_only_when_drain_reports_more_work() -> None:
    async def scenario() -> None:
        service = FakeStrategyService(
            [
                DrainResult(processed_batches=10, has_more=True),
                DrainResult(processed_batches=1, has_more=False),
            ]
        )
        processor = WrongWordEventProcessor(service)

        await processor.start()
        await eventually(lambda: len(service.process_calls) == 2)
        await asyncio.sleep(0.05)

        assert len(service.process_calls) == 2
        await processor.stop()

    asyncio.run(scenario())
