export interface EnglishWord {
  kpId: number
  wordText: string
  moduleCode: string
  moduleName: string
  moduleOrder: number
  kpOrder: number
  needsSenseImage: boolean
  needsSenseImageOverride: boolean | null
  effectiveNeedsSenseImage: boolean
  glyphImageUrl: string
  senseImageUrl: string
  speechAudioUrl: string
}

export interface EnglishGroup {
  moduleCode: string
  moduleName: string
  moduleOrder: number
  words: EnglishWord[]
}

export interface EnglishListResult {
  view: string
  total: number
  groups?: EnglishGroup[]
  words?: EnglishWord[]
}

export interface EnglishSyncResult {
  upserted: number
  total: number
}
