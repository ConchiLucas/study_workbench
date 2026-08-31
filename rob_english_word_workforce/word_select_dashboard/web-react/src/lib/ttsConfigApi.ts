import type { TTSConfig } from "../types/ttsConfig";
import { requestJSON } from "./auth.ts";

export function getTTSConfig(): Promise<TTSConfig> {
  return requestJSON<TTSConfig>("/tts/config");
}

export function saveTTSConfig(config: TTSConfig): Promise<TTSConfig> {
  return requestJSON<TTSConfig>("/tts/config", {
    method: "POST",
    body: JSON.stringify(config),
  });
}
