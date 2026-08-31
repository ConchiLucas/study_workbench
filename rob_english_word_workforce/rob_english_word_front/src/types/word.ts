export interface Word {
  id: number
  libraryId: number
  word: string
  meaning: string
  pronunciationUs?: string
  pronunciationUk?: string
  frequency: number
  difficulty: number
  status: number
  phrase?: string
  phraseTranslation?: string
  sentence?: string
  sentenceTranslation?: string
  // 游戏相关字段
  grabbed?: boolean
  grabbedBy?: number
}
