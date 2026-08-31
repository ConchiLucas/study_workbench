import {
  applyExecutionAIProviderEdit,
  trimLikeGo,
} from "./executionConfig.ts";
import type {
  CLIProviderConfigItem,
  ExecutionAIProviderConfigItem,
  ExecutionConfig,
  ExecutionTarget,
} from "../types/executionConfig.ts";

export interface ResolvedActiveTarget {
  type: "API" | "CLI";
  label: string;
  id: string;
  model: string;
}

export type ExecutionTab = "api" | "cli";
export type ExecutionOperation = "save" | "refresh";

export interface ExecutionOperationTicket {
  operation: ExecutionOperation;
  sequence: number;
}

export type ExecutionRefetchResolution<T> =
  | { status: "error"; error: unknown }
  | { status: "success"; data: T }
  | { status: "empty" };

export interface ExecutionSaveResolution {
  draft: ExecutionConfig | null;
  cleanRevision: number;
  applied: boolean;
}

export function resolveExecutionSaveResult(
  currentDraft: ExecutionConfig | null,
  currentRevision: number,
  currentCleanRevision: number,
  serverResponse: ExecutionConfig,
  submittedRevision: number,
): ExecutionSaveResolution {
  if (currentRevision === submittedRevision) {
    return {
      draft: serverResponse,
      cleanRevision: Math.max(currentCleanRevision, submittedRevision),
      applied: true,
    };
  }
  return {
    draft: currentDraft,
    cleanRevision: Math.max(currentCleanRevision, submittedRevision),
    applied: false,
  };
}

export function resolveExecutionRefetchResult<T>(result: {
  data?: T;
  error?: unknown;
  isError: boolean;
}): ExecutionRefetchResolution<T> {
  if (result.isError || result.error) {
    return { status: "error", error: result.error };
  }
  if (result.data !== undefined) {
    return { status: "success", data: result.data };
  }
  return { status: "empty" };
}

export function beginExecutionOperation(
  current: ExecutionOperationTicket | null,
  currentSequence: number,
  operation: ExecutionOperation,
): ExecutionOperationTicket | null {
  if (current) {
    return null;
  }
  return { operation, sequence: currentSequence + 1 };
}

export function isLatestExecutionOperation(
  current: ExecutionOperationTicket | null,
  expected: ExecutionOperationTicket | null,
): boolean {
  return Boolean(
    current
    && expected
    && current.sequence === expected.sequence
    && current.operation === expected.operation,
  );
}

export function nextExecutionTab(current: ExecutionTab, key: string): ExecutionTab | null {
  if (key === "Home") {
    return "api";
  }
  if (key === "End") {
    return "cli";
  }
  if (key === "ArrowLeft" || key === "ArrowRight") {
    return current === "api" ? "cli" : "api";
  }
  return null;
}

export function resolveActiveTarget(config: ExecutionConfig | null): ResolvedActiveTarget | null {
  const target = config?.active_target;
  if (!config || !target) {
    return null;
  }
  if (target.type === "api") {
    const provider = config.api_providers.find((item) => item.id === target.id);
    return provider ? {
      type: "API",
      label: provider.label,
      id: provider.id,
      model: provider.model,
    } : null;
  }
  const provider = config.cli_providers.find((item) => item.id === target.id);
  return provider ? {
    type: "CLI",
    label: provider.label,
    id: provider.id,
    model: provider.model,
  } : null;
}

export function setExecutionTarget(
  config: ExecutionConfig,
  type: ExecutionTarget["type"],
  id: string,
): ExecutionConfig {
  return { ...config, active_target: { type, id } };
}

export function updateExecutionAPIProvider(
  config: ExecutionConfig,
  index: number,
  patch: Partial<ExecutionAIProviderConfigItem>,
): ExecutionConfig {
  const previous = config.api_providers[index];
  if (!previous) {
    return config;
  }
  const apiProviders = config.api_providers.map((provider, providerIndex) => (
    providerIndex === index ? applyExecutionAIProviderEdit(provider, patch) : provider
  ));
  const activeTarget = patch.id !== undefined
    && config.active_target?.type === "api"
    && config.active_target.id === previous.id
    ? { type: "api" as const, id: patch.id }
    : config.active_target;
  return { ...config, active_target: activeTarget, api_providers: apiProviders };
}

export function applyExecutionCLIProviderEdit(
  provider: CLIProviderConfigItem,
  patch: Partial<CLIProviderConfigItem>,
): CLIProviderConfigItem {
  if (patch.driver !== undefined && patch.driver !== provider.driver) {
    return patch.driver === "codex"
      ? { ...provider, ...patch, model: "gpt-5.6-sol", reasoning_effort: "high" }
      : { ...provider, ...patch, model: "auto", reasoning_effort: "" };
  }
  return { ...provider, ...patch };
}

export function updateExecutionCLIProvider(
  config: ExecutionConfig,
  index: number,
  patch: Partial<CLIProviderConfigItem>,
): ExecutionConfig {
  const previous = config.cli_providers[index];
  if (!previous) {
    return config;
  }
  const cliProviders = config.cli_providers.map((provider, providerIndex) => (
    providerIndex === index ? applyExecutionCLIProviderEdit(provider, patch) : provider
  ));
  const activeTarget = patch.id !== undefined
    && config.active_target?.type === "cli"
    && config.active_target.id === previous.id
    ? { type: "cli" as const, id: patch.id }
    : config.active_target;
  return { ...config, active_target: activeTarget, cli_providers: cliProviders };
}

export function canDisableCLI(config: ExecutionConfig, id: string): boolean {
  const normalizedID = trimLikeGo(id);
  return !(
    config.active_target?.type === "cli"
    && trimLikeGo(config.active_target.id) === normalizedID
  );
}

export function nextExecutionProviderID(
  baseID: string,
  providers: ReadonlyArray<{ id: string }>,
): string {
  const used = new Set(providers.map((provider) => trimLikeGo(provider.id)));
  if (!used.has(baseID)) {
    return baseID;
  }
  let suffix = 2;
  while (used.has(`${baseID}-${suffix}`)) {
    suffix += 1;
  }
  return `${baseID}-${suffix}`;
}
