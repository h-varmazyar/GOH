import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { Strategy } from '../types';
import { mockStrategies } from '../data/mock';
import { normalizeStrategy } from '../utils/strategyDefaults';

interface StrategyStore {
  strategies: Strategy[];
  isLoading: boolean;
  isHydrated: boolean;
  hasSeeded: boolean;
  fetchStrategies: () => Promise<void>;
  addStrategy: (strategy: Strategy) => void;
  updateStrategy: (id: string, data: Partial<Strategy>) => void;
  deleteStrategy: (id: string) => void;
  toggleStrategy: (id: string) => void;
}

export const useStrategyStore = create<StrategyStore>()(
  persist(
    (set, get) => ({
      strategies: [], isLoading: false, isHydrated: false, hasSeeded: false,
      fetchStrategies: async () => {
        if (get().hasSeeded || get().isLoading) return;
        set({ isLoading: true });
        await new Promise((resolve) => setTimeout(resolve, 250));
        set((state) => ({
          strategies: state.strategies.length ? state.strategies : mockStrategies.map((item) => normalizeStrategy(item)),
          isLoading: false,
          hasSeeded: true,
        }));
      },
      addStrategy: (strategy) => set((state) => ({
        strategies: [...state.strategies, normalizeStrategy({ ...strategy, id: crypto.randomUUID(), createdAt: new Date().toISOString() })],
      })),
      updateStrategy: (id, data) => set((state) => ({
        strategies: state.strategies.map((item) => item.id === id
          ? normalizeStrategy({ ...item, ...data, id, updatedAt: new Date().toISOString() })
          : item),
      })),
      deleteStrategy: (id) => set((state) => ({ strategies: state.strategies.filter((item) => item.id !== id) })),
      toggleStrategy: (id) => set((state) => ({
        strategies: state.strategies.map((item) => item.id === id
          ? { ...item, enabled: !item.enabled, updatedAt: new Date().toISOString() }
          : item),
      })),
    }),
    {
      name: 'gate-strategies-v2',
      version: 2,
      partialize: (state) => ({ strategies: state.strategies, hasSeeded: state.hasSeeded }),
      merge: (persisted, current) => {
        const saved = persisted as Partial<StrategyStore>;
        return { ...current, ...saved, strategies: (saved.strategies || []).map((item) => normalizeStrategy(item)) };
      },
      onRehydrateStorage: () => (state) => { if (state) state.isHydrated = true; },
    },
  ),
);
