import assert from "node:assert/strict";
import test from "node:test";

import {
  CLI_MODEL_OPTIONS,
  CLI_REASONING_OPTIONS,
  DEFAULT_WORKING_DIRECTORY,
  applyExecutionAIProviderEdit,
  canDeleteAPI,
  canDeleteCLI,
  createDefaultCLIProvider,
  createDefaultExecutionConfig,
  findGoTrimBounds,
  isGoSpaceCodePoint,
  normalizedExecutionAIOrigin,
  trimLikeGo,
  validateExecutionConfig,
} from "../src/features/executionConfig.ts";
import { getExecutionConfig, saveExecutionConfig } from "../src/lib/executionConfigApi.ts";
import type {
  CLIProviderConfigItem,
  ExecutionAIProviderConfigItem,
  ExecutionConfig,
} from "../src/types/executionConfig.ts";

class MemoryStorage implements Storage {
  private values = new Map<string, string>();
  get length() { return this.values.size; }
  clear() { this.values.clear(); }
  getItem(key: string) { return this.values.get(key) ?? null; }
  key(index: number) { return [...this.values.keys()][index] ?? null; }
  removeItem(key: string) { this.values.delete(key); }
  setItem(key: string, value: string) { this.values.set(key, value); }
}

function configuredAPI(
  patch: Partial<ExecutionAIProviderConfigItem> = {},
): ExecutionAIProviderConfigItem {
  return {
    id: "openai",
    label: "OpenAI",
    type: "openai-compatible",
    base_url: "https://api.openai.com/v1",
    api_key: "",
    api_key_configured: true,
    model: "gpt-test",
    max_tokens: 4096,
    ...patch,
  };
}

function codexCLI(patch: Partial<CLIProviderConfigItem> = {}): CLIProviderConfigItem {
  return { ...createDefaultCLIProvider("codex"), ...patch };
}

function validConfig(): ExecutionConfig {
  const api = configuredAPI();
  return {
    active_target: { type: "api", id: api.id },
    api_providers: [api],
    cli_providers: [createDefaultCLIProvider("codex"), createDefaultCLIProvider("gemini")],
  };
}

test("CLI model and reasoning options exactly match the backend allowlists", () => {
  assert.deepEqual(CLI_MODEL_OPTIONS, {
    codex: [
      { label: "gpt-5.6-sol", value: "gpt-5.6-sol" },
      { label: "gpt-5.6-terra", value: "gpt-5.6-terra" },
      { label: "gpt-5.6-luna", value: "gpt-5.6-luna" },
      { label: "gpt-5.5", value: "gpt-5.5" },
      { label: "gpt-5.4", value: "gpt-5.4" },
      { label: "gpt-5.4-mini", value: "gpt-5.4-mini" },
      { label: "gpt-5.3-codex-spark", value: "gpt-5.3-codex-spark" },
    ],
    gemini: [
      { label: "auto", value: "auto" },
      { label: "pro", value: "pro" },
      { label: "flash", value: "flash" },
      { label: "flash-lite", value: "flash-lite" },
    ],
  });
  assert.deepEqual(CLI_REASONING_OPTIONS, [
    { label: "low", value: "low" },
    { label: "medium", value: "medium" },
    { label: "high", value: "high" },
    { label: "xhigh", value: "xhigh" },
  ]);
});

test("default execution config contains exact Codex and Gemini CLI presets without an active target", () => {
  assert.deepEqual(createDefaultExecutionConfig(), {
    active_target: null,
    api_providers: [],
    cli_providers: [
      {
        id: "codex",
        label: "Codex CLI",
        driver: "codex",
        command_path: "/Applications/ChatGPT.app/Contents/Resources/codex",
        model: "gpt-5.6-sol",
        reasoning_effort: "high",
        working_directory: DEFAULT_WORKING_DIRECTORY,
        timeout_seconds: 300,
        enabled: true,
      },
      {
        id: "gemini",
        label: "Gemini CLI",
        driver: "gemini",
        command_path: "/Users/conchi/.npm-global/bin/gemini",
        model: "auto",
        reasoning_effort: "",
        working_directory: DEFAULT_WORKING_DIRECTORY,
        timeout_seconds: 300,
        enabled: true,
      },
    ],
  });
});

test("default CLI IDs avoid collisions with existing IDs", () => {
  assert.equal(createDefaultCLIProvider("codex", ["codex", "codex-2"]).id, "codex-3");
  assert.equal(createDefaultCLIProvider("gemini", new Set(["gemini"])).id, "gemini-2");
  assert.equal(createDefaultCLIProvider("codex", ["\u0085codex\u0085"]).id, "codex-2");
  assert.equal(createDefaultCLIProvider("codex", ["\uFEFFcodex\uFEFF"]).id, "codex");
});

test("Go-style trimming uses a stable explicit whitespace set", () => {
  const goSpaces = "\u0009\u000a\u000b\u000c\u000d\u0020\u0085\u00a0\u1680"
    + "\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a"
    + "\u2028\u2029\u202f\u205f\u3000";
  assert.equal(trimLikeGo(`${goSpaces}value${goSpaces}`), "value");
  assert.equal(trimLikeGo("\uFEFFvalue\uFEFF"), "\uFEFFvalue\uFEFF");

  const internalSpaces = `left${" ".repeat(100_000)}right`;
  let inspections = 0;
  const bounds = findGoTrimBounds(internalSpaces, (codePoint) => {
    inspections += 1;
    return isGoSpaceCodePoint(codePoint);
  });
  assert.deepEqual(bounds, [0, internalSpaces.length]);
  assert.equal(inspections, 2);
  assert.equal(trimLikeGo(internalSpaces), internalSpaces);
});

test("validation rejects missing, unknown, and disabled active targets", () => {
  const config = validConfig();
  assert.match(validateExecutionConfig({ ...config, active_target: null }) ?? "", /执行器/);
  assert.match(
    validateExecutionConfig({ ...config, active_target: { type: "api", id: "missing" } }) ?? "",
    /不存在/,
  );
  assert.match(
    validateExecutionConfig({
      ...config,
      active_target: { type: "cli", id: "codex" },
      cli_providers: config.cli_providers.map((provider) => ({ ...provider, enabled: false })),
    }) ?? "",
    /停用/,
  );
  assert.match(
    validateExecutionConfig({
      ...config,
      active_target: { type: "worker", id: "openai" } as ExecutionConfig["active_target"],
    }) ?? "",
    /类型/,
  );
  assert.match(
    validateExecutionConfig({
      ...config,
      active_target: { type: "api", id: null } as unknown as ExecutionConfig["active_target"],
    }) ?? "",
    /ID/,
  );
});

test("validation rejects duplicate and blank API fields, missing keys, and non-positive max tokens", () => {
  const config = validConfig();
  const invalidProviders: Array<[Partial<ExecutionAIProviderConfigItem>, RegExp]> = [
    [{ id: " " }, /ID/],
    [{ label: " " }, /label/i],
    [{ type: " " }, /type/i],
    [{ base_url: " " }, /base_url/i],
    [{ model: " " }, /model/i],
    [{ max_tokens: 0 }, /max_tokens/i],
    [{ max_tokens: Number.NaN }, /max_tokens/i],
    [{ max_tokens: Number.POSITIVE_INFINITY }, /max_tokens/i],
    [{ api_key_configured: false, api_key: " " }, /API Key/],
  ];
  for (const [patch, expected] of invalidProviders) {
    assert.match(
      validateExecutionConfig({ ...config, api_providers: [configuredAPI(patch)] }) ?? "",
      expected,
    );
  }

  assert.match(
    validateExecutionConfig({
      ...config,
      api_providers: [configuredAPI(), configuredAPI({ label: "Duplicate" })],
    }) ?? "",
    /重复/,
  );
});

test("validation matches Go trimming for IDs, paths, models, and active targets", () => {
  const duplicateID = validConfig();
  duplicateID.api_providers.push(configuredAPI({ id: "\u0085openai\u0085", label: "Duplicate" }));
  assert.match(validateExecutionConfig(duplicateID) ?? "", /重复/);

  const blankID = validConfig();
  blankID.api_providers[0].id = "\u0085";
  assert.match(validateExecutionConfig(blankID) ?? "", /ID/);

  const padded = validConfig();
  padded.active_target = {
    type: "\u0085cli\u0085",
    id: "\u0085codex\u0085",
  } as ExecutionConfig["active_target"];
  padded.cli_providers[0] = codexCLI({
    id: "\u0085codex\u0085",
    driver: "\u0085codex\u0085" as CLIProviderConfigItem["driver"],
    command_path: "\u0085/opt/bin/codex\u0085",
    working_directory: "\u0085/workspace\u0085",
    model: "\u0085gpt-5.6-sol\u0085",
    reasoning_effort: "\u0085high\u0085",
  });
  assert.equal(validateExecutionConfig(padded), null);
  assert.equal(canDeleteCLI(padded, "\u0085codex\u0085"), false);
});

test("validation applies backend-compatible URL safety rules while allowing paths", () => {
  const config = validConfig();
  for (const base_url of [
    "ftp://api.example.com/v1",
    "https://user@example.com/v1",
    "https://@api.example.com/v1",
    "https://api.example.com/v1?tenant=a",
    "https://api.example.com/v1#models",
    "https:///v1",
    "https://api.example.com/%zz",
    "relative/path",
  ]) {
    assert.match(
      validateExecutionConfig({ ...config, api_providers: [configuredAPI({ base_url })] }) ?? "",
      /base_url/i,
      base_url,
    );
  }
  assert.equal(
    validateExecutionConfig({
      ...config,
      api_providers: [configuredAPI({ base_url: "https://api.example.com/custom/v1" })],
    }),
    null,
  );
  for (const base_url of [
    "HTTPS://API.EXAMPLE.COM/custom/v1",
    "https://api.example.com/custom/v1?",
    "https://api.example.com/custom/v1#",
  ]) {
    assert.equal(
      validateExecutionConfig({ ...config, api_providers: [configuredAPI({ base_url })] }),
      null,
      base_url,
    );
  }
});

test("execution API origins match Go net/url differential fixtures", () => {
  const accepted = new Map<string, string>([
    [" https://api.example.com:00443/v1 ", "https://api.example.com:00443"],
    ["https://api.example.com:99999/v1", "https://api.example.com:99999"],
    ["https://999.1.1.1/v1", "https://999.1.1.1:443"],
    ["HTTPS://BÜCHER.例子/v1", "https://bÜcher.例子:443"],
    ["https://b%C3%9Ccher.example/v1", "https://bÜcher.example:443"],
    ["https://%FF.example.com/v1", "https://\uDCFF.example.com:443"],
    ["https://%FE.example.com/v1", "https://\uDCFE.example.com:443"],
    ["https://%E2%82.example/v1", "https://\uDCE2\uDC82.example:443"],
    ["https://�.example.com/v1", "https://�.example.com:443"],
    ["https://İ.example/v1", "https://İ.example:443"],
    ["https://i\u0307.example/v1", "https://i\u0307.example:443"],
    ["https://ΟΣ.example/v1", "https://ΟΣ.example:443"],
    ["https://\u{1C89}.example/v1", "https://\u{1C89}.example:443"],
    ["https://\u{1C8A}.example/v1", "https://\u{1C8A}.example:443"],
    ["https://[::1]/v1", "https://::1:443"],
    [
      "https://[0:0:0:0:0:0:0:1]:00443/v1",
      "https://0:0:0:0:0:0:0:1:00443",
    ],
    ["https://[fe80::1%25en0]:99999/v1", "https://fe80::1%en0:99999"],
    [
      "https://[fe80::1%25en%5B0%5D]:00443/v1",
      "https://fe80::1%en[0]:00443",
    ],
    [
      "https://[fe80::1%25en%3C0%3E]/v1",
      "https://fe80::1%en<0>:443",
    ],
    [
      "https://[fe80::1%25en%220]/v1",
      "https://fe80::1%en\"0:443",
    ],
    [
      "https://[fe80::1%25en<0>]/v1",
      "https://fe80::1%en<0>:443",
    ],
    [
      "https://[fe80::1%25en\"0]/v1",
      "https://fe80::1%en\"0:443",
    ],
    ["https://api.example.com::443/v1", "https://api.example.com::443"],
    ["https://api.example.com:abc:443/v1", "https://api.example.com:abc:443"],
    ["https://api.example.com/v1?", "https://api.example.com:443"],
    ["https://api.example.com/v1#", "https://api.example.com:443"],
  ]);
  for (const [base_url, origin] of accepted) {
    assert.equal(normalizedExecutionAIOrigin(base_url), origin, base_url);
    const config = validConfig();
    config.api_providers[0].base_url = base_url;
    assert.equal(validateExecutionConfig(config), null, base_url);
  }

  assert.equal(
    normalizedExecutionAIOrigin("\u0085https://api.example.com/v1\u0085"),
    normalizedExecutionAIOrigin("https://api.example.com/v1"),
  );
  assert.equal(normalizedExecutionAIOrigin("\uFEFFhttps://api.example.com/v1"), null);

  assert.notEqual(
    normalizedExecutionAIOrigin("https://%E2%82.example/v1"),
    normalizedExecutionAIOrigin("https://%FF.example/v1"),
  );
  assert.notEqual(
    normalizedExecutionAIOrigin("https://%FF.example/v1"),
    normalizedExecutionAIOrigin("https://%FE.example/v1"),
  );
  assert.notEqual(
    normalizedExecutionAIOrigin("https://%FF.example/v1"),
    normalizedExecutionAIOrigin("https://�.example/v1"),
  );
  assert.equal(normalizedExecutionAIOrigin("https://\uDCFF.example/v1"), null);
  assert.equal(normalizedExecutionAIOrigin("https://example.com/\uD800"), null);
  assert.notEqual(
    normalizedExecutionAIOrigin("https://%FF.example/v1"),
    normalizedExecutionAIOrigin("https://\uDCFF.example/v1"),
  );
  assert.notEqual(
    normalizedExecutionAIOrigin("https://İ.example/v1"),
    normalizedExecutionAIOrigin("https://i\u0307.example/v1"),
  );
  assert.notEqual(
    normalizedExecutionAIOrigin("https://\u{1C89}.example/v1"),
    normalizedExecutionAIOrigin("https://\u{1C8A}.example/v1"),
  );

  for (const base_url of [
    "https://api.example.com/v1?q=1",
    "https://api.example.com/v1#fragment",
    "https://user@api.example.com/v1",
    "https://@api.example.com/v1",
    "https://api.example.com:abc/v1",
    "https://[::1/v1",
    "https://[not-ipv6]/v1",
    "https://[fe80::1%25en%80]/v1",
    "https://[fe80::1%25en%FF]/v1",
    "https://[fe80::1%25en%C3%9Cx]/v1",
    "https://api.example.com/%zz",
    "https://api.example.com::abc/v1",
    "https://api.example.com\\evil/v1",
    "https:///v1",
  ]) {
    assert.equal(normalizedExecutionAIOrigin(base_url), null, base_url);
    const config = validConfig();
    config.api_providers[0].base_url = base_url;
    assert.match(validateExecutionConfig(config) ?? "", /base_url/i, base_url);
  }
});

test("validation rejects invalid CLI fields and driver-specific model or reasoning values", () => {
  const config = validConfig();
  const invalidCLIs: Array<[CLIProviderConfigItem, RegExp]> = [
    [codexCLI({ id: " " }), /ID/],
    [codexCLI({ label: " " }), /label/i],
    [codexCLI({ driver: "claude" as CLIProviderConfigItem["driver"] }), /driver/i],
    [codexCLI({ command_path: "bin/codex" }), /command_path/i],
    [codexCLI({ working_directory: "workspace" }), /working_directory/i],
    [codexCLI({ model: "latest" }), /model/i],
    [codexCLI({ reasoning_effort: "extreme" as CLIProviderConfigItem["reasoning_effort"] }), /reasoning_effort/i],
    [codexCLI({ timeout_seconds: 0 }), /timeout_seconds/i],
    [codexCLI({ timeout_seconds: Number.NaN }), /timeout_seconds/i],
    [codexCLI({ timeout_seconds: Number.POSITIVE_INFINITY }), /timeout_seconds/i],
  ];
  for (const [provider, expected] of invalidCLIs) {
    assert.match(
      validateExecutionConfig({ ...config, cli_providers: [provider] }) ?? "",
      expected,
    );
  }

  const gemini = createDefaultCLIProvider("gemini");
  assert.match(
    validateExecutionConfig({
      ...config,
      cli_providers: [{ ...gemini, reasoning_effort: "high" } as CLIProviderConfigItem],
    }) ?? "",
    /reasoning_effort/i,
  );
  assert.match(
    validateExecutionConfig({ ...config, cli_providers: [{ ...gemini, model: "gemini-pro" }] }) ?? "",
    /model/i,
  );
  assert.match(
    validateExecutionConfig({ ...config, cli_providers: [codexCLI(), codexCLI()] }) ?? "",
    /重复/,
  );
});

test("editing an API provider invalidates stored-key state only when ID or normalized origin changes", () => {
  const provider = configuredAPI();
  assert.equal(
    applyExecutionAIProviderEdit(provider, { base_url: "https://API.OPENAI.com:443/v2" })
      .api_key_configured,
    true,
  );
  assert.equal(
    applyExecutionAIProviderEdit(provider, { base_url: "HTTPS://API.OPENAI.COM/v2" })
      .api_key_configured,
    true,
  );
  assert.equal(applyExecutionAIProviderEdit(provider, { id: "renamed" }).api_key_configured, false);
  assert.equal(
    applyExecutionAIProviderEdit(provider, { base_url: "https://other.example/v1" })
      .api_key_configured,
    false,
  );
  assert.equal(
    applyExecutionAIProviderEdit(provider, { base_url: "https://user@api.openai.com/v1" })
      .api_key_configured,
    false,
  );
  assert.equal(
    applyExecutionAIProviderEdit(
      configuredAPI({ base_url: "https://例子.测试/v1" }),
      { base_url: "https://xn--fsqu00a.xn--0zwm56d/v2" },
    ).api_key_configured,
    false,
  );
  assert.equal(
    applyExecutionAIProviderEdit(
      configuredAPI({ base_url: "https://[0:0:0:0:0:0:0:1]/v1" }),
      { base_url: "https://[::1]/v2" },
    ).api_key_configured,
    false,
  );
  assert.equal(
    applyExecutionAIProviderEdit(provider, { base_url: "https://api.openai.com:00443/v2" })
      .api_key_configured,
    false,
  );
  assert.equal(
    applyExecutionAIProviderEdit(
      configuredAPI({ base_url: "https://%E2%82.example/v1" }),
      { base_url: "https://%FF.example/v2" },
    ).api_key_configured,
    false,
  );
  assert.equal(
    applyExecutionAIProviderEdit(
      configuredAPI({ base_url: "https://%FF.example/v1" }),
      { base_url: "https://%FE.example/v2" },
    ).api_key_configured,
    false,
  );
  assert.equal(
    applyExecutionAIProviderEdit(
      configuredAPI({ base_url: "https://api.openai.com/v1" }),
      { base_url: "\uFEFFhttps://api.openai.com/v2" },
    ).api_key_configured,
    false,
  );
  assert.equal(
    applyExecutionAIProviderEdit(
      configuredAPI({ base_url: "https://[fe80::1%25en0]/v1" }),
      { base_url: "https://[fe80::1%25en%FF]/v2" },
    ).api_key_configured,
    false,
  );
  assert.equal(
    applyExecutionAIProviderEdit(
      configuredAPI({ base_url: "https://%FF.example/v1" }),
      { base_url: "https://�.example/v2" },
    ).api_key_configured,
    false,
  );
  assert.equal(
    applyExecutionAIProviderEdit(
      configuredAPI({ base_url: "https://%FF.example/v1" }),
      { base_url: "https://\uDCFF.example/v2" },
    ).api_key_configured,
    false,
  );
  assert.equal(
    applyExecutionAIProviderEdit(
      configuredAPI({ base_url: "https://İ.example/v1" }),
      { base_url: "https://i.example/v2" },
    ).api_key_configured,
    false,
  );
  assert.equal(
    applyExecutionAIProviderEdit(
      configuredAPI({ base_url: "https://\u{1C89}.example/v1" }),
      { base_url: "https://\u{1C8A}.example/v2" },
    ).api_key_configured,
    false,
  );
});

test("validation, provider edits, and saves do not mutate their inputs", async () => {
  const config = validConfig();
  const configSnapshot = structuredClone(config);
  assert.equal(validateExecutionConfig(config), null);
  assert.deepEqual(config, configSnapshot);

  const provider = configuredAPI();
  const patch: Partial<ExecutionAIProviderConfigItem> = {
    base_url: "https://api.openai.com:00443/v2",
  };
  const providerSnapshot = structuredClone(provider);
  const patchSnapshot = structuredClone(patch);
  applyExecutionAIProviderEdit(provider, patch);
  assert.deepEqual(provider, providerSnapshot);
  assert.deepEqual(patch, patchSnapshot);

  const originalFetch = globalThis.fetch;
  globalThis.fetch = async () => new Response(
    JSON.stringify({ code: 0, data: config, msg: "" }),
    { status: 200, headers: { "Content-Type": "application/json" } },
  );
  try {
    await saveExecutionConfig(config);
  } finally {
    globalThis.fetch = originalFetch;
  }
  assert.deepEqual(config, configSnapshot);
});

test("active providers cannot be deleted and API/CLI IDs use separate namespaces", () => {
  const config = validConfig();
  config.cli_providers[0] = { ...config.cli_providers[0], id: "openai" };
  assert.equal(validateExecutionConfig(config), null);
  assert.equal(canDeleteAPI(config, "openai"), false);
  assert.equal(canDeleteCLI(config, "openai"), true);

  config.active_target = { type: "cli", id: "openai" };
  assert.equal(canDeleteAPI(config, "openai"), true);
  assert.equal(canDeleteCLI(config, "openai"), false);
});

test("execution config GET and POST use the exact authenticated JSON contract and strict payload", async () => {
  const originalFetch = globalThis.fetch;
  const originalStorage = globalThis.sessionStorage;
  const originalConsole = { log: console.log, info: console.info, warn: console.warn, error: console.error };
  const storage = new MemoryStorage();
  storage.setItem("word-agent-admin-token", "jwt-token");
  Object.defineProperty(globalThis, "sessionStorage", { configurable: true, value: storage });
  const config = validConfig();
  const apiWithUnknown = { ...config.api_providers[0], ui_draft: "secret-ui-only" };
  const cliWithUnknown = { ...config.cli_providers[0], ui_expanded: true };
  const input = {
    ...config,
    api_providers: [apiWithUnknown],
    cli_providers: [cliWithUnknown],
    ui_selected_tab: "api",
  } as ExecutionConfig;
  const calls: Array<{ input: RequestInfo | URL; init?: RequestInit }> = [];
  const logs: unknown[][] = [];
  console.log = (...args) => { logs.push(args); };
  console.info = (...args) => { logs.push(args); };
  console.warn = (...args) => { logs.push(args); };
  console.error = (...args) => { logs.push(args); };
  globalThis.fetch = async (request, init) => {
    calls.push({ input: request, init });
    return new Response(JSON.stringify({ code: 0, data: config, msg: "" }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    });
  };

  try {
    await getExecutionConfig();
    await saveExecutionConfig(input);
  } finally {
    globalThis.fetch = originalFetch;
    Object.defineProperty(globalThis, "sessionStorage", { configurable: true, value: originalStorage });
    console.log = originalConsole.log;
    console.info = originalConsole.info;
    console.warn = originalConsole.warn;
    console.error = originalConsole.error;
  }

  assert.equal(calls[0].input, "/api/ai/execution-config");
  assert.equal(calls[0].init?.method, undefined);
  assert.equal(new Headers(calls[0].init?.headers).get("x-token"), "jwt-token");
  assert.equal(new Headers(calls[0].init?.headers).get("Content-Type"), "application/json");
  assert.equal(calls[1].input, "/api/ai/execution-config");
  assert.equal(calls[1].init?.method, "POST");
  assert.equal(new Headers(calls[1].init?.headers).get("x-token"), "jwt-token");
  assert.equal(new Headers(calls[1].init?.headers).get("Content-Type"), "application/json");
  assert.deepEqual(JSON.parse(String(calls[1].init?.body)), {
    active_target: { type: "api", id: "openai" },
    api_providers: [{
      id: "openai",
      label: "OpenAI",
      type: "openai-compatible",
      base_url: "https://api.openai.com/v1",
      api_key: "",
      model: "gpt-test",
      max_tokens: 4096,
    }],
    cli_providers: [{
      id: "codex",
      label: "Codex CLI",
      driver: "codex",
      command_path: "/Applications/ChatGPT.app/Contents/Resources/codex",
      model: "gpt-5.6-sol",
      reasoning_effort: "high",
      working_directory: DEFAULT_WORKING_DIRECTORY,
      timeout_seconds: 300,
      enabled: true,
    }],
  });
  assert.deepEqual(logs, []);
});

test("save rejects invalid configs before fetch and 401 errors retain requestJSON behavior", async () => {
  const originalFetch = globalThis.fetch;
  let calls = 0;
  globalThis.fetch = async () => {
    calls += 1;
    return new Response("", { status: 401 });
  };
  try {
    await assert.rejects(() => saveExecutionConfig(createDefaultExecutionConfig()), /执行器/);
    const invalidMaxTokens = validConfig();
    invalidMaxTokens.api_providers[0].max_tokens = Number.NaN;
    await assert.rejects(() => saveExecutionConfig(invalidMaxTokens), /max_tokens/);
    const invalidTimeout = validConfig();
    invalidTimeout.cli_providers[0].timeout_seconds = Number.POSITIVE_INFINITY;
    await assert.rejects(() => saveExecutionConfig(invalidTimeout), /timeout_seconds/);
    assert.equal(calls, 0);
    await assert.rejects(() => getExecutionConfig(), /登录已失效/);
    assert.equal(calls, 1);
  } finally {
    globalThis.fetch = originalFetch;
  }
});
