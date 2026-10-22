import { useMutation, useQueryClient } from '@tanstack/vue-query'
import { computed } from 'vue'
import { listAnalysesQueryKey, startAnalysisMutation } from '../../../infrastructure/api/@tanstack/vue-query.gen'
import { placeStartError, type StartError } from '../domain/newScan'

/** Starts a scan, and lists the recent scans again once it is queued. */
export function useStartScan() {
  const queryClient = useQueryClient()
  const start = useMutation(startAnalysisMutation())
  return {
    isStarting: start.isPending,
    /** Why the server refused the last scan, and where to say so. */
    error: computed<StartError | null>(() => (start.error.value ? placeStartError(start.error.value) : null)),
    /** Queues a scan of url; onStarted gets the new scan's id. */
    start: (url: string, maxPages: number, onStarted: (id: string) => void) => {
      start.mutate(
        { body: { url, maxPages } },
        {
          onSuccess: (scan) => {
            onStarted(scan.summary.id)
            void queryClient.invalidateQueries({ queryKey: listAnalysesQueryKey() })
          },
        },
      )
    },
    reset: () => {
      start.reset()
    },
  }
}
