import type {
  PinyinBatchResult,
  PinyinItem,
  PinyinListResult,
  PinyinSyncResult,
} from './pinyinTypes'

async function parseJSON<T>(res: Response): Promise<T> {
  const body = await res.json().catch(() => ({}))
  if (!res.ok) {
    const msg = typeof body?.error === 'string' ? body.error : `请求失败 ${res.status}`
    throw new Error(msg)
  }
  return body as T
}

export async function syncPinyin(): Promise<PinyinSyncResult> {
  const res = await fetch('/api/v1/pinyin/sync', { method: 'POST' })
  return parseJSON(res)
}

export async function listPinyin(params: {
  view: 'groups' | 'table'
}): Promise<PinyinListResult> {
  const q = new URLSearchParams({ view: params.view })
  const res = await fetch(`/api/v1/pinyin/items?${q}`)
  return parseJSON(res)
}

export function speechAudioURL(
  kpId: number,
  kind: 'solo' | 'word',
  speechUrl?: string,
): string {
  const base = `/api/v1/pinyin/items/${kpId}/speech/${kind}.mp3`
  if (!speechUrl) return base
  try {
    const u = new URL(speechUrl, 'http://localhost')
    const v = u.searchParams.get('v')
    return v ? `${base}?v=${v}` : base
  } catch {
    return base
  }
}

export async function regenerateSpeech(
  kpId: number,
  kind: 'solo' | 'word',
): Promise<PinyinItem> {
  const res = await fetch(`/api/v1/pinyin/items/${kpId}/speech/${kind}`, { method: 'POST' })
  return parseJSON(res)
}

export async function batchGenerateSpeech(moduleCode: string): Promise<PinyinBatchResult> {
  const q = new URLSearchParams({ moduleCode })
  const res = await fetch(`/api/v1/pinyin/speech/batch?${q}`, { method: 'POST' })
  return parseJSON(res)
}

export function glyphImageURL(kpId: number, glyphUrl?: string): string {
  const base = `/api/v1/pinyin/items/${kpId}/glyph.png`
  if (!glyphUrl) return base
  try {
    const u = new URL(glyphUrl, 'http://localhost')
    const v = u.searchParams.get('v')
    return v ? `${base}?v=${v}` : base
  } catch {
    return base
  }
}

export async function generateGlyph(kpId: number): Promise<PinyinItem> {
  const res = await fetch(`/api/v1/pinyin/items/${kpId}/glyph`, { method: 'POST' })
  return parseJSON(res)
}

export async function batchGenerateGlyphs(moduleCode: string): Promise<PinyinBatchResult> {
  const q = new URLSearchParams({ moduleCode })
  const res = await fetch(`/api/v1/pinyin/glyphs/batch?${q}`, { method: 'POST' })
  return parseJSON(res)
}

