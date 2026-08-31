import {
  CheckCircleOutlined,
  DeleteOutlined,
  PlusOutlined,
  ReloadOutlined,
  SaveOutlined,
} from "@ant-design/icons";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Button, Input, InputNumber, Select, Spin, Switch, message } from "antd";
import { useEffect, useRef, useState } from "react";
import type { KeyboardEvent as ReactKeyboardEvent } from "react";
import {
  CLI_MODEL_OPTIONS,
  CLI_REASONING_OPTIONS,
  canDeleteAPI,
  canDeleteCLI,
  createDefaultExecutionConfig,
  trimLikeGo,
  validateExecutionConfig,
} from "../features/executionConfig";
import {
  beginExecutionOperation,
  canDisableCLI,
  isLatestExecutionOperation,
  nextExecutionTab,
  nextExecutionProviderID,
  resolveActiveTarget,
  resolveExecutionRefetchResult,
  resolveExecutionSaveResult,
  setExecutionTarget,
  updateExecutionAPIProvider,
  updateExecutionCLIProvider,
} from "../features/executionConfigPage";
import type {
  ExecutionOperation,
  ExecutionOperationTicket,
  ExecutionTab,
} from "../features/executionConfigPage";
import { getExecutionConfig, saveExecutionConfig } from "../lib/executionConfigApi";
import {
  captureAuthSessionSnapshot,
  getAuthToken,
  isCurrentAuthSession,
} from "../lib/auth";
import type { AuthSessionSnapshot } from "../lib/auth";
import type {
  CLIDriver,
  CLIProviderConfigItem,
  ExecutionAIProviderConfigItem,
  ExecutionConfig,
} from "../types/executionConfig";

const API_TYPE_OPTIONS = [
  { label: "OpenAI Compatible", value: "openai-compatible" },
  { label: "Anthropic Compatible", value: "anthropic-compatible" },
];

function createAPIProvider(providers: ExecutionAIProviderConfigItem[]): ExecutionAIProviderConfigItem {
  return {
    id: nextExecutionProviderID("api", providers),
    label: "New API",
    type: "openai-compatible",
    base_url: "",
    api_key: "",
    api_key_configured: false,
    model: "",
    max_tokens: 4096,
  };
}

interface ProviderListProps {
  title: string;
  count: number;
  children: React.ReactNode;
  actions: React.ReactNode;
}

function ProviderList({ title, count, children, actions }: ProviderListProps) {
  return (
    <aside className="provider-list-panel">
      <div className="provider-list-title">
        <span>{title}</span>
        <em>{count} 项</em>
      </div>
      <div className="provider-buttons">{children}</div>
      <div className="execution-list-actions">{actions}</div>
    </aside>
  );
}

interface ExecutionConfigPageProps {
  authenticated: boolean;
}

export default function ExecutionConfigPage({ authenticated }: ExecutionConfigPageProps) {
  const [messageApi, contextHolder] = message.useMessage();
  const queryClient = useQueryClient();
  const [draft, setDraft] = useState<ExecutionConfig | null>(null);
  const [activeTab, setActiveTab] = useState<ExecutionTab>("api");
  const [selectedAPIIndex, setSelectedAPIIndex] = useState(0);
  const [selectedCLIIndex, setSelectedCLIIndex] = useState(0);
  const [operation, setOperation] = useState<ExecutionOperation | null>(null);
  const draftRef = useRef<ExecutionConfig | null>(null);
  const draftRevisionRef = useRef(0);
  const cleanRevisionRef = useRef(0);
  const operationRef = useRef<ExecutionOperationTicket | null>(null);
  const operationSequenceRef = useRef(0);
  const mountedRef = useRef(true);
  const authenticatedRef = useRef(authenticated);
  const refreshAuthSnapshotRef = useRef<AuthSessionSnapshot | null>(null);
  const apiTabRef = useRef<HTMLButtonElement>(null);
  const cliTabRef = useRef<HTMLButtonElement>(null);
  authenticatedRef.current = authenticated;
  const configQuery = useQuery({
    queryKey: ["execution-config"],
    queryFn: () => getExecutionConfig(refreshAuthSnapshotRef.current ?? undefined),
    enabled: authenticated,
    retry: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    refetchOnMount: false,
  });

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      operationSequenceRef.current += 1;
      operationRef.current = null;
      refreshAuthSnapshotRef.current = null;
      void queryClient.cancelQueries({ queryKey: ["execution-config"] });
    };
  }, [queryClient]);

  useEffect(() => {
    if (!authenticated) {
      operationSequenceRef.current += 1;
      operationRef.current = null;
      refreshAuthSnapshotRef.current = null;
      setOperation(null);
      void queryClient.cancelQueries({ queryKey: ["execution-config"] });
      draftRef.current = null;
      setDraft(null);
      draftRevisionRef.current = 0;
      cleanRevisionRef.current = 0;
      setSelectedAPIIndex(0);
      setSelectedCLIIndex(0);
      queryClient.removeQueries({ queryKey: ["execution-config"] });
      return;
    }
    if (configQuery.data && draftRef.current === null) {
      hydrateDraftFromServer(configQuery.data);
      setSelectedAPIIndex(0);
      setSelectedCLIIndex(0);
    }
  }, [authenticated, configQuery.data, queryClient]);

  useEffect(() => {
    if (draft && selectedAPIIndex >= draft.api_providers.length) {
      setSelectedAPIIndex(Math.max(0, draft.api_providers.length - 1));
    }
    if (draft && selectedCLIIndex >= draft.cli_providers.length) {
      setSelectedCLIIndex(Math.max(0, draft.cli_providers.length - 1));
    }
  }, [draft, selectedAPIIndex, selectedCLIIndex]);

  const selectedAPI = draft?.api_providers[selectedAPIIndex];
  const selectedCLI = draft?.cli_providers[selectedCLIIndex];
  const activeTarget = resolveActiveTarget(authenticated ? draft : null);
  const hasNoProviders = Boolean(
    draft && draft.api_providers.length === 0 && draft.cli_providers.length === 0,
  );

  function hydrateDraftFromServer(nextDraft: ExecutionConfig) {
    draftRef.current = nextDraft;
    setDraft(nextDraft);
    cleanRevisionRef.current = Math.max(
      cleanRevisionRef.current,
      draftRevisionRef.current,
    );
  }

  function replaceDraftFromUser(nextDraft: ExecutionConfig) {
    draftRevisionRef.current += 1;
    draftRef.current = nextDraft;
    setDraft(nextDraft);
  }

  function updateDraft(updater: (current: ExecutionConfig) => ExecutionConfig) {
    const current = draftRef.current;
    if (!current) {
      return;
    }
    const nextDraft = updater(current);
    if (nextDraft !== current) {
      replaceDraftFromUser(nextDraft);
    }
  }

  function useDefaultDraft() {
    replaceDraftFromUser(createDefaultExecutionConfig());
    setSelectedAPIIndex(0);
    setSelectedCLIIndex(0);
  }

  function startOperation(nextOperation: ExecutionOperation): ExecutionOperationTicket | null {
    if (operation !== null || operationRef.current !== null) {
      return null;
    }
    const ticket = beginExecutionOperation(
      operationRef.current,
      operationSequenceRef.current,
      nextOperation,
    );
    if (!ticket) {
      return null;
    }
    operationSequenceRef.current = ticket.sequence;
    operationRef.current = ticket;
    setOperation(nextOperation);
    return ticket;
  }

  function isCurrentOperation(
    ticket: ExecutionOperationTicket,
    authSnapshot: AuthSessionSnapshot,
  ): boolean {
    return mountedRef.current
      && authenticatedRef.current
      && getAuthToken() !== null
      && isCurrentAuthSession(authSnapshot)
      && isLatestExecutionOperation(operationRef.current, ticket);
  }

  function finishOperation(ticket: ExecutionOperationTicket) {
    if (!isLatestExecutionOperation(operationRef.current, ticket)) {
      return;
    }
    operationRef.current = null;
    if (mountedRef.current) {
      setOperation(null);
    }
  }

  async function handleSave() {
    if (!authenticated) {
      messageApi.warning("请先登录");
      return;
    }
    const currentDraft = draftRef.current;
    if (!currentDraft) {
      messageApi.warning("请先读取或创建配置草稿");
      return;
    }
    const validationError = validateExecutionConfig(currentDraft);
    if (validationError) {
      messageApi.warning(validationError);
      return;
    }
    const ticket = startOperation("save");
    if (!ticket) {
      return;
    }
    const authSnapshot = captureAuthSessionSnapshot();
    const submittedRevision = draftRevisionRef.current;
    let snapshot: ExecutionConfig | null = structuredClone(currentDraft);
    try {
      await queryClient.cancelQueries({ queryKey: ["execution-config"] });
      if (!isCurrentOperation(ticket, authSnapshot) || !snapshot) {
        return;
      }
      const data = await saveExecutionConfig(snapshot, authSnapshot);
      if (!isCurrentOperation(ticket, authSnapshot)) {
        return;
      }
      queryClient.setQueryData(["execution-config"], data);
      const resolution = resolveExecutionSaveResult(
        draftRef.current,
        draftRevisionRef.current,
        cleanRevisionRef.current,
        data,
        submittedRevision,
      );
      cleanRevisionRef.current = resolution.cleanRevision;
      if (resolution.applied && resolution.draft) {
        hydrateDraftFromServer(resolution.draft);
        messageApi.success("保存成功");
      } else {
        messageApi.info("已保存提交时版本，当前未保存编辑仍保留");
      }
    } catch (error) {
      if (isCurrentOperation(ticket, authSnapshot)) {
        messageApi.error(error instanceof Error ? error.message : "保存失败");
      }
    } finally {
      snapshot = null;
      finishOperation(ticket);
    }
  }

  async function handleRefresh() {
    if (!authenticated) {
      return;
    }
    if (configQuery.isLoading) {
      return;
    }
    if (draftRevisionRef.current !== cleanRevisionRef.current) {
      messageApi.warning("当前有未保存编辑，请先保存后再刷新");
      return;
    }
    const ticket = startOperation("refresh");
    if (!ticket) {
      return;
    }
    const authSnapshot = captureAuthSessionSnapshot();
    refreshAuthSnapshotRef.current = authSnapshot;
    const startedRevision = draftRevisionRef.current;
    try {
      const refreshResult = await configQuery.refetch({ cancelRefetch: true });
      if (!isCurrentOperation(ticket, authSnapshot)) {
        return;
      }
      const resolution = resolveExecutionRefetchResult(refreshResult);
      if (resolution.status === "error") {
        messageApi.error(
          resolution.error instanceof Error ? resolution.error.message : "刷新失败",
        );
        return;
      }
      if (resolution.status === "success") {
        if (draftRevisionRef.current === startedRevision) {
          hydrateDraftFromServer(resolution.data);
          setSelectedAPIIndex(0);
          setSelectedCLIIndex(0);
        } else {
          messageApi.info("刷新完成，当前未保存编辑仍保留");
        }
      } else {
        messageApi.error("刷新失败");
      }
    } catch (error) {
      if (isCurrentOperation(ticket, authSnapshot)) {
        messageApi.error(error instanceof Error ? error.message : "刷新失败");
      }
    } finally {
      if (refreshAuthSnapshotRef.current === authSnapshot) {
        refreshAuthSnapshotRef.current = null;
      }
      finishOperation(ticket);
    }
  }

  function addAPIProvider() {
    const current = draftRef.current;
    if (!current) {
      return;
    }
    const provider = createAPIProvider(current.api_providers);
    setSelectedAPIIndex(current.api_providers.length);
    updateDraft((config) => ({
      ...config,
      api_providers: [...config.api_providers, provider],
    }));
  }

  function updateAPIProvider(patch: Partial<ExecutionAIProviderConfigItem>) {
    updateDraft((current) => updateExecutionAPIProvider(current, selectedAPIIndex, patch));
  }

  function setCurrentAPI() {
    if (!draft || !selectedAPI || !trimLikeGo(selectedAPI.id)) {
      messageApi.warning("请先填写 API 配置 ID");
      return;
    }
    updateDraft((current) => setExecutionTarget(current, "api", selectedAPI.id));
  }

  function deleteAPIProvider() {
    if (!draft || !selectedAPI) {
      return;
    }
    if (!canDeleteAPI(draft, selectedAPI.id)) {
      messageApi.warning("当前造句执行器不能删除，请先切换到其他配置");
      return;
    }
    updateDraft((current) => ({
      ...current,
      api_providers: current.api_providers.filter((_, index) => index !== selectedAPIIndex),
    }));
    setSelectedAPIIndex(Math.max(0, selectedAPIIndex - 1));
  }

  function updateCLIProvider(patch: Partial<CLIProviderConfigItem>) {
    if (!draft || !selectedCLI) {
      return;
    }
    if (patch.enabled === false && !canDisableCLI(draft, selectedCLI.id)) {
      messageApi.warning("当前 CLI 造句执行器不能停用，请先切换到其他配置");
      return;
    }
    updateDraft((current) => updateExecutionCLIProvider(current, selectedCLIIndex, patch));
  }

  function setCurrentCLI() {
    if (!draft || !selectedCLI || !trimLikeGo(selectedCLI.id)) {
      messageApi.warning("请先填写 CLI 配置 ID");
      return;
    }
    if (!selectedCLI.enabled) {
      messageApi.warning("停用的 CLI 配置不能设为当前造句执行器");
      return;
    }
    updateDraft((current) => setExecutionTarget(current, "cli", selectedCLI.id));
  }

  function deleteCurrentProvider() {
    if (activeTab === "api") {
      deleteAPIProvider();
      return;
    }
    deleteCLIProvider();
  }

  function deleteCLIProvider() {
    if (!draft || !selectedCLI) {
      return;
    }
    if (!canDeleteCLI(draft, selectedCLI.id)) {
      messageApi.warning("当前造句执行器不能删除，请先切换到其他配置");
      return;
    }
    updateDraft((current) => ({
      ...current,
      cli_providers: current.cli_providers.filter((_, index) => index !== selectedCLIIndex),
    }));
    setSelectedCLIIndex(Math.max(0, selectedCLIIndex - 1));
  }

  function handleTabKeyDown(event: ReactKeyboardEvent<HTMLButtonElement>) {
    const nextTab = nextExecutionTab(activeTab, event.key);
    if (!nextTab) {
      return;
    }
    event.preventDefault();
    setActiveTab(nextTab);
    const targetRef = nextTab === "api" ? apiTabRef : cliTabRef;
    targetRef.current?.focus();
  }

  function renderAPIConfig() {
    if (!draft) {
      return null;
    }
    return (
      <section
        id="execution-api-panel"
        className="config-panel execution-config-panel"
        role="tabpanel"
        aria-labelledby="execution-api-tab"
      >
        <ProviderList
          title="API 配置列表"
          count={draft.api_providers.length}
          actions={(
            <Button icon={<PlusOutlined />} onClick={addAPIProvider} block>
              新增 API
            </Button>
          )}
        >
          {draft.api_providers.map((provider, index) => {
            const isCurrent = draft.active_target?.type === "api" && draft.active_target.id === provider.id;
            return (
              <button
                type="button"
                key={`${provider.id || "empty"}-${index}`}
                className={`provider-row ${index === selectedAPIIndex ? "active" : ""}`}
                aria-pressed={index === selectedAPIIndex}
                onClick={() => setSelectedAPIIndex(index)}
              >
                <span className="provider-name">{provider.label || provider.id || "未命名 API"}</span>
                <span className="provider-meta">{provider.model || provider.type}</span>
                {isCurrent ? (
                  <span className="execution-current-badge"><CheckCircleOutlined /> 当前</span>
                ) : null}
              </button>
            );
          })}
        </ProviderList>

        <section className="provider-form-panel">
          {selectedAPI ? (
            <>
              <div className="provider-form-title">
                <div>
                  <strong>{selectedAPI.label || selectedAPI.id || "未命名 API"}</strong>
                  <span>
                    {draft.active_target?.type === "api" && draft.active_target.id === selectedAPI.id
                      ? "当前全局造句执行器"
                      : "可设为全局造句执行器"}
                  </span>
                </div>
                <Button icon={<CheckCircleOutlined />} onClick={setCurrentAPI}>
                  设为当前
                </Button>
              </div>

              <div className="config-grid">
                <label className="config-field">
                  <span>配置 ID</span>
                  <Input value={selectedAPI.id} onChange={(event) => updateAPIProvider({ id: event.target.value })} />
                </label>
                <label className="config-field">
                  <span>显示名称</span>
                  <Input value={selectedAPI.label} onChange={(event) => updateAPIProvider({ label: event.target.value })} />
                </label>
                <label className="config-field">
                  <span>接口类型</span>
                  <Select
                    options={API_TYPE_OPTIONS}
                    value={selectedAPI.type}
                    onChange={(type) => updateAPIProvider({ type })}
                  />
                </label>
                <label className="config-field">
                  <span>Max Tokens</span>
                  <InputNumber
                    min={1}
                    precision={0}
                    value={selectedAPI.max_tokens}
                    onChange={(value) => updateAPIProvider({ max_tokens: Number(value) || 0 })}
                  />
                </label>
                <label className="config-field wide">
                  <span>Base URL</span>
                  <Input
                    placeholder="https://api.openai.com/v1"
                    value={selectedAPI.base_url}
                    onChange={(event) => updateAPIProvider({ base_url: event.target.value })}
                  />
                </label>
                <label className="config-field wide">
                  <span>API Key{selectedAPI.api_key_configured ? " · API Key 已配置" : ""}</span>
                  <Input.Password
                    autoComplete="new-password"
                    placeholder={selectedAPI.api_key_configured ? "留空表示保持现有 API Key" : "请输入 API Key"}
                    value={selectedAPI.api_key}
                    onChange={(event) => updateAPIProvider({ api_key: event.target.value })}
                  />
                </label>
                <label className="config-field wide">
                  <span>模型名称</span>
                  <Input value={selectedAPI.model} onChange={(event) => updateAPIProvider({ model: event.target.value })} />
                </label>
              </div>

              <div className="config-footer">
                <span>修改 ID 或 Base URL 会要求重新提供对应 API Key。</span>
                <Button type="primary" icon={<SaveOutlined />} disabled={!authenticated || !draft || operation !== null} loading={operation === "save"} onClick={handleSave}>保存配置</Button>
              </div>
            </>
          ) : (
            <div className="execution-inline-empty">
              <strong>还没有 API 配置</strong>
              <span>新增 API 后填写连接信息；不会自动设为当前执行器。</span>
              <Button icon={<PlusOutlined />} onClick={addAPIProvider}>新增 API</Button>
            </div>
          )}
        </section>
      </section>
    );
  }

  function renderCLIConfig() {
    if (!draft) {
      return null;
    }
    return (
      <section
        id="execution-cli-panel"
        className="config-panel execution-config-panel"
        role="tabpanel"
        aria-labelledby="execution-cli-tab"
      >
        <ProviderList
          title="CLI 配置列表"
          count={draft.cli_providers.length}
          actions={null}
        >
          {draft.cli_providers.map((provider, index) => {
            const isCurrent = draft.active_target?.type === "cli" && draft.active_target.id === provider.id;
            return (
              <button
                type="button"
                key={`${provider.id || "empty"}-${index}`}
                className={`provider-row ${index === selectedCLIIndex ? "active" : ""}`}
                aria-pressed={index === selectedCLIIndex}
                onClick={() => setSelectedCLIIndex(index)}
              >
                <span className="provider-name">{provider.label || provider.id || "未命名 CLI"}</span>
                <span className="provider-meta">
                  {provider.driver} · {provider.model}{provider.enabled ? "" : " · 已停用"}
                </span>
                {isCurrent ? (
                  <span className="execution-current-badge"><CheckCircleOutlined /> 当前</span>
                ) : null}
              </button>
            );
          })}
        </ProviderList>

        <section className="provider-form-panel">
          {selectedCLI ? (
            <>
              <div className="provider-form-title">
                <div>
                  <strong>{selectedCLI.label || selectedCLI.id || "未命名 CLI"}</strong>
                  <span>
                    {draft.active_target?.type === "cli" && draft.active_target.id === selectedCLI.id
                      ? "当前全局造句执行器"
                      : selectedCLI.enabled ? "可设为全局造句执行器" : "当前配置已停用"}
                  </span>
                </div>
                <Button icon={<CheckCircleOutlined />} onClick={setCurrentCLI}>设为当前</Button>
              </div>

              <div className="config-grid">
                <label className="config-field">
                  <span>配置 ID</span>
                  <Input value={selectedCLI.id} onChange={(event) => updateCLIProvider({ id: event.target.value })} />
                </label>
                <label className="config-field">
                  <span>显示名称</span>
                  <Input value={selectedCLI.label} onChange={(event) => updateCLIProvider({ label: event.target.value })} />
                </label>
                <label className="config-field">
                  <span>CLI Driver</span>
                  <Select
                    options={[
                      { label: "Codex", value: "codex" },
                      { label: "Gemini", value: "gemini" },
                    ]}
                    value={selectedCLI.driver}
                    onChange={(driver: CLIDriver) => updateCLIProvider({ driver })}
                  />
                </label>
                <label className="config-field">
                  <span>启用配置</span>
                  <Switch
                    checked={selectedCLI.enabled}
                    checkedChildren="启用"
                    unCheckedChildren="停用"
                    onChange={(enabled) => updateCLIProvider({ enabled })}
                  />
                </label>
                <label className="config-field wide">
                  <span>命令路径</span>
                  <Input
                    value={selectedCLI.command_path}
                    onChange={(event) => updateCLIProvider({ command_path: event.target.value })}
                  />
                </label>
                <label className="config-field">
                  <span>模型</span>
                  <Select
                    options={[...CLI_MODEL_OPTIONS[selectedCLI.driver]]}
                    value={selectedCLI.model}
                    onChange={(model) => updateCLIProvider({ model })}
                  />
                </label>
                <label className="config-field">
                  <span>Reasoning Effort</span>
                  <Select
                    disabled={selectedCLI.driver === "gemini"}
                    options={selectedCLI.driver === "codex" ? [...CLI_REASONING_OPTIONS] : []}
                    placeholder={selectedCLI.driver === "gemini" ? "Gemini CLI 不使用 reasoning effort" : undefined}
                    value={selectedCLI.driver === "codex" ? selectedCLI.reasoning_effort || undefined : undefined}
                    onChange={(reasoning_effort) => updateCLIProvider({ reasoning_effort })}
                  />
                  {selectedCLI.driver === "gemini" ? <small>Gemini 必须保持为空。</small> : null}
                </label>
                <label className="config-field wide">
                  <span>工作目录</span>
                  <Input
                    value={selectedCLI.working_directory}
                    onChange={(event) => updateCLIProvider({ working_directory: event.target.value })}
                  />
                </label>
                <label className="config-field">
                  <span>超时（秒）</span>
                  <InputNumber
                    min={1}
                    precision={0}
                    value={selectedCLI.timeout_seconds}
                    onChange={(value) => updateCLIProvider({ timeout_seconds: Number(value) || 0 })}
                  />
                </label>
              </div>

              <div className="config-footer">
                <span>切换 Driver 会恢复该 CLI 的安全默认模型与 reasoning 配置。</span>
                <Button type="primary" icon={<SaveOutlined />} disabled={!authenticated || !draft || operation !== null} loading={operation === "save"} onClick={handleSave}>保存配置</Button>
              </div>
            </>
          ) : (
            <div className="execution-inline-empty">
              <strong>暂无 CLI 配置</strong>
            </div>
          )}
        </section>
      </section>
    );
  }

  return (
    <>
      {contextHolder}
      <header className="content-header execution-header">
        <div>
          <h1>模型配置</h1>
          <p>API 与本地 CLI 共用一个全局造句执行器</p>
        </div>
        <div className="content-actions">
          <Button
            danger
            icon={<DeleteOutlined />}
            disabled={!authenticated || !draft || operation !== null || (activeTab === "api" ? draft?.api_providers.length === 0 : draft?.cli_providers.length === 0)}
            onClick={deleteCurrentProvider}
          >
            删除配置
          </Button>
          <Button
            icon={<ReloadOutlined />}
            disabled={!authenticated || operation !== null || configQuery.isLoading}
            loading={operation === "refresh"}
            onClick={handleRefresh}
          >
            刷新
          </Button>
        </div>
      </header>

      <section className="execution-active-status" aria-live="polite">
        <div className="execution-status-icon"><CheckCircleOutlined /></div>
        <div className="execution-status-copy">
          <strong>当前造句执行器 <em>全局唯一</em></strong>
          {activeTarget ? (
            <span>
              <b>{activeTarget.type}</b>
              <span>{activeTarget.label || "未命名"}</span>
              <code>{activeTarget.id}</code>
              <span>{activeTarget.model || "未填写模型"}</span>
            </span>
          ) : (
            <span>尚未选择</span>
          )}
        </div>
      </section>

      {!authenticated ? (
        <div className="empty-state execution-empty-state">
          <strong>登录后读取/编辑</strong>
          <span>当前未登录，不会主动请求私有配置接口。</span>
        </div>
      ) : configQuery.isLoading && !draft ? (
        <div className="loading-area"><Spin /></div>
      ) : !draft ? (
        <div className="empty-state execution-empty-state">
          <strong>暂时无法读取 Execution Config</strong>
          <span>{configQuery.error instanceof Error ? configQuery.error.message : "请确认 Go server 已启动"}</span>
          <Button onClick={useDefaultDraft}>使用默认草稿</Button>
        </div>
      ) : (
        <>
          {hasNoProviders ? (
            <div className="execution-default-prompt">
              <span>当前配置为空，可从空白开始，或建立 Codex + Gemini 默认草稿（不会自动激活）。</span>
              <Button onClick={useDefaultDraft}>使用默认草稿</Button>
            </div>
          ) : null}
          <div className="execution-tabs" role="tablist" aria-label="造句执行器类型">
            <button
              ref={apiTabRef}
              id="execution-api-tab"
              type="button"
              role="tab"
              aria-selected={activeTab === "api"}
              aria-controls="execution-api-panel"
              tabIndex={activeTab === "api" ? 0 : -1}
              className={activeTab === "api" ? "active" : ""}
              onClick={() => setActiveTab("api")}
              onKeyDown={handleTabKeyDown}
            >
              模型 API
            </button>
            <button
              ref={cliTabRef}
              id="execution-cli-tab"
              type="button"
              role="tab"
              aria-selected={activeTab === "cli"}
              aria-controls="execution-cli-panel"
              tabIndex={activeTab === "cli" ? 0 : -1}
              className={activeTab === "cli" ? "active" : ""}
              onClick={() => setActiveTab("cli")}
              onKeyDown={handleTabKeyDown}
            >
              本地 CLI
            </button>
          </div>
          {activeTab === "api" ? renderAPIConfig() : renderCLIConfig()}
        </>
      )}
    </>
  );
}
