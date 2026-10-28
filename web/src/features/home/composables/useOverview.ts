import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import { getOverviewOptions } from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { Overview } from '../../../infrastructure/api/types.gen'
import { describeError } from '../../../shared/describeError'

/** Server state for the overview's figures: the search term list, the competitors on the list and the newest Prompt run, counted on the server. */
export function useOverview() {
  const query = useQuery(getOverviewOptions())
  return {
    /** The figures; undefined until they load. */
    overview: computed<Overview | undefined>(() => query.data.value),
    isLoading: query.isPending,
    error: computed(() => (query.error.value ? describeError(query.error.value) : null)),
  }
}
