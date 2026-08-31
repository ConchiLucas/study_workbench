import type { WorkspaceConfigCatalog } from './types'

async function parseJSON<T>(res: Response): Promise<T> {
  const body = await res.json().catch(() => ({}))
  if (!res.ok) {
    const msg = typeof body?.error === 'string' ? body.error : `请求失败 ${res.status}`
    throw new Error(msg)
  }
  return body as T
}

export async function getCatalog(): Promise<WorkspaceConfigCatalog> {
  const res = await fetch('/api/v1/runtime-config/catalog')
  return parseJSON(res)
}

export async function refreshCatalog(): Promise<WorkspaceConfigCatalog> {
  const res = await fetch('/api/v1/runtime-config/refresh', { method: 'POST' })
  return parseJSON(res)
}
