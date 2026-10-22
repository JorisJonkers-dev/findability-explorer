import { useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, type Ref, watch } from 'vue'
import { getAnalysisOptions, listAnalysesQueryKey } from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { Analysis } from '../../../infrastructure/api/types.gen'
import { describeError } from '../../../shared/describeError'
import { isInProgress, pollInterval } from '../domain/status'

/**
 * Server state for one scan, asked for again every two seconds while it is queued or running.
 * When it finishes, the recent scans are fetched again, so the sidebar shows it done.
 */
export function useScan(id: Ref<string>) {
  const queryClient = useQueryClient()
  const query = useQuery(
    computed(() => ({
      ...getAnalysisOptions({ path: { analysisId: id.value } }),
      // The form alone, without a scan under it, asks for nothing.
      enabled: id.value !== '',
      refetchInterval: (q: { state: { data: Analysis | undefined } }) => pollInterval(q.state.data?.summary.status),
    })),
  )
  const status = computed(() => query.data.value?.summary.status)
  watch(status, (now, before) => {
    if (before !== undefined && now !== undefined && isInProgress(before) && !isInProgress(now)) {
      void queryClient.invalidateQueries({ queryKey: listAnalysesQueryKey() })
    }
  })
  return {
    scan: computed<Analysis | undefined>(() => query.data.value),
    isLoading: query.isPending,
    error: computed(() => (query.error.value ? describeError(query.error.value) : null)),
  }
}
