import { useQuery } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import type { LocationQuery } from 'vue-router'
import { listPromptSetsOptions } from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { PromptAnswer, PromptSets } from '../../../infrastructure/api/types.gen'
import { latestRun } from '../domain/results'
import { filtersFrom, gridOf, KEY, modelLabel, scopeOptions, shown, stateOf } from '../domain/responses'
import { promptProblem } from '../domain/settings'
import { usePromptAnswers, usePromptResults } from './usePromptTest'

/** The published Prompt sets, newest first, with their questions and translations. */
export function usePromptSets() {
  const query = useQuery(listPromptSetsOptions())
  return {
    sets: computed<PromptSets | undefined>(() => query.data.value),
    error: computed(() => (query.error.value ? promptProblem(query.error.value) : null)),
  }
}

/**
 * Every response: the run the address names, or the newest with responses, its responses, and the
 * ones the filters in the address keep, laid out as the grid.
 */
export function useEveryResponse(query: Ref<LocationQuery>) {
  const { results, isLoading, error } = usePromptResults()
  const runs = computed(() => results.value?.runs ?? [])
  const run = computed(() => runs.value.find((r) => r.id === query.value.run) ?? latestRun(results.value))
  const { answers, isLoading: answersLoading, error: answersError } = usePromptAnswers(
    computed(() => run.value?.id),
    computed(() => ({})),
    computed(() => run.value?.status === 'running'),
  )
  const all = computed<PromptAnswer[]>(() => answers.value?.items ?? [])
  const filters = computed(() => filtersFrom(query.value))
  const list = computed(() => shown(all.value, filters.value))
  return {
    runs,
    run,
    isLoading,
    error,
    answersLoading,
    answersError,
    filters,
    list,
    grid: computed(() => gridOf({ models: run.value?.models ?? [] }, list.value)),
    scopes: computed(() => scopeOptions(all.value)),
    models: computed(() => [{ value: 'all', label: 'All models' }, ...(run.value?.models ?? []).map((m) => ({ value: m.provider, label: modelLabel(m.provider, m.model) }))]),
    key: computed(() => KEY.filter((k) => k.state !== 'unread' || list.value.some((a) => stateOf(a) === 'unread'))),
  }
}

/** One response, found in the run the address names, or else in the newest run with responses. */
export function useOneResponse(id: Ref<string>, runId: Ref<string | undefined>) {
  const { results, isLoading, error } = usePromptResults()
  const { answers, isLoading: answersLoading, error: answersError } = usePromptAnswers(
    computed(() => runId.value ?? latestRun(results.value)?.id),
    computed(() => ({})),
  )
  return {
    run: computed(() => answers.value?.run),
    answer: computed(() => answers.value?.items.find((a) => a.id === id.value)),
    isLoading: computed(() => (runId.value === undefined && isLoading.value) || answersLoading.value),
    error: computed(() => answersError.value ?? (runId.value === undefined ? error.value : null)),
  }
}
