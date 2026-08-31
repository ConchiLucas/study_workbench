import assert from "node:assert/strict";
import test from "node:test";

import { applyAIProviderEdit, validateAIConfig } from "../src/features/aiConfig.ts";
import type { AIConfig, AIProviderConfigItem } from "../src/types/aiConfig.ts";

function configuredProvider(): AIProviderConfigItem {
  return {
    id: "default",
    label: "Default",
    type: "openai-compatible",
    base_url: "https://api.openai.com/v1",
    api_key: "",
    api_key_configured: true,
    model: "gpt-test",
    max_tokens: 4096,
  };
}

test("same effective origin keeps configured-key state when only path changes", () => {
  const edited = applyAIProviderEdit(configuredProvider(), {
    base_url: "https://api.openai.com:443/v2",
  });

  assert.equal(edited.api_key_configured, true);
});

test("renaming a provider or changing origin invalidates configured-key state", () => {
  assert.equal(
    applyAIProviderEdit(configuredProvider(), { id: "renamed" }).api_key_configured,
    false,
  );
  assert.equal(
    applyAIProviderEdit(configuredProvider(), { base_url: "https://attacker.example/v1" }).api_key_configured,
    false,
  );
});

test("validation requires a key when no stored key remains bound", () => {
  const provider = { ...configuredProvider(), api_key_configured: false };
  const config: AIConfig = { active: provider.id, providers: [provider] };

  assert.match(validateAIConfig(config), /API Key/);
  provider.api_key = "new-secret";
  assert.equal(validateAIConfig(config), "");
});
