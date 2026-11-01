import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, type Ref, ref, watch } from 'vue'
import {
  getCompetitorDefinitionOptions,
  getCompetitorDefinitionQueryKey,
  listCompetitorChecksQueryKey,
  restoreCompetitorDefinitionMutation,
  saveCompetitorDefinitionMutation,
  startCandidateReviewMutation,
  startCompetitorCheckMutation,
} from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { CompetitorDefinition } from '../../../infrastructure/api/types.gen'
import { definitionProblem } from '../domain/definition'

/**
 * The competitor definition for an administrator: its text in a box to edit, saved as a new
 * version; a version copied into the box, which saving restores; and Check again, which starts
 * Claude Haiku's review of the candidates and its check of the list.
 */
export function useCompetitorDefinition(enabled: Ref<boolean>) {
  const queryClient = useQueryClient()
  const query = useQuery(computed(() => ({ ...getCompetitorDefinitionOptions(), enabled: enabled.value })))
  const save = useMutation(saveCompetitorDefinitionMutation())
  const restore = useMutation(restoreCompetitorDefinitionMutation())
  const review = useMutation(startCandidateReviewMutation())
  const check = useMutation(startCompetitorCheckMutation())
  const definition = computed<CompetitorDefinition | undefined>(() => query.data.value)
  const draft = ref('')
  const copied = ref<{ version: number; text: string } | null>(null)
  const notice = ref<string | null>(null)
  const refusal = ref<string | null>(null)
  watch(
    () => definition.value?.current.text,
    (text) => {
      if (text !== undefined) draft.value = text
    },
    { immediate: true },
  )

  function done(next: CompetitorDefinition): void {
    queryClient.setQueryData(getCompetitorDefinitionQueryKey(), next)
    copied.value = null
    draft.value = next.current.text
    notice.value = `The definition is saved as version ${String(next.current.version)}. Nothing was rechecked yet.`
  }
  function failed(error: unknown): void {
    refusal.value = definitionProblem(error)
  }

  return {
    definition,
    draft,
    dirty: computed(() => definition.value !== undefined && draft.value !== definition.value.current.text),
    isSaving: computed(() => save.isPending.value || restore.isPending.value),
    isChecking: computed(() => review.isPending.value || check.isPending.value),
    notice,
    error: computed(() => refusal.value ?? (query.error.value ? definitionProblem(query.error.value) : null)),
    restore: (version: number) => {
      const v = definition.value?.versions.find((x) => x.version === version)
      if (!v) return
      draft.value = v.text
      copied.value = { version, text: v.text }
      notice.value = `Version ${String(version)} is copied into the text box. Save it to make a new version.`
    },
    save: () => {
      notice.value = null
      refusal.value = null
      const text = draft.value.trim()
      if (text === '') {
        refusal.value = 'The definition cannot be empty.'
        return
      }
      if (copied.value && copied.value.text === draft.value) {
        restore.mutate({ path: { version: copied.value.version } }, { onSuccess: done, onError: failed })
        return
      }
      save.mutate({ body: { text } }, { onSuccess: done, onError: failed })
    },
    checkAgain: () => {
      notice.value = null
      refusal.value = null
      const started = () => {
        notice.value = 'Claude Haiku is checking the candidates and the list again.'
        void queryClient.invalidateQueries({ queryKey: listCompetitorChecksQueryKey() })
        void queryClient.invalidateQueries({ queryKey: [{ _id: 'listCompetitorCandidates' }] })
      }
      review.mutate({}, { onSuccess: started, onError: failed })
      check.mutate({}, { onSuccess: started, onError: failed })
    },
  }
}
