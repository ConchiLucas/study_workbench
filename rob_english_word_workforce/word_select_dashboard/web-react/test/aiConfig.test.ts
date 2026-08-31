import assert from "node:assert/strict";
import test from "node:test";

import { getAIConfig, saveAIConfig } from "../src/lib/aiConfigApi.ts";
import type { AIConfig } from "../src/types/aiConfig.ts";

class MemoryStorage implements Storage {
  private values = new Map<string, string>();
  get length() { return this.values.size; }
  clear() { this.values.clear(); }
  getItem(key: string) { return this.values.get(key) ?? null; }
  key(index: number) { return [...this.values.keys()][index] ?? null; }
  removeItem(key: string) { this.values.delete(key); }
  setItem(key: string, value: string) { this.values.set(key, value); }
}

function maskedAIConfig(): AIConfig {
  return {
    active: "default",
    providers: [{
      id: "default",
      label: "Default",
      type: "openai-compatible",
      base_url: "https://api.openai.com/v1",
      api_key: "",
      api_key_configured: true,
      model: "gpt-test",
      max_tokens: 4096,
    }],
  };
}

test("AI config uses public GET and authenticated blank-key POST contracts", async () => {
  const originalFetch = globalThis.fetch;
  const originalStorage = globalThis.sessionStorage;
  const storage = new MemoryStorage();
  storage.setItem("word-agent-admin-token", "jwt-token");
  Object.defineProperty(globalThis, "sessionStorage", { configurable: true, value: storage });
  const calls: Array<{ input: RequestInfo | URL; init?: RequestInit }> = [];
  globalThis.fetch = async (input, init) => {
    calls.push({ input, init });
    return new Response(JSON.stringify({ code: 0, data: maskedAIConfig(), msg: "" }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    });
  };

  try {
    await getAIConfig();
    await saveAIConfig(maskedAIConfig());
  } finally {
    globalThis.fetch = originalFetch;
    Object.defineProperty(globalThis, "sessionStorage", { configurable: true, value: originalStorage });
  }

  assert.equal(calls[0].input, "/api/ai/config");
  assert.equal(calls[0].init?.method, undefined);
  assert.equal(calls[1].input, "/api/ai/config");
  assert.equal(calls[1].init?.method, "POST");
  assert.equal(new Headers(calls[1].init?.headers).get("x-token"), "jwt-token");
  const saved = JSON.parse(String(calls[1].init?.body)) as AIConfig;
  assert.equal(saved.providers[0].api_key, "");
  assert.equal(saved.providers[0].api_key_configured, true);
});
