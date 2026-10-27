import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed } from 'vue'
import { getPlanOptions, getPlanQueryKey, setPlanTargetsMutation, tickImprovementMutation } from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { Plan, PlanTargetValue } from '../../../infrastructure/api/types.gen'
import { describeError } from '../../../shared/describeError'

/** Server state for the plan, and the administrator's changes to it, which answer with the plan. */
export function usePlan() {
  const queryClient = useQueryClient()
  const query = useQuery(getPlanOptions())
  const tick = useMutation(tickImprovementMutation())
  const targets = useMutation(setPlanTargetsMutation())
  const changed = (plan: Plan) => {
    queryClient.setQueryData(getPlanQueryKey(), plan)
    // Catching up lists the open improvements.
    void queryClient.invalidateQueries({ queryKey: [{ _id: 'getStandings' }] })
  }
  return {
    plan: computed<Plan | undefined>(() => query.data.value),
    isLoading: query.isPending,
    error: computed(() => (query.error.value ? describeError(query.error.value) : null)),
    isSaving: computed(() => tick.isPending.value || targets.isPending.value),
    saveError: computed(() => {
      const e = tick.error.value ?? targets.error.value
      return e ? describeError(e) : null
    }),
    tick: (improvementId: string, done: boolean) => {
      targets.reset()
      tick.mutate({ path: { improvementId }, body: { done } }, { onSuccess: changed })
    },
    setTargets: (values: PlanTargetValue[]) => {
      tick.reset()
      targets.mutate({ body: { targets: values } }, { onSuccess: changed })
    },
  }
}
