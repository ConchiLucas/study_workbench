import { create } from 'zustand'

interface ChildState {
  childId: number
  kpDrawerId: number | null
  setChild: (id: number) => void
  openKp: (id: number) => void
  closeKp: () => void
}

export const useChildStore = create<ChildState>((set) => ({
  childId: 1,
  kpDrawerId: null,
  setChild: (childId) => set({ childId }),
  openKp: (kpDrawerId) => set({ kpDrawerId }),
  closeKp: () => set({ kpDrawerId: null }),
}))
