import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, ref } from 'vue'
import { addTermMutation, listHandTermsOptions, putBackTermMutation, removeTermMutation } from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { HandTerm, NewTerm, TermRef } from '../../../infrastructure/api/types.gen'
import { describeError } from '../../../shared/describeError'
import { termName } from '../domain/handTerms'

/**
 * The search terms added and removed by hand, and adding, removing, and putting one back. Each
 * change asks for the lists and the whole map again, since every term count follows them.
 */
export function useHandTerms() {
  const queryClient = useQueryClient()
  const query = useQuery(listHandTermsOptions())
  const add = useMutation(addTermMutation())
  const remove = useMutation(removeTermMutation())
  const putBack = useMutation(putBackTermMutation())
  const notice = ref('')
  const failure = ref<string | null>(null)
  const isChanging = ref(false)

  async function run(change: () => Promise<unknown>, message: string): Promise<boolean> {
    notice.value = ''
    failure.value = null
    isChanging.value = true
    try {
      await change()
    } catch (error) {
      failure.value = describeError(error)
      return false
    } finally {
      isChanging.value = false
    }
    notice.value = message
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: [{ _id: 'listHandTerms' }] }),
      queryClient.invalidateQueries({ queryKey: [{ _id: 'searchTerms' }] }),
    ])
    return true
  }

  return {
    added: computed<HandTerm[]>(() => query.data.value?.added ?? []),
    removed: computed<HandTerm[]>(() => query.data.value?.removed ?? []),
    isLoading: query.isPending,
    error: computed(() => (query.error.value ? describeError(query.error.value) : null)),
    isChanging,
    /** What the last change did, for a live region; failure says why it did not. */
    notice,
    failure,
    add: (body: NewTerm) => run(() => add.mutateAsync({ body }), `${termName(body)} is on the list. Scans look for it from the next scan on.`),
    remove: (body: TermRef) => run(() => remove.mutateAsync({ body }), `${termName(body)} is off the list. It is no longer counted or looked for.`),
    putBack: (ref: TermRef) => run(() => putBack.mutateAsync({ query: ref }), `${termName(ref)} is back on the list.`),
  }
}
