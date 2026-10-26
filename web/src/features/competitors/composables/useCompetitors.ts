import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import { listCompetitorsOptions } from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { Competitor, CompetitorList } from '../../../infrastructure/api/types.gen'
import { describeError } from '../../../shared/describeError'
import { POLL_INTERVAL_MS } from '../../scan/domain/status'
import { isScanning } from '../domain/company'

/**
 * Server state for the companies in the terminology map, asked for again while a scan of one is
 * waiting or reading pages, so the list and the card show it as it goes.
 */
export function useCompetitors() {
  const list = useQuery({
    ...listCompetitorsOptions(),
    refetchInterval: (q: { state: { data: CompetitorList | undefined } }) =>
      q.state.data?.items.some(isScanning) ? POLL_INTERVAL_MS : false,
  })
  return {
    competitors: computed<Competitor[]>(() => (list.data.value ? list.data.value.items : [])),
    isLoading: list.isPending,
    error: computed(() => (list.error.value ? describeError(list.error.value) : null)),
  }
}
