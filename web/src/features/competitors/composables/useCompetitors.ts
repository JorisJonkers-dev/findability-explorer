import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import { listCompetitorsOptions } from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { Competitor } from '../../../infrastructure/api/types.gen'
import { describeError } from '../../../shared/describeError'

/** Server state for the companies in the terminology map. */
export function useCompetitors() {
  const list = useQuery(listCompetitorsOptions())
  return {
    competitors: computed<Competitor[]>(() => (list.data.value ? list.data.value.items : [])),
    isLoading: list.isPending,
    error: computed(() => (list.error.value ? describeError(list.error.value) : null)),
  }
}
