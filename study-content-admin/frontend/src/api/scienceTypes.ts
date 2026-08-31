export interface ScienceItem {
  kpId: number
  title: string
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

export interface ScienceGroup {
  moduleCode: string
  moduleName: string
  moduleOrder: number
  items: ScienceItem[]
}

export interface ScienceListResult {
  view: string
  total: number
  groups?: ScienceGroup[]
  items?: ScienceItem[]
}

export interface ScienceSyncResult {
  upserted: number
  total: number
}
