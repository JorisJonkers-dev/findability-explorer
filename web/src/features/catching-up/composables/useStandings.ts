import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import { getStandingsOptions } from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { PlanImprovement, Standing } from '../../../infrastructure/api/types.gen'
import { describeError } from '../../../shared/describeError'

/** Server state for how Tribelt stands against its competitors: one cached request the page and the poster share. */
export function useStandings() {
  const query = useQuery(getStandingsOptions())
  return {
    measures: computed<Standing[]>(() => query.data.value?.measures ?? []),
    trailing: computed(() => query.data.value?.trailing ?? 0),
    /** The first improvements of the plan not done yet. */
    open: computed<PlanImprovement[]>(() => query.data.value?.open ?? []),
    /** What says which search terms tribelt.nl uses: its newest scan, or the research before one. */
    termsFrom: computed(() => query.data.value?.tribeltTermsFrom),
    isLoading: query.isPending,
    error: computed(() => (query.error.value ? describeError(query.error.value) : null)),
  }
}
