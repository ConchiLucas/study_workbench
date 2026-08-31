export type AIProviderType = "openai-compatible" | "anthropic-compatible";

export interface AIProviderConfigItem {
  id: string;
  label: string;
  type: AIProviderType;
  base_url: string;
  api_key: string;
  api_key_configured: boolean;
  model: string;
  max_tokens: number;
}

export interface AIConfig {
  active: string;
  providers: AIProviderConfigItem[];
}
