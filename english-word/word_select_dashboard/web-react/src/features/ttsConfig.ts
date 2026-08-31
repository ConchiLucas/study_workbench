import type { TTSConfig, TTSProviderConfig } from "../types/ttsConfig";

const XIAOMI_MIMO_PROVIDER_ID = "xiaomi-mimo-tts";

export function createDefaultTTSProvider(
  existingIDs: string[] = [],
): TTSProviderConfig {
  const usedIDs = new Set(existingIDs.map((id) => id.trim()));
  let id = XIAOMI_MIMO_PROVIDER_ID;
  let suffix = 2;
  while (usedIDs.has(id)) {
    id = `${XIAOMI_MIMO_PROVIDER_ID}-${suffix}`;
    suffix += 1;
  }

  return {
    id,
    label: "Xiaomi MiMo TTS",
    type: "mimo-tts",
    base_url: "https://api.xiaomimimo.com/v1",
    api_key: "",
    api_key_configured: false,
    model: "mimo-v2.5-tts",
    voice: "Chloe",
    enabled: true,
  };
}

export function validateTTSConfig(config: TTSConfig): string | null {
  if (config.providers.length === 0) {
    return "请至少保留一个 TTS 配置";
  }

  const activeID = config.active.trim();
  if (!activeID) {
    return "请选择默认 TTS 配置";
  }

  const ids = new Set<string>();
  for (const provider of config.providers) {
    const id = provider.id.trim();
    if (!id) {
      return "请填写 TTS 配置 ID";
    }
    if (ids.has(id)) {
      return `TTS 配置 ID「${id}」重复`;
    }
    ids.add(id);

    if (!provider.label.trim()) {
      return `请填写 TTS 配置「${id}」的显示名称`;
    }
    if (provider.type !== "mimo-tts") {
      return `TTS 配置「${id}」的类型不支持`;
    }
    if (!isValidHTTPURL(provider.base_url)) {
      return `TTS 配置「${id}」的 Base URL 无效`;
    }
    if (!provider.model.trim()) {
      return `请填写 TTS 配置「${id}」的模型名称`;
    }
    if (!provider.voice.trim()) {
      return `请填写 TTS 配置「${id}」的默认音色`;
    }
    if (!provider.api_key.trim() && !provider.api_key_configured) {
      return `请填写 TTS 配置「${id}」的 API Key`;
    }
  }

  const activeProvider = config.providers.find(
    (provider) => provider.id.trim() === activeID,
  );
  if (!activeProvider) {
    return `默认 TTS 配置「${activeID}」不存在`;
  }
  if (!activeProvider.enabled) {
    return `默认 TTS 配置「${activeID}」未启用`;
  }
  return null;
}

function isValidHTTPURL(raw: string): boolean {
  try {
    const url = new URL(raw.trim());
    return Boolean(url.host) && (url.protocol === "http:" || url.protocol === "https:");
  } catch {
    return false;
  }
}
