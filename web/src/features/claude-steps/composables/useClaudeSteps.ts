import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import {
  getClaudeStepOptions,
  getClaudeStepQueryKey,
  listClaudeStepsOptions,
  listClaudeStepsQueryKey,
  saveClaudeStepVersionMutation,
} from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { ClaudeStep, ClaudeStepChange, ClaudeStepId, ClaudeStepSummary } from '../../../infrastructure/api/types.gen'
import { describeError } from '../../../shared/describeError'

/** Server state for every Claude step: its model, version, and the cost of one job. */
export function useClaudeSteps() {
  const query = useQuery(listClaudeStepsOptions())
  return {
    steps: computed<ClaudeStepSummary[]>(() => query.data.value?.items ?? []),
    isLoading: query.isPending,
    error: computed(() => (query.error.value ? describeError(query.error.value) : null)),
  }
}

/** Server state for one Claude step, and saving a new version of it, which answers with the step. */
export function useClaudeStep(id: Ref<ClaudeStepId>) {
  const queryClient = useQueryClient()
  const query = useQuery(computed(() => getClaudeStepOptions({ path: { stepId: id.value } })))
  const save = useMutation(saveClaudeStepVersionMutation())
  return {
    step: computed<ClaudeStep | undefined>(() => query.data.value),
    isLoading: query.isPending,
    error: computed(() => (query.error.value ? describeError(query.error.value) : null)),
    isSaving: save.isPending,
    saveError: computed(() => (save.error.value ? describeError(save.error.value) : null)),
    save: (body: ClaudeStepChange, onSaved: (step: ClaudeStep) => void) => {
      save.mutate(
        { path: { stepId: id.value }, body },
        {
          onSuccess: (step) => {
            queryClient.setQueryData(getClaudeStepQueryKey({ path: { stepId: id.value } }), step)
            void queryClient.invalidateQueries({ queryKey: listClaudeStepsQueryKey() })
            onSaved(step)
          },
        },
      )
    },
  }
}
