import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import { getStandingsOptions } from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { Standing } from '../../../infrastructure/api/types.gen'
import { describeError } from '../../../shared/describeError'

/** Server state for how Tribelt stands against its competitors: one cached request the page and the poster share. */
export function useStandings() {
  const query = useQuery(getStandingsOptions())
  return {
    measures: computed<Standing[]>(() => query.data.value?.measures ?? []),
    trailing: computed(() => query.data.value?.trailing ?? 0),
    isLoading: query.isPending,
    error: computed(() => (query.error.value ? describeError(query.error.value) : null)),
  }
}
