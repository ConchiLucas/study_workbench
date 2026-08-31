import type {
  CatalogCliItem,
  CatalogDatabase,
  CatalogMinio,
  CatalogProvider,
  ConfigSection,
  WorkspaceConfigCatalog,
} from '../../api/types'

export function ConfigSectionView({
  section,
  catalog,
}: {
  section: ConfigSection
  catalog: WorkspaceConfigCatalog
}) {
  if (section === 'object-storage') {
    return <MinioView value={catalog.objectStorage} />
  }
  if (section === 'runtime') {
    return (
      <div className="runtime-panel">
        <div className="runtime-meta">
          <span>schemaVersion {catalog.runtime.schemaVersion || '—'}</span>
          <span>generatedAt {catalog.runtime.generatedAt || '—'}</span>
        </div>
        <pre>{catalog.runtime.json || '{}'}</pre>
      </div>
    )
  }

  const items = listItems(catalog, section)
  if (items.length === 0) {
    return <div className="empty-panel">{emptyCopy(section)}</div>
  }

  return (
    <div className="card-list">
      <div className="section-toolbar">
        <h2>{toolbarTitle(section)}</h2>
        <span>{items.length} 个配置</span>
      </div>
      {items.map((item) => (
        <fieldset key={item.id} className={`config-card${item.active ? ' is-active' : ''}`}>
          <legend>{item.legend}</legend>
          {item.active ? <div className="active-tag">当前启用</div> : null}
          <div className="field-grid">
            {item.fields.map((field) => (
              <ReadonlyField key={field.label} label={field.label} value={field.value} wide={field.wide} />
            ))}
          </div>
          {item.strip?.length ? (
            <div className="chip-strip">
              {item.strip.map((chip) => (
                <span key={chip}>{chip}</span>
              ))}
            </div>
          ) : null}
        </fieldset>
      ))}
    </div>
  )
}

function MinioView({ value }: { value: CatalogMinio }) {
  if (!value.configured) {
    return <div className="empty-panel">暂未配置 MinIO</div>
  }
  return (
    <fieldset className="config-card">
      <legend>S3 COMPATIBLE CONTRACT</legend>
      <div className="field-grid">
        <ReadonlyField label="Endpoint" value={value.endpoint} />
        <ReadonlyField label="Bucket Name" value={value.bucketName} />
        <ReadonlyField label="Access Key ID · 明文" value={value.accessKeyId} />
        <ReadonlyField label="Secret Access Key · 明文" value={value.secretAccessKey} />
        <ReadonlyField label="Base Path · 可选" value={value.basePath} wide />
        <ReadonlyField label="使用 SSL" value={value.useSsl ? '是' : '否'} />
        <ReadonlyField label="Enabled" value={value.enabled ? '是' : '否'} />
      </div>
    </fieldset>
  )
}

function ReadonlyField({
  label,
  value,
  wide,
}: {
  label: string
  value?: string | number | null
  wide?: boolean
}) {
  const display = value === undefined || value === null || value === '' ? '—' : String(value)
  return (
    <label className={`field${wide ? ' is-wide' : ''}`}>
      <span>{label}</span>
      <input readOnly value={display} />
    </label>
  )
}

type CardItem = {
  id: string
  legend: string
  active: boolean
  fields: { label: string; value?: string | number | null; wide?: boolean }[]
  strip?: string[]
}

function listItems(catalog: WorkspaceConfigCatalog, section: ConfigSection): CardItem[] {
  if (section === 'databases') {
    return catalog.databases.map((item, index) => toDatabaseItem(item, index))
  }
  if (section === 'ai') {
    return catalog.ai.providers.map((item, index) => toProviderItem(item, index, 'Provider'))
  }
  if (section === 'local-cli') {
    return catalog.localCli.configs.map((item, index) => toCliItem(item, index))
  }
  if (section === 'image-models') {
    return catalog.imageModels.providers.map((item, index) => toProviderItem(item, index, 'Image'))
  }
  if (section === 'video-models') {
    return (catalog.videoModels?.providers ?? []).map((item, index) =>
      toProviderItem(item, index, 'Video'),
    )
  }
  if (section === 'voice-models') {
    return (catalog.voiceModels?.providers ?? []).map((item, index) =>
      toProviderItem(item, index, 'TTS'),
    )
  }
  return []
}

function toDatabaseItem(item: CatalogDatabase, index: number): CardItem {
  return {
    id: item.id || `db-${index}`,
    legend: `Connection ${String(index + 1).padStart(2, '0')}`,
    active: item.active,
    fields: [
      { label: 'ID', value: item.id },
      { label: 'Name', value: item.name },
      { label: 'Type', value: item.type },
      { label: 'Environment', value: item.environment },
      { label: 'Host', value: item.host },
      { label: 'Port', value: item.port },
      { label: 'Database', value: item.database },
      { label: 'Username', value: item.username },
      { label: 'Password · 明文', value: item.password },
    ],
  }
}

function toProviderItem(item: CatalogProvider, index: number, prefix: string): CardItem {
  return {
    id: item.id || `${prefix}-${index}`,
    legend: `${prefix} ${String(index + 1).padStart(2, '0')}`,
    active: item.active,
    fields: [
      { label: 'Provider ID', value: item.id },
      { label: 'Display Name', value: item.label },
      { label: 'Provider Type', value: item.type },
      { label: 'Base URL', value: item.baseUrl, wide: true },
      { label: 'Model', value: item.model },
      { label: 'Max Tokens', value: item.maxTokens },
      { label: 'API Key · 明文', value: item.apiKey },
      { label: 'Voice · 可选', value: item.voice },
      { label: 'Enabled', value: item.enabled ? '是' : '否' },
    ],
    strip: item.capabilities ?? [],
  }
}

function toCliItem(item: CatalogCliItem, index: number): CardItem {
  return {
    id: item.id || `cli-${index}`,
    legend: `CLI ${String(index + 1).padStart(2, '0')}`,
    active: item.active,
    fields: [
      { label: 'ID', value: item.id },
      { label: 'Label', value: item.label },
      { label: 'Command', value: item.command, wide: true },
      { label: 'Default Args', value: (item.defaultArgs ?? []).join(' ') },
      { label: 'Working Directory', value: item.workingDirectory, wide: true },
      { label: 'Model', value: item.model },
      { label: 'Reasoning Effort', value: item.reasoningEffort },
      { label: 'Timeout Seconds', value: item.timeoutSeconds },
      { label: 'Enabled', value: item.enabled ? '是' : '否' },
    ],
    strip: item.capabilities ?? [],
  }
}

function toolbarTitle(section: ConfigSection) {
  if (section === 'databases') return 'Connections'
  if (section === 'local-cli') return 'CLI Configs'
  if (section === 'image-models') return 'Image Providers'
  if (section === 'video-models') return 'Video Providers'
  if (section === 'voice-models') return 'TTS Providers'
  return 'Providers'
}

function emptyCopy(section: ConfigSection) {
  if (section === 'databases') return '尚未配置数据库连接。请到配置中心新增。'
  if (section === 'local-cli') return '尚未配置本地 CLI。请到配置中心新增。'
  if (section === 'image-models') return '尚无可执行的图片模型 Provider。'
  if (section === 'video-models') return '尚无可执行的视频模型 Provider。'
  if (section === 'voice-models') return '尚无可执行的 TTS 语音 Provider。'
  return '尚未配置 AI Provider。请到配置中心新增。'
}
