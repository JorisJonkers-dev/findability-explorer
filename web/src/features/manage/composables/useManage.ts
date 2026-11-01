import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, ref } from 'vue'
import {
  getActivityOptions,
  getPromptTestHistoryOptions,
  listAnalysesOptions,
  listPromptSetsOptions,
  removeQueuedAnalysesMutation,
  stopAnalysisMutation,
  stopPromptRunMutation,
  stopCandidateReviewMutation,
  stopCompetitorCheckMutation,
  stopReadinessCheckMutation,
  stopRescanMutation,
  stopTermReviewMutation,
} from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { Activity, AnalysisList, AnalysisSummary, PromptSets, PromptTestHistory } from '../../../infrastructure/api/types.gen'
import { describeError } from '../../../shared/describeError'
import { promptProblem } from '../../prompt-test/domain/settings'
import { isInProgress, POLL_INTERVAL_MS } from '../../scan/domain/status'
import type { RunningWork } from '../domain/activity'

/** How many scans Activity reads: enough for the queue, what runs, and the last 20 pieces of work. */
export const WORK_SCANS = 60

/** The latest scans, newest first, asked for again while one waits or runs. */
export function useWorkScans() {
  const query = useQuery({
    ...listAnalysesOptions({ query: { limit: WORK_SCANS } }),
    refetchInterval: (q: { state: { data: AnalysisList | undefined } }) =>
      q.state.data?.items.some((s) => isInProgress(s.status)) ? POLL_INTERVAL_MS : false,
  })
  return {
    scans: computed<AnalysisSummary[]>(() => query.data.value?.items ?? []),
    error: computed(() => (query.error.value ? describeError(query.error.value) : null)),
  }
}

/** How often Activity is read again while work runs. */
export const ACTIVITY_POLL_MS = 5000

/** What runs, the last 20 pieces of work that ended, and this month's spending, read again while work runs. */
export function useActivity() {
  const query = useQuery({
    ...getActivityOptions(),
    refetchInterval: (q: { state: { data: Activity | undefined } }) => ((q.state.data?.running.length ?? 0) > 0 ? ACTIVITY_POLL_MS : false),
  })
  return {
    activity: computed<Activity | undefined>(() => query.data.value),
    error: computed(() => (query.error.value ? promptProblem(query.error.value) : null)),
  }
}

/** Every published version of the Prompt set with its questions, newest first. */
export function usePromptSets() {
  const query = useQuery(listPromptSetsOptions())
  return {
    sets: computed<PromptSets | undefined>(() => query.data.value),
    isLoading: query.isPending,
    error: computed(() => (query.error.value ? promptProblem(query.error.value) : null)),
  }
}

/** The published versions, oldest first, and the figures per run. */
export function usePromptHistory() {
  const query = useQuery(getPromptTestHistoryOptions())
  return {
    history: computed<PromptTestHistory | undefined>(() => query.data.value),
    error: computed(() => (query.error.value ? promptProblem(query.error.value) : null)),
  }
}

/**
 * Stops a piece of work, or takes every waiting scan off the queue. The lists it shows in are read
 * again, so the work moves to Recent work once it has stopped.
 */
export function useStopWork() {
  const queryClient = useQueryClient()
  const mutations = {
    promptRun: useMutation(stopPromptRunMutation()),
    scan: useMutation(stopAnalysisMutation()),
    rescan: useMutation(stopRescanMutation()),
    readinessCheck: useMutation(stopReadinessCheckMutation()),
    termReview: useMutation(stopTermReviewMutation()),
    candidateReview: useMutation(stopCandidateReviewMutation()),
    competitorCheck: useMutation(stopCompetitorCheckMutation()),
    queue: useMutation(removeQueuedAnalysesMutation()),
  }
  const last = ref<keyof typeof mutations | null>(null)
  const notice = ref<string | null>(null)

  function done(message: string, ...ids: string[]): { onSuccess: () => void } {
    return {
      onSuccess: () => {
        notice.value = message
        for (const id of ['getActivity', ...ids]) void queryClient.invalidateQueries({ queryKey: [{ _id: id }] })
      },
    }
  }

  function begin(which: keyof typeof mutations): void {
    last.value = which
    notice.value = null
  }

  return {
    isStopping: computed(() => Object.values(mutations).some((m) => m.isPending.value)),
    notice,
    error: computed(() => {
      const error = last.value ? mutations[last.value].error.value : null
      return error ? promptProblem(error) : null
    }),
    stop: (work: Pick<RunningWork, 'kind' | 'id' | 'label'>) => {
      switch (work.kind) {
        case 'promptRun':
          begin('promptRun')
          mutations.promptRun.mutate(
            { path: { runId: work.id } },
            done('The Prompt run stops within a minute. Its responses so far are kept.', 'getPromptTestResults', 'getPromptTestSettings'),
          )
          return
        case 'scan':
          begin('scan')
          mutations.scan.mutate({ path: { analysisId: work.id } }, done('The scan stops after the page it reads.', 'listAnalyses'))
          return
        case 'rescan':
          begin('rescan')
          mutations.rescan.mutate({}, done('The rescan stops. The waiting scans are taken off the queue.', 'listAnalyses'))
          return
        case 'readinessCheck':
          begin('readinessCheck')
          mutations.readinessCheck.mutate({}, done('The AI readiness check is stopped.', 'getAiReadiness'))
          return
        case 'termReview':
          begin('termReview')
          mutations.termReview.mutate({}, done('The term review is stopped. What it decided is kept.', 'listTermCandidates'))
          return
        case 'candidateReview':
          begin('candidateReview')
          mutations.candidateReview.mutate({}, done('The review of the competitor candidates is stopped. What it decided is kept.', 'listCompetitorCandidates'))
          return
        case 'competitorCheck':
          begin('competitorCheck')
          mutations.competitorCheck.mutate({}, done('The check of the competitor list is stopped. What it decided is kept.', 'listCompetitorChecks'))
          return
        case 'recommendationRun':
          // A Recommendation run has no Stop yet.
          return
      }
    },
    removeQueue: () => {
      begin('queue')
      mutations.queue.mutate({}, done('The waiting scans are taken off the queue.', 'listAnalyses'))
    },
  }
}
