import { useQuery } from '@tanstack/react-query'
import { api } from './client'

export interface Overview {
  child: { id: number; name: string; grade: string; flowers: number }
}

export const useOverview = (childId: number) =>
  useQuery({
    queryKey: ['overview', childId],
    queryFn: () => api.get<Overview>(`/children/${childId}/overview`),
  })
