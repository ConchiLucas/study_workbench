import type {
  EnglishListResult,
  EnglishSyncResult,
  EnglishWord,
} from './englishTypes'

async function parseJSON<T>(res: Response): Promise<T> {
  const body = await res.json().catch(() => ({}))
  if (!res.ok) {
    const msg = typeof body?.error === 'string' ? body.error : `请求失败 ${res.status}`
    throw new Error(msg)
  }
  return body as T
}

export async function syncEnglish(): Promise<EnglishSyncResult> {
  const res = await fetch('/api/v1/english/sync', { method: 'POST' })
  return parseJSON(res)
}

export async function listEnglish(params: {
  view: 'groups' | 'table'
  needsSenseImage?: '' | 'true' | 'false'
}): Promise<EnglishListResult> {
  const q = new URLSearchParams({ view: params.view })
  if (params.needsSenseImage) q.set('needsSenseImage', params.needsSenseImage)
  const res = await fetch(`/api/v1/english/words?${q}`)
  return parseJSON(res)
}

export function speechAudioURL(kpId: number, speechAudioUrl?: string): string {
  const base = `/api/v1/english/words/${kpId}/speech.mp3`
  if (!speechAudioUrl) return base
  try {
    const u = new URL(speechAudioUrl, 'http://localhost')
    const v = u.searchParams.get('v')
    return v ? `${base}?v=${v}` : base
  } catch {
    return base
  }
}

export async function patchEnglishWord(
  kpId: number,
  needsSenseImageOverride: boolean | null,
): Promise<EnglishWord> {
  const res = await fetch(`/api/v1/english/words/${kpId}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ needsSenseImageOverride }),
  })
  return parseJSON(res)
}

export async function batchGenerateGlyphs(): Promise<{
  generated: number
  skipped: number
  failed: number
  errors?: string[]
}> {
  const res = await fetch('/api/v1/english/glyphs/batch', { method: 'POST' })
  return parseJSON(res)
}

export async function batchGenerateSenses(
  moduleCode?: string,
  opts?: { workers?: number; maxRetries?: number },
): Promise<{
  generated: number
  skipped: number
  failed: number
  retried?: number
  workers?: number
  errors?: string[]
}> {
  const q = new URLSearchParams()
  if (moduleCode) q.set('moduleCode', moduleCode)
  if (opts?.workers) q.set('workers', String(opts.workers))
  if (opts?.maxRetries != null) q.set('maxRetries', String(opts.maxRetries))
  const res = await fetch(`/api/v1/english/senses/batch?${q}`, { method: 'POST' })
  return parseJSON(res)
}

export async function batchGenerateSpeech(moduleCode: string): Promise<{
  generated: number
  skipped: number
  failed: number
  errors?: string[]
}> {
  const q = new URLSearchParams({ moduleCode })
  const res = await fetch(`/api/v1/english/speech/batch?${q}`, { method: 'POST' })
  return parseJSON(res)
}
