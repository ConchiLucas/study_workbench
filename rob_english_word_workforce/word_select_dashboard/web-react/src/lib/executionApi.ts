import type { ExecutionRunRecord } from "../types/execution";
import type { PaginatedResult } from "../types/sentence";

interface ApiResponse<T> {
  code: number;
  data: T;
  msg: string;
}

interface ListExecutionRunsParams {
  businessType?: string;
  keyword?: string;
  page?: number;
  pageSize?: number;
  startedAtFrom?: number;
  startedAtTo?: number;
}

async function requestJSON<T>(path: string): Promise<T> {
  const response = await fetch(`/api${path}`);
  if (!response.ok) {
    throw new Error(`请求失败 (${response.status})`);
  }

  const result = (await response.json()) as ApiResponse<T>;
  if (result.code !== 0) {
    throw new Error(result.msg || "请求失败");
  }
  return result.data;
}

export function listExecutionRuns(params: ListExecutionRunsParams = {}): Promise<PaginatedResult<ExecutionRunRecord>> {
  const searchParams = new URLSearchParams();
  if (params.businessType) {
    searchParams.set("businessType", params.businessType);
  }
  if (params.keyword) {
    searchParams.set("keyword", params.keyword);
  }
  if (params.page) {
    searchParams.set("page", String(params.page));
  }
  if (params.pageSize) {
    searchParams.set("pageSize", String(params.pageSize));
  }
  if (params.startedAtFrom) {
    searchParams.set("startedAtFrom", String(params.startedAtFrom));
  }
  if (params.startedAtTo) {
    searchParams.set("startedAtTo", String(params.startedAtTo));
  }

  const query = searchParams.toString();
  return requestJSON<PaginatedResult<ExecutionRunRecord>>(`/executions/runs${query ? `?${query}` : ""}`);
}
