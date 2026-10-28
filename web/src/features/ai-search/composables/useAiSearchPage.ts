import { computed } from 'vue'
import { usePlan } from '../../plan/composables/usePlan'
import { usePromptResults } from '../../prompt-test/composables/usePromptTest'
import { figures, newestCells, readinessCard, voiceHeadline, voiceLede, voiceRows } from '../domain/voice'
import { useAiSearchSummary } from './useAiSearchSummary'

/** What the AI search page shows: the summary, the newest Prompt run's counts and the plan's targets, read once. */
export function useAiSearchPage() {
  const { summary, isLoading: summaryLoading, error: summaryError } = useAiSearchSummary()
  const { results, isLoading: resultsLoading, error: resultsError } = usePromptResults()
  const { plan } = usePlan()
  const cells = computed(() => newestCells(results.value))
  const rows = computed(() => (summary.value ? voiceRows(summary.value) : []))
  return {
    summary,
    cells,
    plan,
    rows,
    isLoading: computed(() => summaryLoading.value || resultsLoading.value),
    error: computed(() => summaryError.value ?? resultsError.value),
    headline: computed(() => (summary.value ? voiceHeadline(rows.value, summary.value.followed.length) : 'AI assistants, and what they say about Tribelt.')),
    lede: computed(() => voiceLede(cells.value)),
    figures: computed(() => (summary.value ? figures(summary.value, cells.value, plan.value) : [])),
    readinessLine: computed(() => (summary.value ? readinessCard(summary.value) : '')),
  }
}
