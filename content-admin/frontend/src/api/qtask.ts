import type { LiteracyModuleOption, QuestionTask } from './qtaskTypes'

async function parseJSON<T>(res: Response): Promise<T> {
  const body = await res.json().catch(() => ({}))
  if (!res.ok) {
    const msg = typeof body?.error === 'string' ? body.error : `请求失败 ${res.status}`
    throw new Error(msg)
  }
  return body as T
}

export async function listQuestionTasks(params?: {
  subject?: string
  status?: string
}): Promise<QuestionTask[]> {
  const q = new URLSearchParams()
  if (params?.subject) q.set('subject', params.subject)
  if (params?.status) q.set('status', params.status)
  const qs = q.toString()
  const res = await fetch(`/api/v1/question-tasks${qs ? `?${qs}` : ''}`)
  return parseJSON(res)
}

export async function createQuestionTask(body: {
  subjectCode: string
  moduleCode: string
  title?: string
}): Promise<QuestionTask> {
  const res = await fetch('/api/v1/question-tasks', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  return parseJSON(res)
}

export async function getQuestionTask(id: number): Promise<QuestionTask> {
  const res = await fetch(`/api/v1/question-tasks/${id}`)
  return parseJSON(res)
}

export async function reshuffleQuestionTask(id: number): Promise<QuestionTask> {
  const res = await fetch(`/api/v1/question-tasks/${id}/reshuffle`, { method: 'POST' })
  return parseJSON(res)
}

export async function publishQuestionTask(id: number): Promise<QuestionTask> {
  const res = await fetch(`/api/v1/question-tasks/${id}/publish`, { method: 'POST' })
  return parseJSON(res)
}

export async function unpublishQuestionTask(id: number): Promise<QuestionTask> {
  const res = await fetch(`/api/v1/question-tasks/${id}/unpublish`, { method: 'POST' })
  return parseJSON(res)
}

export async function deleteQuestionTask(id: number): Promise<void> {
  const res = await fetch(`/api/v1/question-tasks/${id}`, { method: 'DELETE' })
  if (!res.ok) {
    await parseJSON(res)
  }
}

export async function listLiteracyModules(): Promise<LiteracyModuleOption[]> {
  const res = await fetch('/api/v1/question-tasks/literacy-modules')
  return parseJSON(res)
}
