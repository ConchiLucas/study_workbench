import logging

import httpx

from word_agent.core.config import Settings
from word_agent.domain.schemas import StepEvent

logger = logging.getLogger(__name__)


class CallbackClient:
    def __init__(self, settings: Settings) -> None:
        self._settings = settings

    async def post_event(self, callback_url: str | None, event: StepEvent) -> None:
        if not callback_url:
            logger.info("callback url empty, event retained locally: %s", event.step_id)
            return

        payload = event.model_dump(mode="json", by_alias=True)
        timeout = httpx.Timeout(self._settings.callback_timeout_seconds)
        try:
            async with httpx.AsyncClient(timeout=timeout) as client:
                response = await client.post(callback_url, json=payload)
                response.raise_for_status()
        except httpx.HTTPError:
            logger.exception("failed to post step event to callback: %s", event.step_id)
