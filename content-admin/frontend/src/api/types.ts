export interface CatalogDatabase {
  id: string
  name: string
  type: string
  environment: string
  host: string
  port: number
  database: string
  username: string
  password: string
  parameters?: Record<string, string>
  active: boolean
}

export interface CatalogProvider {
  id: string
  label: string
  type: string
  baseUrl: string
  apiKey: string
  model: string
  maxTokens: number
  voice?: string
  capabilities: string[]
  options: Record<string, string>
  enabled: boolean
  active: boolean
}

export interface CatalogAi {
  active: string
  providers: CatalogProvider[]
}

export interface CatalogCliItem {
  id: string
  label: string
  enabled: boolean
  command: string
  defaultArgs: string[]
  model?: string
  reasoningEffort?: string
  workingDirectory: string
  timeoutSeconds: number
  capabilities: string[]
  active: boolean
}

export interface CatalogCli {
  active: string
  configs: CatalogCliItem[]
}

export interface CatalogMinio {
  configured: boolean
  enabled: boolean
  endpoint: string
  accessKeyId: string
  secretAccessKey: string
  useSsl: boolean
  bucketName: string
  basePath: string
}

export interface CatalogRuntime {
  schemaVersion: string
  generatedAt: string
  json: string
}

export interface WorkspaceConfigCatalog {
  databases: CatalogDatabase[]
  ai: CatalogAi
  localCli: CatalogCli
  objectStorage: CatalogMinio
  imageModels: CatalogAi
  videoModels: CatalogAi
  voiceModels: CatalogAi
  runtime: CatalogRuntime
}

export type ConfigSection =
  | 'databases'
  | 'ai'
  | 'local-cli'
  | 'object-storage'
  | 'image-models'
  | 'video-models'
  | 'voice-models'
  | 'runtime'
