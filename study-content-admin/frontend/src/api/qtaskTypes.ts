export interface QTaskItem {
  seq: number
  kpId: number
  questionId: number
  charText: string
  code: string
  stem: string
  options: { label?: string }[]
  answerIndex: number
  speech?: { text?: string; lang?: string }
}

export interface QuestionTask {
  id: number
  subjectCode: string
  title: string
  moduleCode: string
  moduleName: string
  targetCount: number
  status: 'draft' | 'published'
  createdAt: string
  updatedAt: string
  items?: QTaskItem[]
}

export interface LiteracyModuleOption {
  code: string
  name: string
  order: number
}
