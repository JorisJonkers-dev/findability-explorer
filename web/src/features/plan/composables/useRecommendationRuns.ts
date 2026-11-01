import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, ref, watch } from 'vue'
import {
  getPlanQueryKey,
  getRecommendationRunsOptions,
  getRecommendationRunsQueryKey,
  startRecommendationRunMutation,
  stopRecommendationRunMutation,
} from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { RecommendationRuns } from '../../../infrastructure/api/types.gen'
import { runProblem } from '../domain/runs'

/** How often the runs are asked for again while one runs. */
export const RUN_POLL_MS = 3000

/** The Recommendation run for an administrator: what a run would do, Start, Stop, and the past runs, asked for again while one runs. */
export function useRecommendationRuns() {
  const queryClient = useQueryClient()
  const query = useQuery({
    ...getRecommendationRunsOptions(),
    refetchInterval: (q: { state: { data?: RecommendationRuns } }) => (q.state.data?.running ? RUN_POLL_MS : false),
  })
  const start = useMutation(startRecommendationRunMutation())
  const stop = useMutation(stopRecommendationRunMutation())
  const notice = ref<string | null>(null)
  const refused = computed(() => start.error.value ?? stop.error.value ?? query.error.value)
  const changed = () => {
    void queryClient.invalidateQueries({ queryKey: getRecommendationRunsQueryKey() })
    void queryClient.invalidateQueries({ queryKey: getPlanQueryKey() })
  }
  // A run that ends replaced the recommendations: the plan is asked for again.
  watch(
    () => query.data.value?.running,
    (now, before) => {
      if (before === true && now === false) void queryClient.invalidateQueries({ queryKey: getPlanQueryKey() })
    },
  )
  return {
    runs: computed<RecommendationRuns | undefined>(() => query.data.value),
    isStarting: start.isPending,
    isStopping: stop.isPending,
    notice,
    error: computed(() => (refused.value ? runProblem(refused.value) : null)),
    start: () => {
      stop.reset()
      notice.value = null
      start.mutate(
        {},
        {
          onSuccess: (r) => {
            queryClient.setQueryData(getRecommendationRunsQueryKey(), r)
            notice.value = 'The Recommendation run started. It replaces the open recommendations only when it ends.'
          },
        },
      )
    },
    stop: () => {
      start.reset()
      notice.value = null
      stop.mutate(
        {},
        {
          onSuccess: () => {
            notice.value = 'The run stops after the batch it writes. The recommendations stay as they are.'
            changed()
          },
        },
      )
    },
  }
}
