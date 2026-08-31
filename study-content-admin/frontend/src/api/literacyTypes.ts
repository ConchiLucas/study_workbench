export interface LiteracyChar {
  kpId: number
  charText: string
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

export interface LiteracyGroup {
  moduleCode: string
  moduleName: string
  moduleOrder: number
  chars: LiteracyChar[]
}

export interface LiteracyListResult {
  view: string
  total: number
  groups?: LiteracyGroup[]
  chars?: LiteracyChar[]
}

export interface LiteracySyncResult {
  upserted: number
  total: number
}
