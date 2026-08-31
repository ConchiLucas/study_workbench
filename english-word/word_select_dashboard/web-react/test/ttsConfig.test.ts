import assert from "node:assert/strict";
import test from "node:test";

import {
  createDefaultTTSProvider,
  validateTTSConfig,
} from "../src/features/ttsConfig.ts";
import { getTTSConfig, saveTTSConfig } from "../src/lib/ttsConfigApi.ts";
import type { TTSConfig } from "../src/types/ttsConfig.ts";

function validTTSConfig(): TTSConfig {
  return {
    active: "xiaomi-mimo-tts",
    providers: [
      {
        id: "xiaomi-mimo-tts",
        label: "Xiaomi MiMo TTS",
        type: "mimo-tts",
        base_url: "https://api.xiaomimimo.com/v1",
        api_key: "",
        api_key_configured: true,
        model: "mimo-v2.5-tts",
        voice: "Chloe",
        enabled: true,
      },
    ],
  };
}

test("creates a non-secret Xiaomi MiMo provider with a unique ID", () => {
  const provider = createDefaultTTSProvider(["xiaomi-mimo-tts"]);

  assert.equal(provider.id, "xiaomi-mimo-tts-2");
  assert.equal(provider.type, "mimo-tts");
  assert.equal(provider.base_url, "https://api.xiaomimimo.com/v1");
  assert.equal(provider.model, "mimo-v2.5-tts");
  assert.equal(provider.voice, "Chloe");
  assert.equal(provider.api_key, "");
  assert.equal(provider.api_key_configured, false);
});

test("accepts a valid provider whose stored key is configured", () => {
  assert.equal(validateTTSConfig(validTTSConfig()), null);
});

test("rejects duplicate provider IDs", () => {
  const config = validTTSConfig();
  config.providers.push({ ...config.providers[0] });

  assert.match(validateTTSConfig(config) ?? "", /重复/);
});

test("rejects a disabled active provider", () => {
  const config = validTTSConfig();
  config.providers[0].enabled = false;

  assert.match(validateTTSConfig(config) ?? "", /未启用/);
});

test("requires a key for a newly added provider", () => {
  const config = validTTSConfig();
  config.providers[0].api_key_configured = false;

  assert.match(validateTTSConfig(config) ?? "", /API Key/);
});

test("uses dedicated TTS endpoints and keeps blank keys in save payloads", async () => {
  const originalFetch = globalThis.fetch;
  const calls: Array<{ input: RequestInfo | URL; init?: RequestInit }> = [];
  globalThis.fetch = async (input, init) => {
    calls.push({ input, init });
    return new Response(
      JSON.stringify({ code: 0, data: validTTSConfig(), msg: "" }),
      { status: 200, headers: { "Content-Type": "application/json" } },
    );
  };

  try {
    await getTTSConfig();
    await saveTTSConfig(validTTSConfig());
  } finally {
    globalThis.fetch = originalFetch;
  }

  assert.equal(calls[0].input, "/api/tts/config");
  assert.equal(calls[0].init?.method, undefined);
  assert.equal(calls[1].input, "/api/tts/config");
  assert.equal(calls[1].init?.method, "POST");
  const body = JSON.parse(String(calls[1].init?.body)) as TTSConfig;
  assert.equal(body.providers[0].api_key, "");
});
