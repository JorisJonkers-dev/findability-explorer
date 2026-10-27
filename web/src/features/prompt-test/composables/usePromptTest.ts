import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, type Ref, ref } from 'vue'
import {
  addCompetitorCandidateMutation,
  dismissCompetitorCandidateMutation,
  getPromptTestResultsOptions,
  getPromptTestSettingsOptions,
  getPromptTestSettingsQueryKey,
  listCompetitorCandidatesOptions,
  listPromptAnswersOptions,
  retryPromptRunMutation,
  startPromptRunMutation,
  undoCompetitorCandidateMutation,
  updatePromptTestSettingsMutation,
} from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type {
  CompetitorCandidate,
  CompetitorCandidateCounts,
  CompetitorCandidateStatus,
  ListPromptAnswersData,
  PromptAnswerList,
  PromptMode,
  PromptTestResults,
  PromptTestSettings,
  PromptTestSettingsChange,
} from '../../../infrastructure/api/types.gen'
import { isForbidden } from '../../settings/domain/problems'
import { costLabel } from '../domain/results'
import { promptProblem } from '../domain/settings'

/** How often the results are asked for again while a run is running. */
export const RUN_POLL_MS = 5000

/** Server state for the Prompt test's results, asked for again while a run is running. */
export function usePromptResults() {
  const query = useQuery({
    ...getPromptTestResultsOptions(),
    refetchInterval: (q: { state: { data: PromptTestResults | undefined } }) =>
      q.state.data?.runs.some((r) => r.status === 'running') ? RUN_POLL_MS : false,
  })
  return {
    results: computed<PromptTestResults | undefined>(() => query.data.value),
    isLoading: query.isPending,
    error: computed(() => (query.error.value ? promptProblem(query.error.value) : null)),
  }
}

/** The answers of a run the filter keeps, asked for again while poll is true; nothing is asked until a run is named. */
export function usePromptAnswers(runId: Ref<string | undefined>, filter: Ref<NonNullable<ListPromptAnswersData['query']>>, poll: Ref<boolean> = ref(false)) {
  const query = useQuery(
    computed(() => ({
      ...listPromptAnswersOptions({ path: { runId: runId.value ?? '' }, query: filter.value }),
      enabled: runId.value !== undefined,
      refetchInterval: poll.value ? RUN_POLL_MS : (false as const),
    })),
  )
  return {
    answers: computed<PromptAnswerList | undefined>(() => query.data.value),
    isLoading: computed(() => query.isFetching.value && query.data.value === undefined),
    error: computed(() => (query.error.value ? promptProblem(query.error.value) : null)),
  }
}

/** Server state for the Prompt test's settings; a 403 means the account is no administrator. */
export function usePromptSettings() {
  const query = useQuery(getPromptTestSettingsOptions())
  return {
    settings: computed<PromptTestSettings | undefined>(() => query.data.value),
    isLoading: query.isPending,
    forbidden: computed(() => isForbidden(query.error.value)),
    error: computed(() => (query.error.value ? promptProblem(query.error.value) : null)),
  }
}

/** Changes the settings; they answer with the settings, which replace the ones shown. */
export function useUpdatePromptSettings() {
  const queryClient = useQueryClient()
  const update = useMutation(updatePromptTestSettingsMutation())
  const notice = ref<string | null>(null)
  return {
    isSaving: update.isPending,
    notice,
    error: computed(() => (update.error.value ? promptProblem(update.error.value) : null)),
    save: (body: PromptTestSettingsChange, done: string) => {
      notice.value = null
      update.mutate(
        { body },
        {
          onSuccess: (settings) => {
            queryClient.setQueryData(getPromptTestSettingsQueryKey(), settings)
            notice.value = done
          },
        },
      )
    },
  }
}

/** Starts a Prompt run now; the results are asked for again, and follow it while it runs. */
export function useStartRun() {
  const queryClient = useQueryClient()
  const start = useMutation(startPromptRunMutation())
  const notice = ref<string | null>(null)
  return {
    isStarting: start.isPending,
    notice,
    error: computed(() => (start.error.value ? promptProblem(start.error.value) : null)),
    start: () => {
      notice.value = null
      start.mutate(
        {},
        {
          onSuccess: (run) => {
            notice.value = `A Prompt run started: ${String(run.tasks)} answers to ask.`
            void queryClient.invalidateQueries({ queryKey: [{ _id: 'getPromptTestResults' }] })
            void queryClient.invalidateQueries({ queryKey: getPromptTestSettingsQueryKey() })
          },
        },
      )
    },
  }
}

/** Asks a run's failed answers again, or one; the results and the run's answers follow it. */
export function useRetryRun() {
  const queryClient = useQueryClient()
  const retry = useMutation(retryPromptRunMutation())
  const notice = ref<string | null>(null)
  return {
    isRetrying: retry.isPending,
    notice,
    error: computed(() => (retry.error.value ? promptProblem(retry.error.value) : null)),
    retry: (runId: string, task?: { promptId: string; provider: string; mode: PromptMode }) => {
      notice.value = null
      retry.mutate(
        { path: { runId }, body: task ?? {} },
        {
          onSuccess: (r) => {
            const answers = r.tasks === 1 ? 'one answer' : `${String(r.tasks)} answers`
            notice.value = `Asking ${answers} again, for about ${costLabel(r.estimatedCostUsd)}.`
            void queryClient.invalidateQueries({ queryKey: [{ _id: 'getPromptTestResults' }] })
            void queryClient.invalidateQueries({ queryKey: [{ _id: 'listPromptAnswers' }] })
          },
        },
      )
    },
  }
}

/** Server state for the competitor candidates with one status; a 403 means no administrator. */
export function useCompetitorCandidates(status: Ref<CompetitorCandidateStatus>) {
  const query = useQuery(computed(() => listCompetitorCandidatesOptions({ query: { status: status.value } })))
  return {
    candidates: computed<CompetitorCandidate[]>(() => query.data.value?.items ?? []),
    counts: computed<CompetitorCandidateCounts | undefined>(() => query.data.value?.counts),
    isLoading: query.isPending,
    forbidden: computed(() => isForbidden(query.error.value)),
    error: computed(() => (query.error.value ? promptProblem(query.error.value) : null)),
  }
}

/**
 * Adds, dismisses or undoes the decision on a candidate. Each moves it to another tab, so every
 * list is asked for again, and so are the companies: an added candidate joins them.
 */
export function useCandidateDecision() {
  const queryClient = useQueryClient()
  const add = useMutation(addCompetitorCandidateMutation())
  const dismiss = useMutation(dismissCompetitorCandidateMutation())
  const undo = useMutation(undoCompetitorCandidateMutation())
  const notice = ref<string | null>(null)
  const last = ref<'add' | 'dismiss' | 'undo' | null>(null)

  function done(message: string): void {
    notice.value = message
    void queryClient.invalidateQueries({ queryKey: [{ _id: 'listCompetitorCandidates' }] })
    void queryClient.invalidateQueries({ queryKey: [{ _id: 'listCompetitors' }] })
  }

  function begin(action: 'add' | 'dismiss' | 'undo'): void {
    last.value = action
    notice.value = null
  }

  return {
    isDeciding: computed(() => add.isPending.value || dismiss.isPending.value || undo.isPending.value),
    notice,
    error: computed(() => {
      const error = last.value === 'add' ? add.error.value : last.value === 'dismiss' ? dismiss.error.value : undo.error.value
      return error ? promptProblem(error) : null
    }),
    add: (id: string, name: string, url: string) => {
      begin('add')
      add.mutate({ path: { competitorCandidateId: id }, body: { name, url } }, { onSuccess: () => { done(`Added ${name} to the competitors.`) } })
    },
    dismiss: (id: string, name: string) => {
      begin('dismiss')
      dismiss.mutate({ path: { competitorCandidateId: id } }, { onSuccess: () => { done(`Dismissed ${name}.`) } })
    },
    undo: (id: string, name: string) => {
      begin('undo')
      undo.mutate({ path: { competitorCandidateId: id } }, { onSuccess: () => { done(`${name} is back among the candidates to review.`) } })
    },
  }
}
