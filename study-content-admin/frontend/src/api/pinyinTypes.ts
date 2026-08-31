export interface PinyinItem {
  kpId: number
  letter: string
  moduleCode: string
  moduleName: string
  moduleOrder: number
  kpOrder: number
  soloText: string
  wordText: string
  soloSpeechUrl: string
  wordSpeechUrl: string
  glyphImageUrl: string
}

export interface PinyinGroup {
  moduleCode: string
  moduleName: string
  moduleOrder: number
  items: PinyinItem[]
}

export interface PinyinListResult {
  view: string
  total: number
  groups?: PinyinGroup[]
  items?: PinyinItem[]
}

export interface PinyinSyncResult {
  upserted: number
  total: number
}

export interface PinyinBatchResult {
  generated: number
  skipped: number
  failed: number
  errors?: string[]
}
