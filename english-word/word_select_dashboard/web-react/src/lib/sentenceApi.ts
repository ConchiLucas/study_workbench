import type { GenerateSentenceRequest, GenerateSentenceResponse, SentenceHistoryItem, PaginatedResult } from "../types/sentence";

interface ApiResponse<T> {
  code: number;
  data: T;
  msg: string;
}

export class SentenceApiError extends Error {
  data?: GenerateSentenceResponse;

  constructor(message: string, data?: GenerateSentenceResponse) {
    super(message);
    this.name = "SentenceApiError";
    this.data = data;
  }
}

export async function generateSentence(request: GenerateSentenceRequest): Promise<GenerateSentenceResponse> {
  const response = await fetch("/api/sentences/generate", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(request),
  });

  if (!response.ok) {
    throw new SentenceApiError(`请求失败 (${response.status})`);
  }

  const result = (await response.json()) as ApiResponse<GenerateSentenceResponse>;
  if (result.code !== 0) {
    throw new SentenceApiError(result.msg || "生成失败", result.data);
  }

  return result.data;
}

export async function listSentenceHistory(page?: number, pageSize?: number): Promise<PaginatedResult<SentenceHistoryItem> | SentenceHistoryItem[]> {
  const url = page !== undefined && pageSize !== undefined
    ? `/api/sentences/history?page=${page}&pageSize=${pageSize}`
    : `/api/sentences/history?limit=5`;
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error(`请求失败 (${response.status})`);
  }

  const result = (await response.json()) as ApiResponse<PaginatedResult<SentenceHistoryItem> | SentenceHistoryItem[]>;
  if (result.code !== 0) {
    throw new Error(result.msg || "获取造句记录失败");
  }

  return result.data;
}
