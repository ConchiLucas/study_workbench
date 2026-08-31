import type { PaginatedResult } from "../types/wordLibrary";
import type { AppUserItem, UserClozeWrongItem, UserMasteredWordItem, UserTrainingRound, UserWrongWordItem } from "../types/user";

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

export function listUsers(params: {
  keyword?: string;
  page?: number;
  pageSize?: number;
}): Promise<PaginatedResult<AppUserItem>> {
  const query = buildQuery(params);
  return requestJSON<PaginatedResult<AppUserItem>>(`/users${query ? `?${query}` : ""}`);
}

export function listUserTrainingResults(
  userId: number,
  mode: "solo_training" | "match" = "solo_training",
): Promise<UserTrainingRound[]> {
  return requestJSON<UserTrainingRound[]>(`/users/${userId}/training-results?mode=${mode}`);
}

export function listUserWrongWords(params: {
  userId?: number;
  keyword?: string;
  page?: number;
  pageSize?: number;
}): Promise<PaginatedResult<UserWrongWordItem>> {
  const query = buildQuery(params);
  return requestJSON<PaginatedResult<UserWrongWordItem>>(`/users/wrong-words${query ? `?${query}` : ""}`);
}

export function listUserClozeWrongWords(params: {
  userId?: number;
  keyword?: string;
  page?: number;
  pageSize?: number;
}): Promise<PaginatedResult<UserClozeWrongItem>> {
  const query = buildQuery(params);
  return requestJSON<PaginatedResult<UserClozeWrongItem>>(`/users/cloze-wrong-words${query ? `?${query}` : ""}`);
}

export function listUserMasteredWords(params: {
  userId?: number;
  keyword?: string;
  status?: "learning" | "mastered" | "";
  page?: number;
  pageSize?: number;
}): Promise<PaginatedResult<UserMasteredWordItem>> {
  const query = buildQuery(params);
  return requestJSON<PaginatedResult<UserMasteredWordItem>>(`/users/mastered-words${query ? `?${query}` : ""}`);
}
