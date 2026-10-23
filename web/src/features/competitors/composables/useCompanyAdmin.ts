import { useMutation, useQueryClient } from '@tanstack/vue-query'
import { computed } from 'vue'
import {
  addCompetitorMutation,
  deleteCompetitorMutation,
  listAnalysesQueryKey,
  listCompetitorsQueryKey,
  startRescansMutation,
  updateCompetitorMutation,
} from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { Competitor, CompetitorUpdate, RescanBatch, RescanRequest } from '../../../infrastructure/api/types.gen'
import { companyProblem } from '../domain/admin'

/** After a change the company list, its detail and the recent scans are asked for again. */
function useRefresh(): () => void {
  const queryClient = useQueryClient()
  return () => {
    void queryClient.invalidateQueries({ queryKey: listCompetitorsQueryKey() })
    void queryClient.invalidateQueries({ queryKey: listAnalysesQueryKey() })
    void queryClient.invalidateQueries({ queryKey: [{ _id: 'getCompetitor' }] })
    void queryClient.invalidateQueries({ queryKey: [{ _id: 'getCompetitorProfile' }] })
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
            refresh()
            onRemoved()
          },
        },
      )
    },
  }
}

/** Queues scans of named companies, or of every company with data; administrators only. */
export function useRescan() {
  const refresh = useRefresh()
  const rescan = useMutation(startRescansMutation())
  return {
    isQueuing: rescan.isPending,
    batch: computed<RescanBatch | undefined>(() => rescan.data.value),
    error: computed(() => (rescan.error.value ? companyProblem(rescan.error.value) : null)),
    rescan: (body: RescanRequest) => {
      rescan.mutate({ body }, { onSuccess: refresh })
    },
  }
}
