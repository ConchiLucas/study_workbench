import { listExecutionRuns } from "./executionApi";
import type { ExecutionRunRecord } from "../types/execution";
import type { PaginatedResult } from "../types/sentence";
import type { RunStatus, WorkflowRun, WorkflowStep } from "../types/workflow";

const runStatuses = new Set<RunStatus>(["pending", "running", "success", "failed", "skipped", "retrying"]);

function normalizeStatus(status: string): RunStatus {
  return runStatuses.has(status as RunStatus) ? (status as RunStatus) : "failed";
}

function normalizeDate(value: string, fallback: string) {
  if (!value || value.startsWith("0001-01-01")) {
    return fallback;
  }
  return value;
}

function stepDuration(startedAt: string, finishedAt: string) {
  return Math.max(0, new Date(finishedAt).getTime() - new Date(startedAt).getTime());
}

function buildSteps(record: ExecutionRunRecord): WorkflowStep[] {
  const startedAt = normalizeDate(record.startedAt, record.createdAt);
  const finishedAt = normalizeDate(record.finishedAt, startedAt);
  const status = normalizeStatus(record.status);
  const failed = status === "failed";
  const businessTitle = record.title || record.businessType || "未命名业务";

  return [
    {
      id: "receive_request",
      name: "接收任务",
      stage: "input",
      status: "success",
      description: "Go 接收到一个业务执行请求。",
      startedAt,
      finishedAt: startedAt,
      durationMs: 0,
      input: { businessType: record.businessType },
      output: { title: businessTitle },
      logs: ["任务已接收"],
    },
    {
      id: "execute_business",
      name: "执行业务",
      stage: "generation",
      status: failed ? "failed" : "success",
      description: "执行业务自身流程，业务数据由各业务模块单独保存。",
      startedAt,
      finishedAt,
      durationMs: stepDuration(startedAt, finishedAt),
      input: { businessType: record.businessType },
      output: { status },
      error: record.error || undefined,
      logs: failed ? [record.error || "业务执行失败"] : ["业务执行完成"],
    },
    {
      id: "save_result",
      name: "保存记录",
      stage: "delivery",
      status: failed ? "skipped" : "success",
      description: "保存通用执行记录，业务结果由业务记录表承载。",
      startedAt: failed ? undefined : finishedAt,
      finishedAt: failed ? undefined : finishedAt,
      durationMs: failed ? undefined : 0,
      input: {},
      output: { runId: record.runId },
      logs: failed ? [] : ["执行记录已保存"],
    },
  ];
}

function mapExecutionRun(record: ExecutionRunRecord): WorkflowRun {
  const startedAt = normalizeDate(record.startedAt, record.createdAt);
  const finishedAt = normalizeDate(record.finishedAt, startedAt);
  const status = normalizeStatus(record.status);
  const currentStepId = record.currentStepId || (status === "failed" ? "execute_business" : "save_result");
  const title = record.title || record.businessType || "未命名业务";
  const steps = buildSteps(record);

  return {
    id: record.runId,
    businessType: record.businessType || "-",
    title,
    owner: "system",
    status,
    currentStepId,
    summary: record.error || `${title}执行${status === "success" ? "成功" : "结束"}`,
    error: record.error || undefined,
    durationMs: record.durationMs,
    startedAt,
    finishedAt,
    steps,
    edges: [
      { source: "receive_request", target: "execute_business" },
      { source: "execute_business", target: "save_result" },
    ],
    events: [
      {
        id: `${record.runId}-received`,
        stepId: "receive_request",
        level: "info",
        message: "执行任务已提交",
        createdAt: startedAt,
      },
      {
        id: `${record.runId}-finished`,
        stepId: currentStepId,
        level: status === "failed" ? "error" : "info",
        message: status === "failed" ? record.error || "执行失败" : "执行完成",
        createdAt: finishedAt,
      },
    ],
  };
}

export async function listWorkflowRuns(params: {
  keyword?: string;
  page?: number;
  pageSize?: number;
  startedAtFrom?: number;
  startedAtTo?: number;
} = {}): Promise<PaginatedResult<WorkflowRun>> {
  const result = await listExecutionRuns(params);
  return {
    ...result,
    list: result.list.map(mapExecutionRun),
  };
}
