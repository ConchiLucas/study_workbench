import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

import {
  applyExecutionCLIProviderEdit,
  beginExecutionOperation,
  canDisableCLI,
  isLatestExecutionOperation,
  nextExecutionTab,
  resolveActiveTarget,
  resolveExecutionRefetchResult,
  resolveExecutionSaveResult,
  setExecutionTarget,
  updateExecutionAPIProvider,
  updateExecutionCLIProvider,
} from "../src/features/executionConfigPage.ts";
import { canDeleteAPI, canDeleteCLI, createDefaultExecutionConfig } from "../src/features/executionConfig.ts";
import type { ExecutionConfig } from "../src/types/executionConfig.ts";

const appSource = readFileSync(new URL("../src/App.tsx", import.meta.url), "utf8");
const componentSource = readFileSync(new URL("../src/components/ExecutionConfigPage.tsx", import.meta.url), "utf8");
const helperSource = readFileSync(new URL("../src/features/executionConfigPage.ts", import.meta.url), "utf8");
const stylesSource = readFileSync(new URL("../src/styles/app.css", import.meta.url), "utf8");

function configFixture(): ExecutionConfig {
  return {
    active_target: { type: "api", id: "api-main" },
    api_providers: [{
      id: "api-main",
      label: "Main API",
      type: "openai-compatible",
      base_url: "https://api.example.com/v1",
      api_key: "",
      api_key_configured: true,
      model: "gpt-api",
      max_tokens: 4096,
    }],
    cli_providers: [{
      id: "codex-local",
      label: "Local Codex",
      driver: "codex",
      command_path: "/usr/local/bin/codex",
      model: "gpt-5.6-sol",
      reasoning_effort: "high",
      working_directory: "/workspace",
      timeout_seconds: 300,
      enabled: true,
    }],
  };
}

function deferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void;
  const promise = new Promise<T>((resolvePromise) => {
    resolve = resolvePromise;
  });
  return { promise, resolve };
}

test("App replaces the inline AI page with the authenticated execution config page", () => {
  assert.match(appSource, /import ExecutionConfigPage from "\.\/components\/ExecutionConfigPage"/);
  assert.match(appSource, /<ExecutionConfigPage authenticated=\{authenticated\}\s*\/>/);
  assert.doesNotMatch(appSource, /function AIConfigPage\(/);
  assert.doesNotMatch(appSource, /<span>本地 CLI 配置<\/span>/);
});

test("execution page exposes unified status, API and CLI tabs, and accessible controls", () => {
  assert.match(componentSource, /API 与本地 CLI 共用一个全局造句执行器/);
  assert.match(componentSource, /当前造句执行器/);
  assert.match(componentSource, /全局唯一/);
  assert.match(componentSource, /模型 API/);
  assert.match(componentSource, /本地 CLI/);
  assert.match(componentSource, /role="tablist"/);
  assert.match(componentSource, /role="tab"/);
  assert.match(componentSource, /aria-selected=/);
  assert.match(componentSource, /type="button"/);
});

test("CLI tab does not expose manual Codex or Gemini creation controls", () => {
  assert.doesNotMatch(componentSource, /新增 Codex/);
  assert.doesNotMatch(componentSource, /新增 Gemini/);
  assert.doesNotMatch(componentSource, /function addCLIProvider/);
  assert.match(componentSource, /<strong>暂无 CLI 配置<\/strong>/);
  assert.match(componentSource, /onClick=\{deleteCLIProvider\}>删除配置<\/Button>/);
});

test("status resolution depends only on active_target and the exact matching provider", () => {
  const config = configFixture();
  assert.deepEqual(resolveActiveTarget(config), {
    type: "API",
    label: "Main API",
    id: "api-main",
    model: "gpt-api",
  });

  assert.deepEqual(resolveActiveTarget({ ...config, active_target: { type: "cli", id: "codex-local" } }), {
    type: "CLI",
    label: "Local Codex",
    id: "codex-local",
    model: "gpt-5.6-sol",
  });
  assert.equal(resolveActiveTarget({ ...config, active_target: { type: "cli", id: "missing" } }), null);
  assert.equal(resolveActiveTarget({ ...config, active_target: null }), null);
  assert.match(componentSource, /resolveActiveTarget\(authenticated \? draft : null\)/);
});

test("setting current writes one typed target and provider renames keep that target exact", () => {
  const config = configFixture();
  const cliActive = setExecutionTarget(config, "cli", "codex-local");
  assert.deepEqual(cliActive.active_target, { type: "cli", id: "codex-local" });

  const renamedAPI = updateExecutionAPIProvider(config, 0, { id: "api-renamed" });
  assert.deepEqual(renamedAPI.active_target, { type: "api", id: "api-renamed" });
  assert.equal(renamedAPI.api_providers[0].api_key_configured, false);

  const renamedCLI = updateExecutionCLIProvider(cliActive, 0, { id: "cli-renamed" });
  assert.deepEqual(renamedCLI.active_target, { type: "cli", id: "cli-renamed" });
});

test("current providers cannot be deleted or disabled", () => {
  const apiActive = configFixture();
  assert.equal(canDeleteAPI(apiActive, "api-main"), false);
  assert.equal(canDeleteCLI(apiActive, "codex-local"), true);

  const cliActive = setExecutionTarget(apiActive, "cli", "codex-local");
  assert.equal(canDeleteAPI(cliActive, "api-main"), true);
  assert.equal(canDeleteCLI(cliActive, "codex-local"), false);
  assert.equal(canDisableCLI(cliActive, "codex-local"), false);
  assert.equal(canDisableCLI(apiActive, "codex-local"), true);
});

test("switching CLI drivers applies the required safe defaults", () => {
  const codex = configFixture().cli_providers[0];
  const gemini = applyExecutionCLIProviderEdit(codex, { driver: "gemini" });
  assert.equal(gemini.model, "auto");
  assert.equal(gemini.reasoning_effort, "");

  const nextCodex = applyExecutionCLIProviderEdit(gemini, { driver: "codex" });
  assert.equal(nextCodex.model, "gpt-5.6-sol");
  assert.equal(nextCodex.reasoning_effort, "high");
});

test("save responses only replace the draft revision that was submitted", () => {
  const submitted = configFixture();
  const serverResponse = structuredClone(submitted);
  serverResponse.api_providers[0].api_key = "";
  serverResponse.api_providers[0].api_key_configured = true;

  const editedAfterSubmit = updateExecutionAPIProvider(submitted, 0, { label: "Still editing" });
  const staleResolution = resolveExecutionSaveResult(editedAfterSubmit, 8, 6, serverResponse, 7);
  assert.equal(staleResolution.applied, false);
  assert.equal(staleResolution.draft, editedAfterSubmit);
  assert.equal(staleResolution.cleanRevision, 7);

  const currentResolution = resolveExecutionSaveResult(submitted, 7, 6, serverResponse, 7);
  assert.equal(currentResolution.applied, true);
  assert.equal(currentResolution.draft, serverResponse);
  assert.equal(currentResolution.cleanRevision, 7);

  const olderResolution = resolveExecutionSaveResult(editedAfterSubmit, 10, 9, serverResponse, 7);
  assert.equal(olderResolution.applied, false);
  assert.equal(olderResolution.cleanRevision, 9);
});

test("refresh errors win over cached data and never yield hydration data", () => {
  const oldData = configFixture();
  const error = new Error("refresh failed");
  assert.deepEqual(
    resolveExecutionRefetchResult({ data: oldData, error, isError: true }),
    { status: "error", error },
  );
});

test("execution operations are mutually exclusive and stale completions cannot become latest", async () => {
  let current = beginExecutionOperation(null, 0, "save");
  assert.deepEqual(current, { operation: "save", sequence: 1 });
  assert.equal(beginExecutionOperation(current, 1, "refresh"), null);

  const pendingSave = deferred<void>();
  const saveTicket = current;
  const saveWasLatest = pendingSave.promise.then(() => isLatestExecutionOperation(current, saveTicket));

  current = null;
  current = beginExecutionOperation(current, 2, "refresh");
  assert.deepEqual(current, { operation: "refresh", sequence: 3 });
  pendingSave.resolve();
  assert.equal(await saveWasLatest, false);
});

test("tab keyboard navigation wraps and supports Home or End", () => {
  assert.equal(nextExecutionTab("api", "ArrowRight"), "cli");
  assert.equal(nextExecutionTab("cli", "ArrowRight"), "api");
  assert.equal(nextExecutionTab("api", "ArrowLeft"), "cli");
  assert.equal(nextExecutionTab("cli", "ArrowLeft"), "api");
  assert.equal(nextExecutionTab("cli", "Home"), "api");
  assert.equal(nextExecutionTab("api", "End"), "cli");
  assert.equal(nextExecutionTab("api", "Enter"), null);
});

test("execution page uses authenticated execution APIs and validates before save", () => {
  assert.match(componentSource, /queryKey:\s*\["execution-config"\]/);
  assert.match(componentSource, /queryFn: \(\) => getExecutionConfig\(refreshAuthSnapshotRef\.current \?\? undefined\)/);
  assert.match(componentSource, /enabled:\s*authenticated/);
  assert.doesNotMatch(componentSource, /useMutation/);
  assert.match(componentSource, /await saveExecutionConfig\(snapshot, authSnapshot\)/);
  assert.match(componentSource, /validateExecutionConfig\(currentDraft\)/);
  assert.match(componentSource, /createDefaultExecutionConfig\(\)/);
  assert.match(componentSource, /CLI_MODEL_OPTIONS/);
  assert.match(helperSource, /applyExecutionAIProviderEdit/);
  assert.match(componentSource, /API Key 已配置/);
  assert.match(componentSource, /登录后读取\/编辑/);
  assert.deepEqual(createDefaultExecutionConfig().active_target, null);
});

test("logout clears private draft/cache and refresh replaces unsaved local edits", () => {
  assert.match(
    componentSource,
    /if \(!authenticated\) \{[\s\S]*setDraft\(null\);[\s\S]*removeQueries\(\{ queryKey: \["execution-config"\] \}\)/,
  );
  assert.match(componentSource, /const activeTarget = resolveActiveTarget\(authenticated \? draft : null\)/);
  assert.match(componentSource, /const refreshResult = await configQuery\.refetch\(\{ cancelRefetch: true \}\)/);
  assert.match(
    componentSource,
    /const authSnapshot = captureAuthSessionSnapshot\(\);[\s\S]*refreshAuthSnapshotRef\.current = authSnapshot;[\s\S]*const refreshResult = await configQuery\.refetch\(\{ cancelRefetch: true \}\)/,
  );
  assert.match(componentSource, /resolveExecutionRefetchResult\(refreshResult\)/);
  assert.match(componentSource, /onClick=\{handleRefresh\}/);
});

test("draft revisions preserve edits made while a save is in flight", () => {
  assert.match(componentSource, /const draftRevisionRef = useRef\(0\)/);
  assert.match(componentSource, /if \(configQuery\.data && draftRef\.current === null\)/);
  assert.match(componentSource, /function replaceDraftFromUser\(/);
  assert.match(componentSource, /function updateDraft\(/);
  assert.match(componentSource, /draftRevisionRef\.current \+= 1/);
  assert.match(componentSource, /const currentDraft = draftRef\.current/);
  assert.match(componentSource, /let snapshot: ExecutionConfig \| null = structuredClone\(currentDraft\)/);
  assert.match(componentSource, /const submittedRevision = draftRevisionRef\.current/);
  assert.match(componentSource, /await queryClient\.cancelQueries\(\{ queryKey: \["execution-config"\] \}\)/);
  assert.match(componentSource, /resolveExecutionSaveResult\(/);
  assert.match(componentSource, /已保存提交时版本，当前未保存编辑仍保留/);
  assert.match(componentSource, /当前有未保存编辑，请先保存后再刷新/);
  assert.match(componentSource, /refetchOnWindowFocus: false/);
  assert.match(componentSource, /refetchOnReconnect: false/);
  assert.match(componentSource, /refetchOnMount: false/);
  assert.match(componentSource, /captureAuthSessionSnapshot\(\)/);
  assert.match(componentSource, /isCurrentAuthSession\(authSnapshot\)/);
  assert.match(componentSource, /getAuthToken\(\) !== null/);
  assert.match(componentSource, /saveExecutionConfig\(snapshot, authSnapshot\)/);
  assert.match(componentSource, /refreshAuthSnapshotRef\.current = authSnapshot/);
  assert.match(componentSource, /if \(configQuery\.isLoading\) \{\s*return;\s*\}/);
  assert.match(componentSource, /disabled=\{!authenticated \|\| operation !== null \|\| configQuery\.isLoading\}/);
  assert.match(componentSource, /const \[operation, setOperation\] = useState<ExecutionOperation \| null>\(null\)/);
  assert.match(componentSource, /const operationSequenceRef = useRef\(0\)/);
  assert.match(componentSource, /const mountedRef = useRef\(true\)/);
});

test("tabs handle arrow, Home, and End keys and focus the activated tab", () => {
  assert.match(componentSource, /function handleTabKeyDown\(/);
  assert.match(componentSource, /nextExecutionTab\(activeTab, event\.key\)/);
  assert.match(componentSource, /event\.preventDefault\(\)/);
  assert.match(componentSource, /\.current\?\.focus\(\)/);
  assert.equal(componentSource.match(/onKeyDown=\{handleTabKeyDown\}/g)?.length, 2);
});

test("narrow execution layout keeps expanded and collapsed sidebar widths distinct", () => {
  assert.match(
    stylesSource,
    /\.admin-shell:has\(\.execution-active-status\) \{\s*grid-template-columns: 240px minmax\(0, 1fr\)/,
  );
  assert.match(
    stylesSource,
    /\.admin-shell\.sidebar-collapsed:has\(\.execution-active-status\) \{\s*grid-template-columns: 72px minmax\(0, 1fr\)/,
  );
  assert.match(
    stylesSource,
    /\.admin-shell\.sidebar-collapsed:has\(\.execution-active-status\) \.brand-copy/,
  );
});

test("375px execution layout reserves a usable content column behind an overlay sidebar", () => {
  const mobileStyles = stylesSource.slice(stylesSource.indexOf("@media (max-width: 560px)"));
  assert.match(
    mobileStyles,
    /\.admin-shell:has\(\.execution-active-status\) \{[\s\S]*grid-template-columns: 72px minmax\(0, 1fr\)/,
  );
  assert.match(
    mobileStyles,
    /\.admin-shell:has\(\.execution-active-status\):not\(\.sidebar-collapsed\) \.sidebar \{[\s\S]*position: absolute;[\s\S]*width: 240px;[\s\S]*z-index:/,
  );
  assert.match(
    mobileStyles,
    /\.admin-shell\.sidebar-collapsed:has\(\.execution-active-status\) \.sidebar \{[\s\S]*width: 72px/,
  );
  assert.match(
    mobileStyles,
    /\.admin-shell:has\(\.execution-active-status\) \.content-area \{[\s\S]*grid-column: 2;[\s\S]*min-width: 0/,
  );
  assert.match(mobileStyles, /\.execution-config-panel \.provider-buttons \{\s*grid-template-columns: minmax\(0, 1fr\)/);
  assert.match(mobileStyles, /\.execution-header \{\s*padding: 16px 12px/);
  assert.match(appSource, /aria-label=\{sidebarCollapsed \? "展开菜单" : "收起菜单"\}/);
});
