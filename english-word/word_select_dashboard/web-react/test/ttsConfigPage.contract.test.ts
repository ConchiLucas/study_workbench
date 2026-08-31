import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const appSource = readFileSync(new URL("../src/App.tsx", import.meta.url), "utf8");

test("keeps text-model and TTS configuration as separate pages", () => {
  assert.match(appSource, /type PageKey[\s\S]*"ai-config"[\s\S]*"tts-config"/);
  assert.match(appSource, /import ExecutionConfigPage from "\.\/components\/ExecutionConfigPage"/);
  assert.match(appSource, /<ExecutionConfigPage authenticated=\{authenticated\}\s*\/>/);
  assert.doesNotMatch(appSource, /function AIConfigPage\(\)/);
  assert.match(appSource, /function TTSConfigPage\(\)/);
  assert.match(appSource, /activePage === "ai-config"/);
  assert.match(appSource, /activePage === "tts-config"/);
});

test("places TTS configuration immediately after model configuration", () => {
  assert.match(
    appSource,
    /onClick=\{\(\) => setActivePage\("ai-config"\)\}[\s\S]*<span>模型配置<\/span>[\s\S]*onClick=\{\(\) => setActivePage\("tts-config"\)\}[\s\S]*<span>TTS 模型配置<\/span>/,
  );
});

test("TTS page uses its dedicated API and complete Xiaomi fields", () => {
  assert.match(appSource, /getTTSConfig/);
  assert.match(appSource, /saveTTSConfig/);
  assert.match(appSource, /createDefaultTTSProvider/);
  assert.match(appSource, /validateTTSConfig/);
  assert.match(appSource, /API Key 已配置/);
  assert.match(appSource, /默认音色/);
  assert.match(appSource, /启用配置/);
  assert.match(appSource, /设为默认/);
  assert.match(appSource, /删除配置/);
  assert.match(appSource, /保存配置/);
});
