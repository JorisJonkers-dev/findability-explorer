import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import { getAiSearchSummaryOptions } from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { AiSearchSummary } from '../../../infrastructure/api/types.gen'
import { describeError } from '../../../shared/describeError'

/** Server state for the AI search summary: one cached request the page and the poster share. */
export function useAiSearchSummary() {
  const query = useQuery(getAiSearchSummaryOptions())
  return {
    summary: computed<AiSearchSummary | undefined>(() => query.data.value),
    isLoading: query.isPending,
    error: computed(() => (query.error.value ? describeError(query.error.value) : null)),
  }
}
