import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { RefreshCw } from 'lucide-react'
import { Navigate, useParams } from 'react-router-dom'
import { getCatalog, refreshCatalog } from '../../api/config'
import type { ConfigSection, WorkspaceConfigCatalog } from '../../api/types'
import { ConfigSectionView } from './ConfigSectionView'

const SECTIONS: ConfigSection[] = [
  'databases',
  'ai',
  'local-cli',
  'object-storage',
  'image-models',
  'video-models',
  'voice-models',
  'runtime',
]

const HEADINGS: Record<ConfigSection, { eyebrow: string; title: string; description: string }> = {
  databases: {
    eyebrow: 'CONFIGURATION / DATABASES',
    title: 'Database Connections',
    description: '展示跨语言通用的连接参数。',
  },
  ai: {
    eyebrow: 'CONFIGURATION / AI',
    title: 'AI Provider',
    description: '展示兼容 OpenAI 协议的服务提供商配置。',
  },
  'local-cli': {
    eyebrow: 'CONFIGURATION / LOCAL CLI',
    title: '本地 CLI 配置',
    description: '展示命令、参数与执行上下文配置。',
  },
  'object-storage': {
    eyebrow: 'CONFIGURATION / MINIO',
    title: 'MinIO 配置',
    description: '展示 S3-compatible 对象存储连接信息。',
  },
  'image-models': {
    eyebrow: 'CONFIGURATION / IMAGE MODEL',
    title: '图片模型配置',
    description: '展示可执行的图片 Provider；数据归属于统一 AI 配置。',
  },
  'video-models': {
    eyebrow: 'CONFIGURATION / VIDEO MODEL',
    title: '视频模型配置',
    description: '展示可执行的视频 Provider；数据归属于统一 AI 配置。',
  },
  'voice-models': {
    eyebrow: 'CONFIGURATION / TTS VOICE',
    title: 'TTS 语音配置',
    description: '展示可执行的 TTS Provider；数据归属于统一 AI 配置。',
  },
  runtime: {
    eyebrow: 'RUNTIME / VERSION 1',
    title: '运行契约',
    description: '这是当前工作台实际使用的 JSON 快照。',
  },
}

export function ConfigPage() {
  const { section } = useParams()
  const queryClient = useQueryClient()
  const catalogQuery = useQuery({
    queryKey: ['runtime-config', 'catalog'],
    queryFn: getCatalog,
  })
  const refreshMutation = useMutation({
    mutationFn: refreshCatalog,
    onSuccess: (data) => {
      queryClient.setQueryData(['runtime-config', 'catalog'], data)
    },
  })

  if (!section || !SECTIONS.includes(section as ConfigSection)) {
    return <Navigate to="/config/databases" replace />
  }
  const current = section as ConfigSection
  const heading = HEADINGS[current]
  const error =
    (catalogQuery.error instanceof Error && catalogQuery.error.message) ||
    (refreshMutation.error instanceof Error && refreshMutation.error.message) ||
    ''

  return (
    <section className="config-page" aria-label="配置管理">
      <div className="page-heading">
        <div>
          <p className="eyebrow">{heading.eyebrow}</p>
          <h1>{heading.title}</h1>
          <p className="page-description">{heading.description}</p>
        </div>
        <button
          type="button"
          className="refresh-button"
          onClick={() => refreshMutation.mutate()}
          disabled={refreshMutation.isPending}
        >
          <RefreshCw size={16} />
          {refreshMutation.isPending ? '刷新中…' : '刷新配置'}
        </button>
      </div>

      {error ? <div className="error-panel" role="alert">{error}</div> : null}

      {catalogQuery.isLoading ? (
        <div className="loading-panel">正在读取共享配置…</div>
      ) : catalogQuery.data ? (
        <ConfigSectionView section={current} catalog={catalogQuery.data as WorkspaceConfigCatalog} />
      ) : !error ? (
        <div className="empty-panel">暂无配置</div>
      ) : null}

      <p className="edit-hint">修改请到 Shared Config Center（http://localhost:18427）。本页只读。</p>
    </section>
  )
}
