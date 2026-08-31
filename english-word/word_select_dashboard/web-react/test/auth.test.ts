import assert from "node:assert/strict";
import test from "node:test";

import {
  AUTH_REQUIRED_EVENT,
  LoginAttemptCancelledError,
  authEvents,
  cancelPendingLoginAttempt,
  captureAuthSessionSnapshot,
  clearAuthSession,
  getAuthSessionGeneration,
  getAuthToken,
  isCurrentAuthSession,
  login,
  requestJSON,
} from "../src/lib/auth.ts";

class MemoryStorage implements Storage {
  private values = new Map<string, string>();
  get length() { return this.values.size; }
  clear() { this.values.clear(); }
  getItem(key: string) { return this.values.get(key) ?? null; }
  key(index: number) { return [...this.values.keys()][index] ?? null; }
  removeItem(key: string) { this.values.delete(key); }
  setItem(key: string, value: string) { this.values.set(key, value); }
}

function deferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise;
    reject = rejectPromise;
  });
  return { promise, resolve, reject };
}

function successfulResponse(data: unknown, headers?: HeadersInit): Response {
  return new Response(JSON.stringify({ code: 0, data, msg: "" }), {
    status: 200,
    headers: { "Content-Type": "application/json", ...headers },
  });
}

test("login stores only the returned token and authenticated requests use x-token", async () => {
  const originalFetch = globalThis.fetch;
  const originalStorage = globalThis.sessionStorage;
  const storage = new MemoryStorage();
  Object.defineProperty(globalThis, "sessionStorage", { configurable: true, value: storage });
  const calls: Array<{ input: RequestInfo | URL; init?: RequestInit }> = [];
  const sessionGeneration = getAuthSessionGeneration();
  globalThis.fetch = async (input, init) => {
    calls.push({ input, init });
    if (String(input).endsWith("/base/login")) {
      return new Response(JSON.stringify({
        code: 0,
        data: { token: "jwt-token", expiresAt: 123, user: { userName: "admin" } },
        msg: "",
      }), { status: 200, headers: { "Content-Type": "application/json" } });
    }
    return new Response(JSON.stringify({ code: 0, data: {}, msg: "" }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    });
  };

  try {
    await login("admin", "password-secret");
    assert.equal(getAuthSessionGeneration(), sessionGeneration + 1);
    await requestJSON("/tts/config", { method: "POST", body: "{}" });
  } finally {
    globalThis.fetch = originalFetch;
    Object.defineProperty(globalThis, "sessionStorage", { configurable: true, value: originalStorage });
  }

  assert.equal(getAuthToken(storage), "jwt-token");
  assert.equal(storage.getItem("password"), null);
  assert.equal(calls[0].input, "/api/base/login");
  assert.deepEqual(JSON.parse(String(calls[0].init?.body)), { username: "admin", password: "password-secret" });
  assert.equal(new Headers(calls[1].init?.headers).get("x-token"), "jwt-token");
});

test("a 401 clears the token and emits the auth-required event", async () => {
  const originalFetch = globalThis.fetch;
  const originalStorage = globalThis.sessionStorage;
  const storage = new MemoryStorage();
  storage.setItem("word-agent-admin-token", "expired-token");
  Object.defineProperty(globalThis, "sessionStorage", { configurable: true, value: storage });
  globalThis.fetch = async () => new Response("", { status: 401 });
  let events = 0;
  const listener = () => { events += 1; };
  authEvents.addEventListener(AUTH_REQUIRED_EVENT, listener);

  try {
    await assert.rejects(() => requestJSON("/ai/config", { method: "POST", body: "{}" }), /登录/);
  } finally {
    authEvents.removeEventListener(AUTH_REQUIRED_EVENT, listener);
    globalThis.fetch = originalFetch;
    Object.defineProperty(globalThis, "sessionStorage", { configurable: true, value: originalStorage });
  }

  assert.equal(getAuthToken(storage), null);
  assert.equal(events, 1);
  clearAuthSession(storage);
});

test("a refreshed JWT response replaces the stored x-token", async () => {
  const originalFetch = globalThis.fetch;
  const originalStorage = globalThis.sessionStorage;
  const storage = new MemoryStorage();
  storage.setItem("word-agent-admin-token", "old-token");
  Object.defineProperty(globalThis, "sessionStorage", { configurable: true, value: storage });
  globalThis.fetch = async () => new Response(
    JSON.stringify({ code: 0, data: {}, msg: "" }),
    {
      status: 200,
      headers: { "Content-Type": "application/json", "new-token": "refreshed-token" },
    },
  );

  try {
    await requestJSON("/tts/config", { method: "POST", body: "{}" });
  } finally {
    globalThis.fetch = originalFetch;
    Object.defineProperty(globalThis, "sessionStorage", { configurable: true, value: originalStorage });
  }

  assert.equal(getAuthToken(storage), "refreshed-token");
});

test("a token rotation keeps the request business snapshot in the same session", async () => {
  const originalFetch = globalThis.fetch;
  const originalStorage = globalThis.sessionStorage;
  const storage = new MemoryStorage();
  storage.setItem("word-agent-admin-token", "old-token");
  Object.defineProperty(globalThis, "sessionStorage", { configurable: true, value: storage });
  globalThis.fetch = async () => successfulResponse({}, { "new-token": "refreshed-token" });

  try {
    const authSnapshot = captureAuthSessionSnapshot();
    await requestJSON("/ai/execution-config", undefined, authSnapshot);

    assert.equal(authSnapshot.startedToken, "old-token");
    assert.equal(isCurrentAuthSession(authSnapshot), true);
  } finally {
    clearAuthSession(storage);
    globalThis.fetch = originalFetch;
    Object.defineProperty(globalThis, "sessionStorage", { configurable: true, value: originalStorage });
  }
});

test("a stale successful response cannot replace a newer session that reused the same token", async () => {
  const originalFetch = globalThis.fetch;
  const originalStorage = globalThis.sessionStorage;
  const storage = new MemoryStorage();
  const staleRequest = deferred<Response>();
  Object.defineProperty(globalThis, "sessionStorage", { configurable: true, value: storage });
  globalThis.fetch = async (input) => {
    const path = String(input);
    if (path.endsWith("/base/login")) {
      return successfulResponse({ token: "shared-token", expiresAt: 123, user: {} });
    }
    return staleRequest.promise;
  };

  try {
    await login("admin-a", "password-a");
    const oldSnapshot = captureAuthSessionSnapshot();
    const oldResponse = requestJSON("/ai/config", undefined, oldSnapshot);
    clearAuthSession();
    await login("admin-b", "password-b");

    staleRequest.resolve(successfulResponse({}, { "new-token": "refreshed-A" }));
    await oldResponse;

    assert.equal(getAuthToken(storage), "shared-token");
    assert.equal(isCurrentAuthSession(oldSnapshot), false);
  } finally {
    clearAuthSession(storage);
    globalThis.fetch = originalFetch;
    Object.defineProperty(globalThis, "sessionStorage", { configurable: true, value: originalStorage });
  }
});

test("a stale 401 cannot clear or notify a newer login session", async () => {
  const originalFetch = globalThis.fetch;
  const originalStorage = globalThis.sessionStorage;
  const storage = new MemoryStorage();
  const staleRequest = deferred<Response>();
  let loginCount = 0;
  let events = 0;
  const listener = () => { events += 1; };
  Object.defineProperty(globalThis, "sessionStorage", { configurable: true, value: storage });
  globalThis.fetch = async (input) => {
    if (String(input).endsWith("/base/login")) {
      loginCount += 1;
      return successfulResponse({ token: `token-${loginCount}`, expiresAt: 123, user: {} });
    }
    return staleRequest.promise;
  };
  authEvents.addEventListener(AUTH_REQUIRED_EVENT, listener);

  try {
    await login("admin-a", "password-a");
    const oldResponse = requestJSON("/ai/config");
    clearAuthSession();
    await login("admin-b", "password-b");

    staleRequest.resolve(new Response("", { status: 401 }));
    await assert.rejects(() => oldResponse, /登录/);

    assert.equal(getAuthToken(storage), "token-2");
    assert.equal(events, 0);
  } finally {
    authEvents.removeEventListener(AUTH_REQUIRED_EVENT, listener);
    clearAuthSession(storage);
    globalThis.fetch = originalFetch;
    Object.defineProperty(globalThis, "sessionStorage", { configurable: true, value: originalStorage });
  }
});

test("only the first concurrent response may refresh an unchanged session token", async () => {
  const originalFetch = globalThis.fetch;
  const originalStorage = globalThis.sessionStorage;
  const storage = new MemoryStorage();
  const requests = [deferred<Response>(), deferred<Response>(), deferred<Response>()];
  let requestIndex = 0;
  let events = 0;
  const listener = () => { events += 1; };
  Object.defineProperty(globalThis, "sessionStorage", { configurable: true, value: storage });
  globalThis.fetch = async (input) => {
    if (String(input).endsWith("/base/login")) {
      return successfulResponse({ token: "old-token", expiresAt: 123, user: {} });
    }
    const request = requests[requestIndex];
    requestIndex += 1;
    return request.promise;
  };
  authEvents.addEventListener(AUTH_REQUIRED_EVENT, listener);

  try {
    await login("admin", "password");
    const firstSnapshot = captureAuthSessionSnapshot();
    const secondSnapshot = captureAuthSessionSnapshot();
    const thirdSnapshot = captureAuthSessionSnapshot();
    const first = requestJSON("/first", undefined, firstSnapshot);
    const second = requestJSON("/second", undefined, secondSnapshot);
    const third = requestJSON("/third", undefined, thirdSnapshot);

    requests[0].resolve(successfulResponse({}, { "new-token": "refreshed-1" }));
    await first;
    requests[1].resolve(successfulResponse({}, { "new-token": "refreshed-2" }));
    await second;
    requests[2].resolve(new Response("", { status: 401 }));
    await assert.rejects(() => third, /登录/);

    assert.equal(getAuthToken(storage), "refreshed-1");
    assert.equal(isCurrentAuthSession(firstSnapshot), true);
    assert.equal(isCurrentAuthSession(secondSnapshot), true);
    assert.equal(isCurrentAuthSession(thirdSnapshot), true);
    assert.equal(events, 0);
  } finally {
    authEvents.removeEventListener(AUTH_REQUIRED_EVENT, listener);
    clearAuthSession(storage);
    globalThis.fetch = originalFetch;
    Object.defineProperty(globalThis, "sessionStorage", { configurable: true, value: originalStorage });
  }
});

test("a concurrent token refresh does not invalidate another successful business result", async () => {
  const originalFetch = globalThis.fetch;
  const originalStorage = globalThis.sessionStorage;
  const storage = new MemoryStorage();
  const requests = [deferred<Response>(), deferred<Response>()];
  let requestIndex = 0;
  Object.defineProperty(globalThis, "sessionStorage", { configurable: true, value: storage });
  globalThis.fetch = async (input) => {
    if (String(input).endsWith("/base/login")) {
      return successfulResponse({ token: "old-token", expiresAt: 123, user: {} });
    }
    const request = requests[requestIndex];
    requestIndex += 1;
    return request.promise;
  };

  try {
    await login("admin", "password");
    const ordinarySnapshot = captureAuthSessionSnapshot();
    const saveSnapshot = captureAuthSessionSnapshot();
    const ordinary = requestJSON("/ordinary", undefined, ordinarySnapshot);
    const save = requestJSON<{ saved: boolean }>("/save", undefined, saveSnapshot);

    requests[0].resolve(successfulResponse({}, { "new-token": "refreshed-token" }));
    await ordinary;
    requests[1].resolve(successfulResponse({ saved: true }));

    assert.deepEqual(await save, { saved: true });
    assert.equal(getAuthToken(storage), "refreshed-token");
    assert.equal(isCurrentAuthSession(saveSnapshot), true);
  } finally {
    clearAuthSession(storage);
    globalThis.fetch = originalFetch;
    Object.defineProperty(globalThis, "sessionStorage", { configurable: true, value: originalStorage });
  }
});

test("cancelling a pending login prevents a late success and success callback", async () => {
  const originalFetch = globalThis.fetch;
  const originalStorage = globalThis.sessionStorage;
  const storage = new MemoryStorage();
  const pendingLogin = deferred<Response>();
  let successes = 0;
  Object.defineProperty(globalThis, "sessionStorage", { configurable: true, value: storage });
  globalThis.fetch = async () => pendingLogin.promise;

  try {
    const attempt = login("admin", "password").then(() => { successes += 1; });
    cancelPendingLoginAttempt();
    pendingLogin.resolve(successfulResponse({ token: "late-token", expiresAt: 123, user: {} }));

    await assert.rejects(() => attempt, LoginAttemptCancelledError);
    assert.equal(getAuthToken(storage), null);
    assert.equal(successes, 0);
  } finally {
    cancelPendingLoginAttempt();
    clearAuthSession(storage);
    globalThis.fetch = originalFetch;
    Object.defineProperty(globalThis, "sessionStorage", { configurable: true, value: originalStorage });
  }
});

test("a later concurrent login wins when its response arrives first", async () => {
  const originalFetch = globalThis.fetch;
  const originalStorage = globalThis.sessionStorage;
  const storage = new MemoryStorage();
  const pendingLogins = [deferred<Response>(), deferred<Response>()];
  let loginIndex = 0;
  Object.defineProperty(globalThis, "sessionStorage", { configurable: true, value: storage });
  globalThis.fetch = async () => {
    const pending = pendingLogins[loginIndex];
    loginIndex += 1;
    return pending.promise;
  };

  try {
    const first = login("admin-a", "password-a");
    const second = login("admin-b", "password-b");
    pendingLogins[1].resolve(successfulResponse({ token: "token-B", expiresAt: 123, user: {} }));
    await second;
    pendingLogins[0].resolve(successfulResponse({ token: "token-A", expiresAt: 123, user: {} }));

    await assert.rejects(() => first, LoginAttemptCancelledError);
    assert.equal(getAuthToken(storage), "token-B");
  } finally {
    cancelPendingLoginAttempt();
    clearAuthSession(storage);
    globalThis.fetch = originalFetch;
    Object.defineProperty(globalThis, "sessionStorage", { configurable: true, value: originalStorage });
  }
});

test("a cancelled pending login ignores a late 401 without an auth-required event", async () => {
  const originalFetch = globalThis.fetch;
  const originalStorage = globalThis.sessionStorage;
  const storage = new MemoryStorage();
  const pendingLogin = deferred<Response>();
  let events = 0;
  const listener = () => { events += 1; };
  Object.defineProperty(globalThis, "sessionStorage", { configurable: true, value: storage });
  globalThis.fetch = async () => pendingLogin.promise;
  authEvents.addEventListener(AUTH_REQUIRED_EVENT, listener);

  try {
    const attempt = login("admin", "password");
    cancelPendingLoginAttempt();
    pendingLogin.resolve(new Response("", { status: 401 }));

    await assert.rejects(() => attempt, LoginAttemptCancelledError);
    assert.equal(events, 0);
    assert.equal(getAuthToken(storage), null);
  } finally {
    authEvents.removeEventListener(AUTH_REQUIRED_EVENT, listener);
    cancelPendingLoginAttempt();
    clearAuthSession(storage);
    globalThis.fetch = originalFetch;
    Object.defineProperty(globalThis, "sessionStorage", { configurable: true, value: originalStorage });
  }
});

test("cancelling after a settled login does not invalidate the successful session", async () => {
  const originalFetch = globalThis.fetch;
  const originalStorage = globalThis.sessionStorage;
  const storage = new MemoryStorage();
  Object.defineProperty(globalThis, "sessionStorage", { configurable: true, value: storage });
  globalThis.fetch = async () => successfulResponse({
    token: "settled-token",
    expiresAt: 123,
    user: {},
  });

  try {
    await login("admin", "password");
    const settledGeneration = getAuthSessionGeneration();
    cancelPendingLoginAttempt();

    assert.equal(getAuthSessionGeneration(), settledGeneration);
    assert.equal(getAuthToken(storage), "settled-token");
  } finally {
    clearAuthSession(storage);
    globalThis.fetch = originalFetch;
    Object.defineProperty(globalThis, "sessionStorage", { configurable: true, value: originalStorage });
  }
});
