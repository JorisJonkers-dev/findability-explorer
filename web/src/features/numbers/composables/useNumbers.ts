import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import { getPlanOptions, getStandingsOptions } from '../../../infrastructure/api/@tanstack/vue-query.gen'
import { describeError } from '../../../shared/describeError'
import type { Facts } from '../domain/numbers'

/** Server state for the numbers to watch: the plan holds their targets, the standings the competitors. Both are shared, cached requests. */
export function useNumbers() {
  const plan = useQuery(getPlanOptions())
  const standings = useQuery(getStandingsOptions())
  const failure = computed(() => plan.error.value ?? standings.error.value)
  return {
    /** The plan and the standings together; undefined until both load. */
    facts: computed<Facts | undefined>(() => (plan.data.value && standings.data.value ? { plan: plan.data.value, standings: standings.data.value } : undefined)),
    isLoading: computed(() => plan.isPending.value || standings.isPending.value),
    error: computed(() => (failure.value ? describeError(failure.value) : null)),
  }
}
