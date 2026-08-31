import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from './client'
import type {
  AttentionItem, CalendarDay, KpDetail, Matrix, Overview, SubjectSummary, TrendPoint,
} from './types'

export const useOverview = (childId: number) =>
  useQuery({ queryKey: ['overview', childId],
    queryFn: () => api.get<Overview>(`/children/${childId}/overview`) })

export const useSubjects = (childId: number) =>
  useQuery({ queryKey: ['subjects', childId],
    queryFn: () => api.get<SubjectSummary[]>(`/children/${childId}/subjects`) })

export const useMatrix = (childId: number, subject: string) =>
  useQuery({ queryKey: ['matrix', childId, subject], enabled: !!subject,
    queryFn: () => api.get<Matrix>(`/children/${childId}/mastery/matrix?subject=${subject}`) })

export const useAttention = (childId: number, limit = 10) =>
  useQuery({ queryKey: ['attention', childId, limit],
    queryFn: () => api.get<AttentionItem[]>(`/children/${childId}/attention?limit=${limit}`) })

export const useKpDetail = (childId: number, kpId: number | null) =>
  useQuery({ queryKey: ['kp', childId, kpId], enabled: kpId !== null,
    queryFn: () => api.get<KpDetail>(`/children/${childId}/knowledge-points/${kpId}`) })

export const useTrend = (childId: number, days = 30) =>
  useQuery({ queryKey: ['trend', childId, days],
    queryFn: () => api.get<TrendPoint[]>(`/children/${childId}/stats/trend?days=${days}`) })

export const useCalendar = (childId: number, months = 3) =>
  useQuery({ queryKey: ['calendar', childId, months],
    queryFn: () => api.get<CalendarDay[]>(`/children/${childId}/stats/calendar?months=${months}`) })

export function useMarkMastered(childId: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (kpId: number) =>
      api.post(`/children/${childId}/knowledge-points/${kpId}/mark`),
    onSuccess: () => { void qc.invalidateQueries() },
  })
}

export function useUndoMark(childId: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (kpId: number) =>
      api.del(`/children/${childId}/knowledge-points/${kpId}/mark`),
    onSuccess: () => { void qc.invalidateQueries() },
  })
}
