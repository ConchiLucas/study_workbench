export type TTSProviderType = "mimo-tts";

export interface TTSProviderConfig {
  id: string;
  label: string;
  type: TTSProviderType;
  base_url: string;
  api_key: string;
  api_key_configured: boolean;
  model: string;
  voice: string;
  enabled: boolean;
}

export interface TTSConfig {
  active: string;
  providers: TTSProviderConfig[];
}
