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

export interface ReviewQuestion {
  id: number
  type: string
  stem: string
  options: QuestionOption[]
  visual: QuestionVisual
  answer_index: number
}

export interface PlanReviewItem {
  seq: number
  status: ItemStatus
  bucket: Bucket
  tries: number
  cost_ms: number
  picks: number[]
  answered_at?: string | null
  kp_id: number
  kp_title: string
  subject_code: string
  subject_name: string
  question: ReviewQuestion
}

export interface PlanReview {
  plan: Plan
  items: PlanReviewItem[]
}

export const usePlanHistory = (childId: number, status?: PlanStatus | '') =>
  useQuery({
    queryKey: ['plan', 'history', childId, status ?? ''],
    queryFn: () => {
      const q = status ? `?status=${status}` : ''
      return api.get<PlanSummary[]>(`/children/${childId}/plans${q}`)
    },
  })

export const useTaskReview = (childId: number, planId: number) =>
  useQuery({
    queryKey: ['plan', 'review', childId, planId],
    queryFn: () => api.get<PlanReview>(`/children/${childId}/plans/${planId}/review`),
    enabled: planId > 0,
  })

export function useGenerateToday(childId: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api.post<{ plan: Plan }>(`/children/${childId}/plans/today`),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['plan', 'history'] })
      void qc.invalidateQueries({ queryKey: ['overview'] })
    },
  })
}