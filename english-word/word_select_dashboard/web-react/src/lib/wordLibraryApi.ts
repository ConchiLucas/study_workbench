import type {
  PaginatedResult,
  ScoreWordCleanSentencesResponse,
  WordCleanItem,
  WordCleanSentenceItem,
  WordLibraryItem,
  WordLibraryWordItem,
} from "../types/wordLibrary";

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

async function requestJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers);
  if (init?.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  const response = await fetch(`/api${path}`, {
    ...init,
    headers,
  });
  if (!response.ok) {
    throw new Error(`请求失败 (${response.status})`);
  }

  const result = (await response.json()) as ApiResponse<T>;
  if (result.code !== 0) {
    throw new Error(result.msg || "获取数据失败");
  }
  return result.data;
}

export function listWordLibraries(params: {
  keyword?: string;
  page?: number;
  pageSize?: number;
}): Promise<PaginatedResult<WordLibraryItem>> {
  const query = buildQuery(params);
  return requestJSON<PaginatedResult<WordLibraryItem>>(`/word-libraries${query ? `?${query}` : ""}`);
}

export function listWordLibraryWords(params: {
  libraryId: number;
  keyword?: string;
  page?: number;
  pageSize?: number;
  sortBy?: "difficulty" | "frequency";
  sortOrder?: "asc" | "desc";
}): Promise<PaginatedResult<WordLibraryWordItem>> {
  const { libraryId, ...queryParams } = params;
  const query = buildQuery(queryParams);
  return requestJSON<PaginatedResult<WordLibraryWordItem>>(
    `/word-libraries/${libraryId}/words${query ? `?${query}` : ""}`,
  );
}

export function listWordCleanWords(params: {
  keyword?: string;
  page?: number;
  pageSize?: number;
  pepDifficulty?: number;
  sourceGroup?: string;
  difficultyMin?: number;
  difficultyMax?: number;
  sortBy?: "difficulty" | "frequency" | "pepDifficulty" | "sourceDifficulty";
  sortOrder?: "asc" | "desc";
}): Promise<PaginatedResult<WordCleanItem>> {
  const query = buildQuery(params);
  return requestJSON<PaginatedResult<WordCleanItem>>(`/word-libraries/clean-words${query ? `?${query}` : ""}`);
}

export function listWordCleanSentences(wordCleanId: number): Promise<WordCleanSentenceItem[]> {
  return requestJSON<WordCleanSentenceItem[]>(`/word-libraries/clean-words/${wordCleanId}/sentences`);
}

export function scoreWordCleanSentences(params: {
  ids?: number[];
  wordCleanIds?: number[];
  modelNames?: string[];
  judgeModel?: string;
  limit?: number;
  overwrite?: boolean;
}): Promise<ScoreWordCleanSentencesResponse> {
  return requestJSON<ScoreWordCleanSentencesResponse>("/word-libraries/clean-sentences/score", {
    method: "POST",
    body: JSON.stringify(params),
  });
}
