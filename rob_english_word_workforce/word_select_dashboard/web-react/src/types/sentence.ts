export interface GenerateSentenceRequest {
  words: string[];
}

export interface GenerateSentenceResponse {
  runId: string;
  status: "success" | "failed";
  words: string[];
  sentence: string;
  translationZh: string;
  explanationZh: string;
  providerId: string;
  providerLabel: string;
  model: string;
  durationMs: number;
}

export interface SentenceHistoryItem extends GenerateSentenceResponse {
  createdAt: string;
}

export interface PaginatedResult<T> {
  list: T[];
  total: number;
  page: number;
  pageSize: number;
}
