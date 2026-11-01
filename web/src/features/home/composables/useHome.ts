import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import { getPromptTestSettingsOptions } from '../../../infrastructure/api/@tanstack/vue-query.gen'
import { useGoals } from '../../goals/composables/useGoals'
import { goalRows, overviewSummary } from '../../goals/domain/goals'
import { usePromptResults } from '../../prompt-test/composables/usePromptTest'
import { useReadinessTable } from '../../readiness/composables/useReadiness'
import { useMe } from '../../shell/composables/useMe'
import { sectionTiles } from '../domain/cards'
import { latestRows, overviewDataLine } from '../domain/latest'
import { useOverview } from './useOverview'

/**
 * What the Overview shows: the summary, the Goals table, the numbered tiles, and Latest runs and
 * scans, from the plan, the standings, the overview's figures, the Prompt runs, and the scans.
 * Only an administrator may read when the next Prompt run is, so only they ask.
 */
export function useHome() {
  const goals = useGoals()
  const figures = useOverview()
  const results = usePromptResults()
  const readiness = useReadinessTable()
  const { isAdmin } = useMe()
  const settings = useQuery({ ...getPromptTestSettingsOptions(), enabled: isAdmin })

  const facts = goals.facts
  const overview = figures.overview
  return {
    isLoading: computed(() => goals.isLoading.value || figures.isLoading.value || results.isLoading.value || readiness.isLoading.value),
    error: computed(() => goals.error.value ?? figures.error.value ?? results.error.value ?? readiness.error.value),
    dataLine: computed(() => overviewDataLine(facts.value?.standings.runStartedAt, readiness.rows.value)),
    summary: computed(() => (facts.value && overview.value ? overviewSummary(facts.value, overview.value) : '')),
    rows: computed(() => (facts.value ? goalRows(facts.value) : [])),
    tiles: computed(() => (facts.value && overview.value ? sectionTiles(overview.value, facts.value.plan) : [])),
    latest: computed(() =>
      facts.value && overview.value
        ? latestRows({
            runs: results.results.value?.runs ?? [],
            nextRunAt: settings.data.value?.nextDueAt,
            readiness: readiness.rows.value,
            overview: overview.value,
            plan: facts.value.plan,
            now: new Date(),
          })
        : [],
    ),
  }
}
