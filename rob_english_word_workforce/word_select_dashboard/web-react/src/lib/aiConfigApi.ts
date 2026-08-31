import type { AIConfig } from "../types/aiConfig";
import { requestJSON } from "./auth.ts";

export function getAIConfig(): Promise<AIConfig> {
  return requestJSON<AIConfig>("/ai/config");
}

export function saveAIConfig(config: AIConfig): Promise<AIConfig> {
  return requestJSON<AIConfig>("/ai/config", {
    method: "POST",
    body: JSON.stringify(config),
  });
}
