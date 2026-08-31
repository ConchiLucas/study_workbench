import type {
  AuthResponse,
  ClozePracticeAnswerResponse,
  ClozePracticeHistoryItem,
  ClozePracticePreference,
  ClozePracticeStats,
  ClozePracticeTask,
  WrongSentenceDetail,
  WrongSentencePageResponse,
  WrongSentenceQuery,
} from "../types/cloze";

export interface Credentials {
  username: string;
  password: string;
  nickname?: string;
}

export interface DifficultyBatchRequest {
  difficultyGroup: string;
  difficultyLevel: string;
  limit?: number;
}

export class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

async function request<T>(path: string, options: RequestInit = {}, token?: string): Promise<T> {
  const headers = new Headers(options.headers);
  if (!headers.has("Content-Type") && options.body) {
    headers.set("Content-Type", "application/json");
  }
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  const response = await fetch(path, {
    ...options,
    headers,
  });

  if (response.status === 204) {
    return null as T;
  }

  const contentType = response.headers.get("content-type") || "";
  const data = contentType.includes("application/json") ? await response.json() : await response.text();
  if (!response.ok) {
    const message =
      response.status === 401 || response.status === 403
        ? "登录已过期，请重新登录"
        : typeof data === "object" && data && "message" in data
          ? String(data.message)
          : "请求失败";
    throw new ApiError(message, response.status);
  }
  return data as T;
}

export function login(credentials: Credentials): Promise<AuthResponse> {
  return request<AuthResponse>("/api/auth/login", {
    method: "POST",
    body: JSON.stringify({
      username: credentials.username,
      password: credentials.password,
    }),
  });
}

export function register(credentials: Credentials): Promise<AuthResponse> {
  return request<AuthResponse>("/api/auth/register", {
    method: "POST",
    body: JSON.stringify({
      username: credentials.username,
      password: credentials.password,
      nickname: credentials.nickname || credentials.username,
    }),
  });
}

export function getNextTask(token: string): Promise<ClozePracticeTask | null> {
  return request<ClozePracticeTask | null>("/api/cloze-practice/tasks/next", {}, token);
}

export function getDueReviewTasks(token: string, limit = 10): Promise<ClozePracticeTask[]> {
  return request<ClozePracticeTask[]>(`/api/cloze-practice/tasks/review-due?limit=${limit}`, {}, token);
}

export function getClozePreferences(token: string): Promise<ClozePracticePreference> {
  return request<ClozePracticePreference>("/api/cloze-practice/preferences", {}, token);
}

export function updateSoloDifficulty(
  token: string,
  difficultyGroup: string,
  difficultyLevel: string,
): Promise<ClozePracticePreference> {
  return request<ClozePracticePreference>(
    "/api/cloze-practice/preferences/solo-difficulty",
    { method: "PUT", body: JSON.stringify({ difficultyGroup, difficultyLevel }) },
    token,
  );
}

export function getPendingTasks(token: string, limit = 100): Promise<ClozePracticeTask[]> {
  return request<ClozePracticeTask[]>(`/api/cloze-practice/tasks/pending?limit=${limit}`, {}, token);
}

export function getAnsweredTasks(token: string, status: "mastered" | "wrong" | "review", limit = 100): Promise<ClozePracticeTask[]> {
  return request<ClozePracticeTask[]>(`/api/cloze-practice/tasks/answered?status=${status}&limit=${limit}`, {}, token);
}

export function createDifficultyBatch(token: string, payload: DifficultyBatchRequest): Promise<ClozePracticeTask[]> {
  return request<ClozePracticeTask[]>(
    "/api/cloze-practice/tasks/difficulty-batch",
    {
      method: "POST",
      body: JSON.stringify(payload),
    },
    token,
  );
}

export function submitAnswer(
  token: string,
  payload: {
    clozeItemId: number;
    answers: string[];
    answerText: string;
    costMs: number;
    submissionKey?: string;
    practiceContext?: "review" | "solo";
    actionType?: "answer" | "reveal";
  },
): Promise<ClozePracticeAnswerResponse> {
  return request<ClozePracticeAnswerResponse>(
    "/api/cloze-practice/answers",
    {
      method: "POST",
      body: JSON.stringify(payload),
    },
    token,
  );
}

export function getStats(token: string): Promise<ClozePracticeStats> {
  return request<ClozePracticeStats>("/api/cloze-practice/stats", {}, token);
}

export function getHistory(token: string, limit = 20): Promise<ClozePracticeHistoryItem[]> {
  return request<ClozePracticeHistoryItem[]>(`/api/cloze-practice/history?limit=${limit}`, {}, token);
}

export function getWrongSentences(
  token: string,
  query: WrongSentenceQuery,
): Promise<WrongSentencePageResponse> {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) {
    if (value !== undefined && value !== null && value !== "") {
      params.set(key, String(value));
    }
  }
  return request<WrongSentencePageResponse>(
    `/api/cloze-practice/wrong-sentences?${params.toString()}`,
    {},
    token,
  );
}

export function getWrongSentenceDetail(
  token: string,
  progressId: number,
): Promise<WrongSentenceDetail> {
  return request<WrongSentenceDetail>(
    `/api/cloze-practice/wrong-sentences/${progressId}`,
    {},
    token,
  );
}
