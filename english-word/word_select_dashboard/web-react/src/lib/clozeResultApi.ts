import type { ClozeResultItem, ClozeResultUserSummary, PaginatedResult } from "../types/clozeResult";

interface ApiResponse<T> {
  code: number;
  data: T;
  msg: string;
}

function buildQuery(params: Record<string, string | number | undefined>) {
  const query = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== "") {
      query.set(key, String(value));
    }
  });
  return query.toString();
}

async function requestJSON<T>(path: string): Promise<T> {
  const response = await fetch(`/api${path}`);
  if (!response.ok) {
    throw new Error(`请求失败 (${response.status})`);
  }

  const result = (await response.json()) as ApiResponse<T>;
  if (result.code !== 0) {
    throw new Error(result.msg || "获取数据失败");
  }
  return result.data;
}

export function listClozeResultUsers(params: {
  keyword?: string;
  page?: number;
  pageSize?: number;
}): Promise<PaginatedResult<ClozeResultUserSummary>> {
  const query = buildQuery(params);
  return requestJSON<PaginatedResult<ClozeResultUserSummary>>(`/cloze-results/users${query ? `?${query}` : ""}`);
}

export function listClozeResultItems(params: {
  userId?: number;
  keyword?: string;
  page?: number;
  pageSize?: number;
}): Promise<PaginatedResult<ClozeResultItem>> {
  const query = buildQuery(params);
  return requestJSON<PaginatedResult<ClozeResultItem>>(`/cloze-results/items${query ? `?${query}` : ""}`);
}
