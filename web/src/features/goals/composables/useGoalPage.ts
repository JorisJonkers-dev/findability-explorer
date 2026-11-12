import { computed, type Ref } from 'vue'
import { useAiSearchSummary } from '../../ai-search/composables/useAiSearchSummary'
import { useOverview } from '../../home/composables/useOverview'
import { usePromptResults } from '../../prompt-test/composables/usePromptTest'
import { latestRun } from '../../prompt-test/domain/results'
import { type BreakdownGroup, type Bar, goalPage, historyMeasure, isPromptGoal, promptBreakdown, promptDataLine, readinessBreakdown, targetOfSlug, termsBreakdown } from '../domain/goals'
import { useGoals } from './useGoals'

/** What Broken down shows: per model, scope, and family; per readiness check; or per language. */
export type Breakdown =
  | { kind: 'groups'; groups: BreakdownGroup[] }
  | { kind: 'checks'; checks: ReturnType<typeof readinessBreakdown> }
  | { kind: 'terms'; rows: Bar[]; max: number }

/** One goal's page: the goals, and the figures its data line and Broken down need. */
export function useGoalPage(measure: Ref<string>) {
  const goals = useGoals()
  const results = usePromptResults()
  const summary = useAiSearchSummary()
  const figures = useOverview()
  const target = computed(() => targetOfSlug(measure.value))
  const page = computed(() => (goals.facts.value && target.value ? goalPage(goals.facts.value, target.value) : undefined))

  return {
    target,
    page,
    isLoading: goals.isLoading,
    error: goals.error,
    missing: computed(() => !goals.isLoading.value && !goals.error.value && page.value === undefined),
    history: computed(() => (target.value ? historyMeasure(target.value) : undefined)),
    helpTopic: computed(() => {
      const t = target.value
      if (!t) return undefined
      if (isPromptGoal(t)) return 'prompt-test'
      return t === 'readinessChecks' ? 'readiness' : 'terms'
    }),
    dataLine: computed(() => {
      const t = target.value
      if (!t) return undefined
      if (!isPromptGoal(t)) return 'From the newest scan of each site.'
      return results.results.value ? promptDataLine(latestRun(results.results.value)) : undefined
    }),
    breakdown: computed<Breakdown | undefined>(() => {
      const t = target.value
      if (!t) return undefined
      if (isPromptGoal(t)) return results.results.value ? { kind: 'groups', groups: promptBreakdown(t, latestRun(results.results.value)?.cells ?? []) } : undefined
      if (t === 'readinessChecks') return summary.summary.value ? { kind: 'checks', checks: readinessBreakdown(summary.summary.value.checks) } : undefined
      return figures.overview.value ? { kind: 'terms', ...termsBreakdown(figures.overview.value.terms.languages) } : undefined
    }),
    breakdownError: computed(() => {
      const t = target.value
      if (!t) return null
      if (isPromptGoal(t)) return results.error.value
      return t === 'readinessChecks' ? summary.error.value : figures.error.value
    }),
  }
}
