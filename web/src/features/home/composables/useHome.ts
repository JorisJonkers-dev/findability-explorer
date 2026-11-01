import { computed } from 'vue'
import { useGoals } from '../../goals/composables/useGoals'
import { useOverview } from './useOverview'

/** Server state for the overview page: the numbers to watch and the figures of the sections. */
export function useHome() {
  const numbers = useGoals()
  const figures = useOverview()
  return {
    facts: numbers.facts,
    overview: figures.overview,
    isLoading: computed(() => numbers.isLoading.value || figures.isLoading.value),
    error: computed(() => numbers.error.value ?? figures.error.value),
  }
}
