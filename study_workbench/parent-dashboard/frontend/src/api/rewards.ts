import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from './client'

export interface Reward {
  id: number
  name: string
  cost: number
  stock: number
}

export const useRewards = (childId: number) =>
  useQuery({
    queryKey: ['rewards', childId],
    queryFn: () => api.get<Reward[]>(`/children/${childId}/rewards`),
  })

export function useRedeem(childId: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (rewardId: number) =>
      api.post(`/children/${childId}/rewards/${rewardId}/redeem`),
    onSuccess: () => {
      void qc.invalidateQueries()
    },
  })
}
