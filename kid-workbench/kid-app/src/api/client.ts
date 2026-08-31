const BASE = '/api/v1'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(BASE + path, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })
  const body = await res.json().catch(() => ({ data: null, error: res.statusText }))
  if (!res.ok || body.error) throw new Error(body.error ?? `请求失败 ${res.status}`)
  return body.data as T
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, payload?: unknown) =>
    request<T>(path, { method: 'POST', body: payload ? JSON.stringify(payload) : undefined }),
}
