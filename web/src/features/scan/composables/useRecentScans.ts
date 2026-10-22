import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import { listAnalysesOptions } from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { AnalysisList, AnalysisSummary } from '../../../infrastructure/api/types.gen'
import { describeError } from '../../../shared/describeError'
import { isInProgress, POLL_INTERVAL_MS } from '../domain/status'

/** How many recent scans the sidebar lists. */
export const RECENT_LIMIT = 8

/** Server state for the most recent scans, newest first, asked for again while one is in progress. */
export function useRecentScans() {
  const list = useQuery({
    ...listAnalysesOptions({ query: { limit: RECENT_LIMIT } }),
    refetchInterval: (q: { state: { data: AnalysisList | undefined } }) =>
      q.state.data?.items.some((s) => isInProgress(s.status)) ? POLL_INTERVAL_MS : false,
  })
  return {
    scans: computed<AnalysisSummary[]>(() => (list.data.value ? list.data.value.items : [])),
    isLoading: list.isPending,
    error: computed(() => (list.error.value ? describeError(list.error.value) : null)),
  }
}
