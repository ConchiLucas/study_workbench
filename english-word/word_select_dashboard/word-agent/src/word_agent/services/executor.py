import logging
from collections.abc import Awaitable, Callable

from word_agent.domain.schemas import RunExecutionRequest, StepEvent, StepStatus

logger = logging.getLogger(__name__)

EmitEvent = Callable[[StepEvent], Awaitable[None]]


class WordRunExecutor:
    def __init__(self, emit_event: EmitEvent) -> None:
        self._emit_event = emit_event

    async def execute(self, request: RunExecutionRequest) -> None:
        try:
            await self._emit(
                request,
                step_id="receive_request",
                status=StepStatus.RUNNING,
                message="开始接收 Go 提交的单词任务",
                input_data={"word": request.word, "meaning": request.meaning},
            )
            await self._emit(
                request,
                step_id="receive_request",
                status=StepStatus.SUCCESS,
                message="任务参数校验完成",
                output_data={"metadata": request.metadata},
            )
            await self._emit(
                request,
                step_id="prompt_plan",
                status=StepStatus.RUNNING,
                message="开始规划提示词",
                input_data={"word": request.word, "meaning": request.meaning},
            )
            await self._emit(
                request,
                step_id="llm_call",
                status=StepStatus.SKIPPED,
                message="大模型调用尚未实现，当前为框架占位",
            )
            await self._emit(
                request,
                step_id="result_packaging",
                status=StepStatus.SKIPPED,
                message="结果封装尚未实现，当前为框架占位",
            )
        except Exception:
            logger.exception("word run execution failed: %s", request.run_id)
            await self._emit(
                request,
                step_id="agent_runtime",
                status=StepStatus.FAILED,
                message="Python agent 执行失败",
                error="unexpected runtime error",
            )

    async def _emit(
        self,
        request: RunExecutionRequest,
        *,
        step_id: str,
        status: StepStatus,
        message: str,
        input_data: dict | None = None,
        output_data: dict | None = None,
        error: str | None = None,
    ) -> None:
        event = StepEvent(
            run_id=request.run_id,
            step_id=step_id,
            status=status,
            message=message,
            input_data=input_data or {},
            output_data=output_data or {},
            error=error,
        )
        await self._emit_event(event)
