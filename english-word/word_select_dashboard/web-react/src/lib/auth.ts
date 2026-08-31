interface ApiResponse<T> {
  code: number;
  data: T;
  msg: string;
}

interface LoginResponse {
  token: string;
  expiresAt: number;
  user: unknown;
}

const AUTH_TOKEN_STORAGE_KEY = "word-agent-admin-token";
let authSessionGeneration = 0;
let loginAttemptSequence = 0;
let activeLoginAttempt: number | null = null;

export const AUTH_REQUIRED_EVENT = "word-agent-auth-required";
export const authEvents = new EventTarget();

export interface AuthSessionSnapshot {
  readonly generation: number;
  readonly startedToken: string | null;
  readonly storage: Storage | null;
}

export class LoginAttemptCancelledError extends Error {
  constructor() {
    super("登录已取消");
    this.name = "LoginAttemptCancelledError";
  }
}

export function getAuthSessionGeneration(): number {
  return authSessionGeneration;
}

function browserSessionStorage(): Storage | null {
  try {
    return globalThis.sessionStorage ?? null;
  } catch {
    return null;
  }
}

export function getAuthToken(storage: Storage | null = browserSessionStorage()): string | null {
  return storage?.getItem(AUTH_TOKEN_STORAGE_KEY)?.trim() || null;
}

export function captureAuthSessionSnapshot(
  storage: Storage | null = browserSessionStorage(),
): AuthSessionSnapshot {
  return {
    generation: getAuthSessionGeneration(),
    startedToken: getAuthToken(storage),
    storage,
  };
}

export function isCurrentAuthSession(snapshot: AuthSessionSnapshot): boolean {
  return getAuthSessionGeneration() === snapshot.generation;
}

function canApplyAuthSideEffects(snapshot: AuthSessionSnapshot): boolean {
  return isCurrentAuthSession(snapshot)
    && getAuthToken(snapshot.storage) === snapshot.startedToken;
}

function setAuthToken(token: string, storage: Storage | null = browserSessionStorage()): void {
  storage?.setItem(AUTH_TOKEN_STORAGE_KEY, token);
}

export function clearAuthSession(storage: Storage | null = browserSessionStorage()): void {
  authSessionGeneration += 1;
  storage?.removeItem(AUTH_TOKEN_STORAGE_KEY);
  if (typeof document === "undefined") {
    return;
  }
  document.cookie = "x-token=; Max-Age=0; Path=/; SameSite=Lax";
  const hostname = globalThis.location?.hostname;
  if (hostname) {
    document.cookie = `x-token=; Max-Age=0; Path=/; Domain=${hostname}; SameSite=Lax`;
  }
}

export function cancelPendingLoginAttempt(): void {
  if (activeLoginAttempt === null) {
    return;
  }
  loginAttemptSequence += 1;
  activeLoginAttempt = null;
  authSessionGeneration += 1;
}

export async function requestJSON<T>(
  path: string,
  init?: RequestInit,
  authSnapshot?: AuthSessionSnapshot,
): Promise<T> {
  const storage = authSnapshot?.storage ?? browserSessionStorage();
  const authSideEffectSnapshot = captureAuthSessionSnapshot(storage);
  const startedToken = authSideEffectSnapshot.startedToken;
  const headers = new Headers(init?.headers);
  headers.set("Content-Type", "application/json");
  if (startedToken) {
    headers.set("x-token", startedToken);
  }

  const response = await fetch(`/api${path}`, {
    ...init,
    credentials: "same-origin",
    headers,
  });

  if (response.status === 401) {
    if (canApplyAuthSideEffects(authSideEffectSnapshot)) {
      clearAuthSession(storage);
      authEvents.dispatchEvent(new Event(AUTH_REQUIRED_EVENT));
    }
    throw new Error("登录已失效，请重新登录");
  }
  if (!response.ok) {
    throw new Error(`请求失败 (${response.status})`);
  }

  const refreshedToken = response.headers.get("new-token")?.trim();
  if (refreshedToken && canApplyAuthSideEffects(authSideEffectSnapshot)) {
    setAuthToken(refreshedToken, storage);
  }

  const result = (await response.json()) as ApiResponse<T>;
  if (result.code !== 0) {
    throw new Error(result.msg || "请求失败");
  }
  return result.data;
}

export async function login(username: string, password: string): Promise<void> {
  if (activeLoginAttempt !== null) {
    authSessionGeneration += 1;
  }
  const attempt = loginAttemptSequence + 1;
  loginAttemptSequence = attempt;
  activeLoginAttempt = attempt;
  const storage = browserSessionStorage();
  const startedGeneration = getAuthSessionGeneration();
  const startedToken = getAuthToken(storage);

  try {
    const result = await requestJSON<LoginResponse>("/base/login", {
      method: "POST",
      body: JSON.stringify({ username, password }),
    });
    if (
      activeLoginAttempt !== attempt
      || loginAttemptSequence !== attempt
      || getAuthSessionGeneration() !== startedGeneration
      || getAuthToken(storage) !== startedToken
    ) {
      throw new LoginAttemptCancelledError();
    }
    const token = result.token?.trim();
    if (!token) {
      throw new Error("登录响应缺少 token");
    }
    setAuthToken(token, storage);
    authSessionGeneration += 1;
  } catch (error) {
    if (activeLoginAttempt !== attempt || loginAttemptSequence !== attempt) {
      throw new LoginAttemptCancelledError();
    }
    throw error;
  } finally {
    if (activeLoginAttempt === attempt) {
      activeLoginAttempt = null;
    }
  }
}
