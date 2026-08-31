import type { MathBatchResult, MathItem, MathListResult, MathSyncResult } from './mathTypes'

async function parseJSON<T>(res: Response): Promise<T> {
  const body = await res.json().catch(() => ({}))
  if (!res.ok) {
    const msg = typeof body?.error === 'string' ? body.error : `请求失败 ${res.status}`
    throw new Error(msg)
  }
  return body as T
}

export async function syncMath(): Promise<MathSyncResult> {
  const res = await fetch('/api/v1/math/sync', { method: 'POST' })
  return parseJSON(res)
}

export async function listMath(params: {
  view: 'groups' | 'table'
}): Promise<MathListResult> {
  const q = new URLSearchParams({ view: params.view })
  const res = await fetch(`/api/v1/math/items?${q}`)
  return parseJSON(res)
}

export function glyphImageURL(kpId: number, glyphUrl?: string): string {
  const base = `/api/v1/math/items/${kpId}/glyph.png`
  if (!glyphUrl) return base
  try {
    const u = new URL(glyphUrl, 'http://localhost')
    const v = u.searchParams.get('v')
    return v ? `${base}?v=${v}` : base
  } catch {
    return base
  }
}

export async function generateGlyph(kpId: number): Promise<MathItem> {
  const res = await fetch(`/api/v1/math/items/${kpId}/glyph`, { method: 'POST' })
  return parseJSON(res)
}

export async function batchGenerateGlyphs(moduleCode: string): Promise<MathBatchResult> {
  const q = new URLSearchParams({ moduleCode })
  const res = await fetch(`/api/v1/math/glyphs/batch?${q}`, { method: 'POST' })
  return parseJSON(res)
}

export function speechAudioURL(kpId: number, speechUrl?: string): string {
  const base = `/api/v1/math/items/${kpId}/speech.mp3`
  if (!speechUrl) return base
  try {
    const u = new URL(speechUrl, 'http://localhost')
    const v = u.searchParams.get('v')
    return v ? `${base}?v=${v}` : base
  } catch {
    return base
  }
}

export async function regenerateSpeech(kpId: number): Promise<MathItem> {
  const res = await fetch(`/api/v1/math/items/${kpId}/speech`, { method: 'POST' })
  return parseJSON(res)
}

export async function batchGenerateSpeech(moduleCode: string): Promise<MathBatchResult> {
  const q = new URLSearchParams({ moduleCode })
  const res = await fetch(`/api/v1/math/speech/batch?${q}`, { method: 'POST' })
  return parseJSON(res)
}
