import { validateExecutionConfig } from "../features/executionConfig.ts";
import type { ExecutionConfig } from "../types/executionConfig.ts";
import { requestJSON } from "./auth.ts";
import type { AuthSessionSnapshot } from "./auth.ts";

export function getExecutionConfig(authSnapshot?: AuthSessionSnapshot): Promise<ExecutionConfig> {
  return requestJSON<ExecutionConfig>("/ai/execution-config", undefined, authSnapshot);
}

export function saveExecutionConfig(
  config: ExecutionConfig,
  authSnapshot?: AuthSessionSnapshot,
): Promise<ExecutionConfig> {
  const validationError = validateExecutionConfig(config);
  if (validationError) {
    return Promise.reject(new Error(validationError));
  }

  const payload = {
    active_target: config.active_target && {
      type: config.active_target.type,
      id: config.active_target.id,
    },
    api_providers: config.api_providers.map((provider) => ({
      id: provider.id,
      label: provider.label,
      type: provider.type,
      base_url: provider.base_url,
      api_key: provider.api_key,
      model: provider.model,
      max_tokens: provider.max_tokens,
    })),
    cli_providers: config.cli_providers.map((provider) => ({
      id: provider.id,
      label: provider.label,
      driver: provider.driver,
      command_path: provider.command_path,
      model: provider.model,
      reasoning_effort: provider.reasoning_effort,
      working_directory: provider.working_directory,
      timeout_seconds: provider.timeout_seconds,
      enabled: provider.enabled,
    })),
  };

  return requestJSON<ExecutionConfig>("/ai/execution-config", {
    method: "POST",
    body: JSON.stringify(payload),
  }, authSnapshot);
}
