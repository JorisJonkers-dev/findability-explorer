import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, watch } from 'vue'
import {
  addCompetitorMutation,
  deleteCompetitorMutation,
  listAnalysesOptions,
  listAnalysesQueryKey,
  listCompetitorsQueryKey,
  startRescansMutation,
  updateCompetitorMutation,
} from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { AnalysisList, Competitor, CompetitorUpdate, RescanBatch, RescanRequest } from '../../../infrastructure/api/types.gen'
import { POLL_INTERVAL_MS } from '../../scan/domain/status'
import { companyProblem, rescanProgress } from '../domain/admin'

/** The most scans one list returns: more than the server's queue holds, so a whole re-scan fits. */
const RESCAN_LIST_LIMIT = 300

/** Whether a query asks about this company: its detail, profile or scans. */
function isAbout(competitorId: string, queryKey: readonly unknown[]): boolean {
  const key = queryKey[0] as { path?: { competitorId?: string } } | undefined
  return key?.path?.competitorId === competitorId
}

/**
 * After a change the company list, its detail and the recent scans are asked for again. A removed
 * company is left out: its card is still on screen until the address changes, and asking for it
 * again would only answer 404.
 */
function useRefresh(): (removed?: string) => void {
  const queryClient = useQueryClient()
  return (removed) => {
    const kept = (query: { queryKey: readonly unknown[] }) => removed === undefined || !isAbout(removed, query.queryKey)
    void queryClient.invalidateQueries({ queryKey: listCompetitorsQueryKey() })
    void queryClient.invalidateQueries({ queryKey: listAnalysesQueryKey() })
    void queryClient.invalidateQueries({ queryKey: [{ _id: 'getCompetitor' }], predicate: kept })
    void queryClient.invalidateQueries({ queryKey: [{ _id: 'getCompetitorProfile' }], predicate: kept })
  }
}

/** Adds a company by its name and website; administrators only. */
export function useAddCompany() {
  const refresh = useRefresh()
  const add = useMutation(addCompetitorMutation())
  return {
    isAdding: add.isPending,
    error: computed(() => (add.error.value ? companyProblem(add.error.value) : null)),
    add: (name: string, url: string, onAdded: (company: Competitor) => void) => {
      add.mutate(
        { body: { name, url } },
        {
          onSuccess: (company) => {
            refresh()
            onAdded(company)
          },
        },
      )
    },
    reset: () => {
      add.reset()
    },
  }
}

/** Changes a company's name or website, or archives or restores it; administrators only. */
export function useUpdateCompany() {
  const refresh = useRefresh()
  const update = useMutation(updateCompetitorMutation())
  return {
    isSaving: update.isPending,
    error: computed(() => (update.error.value ? companyProblem(update.error.value) : null)),
    update: (competitorId: string, body: CompetitorUpdate, onSaved: () => void) => {
      update.mutate(
        { path: { competitorId }, body },
        {
          onSuccess: () => {
            refresh()
            onSaved()
          },
        },
      )
    },
    reset: () => {
      update.reset()
    },
  }
}

/** Deletes an added company with its scans, or archives a researched one; administrators only. */
export function useRemoveCompany() {
  const refresh = useRefresh()
  const remove = useMutation(deleteCompetitorMutation())
  return {
    isRemoving: remove.isPending,
    error: computed(() => (remove.error.value ? companyProblem(remove.error.value) : null)),
    remove: (competitorId: string, onRemoved: () => void) => {
      remove.mutate(
        { path: { competitorId } },
        {
          onSuccess: () => {
            refresh(competitorId)
            onRemoved()
          },
        },
      )
    },
  }
}

/**
 * Queues scans of named companies, or of every company with data; administrators only. Once
 * queued, the scans are followed until each has finished, and the companies are then asked for
 * again so their scan counts show the new scans.
 */
export function useRescan() {
  const refresh = useRefresh()
  const rescan = useMutation(startRescansMutation())
  const batch = computed<RescanBatch | undefined>(() => rescan.data.value)
  const queuedIds = computed(() => (batch.value ? batch.value.queued.map((s) => s.id) : []))
  const scans = useQuery(
    computed(() => ({
      ...listAnalysesOptions({ query: { limit: RESCAN_LIST_LIMIT } }),
      enabled: queuedIds.value.length > 0,
      refetchInterval: (q: { state: { data: AnalysisList | undefined } }) =>
        q.state.data && rescanProgress(queuedIds.value, q.state.data.items).finished ? false : POLL_INTERVAL_MS,
    })),
  )
  const progress = computed(() =>
    queuedIds.value.length > 0 && scans.data.value ? rescanProgress(queuedIds.value, scans.data.value.items) : null,
  )
  watch(
    () => progress.value?.finished,
    (finished) => {
      if (finished) refresh()
    },
  )
  return {
    isQueuing: rescan.isPending,
    batch,
    progress,
    error: computed(() => (rescan.error.value ? companyProblem(rescan.error.value) : null)),
    rescan: (body: RescanRequest) => {
      rescan.mutate({ body }, { onSuccess: () => { refresh() } })
    },
  }
}
