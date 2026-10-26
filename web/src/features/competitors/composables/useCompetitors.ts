import { type QueryClient, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, watch } from 'vue'
import { listCompetitorsOptions } from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { Competitor, CompetitorList } from '../../../infrastructure/api/types.gen'
import { describeError } from '../../../shared/describeError'
import { POLL_INTERVAL_MS } from '../../scan/domain/status'
import { isScanning } from '../domain/company'

/** What a finished scan changes: the map's makers, the companies' details and profiles, the review queue. */
const FINDINGS = ['searchTerms', 'getCompetitor', 'getCompetitorProfile', 'compareScans', 'listTermCandidates'] as const

/** The company list answer each query client refreshed the findings for last. */
const handled = new WeakMap<QueryClient, number>()

/**
 * Server state for the companies in the terminology map, asked for again while a scan of one is
 * waiting or reading pages, so the list and the card show it as it goes. When a scan finishes,
 * what it found is asked for again too, so the search terms, the table of who makes which belt and
 * the company cards follow every scan without a reload.
 */
export function useCompetitors() {
  const queryClient = useQueryClient()
  const list = useQuery({
    ...listCompetitorsOptions(),
    refetchInterval: (q: { state: { data: CompetitorList | undefined } }) =>
      q.state.data?.items.some(isScanning) ? POLL_INTERVAL_MS : false,
  })
  const competitors = computed<Competitor[]>(() => (list.data.value ? list.data.value.items : []))
  const scanning = computed(() => competitors.value.filter(isScanning).map((c) => c.id))
  watch(scanning, (now, before) => {
    // Every view that lists companies watches the same answer; the first to see it refreshes.
    const answer = list.dataUpdatedAt.value
    if (before.some((id) => !now.includes(id)) && handled.get(queryClient) !== answer) {
      handled.set(queryClient, answer)
      for (const id of FINDINGS) void queryClient.invalidateQueries({ queryKey: [{ _id: id }] })
    }
  })
  return {
    competitors,
    isLoading: list.isPending,
    error: computed(() => (list.error.value ? describeError(list.error.value) : null)),
  }
}
