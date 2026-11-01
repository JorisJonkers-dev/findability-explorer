import { computed } from 'vue'
import { useCompetitors } from '../../competitors/composables/useCompetitors'
import { usePlan } from '../../plan/composables/usePlan'
import { usePromptAnswers, usePromptResults } from '../../prompt-test/composables/usePromptTest'
import { usePromptSets } from '../../prompt-test/composables/useResponses'
import { latestRun, tally } from '../../prompt-test/domain/results'
import { questionsOf, runDataLine } from '../../prompt-test/domain/responses'
import { byModel, familyScopeRows, figuresOf, whoNamed } from '../domain/sections'

/** AI responses: the newest run with responses, its responses and questions, the goals, and the companies its responses name. */
export function useAiResponsesPage() {
  const { results, isLoading, error } = usePromptResults()
  const run = computed(() => latestRun(results.value))
  const { answers, isLoading: answersLoading, error: answersError } = usePromptAnswers(computed(() => run.value?.id), computed(() => ({})))
  const { sets } = usePromptSets()
  const { plan } = usePlan()
  const { competitors } = useCompetitors()
  const list = computed(() => answers.value?.items ?? [])
  const asked = computed(() => sets.value?.versions.find((v) => v.version === run.value?.promptSetVersion)?.questions)
  return {
    run,
    isLoading,
    error,
    answersLoading,
    answersError,
    dataLine: computed(() => (run.value ? runDataLine(run.value) : undefined)),
    figures: computed(() => figuresOf(run.value?.cells ?? [], plan.value)),
    read: computed(() => tally(run.value?.cells ?? []).answers),
    named: computed(() => whoNamed(results.value?.mentions ?? [], competitors.value)),
    models: computed(() => (run.value ? byModel(run.value) : [])),
    questions: computed(() => questionsOf(asked.value, list.value)),
    families: computed(() => familyScopeRows(list.value)),
  }
}
