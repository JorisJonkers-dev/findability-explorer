import { useQuery } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import { compareScansOptions, getCompetitorProfileOptions } from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { CompetitorProfile, ScanComparison } from '../../../infrastructure/api/types.gen'
import { describeError } from '../../../shared/describeError'

/** Server state for a company's search terms, from its latest finished scan. */
export function useCompanyProfile(id: Ref<string>) {
  const query = useQuery(computed(() => getCompetitorProfileOptions({ path: { competitorId: id.value } })))
  return {
    profile: computed<CompetitorProfile | undefined>(() => query.data.value),
    isLoading: query.isPending,
    error: computed(() => (query.error.value ? describeError(query.error.value) : null)),
  }
}

/**
 * Server state for two of a company's finished scans compared. Nothing is asked until both are
 * picked; a comparison never changes, so it is asked for once.
 */
export function useScanComparison(id: Ref<string>, from: Ref<string>, to: Ref<string>) {
  const query = useQuery(
    computed(() => ({
      ...compareScansOptions({ path: { competitorId: id.value }, query: { from: from.value, to: to.value } }),
      enabled: from.value !== '' && to.value !== '' && from.value !== to.value,
      staleTime: Number.POSITIVE_INFINITY,
    })),
  )
  return {
    comparison: computed<ScanComparison | undefined>(() => query.data.value),
    isLoading: computed(() => query.isFetching.value && query.data.value === undefined),
    error: computed(() => (query.error.value ? describeError(query.error.value) : null)),
  }
}
