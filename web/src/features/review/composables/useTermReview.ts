import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, type Ref, ref } from 'vue'
import {
  acceptTermCandidateMutation,
  listTermCandidatesOptions,
  rejectTermCandidateMutation,
  undoTermDecisionMutation,
} from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { TermAcceptance, TermCandidate, TermCandidateCounts, TermCandidateStatus } from '../../../infrastructure/api/types.gen'
import { isForbidden } from '../../settings/domain/problems'
import { reviewProblem, shouldRetry } from '../domain/candidates'

/** Server state for the candidates with one status; a 403 means the account is no administrator. */
export function useTermCandidates(status: Ref<TermCandidateStatus>) {
  const query = useQuery(computed(() => ({ ...listTermCandidatesOptions({ query: { status: status.value } }), retry: shouldRetry })))
  return {
    candidates: computed<TermCandidate[]>(() => (query.data.value ? query.data.value.items : [])),
    counts: computed<TermCandidateCounts | undefined>(() => query.data.value?.counts),
    isLoading: query.isPending,
    forbidden: computed(() => isForbidden(query.error.value)),
    error: computed(() => (query.error.value ? reviewProblem(query.error.value) : null)),
  }
}

/**
 * Accepts, rejects or undoes the decision on a candidate. Each moves it to another tab, so every
 * list is asked for again, and so is the map: an accepted term joins it, an undone one leaves it.
 */
export function useTermDecision() {
  const queryClient = useQueryClient()
  const accept = useMutation(acceptTermCandidateMutation())
  const reject = useMutation(rejectTermCandidateMutation())
  const undo = useMutation(undoTermDecisionMutation())
  const notice = ref<string | null>(null)
  const last = ref<'accept' | 'reject' | 'undo' | null>(null)

  function done(message: string): void {
    notice.value = message
    void queryClient.invalidateQueries({ queryKey: [{ _id: 'listTermCandidates' }] })
    void queryClient.invalidateQueries({ queryKey: [{ _id: 'searchTerms' }] })
  }

  function begin(action: 'accept' | 'reject' | 'undo'): void {
    last.value = action
    notice.value = null
  }

  return {
    isDeciding: computed(() => accept.isPending.value || reject.isPending.value || undo.isPending.value),
    /** What the last decision did, for a live region. */
    notice,
    error: computed(() => {
      const error = last.value === 'accept' ? accept.error.value : last.value === 'reject' ? reject.error.value : undo.error.value
      return error ? reviewProblem(error) : null
    }),
    accept: (candidateId: string, body: TermAcceptance) => {
      begin('accept')
      accept.mutate({ path: { candidateId }, body }, { onSuccess: () => { done(`Added “${body.term}” to the terminology map.`) } })
    },
    reject: (candidateId: string, phrase: string) => {
      begin('reject')
      reject.mutate({ path: { candidateId } }, { onSuccess: () => { done(`Rejected “${phrase}”.`) } })
    },
    undo: (candidateId: string, phrase: string) => {
      begin('undo')
      undo.mutate({ path: { candidateId } }, { onSuccess: () => { done(`“${phrase}” is back in the queue.`) } })
    },
  }
}
