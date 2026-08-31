import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from './client'

export type PlanStatus = 'pending' | 'doing' | 'done'
export type ItemStatus = 'pending' | 'correct' | 'wrong' | 'skipped'
export type Bucket = 'review' | 'shaky' | 'learning' | 'new'

export interface Plan {
  id: number
  plan_date: string
  seq_no: number
  status: PlanStatus
  target_count: number
  done_count: number
  correct_count: number
  stars: number
  duration_sec: number
  flowers: number
}

export interface SubjectCount {
  code: string
  name: string
  icon: string
  count: number
}

export interface PlanSummary extends Plan {
  subjects: SubjectCount[]
}

export interface QuestionOption {
  label?: string
  emoji?: string
  shape?: string
}

export interface QuestionVisual {
  kind?: 'count' | 'add' | 'sub' | 'compare' | 'shape' | 'char' | 'emoji' | 'seq'
  a?: number
  b?: number
  emoji?: string
  text?: string
  items?: string[]
}

export interface QuestionSpeech {
  text?: string
  lang?: 'zh-CN' | 'en-US'
}

export interface Question {
  id: number
  type: string
  stem: string
  options: QuestionOption[]
  visual: QuestionVisual
  speech: QuestionSpeech
}

export interface PlanItem {
  id: number
  seq: number
  status: ItemStatus
  bucket: Bucket
  tries: number
  kp_id: number
  kp_title: string
  subject_code: string
  subject_name: string
  question: Question
}

export interface PlanDetail {
  plan: Plan
  items: PlanItem[]
}

export interface AnswerResult {
  correct: boolean
  answer_index: number
  can_retry: boolean
  tries: number
  status: ItemStatus
  plan: Plan
}

export interface FinishResult {
  plan: Plan
  stars: number
  flowers: number
}

export const useKidTodo = (childId: number) =>
  useQuery({
    queryKey: ['plan', 'todo', childId],
    queryFn: () => api.get<PlanSummary[]>(`/children/${childId}/plans/todo`),
  })

export const usePlanDetail = (childId: number, planId: number) =>
  useQuery({
    queryKey: ['plan', 'detail', childId, planId],
    queryFn: () => api.get<PlanDetail>(`/children/${childId}/plans/${planId}`),
    enabled: planId > 0,
    staleTime: Infinity,
    refetchOnWindowFocus: false,
  })

export const useTodayPlan = (childId: number) =>
  useQuery({
    queryKey: ['plan', 'today', childId],
    queryFn: () => api.get<PlanDetail>(`/children/${childId}/plans/today`),
    staleTime: Infinity,
    refetchOnWindowFocus: false,
  })

export const usePlanHistory = (childId: number) =>
  useQuery({
    queryKey: ['plan', 'history', childId],
    queryFn: () => api.get<PlanSummary[]>(`/children/${childId}/plans`),
  })

export function useGenerateToday(childId: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api.post<PlanDetail>(`/children/${childId}/plans/today`),
    onSuccess: (data) => {
      qc.setQueryData(['plan', 'detail', childId, data.plan.id], data)
      void qc.invalidateQueries({ queryKey: ['plan', 'todo', childId] })
      void qc.invalidateQueries({ queryKey: ['plan', 'history', childId] })
      void qc.invalidateQueries({ queryKey: ['overview', childId] })
    },
  })
}

export function useStartPlan(childId: number) {
  return useMutation({
    mutationFn: (planId: number) => api.post<Plan>(`/children/${childId}/plans/${planId}/start`),
  })
}

export function useAnswerItem(childId: number, planId: number) {
  return useMutation({
    mutationFn: (v: { itemId: number; optionIndex: number; costMs: number }) =>
      api.post<AnswerResult>(
        `/children/${childId}/plans/${planId}/items/${v.itemId}/answer`,
        { option_index: v.optionIndex, cost_ms: v.costMs },
      ),
  })
}

export function useFinishPlan(childId: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (planId: number) =>
      api.post<FinishResult>(`/children/${childId}/plans/${planId}/finish`),
    onSuccess: (res, planId) => {
      qc.setQueryData<PlanDetail>(['plan', 'detail', childId, planId], (prev) =>
        prev ? { ...prev, plan: res.plan } : prev,
      )
      void qc.invalidateQueries({ queryKey: ['plan', 'todo', childId] })
      void qc.invalidateQueries({ queryKey: ['plan', 'history', childId] })
      void qc.invalidateQueries({ queryKey: ['overview', childId] })
    },
  })
}

export function useExtraPlan(childId: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api.post<PlanDetail>(`/children/${childId}/plans/extra`),
    onSuccess: (data) => {
      qc.setQueryData(['plan', 'detail', childId, data.plan.id], data)
      void qc.invalidateQueries({ queryKey: ['plan', 'todo', childId] })
      void qc.invalidateQueries({ queryKey: ['overview', childId] })
      void qc.invalidateQueries({ queryKey: ['plan', 'history', childId] })
    },
  })
}
