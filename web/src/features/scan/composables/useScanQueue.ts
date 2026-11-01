import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, ref } from 'vue'
import {
  getAiReadinessQueryKey,
  listAnalysesOptions,
  listAnalysesQueryKey,
  listCompetitorsQueryKey,
  removeQueuedAnalysesMutation,
  stopAnalysisMutation,
  stopReadinessCheckMutation,
  stopRescanMutation,
} from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { AnalysisList, AnalysisSummary } from '../../../infrastructure/api/types.gen'
import { describeError } from '../../../shared/describeError'
import { scanQueue } from '../domain/queue'
import { isInProgress, POLL_INTERVAL_MS } from '../domain/status'

/** How many scans the queue reads: more than a rescan of every company queues. */
export const QUEUE_LIMIT = 300

/** After a scan stops or leaves the queue, the scans, the companies, and the menu's count are read again. */
function useRefresh(): () => void {
  const queryClient = useQueryClient()
  return () => {
    void queryClient.invalidateQueries({ queryKey: listAnalysesQueryKey() })
    void queryClient.invalidateQueries({ queryKey: listCompetitorsQueryKey() })
  }
}

/**
 * The scans that run and the scans that wait, read again every two seconds while any is in
 * progress, and the ways to stop them: one scan, every waiting scan, or every scan of a rescan.
 */
export function useScanQueue() {
  const refresh = useRefresh()
  const list = useQuery({
    ...listAnalysesOptions({ query: { limit: QUEUE_LIMIT } }),
    refetchInterval: (q: { state: { data: AnalysisList | undefined } }) => (q.state.data?.items.some((s) => isInProgress(s.status)) ? POLL_INTERVAL_MS : false),
  })
  const stopOne = useMutation(stopAnalysisMutation())
  const removeAll = useMutation(removeQueuedAnalysesMutation())
  const stopAll = useMutation(stopRescanMutation())
  const notice = ref<string | null>(null)
  const queue = computed(() => scanQueue<AnalysisSummary>(list.data.value?.items ?? []))
  const failure = computed(() => stopOne.error.value ?? removeAll.error.value ?? stopAll.error.value)

  function done(message: string): () => void {
    return () => {
      notice.value = message
      refresh()
    }
  }

  function begin(): void {
    notice.value = null
    stopOne.reset()
    removeAll.reset()
    stopAll.reset()
  }

  return {
    running: computed(() => queue.value.running),
    waiting: computed(() => queue.value.waiting),
    isLoading: list.isPending,
    error: computed(() => (list.error.value ? describeError(list.error.value) : null)),
    isStopping: computed(() => stopOne.isPending.value || removeAll.isPending.value || stopAll.isPending.value),
    /** What the last stop did, for a live region. */
    notice,
    stopError: computed(() => (failure.value ? describeError(failure.value) : null)),
    /** Stops a running scan, which keeps the pages it read, or takes a waiting one off the queue. */
    stop: (scan: Pick<AnalysisSummary, 'id' | 'status'>, name: string) => {
      begin()
      const message = scan.status === 'queued' ? `The scan of ${name} is removed from the queue.` : `The scan of ${name} is stopped. It keeps the pages it read.`
      stopOne.mutate({ path: { analysisId: scan.id } }, { onSuccess: done(message) })
    },
    /** Takes every waiting scan off the queue; the running ones run on. */
    removeWaiting: () => {
      begin()
      removeAll.mutate({}, { onSuccess: done('Every waiting scan is removed from the queue.') })
    },
    /** Stops every scan a rescan queued; a scan started on its own runs on. */
    stopRescan: () => {
      begin()
      stopAll.mutate({}, { onSuccess: done('The rescan is stopped. The scans it finished are kept.') })
    },
  }
}

/** Stops the AI readiness check: the sites it checked keep what it recorded. */
export function useStopReadinessCheck() {
  const queryClient = useQueryClient()
  const stop = useMutation(stopReadinessCheckMutation())
  return {
    isStopping: stop.isPending,
    error: computed(() => (stop.error.value ? describeError(stop.error.value) : null)),
    stop: () => {
      stop.mutate({}, { onSuccess: () => void queryClient.invalidateQueries({ queryKey: getAiReadinessQueryKey() }) })
    },
  }
}
