import type { AIConfig, AIProviderConfigItem } from "../types/aiConfig.ts";

function normalizedOrigin(raw: string): string | null {
  try {
    const parsed = new URL(raw.trim());
    if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
      return null;
    }
    return parsed.origin;
  } catch {
    return null;
  }
}

export function applyAIProviderEdit(
  provider: AIProviderConfigItem,
  patch: Partial<AIProviderConfigItem>,
): AIProviderConfigItem {
  let apiKeyConfigured = provider.api_key_configured;
  if (patch.id !== undefined && patch.id.trim() !== provider.id.trim()) {
    apiKeyConfigured = false;
  }
  if (patch.base_url !== undefined) {
    const previousOrigin = normalizedOrigin(provider.base_url);
    const nextOrigin = normalizedOrigin(patch.base_url);
    if (!previousOrigin || !nextOrigin || previousOrigin !== nextOrigin) {
      apiKeyConfigured = false;
    }
  }
  return { ...provider, ...patch, api_key_configured: apiKeyConfigured };
}

export function validateAIConfig(config: AIConfig): string {
  if (config.providers.length === 0) {
    return "请至少保留一个模型配置";
  }

  const ids = new Set<string>();
  for (const provider of config.providers) {
    const id = provider.id.trim();
    if (!id) {
      return "请填写配置 ID";
    }
    if (ids.has(id)) {
      return `配置 ID「${id}」重复`;
    }
    ids.add(id);
    if (!provider.base_url.trim()) {
      return `请填写「${id}」的 Base URL`;
    }
    if (!provider.model.trim()) {
      return `请填写「${id}」的模型名称`;
    }
    if (!provider.api_key_configured && !provider.api_key.trim()) {
      return `请填写「${id}」的 API Key`;
    }
  }

  if (!config.providers.some((provider) => provider.id === config.active)) {
    return "请选择默认模型配置";
  }
  return "";
}
