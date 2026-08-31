export type ExecutionTarget =
  | { type: "api"; id: string }
  | { type: "cli"; id: string };

export type CLIDriver = "codex" | "gemini";

export type CodexReasoningEffort = "low" | "medium" | "high" | "xhigh";

export interface ExecutionAIProviderConfigItem {
  id: string;
  label: string;
  type: string;
  base_url: string;
  api_key: string;
  api_key_configured: boolean;
  model: string;
  max_tokens: number;
}

export interface CLIProviderConfigItem {
  id: string;
  label: string;
  driver: CLIDriver;
  command_path: string;
  model: string;
  reasoning_effort: "" | CodexReasoningEffort;
  working_directory: string;
  timeout_seconds: number;
  enabled: boolean;
}

export interface ExecutionConfig {
  active_target: ExecutionTarget | null;
  api_providers: ExecutionAIProviderConfigItem[];
  cli_providers: CLIProviderConfigItem[];
}
