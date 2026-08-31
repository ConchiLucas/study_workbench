import { create } from 'zustand'

interface ChildState {
  childId: number
  setChild: (id: number) => void
}

export const useChildStore = create<ChildState>((set) => ({
  childId: 1,
  setChild: (childId) => set({ childId }),
}))
