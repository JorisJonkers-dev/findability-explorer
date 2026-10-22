import { useMutation, useQueryClient } from '@tanstack/vue-query'
import { computed } from 'vue'
import {
  listAnalysesQueryKey,
  listCompetitorsQueryKey,
  startAnalysisMutation,
} from '../../../infrastructure/api/@tanstack/vue-query.gen'
import { placeStartError, type StartError } from '../domain/newScan'

/**
 * Starts a scan. Once it is queued, the recent scans and the companies are fetched again: the
 * company's scan count went up, and an unknown host was added as a new company.
 */
export function useStartScan() {
  const queryClient = useQueryClient()
  const start = useMutation(startAnalysisMutation())
  return {
    isStarting: start.isPending,
    /** Why the server refused the last scan, and where to say so. */
    error: computed<StartError | null>(() => (start.error.value ? placeStartError(start.error.value) : null)),
    /** Queues a scan of url; companyName names the company to add when its host is unknown. */
    start: (url: string, maxPages: number, onStarted: (id: string) => void, companyName?: string) => {
      const name = companyName?.trim() ?? ''
      start.mutate(
        { body: { url, maxPages, ...(name === '' ? {} : { companyName: name }) } },
        {
          onSuccess: (scan) => {
            onStarted(scan.summary.id)
            void queryClient.invalidateQueries({ queryKey: listAnalysesQueryKey() })
            void queryClient.invalidateQueries({ queryKey: listCompetitorsQueryKey() })
            void queryClient.invalidateQueries({ queryKey: [{ _id: 'getCompetitor' }] })
          },
        },
      )
    },
    reset: () => {
      start.reset()
    },
  }
}
