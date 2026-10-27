import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, type Ref, ref, watch } from 'vue'
import {
  listCompetitorChecksOptions,
  listCompetitorChecksQueryKey,
  listCompetitorsQueryKey,
  startCompetitorCheckMutation,
} from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { CandidateReviewState, Competitor, CompetitorCheck } from '../../../infrastructure/api/types.gen'
import { reviewPoll } from '../../review/domain/candidates'
import { companyProblem } from '../domain/admin'
import { checksById } from '../domain/checks'
import { useUpdateCompany } from './useCompanyAdmin'

/**
 * Claude Haiku's checks of the companies on the list, for an administrator, asked for again while
 * it checks. When a check ends, the companies are asked for again too: it archives as it goes.
 */
export function useCompetitorChecks(enabled: Ref<boolean>) {
  const queryClient = useQueryClient()
  const query = useQuery(
    computed(() => ({
      ...listCompetitorChecksOptions(),
      enabled: enabled.value,
      refetchInterval: (q: { state: { data?: { review: CandidateReviewState } } }) => reviewPoll(q.state.data?.review),
    })),
  )
  const start = useMutation(startCompetitorCheckMutation())
  const review = computed<CandidateReviewState | undefined>(() => query.data.value?.review)
  watch(
    () => review.value?.running,
    (now, before) => {
      if (before === true && now === false) void queryClient.invalidateQueries({ queryKey: listCompetitorsQueryKey() })
    },
  )
  const error = computed(() => query.error.value ?? start.error.value)
  return {
    checks: computed<Map<string, CompetitorCheck>>(() => checksById(query.data.value?.items ?? [])),
    unchecked: computed(() => query.data.value?.unchecked ?? 0),
    review,
    isStarting: start.isPending,
    error: computed(() => (error.value ? companyProblem(error.value) : null)),
    start: () => {
      start.mutate({}, { onSuccess: () => void queryClient.invalidateQueries({ queryKey: listCompetitorChecksQueryKey() }) })
    },
  }
}

/** Keeps a company Haiku is unsure of on the list, archives it, or restores one Haiku archived. */
export function useListDecision() {
  const queryClient = useQueryClient()
  const { isSaving, error, update } = useUpdateCompany()
  const notice = ref<string | null>(null)
  function send(c: Pick<Competitor, 'id'>, body: { kept: true } | { archived: boolean }, done: string): void {
    notice.value = null
    update(c.id, body, () => {
      notice.value = done
      void queryClient.invalidateQueries({ queryKey: listCompetitorChecksQueryKey() })
    })
  }
  return {
    isSaving,
    error,
    notice,
    keep: (c: Pick<Competitor, 'id' | 'name'>) => {
      send(c, { kept: true }, `${c.name} stays on the list, and Haiku will not archive it.`)
    },
    archive: (c: Pick<Competitor, 'id' | 'name'>) => {
      send(c, { archived: true }, `Archived ${c.name}. Its scans stay.`)
    },
    restore: (c: Pick<Competitor, 'id' | 'name'>) => {
      send(c, { archived: false }, `${c.name} is back on the list, and Haiku will not archive it again.`)
    },
  }
}
