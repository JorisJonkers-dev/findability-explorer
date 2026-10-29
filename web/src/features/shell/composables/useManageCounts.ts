import { computed, ref } from 'vue'
import { useCompetitorCandidates, usePromptResults } from '../../prompt-test/composables/usePromptTest'
import { useTermCandidates } from '../../review/composables/useTermReview'
import { useRecentScans } from '../../scan/composables/useRecentScans'
import { isInProgress } from '../../scan/domain/status'

/**
 * The Manage menu's counts, by the address of the section each belongs to: Prompt runs and scans
 * running, competitor candidates and search terms waiting. Only an administrator may ask for them.
 */
export function useManageCounts() {
  const { results } = usePromptResults()
  const { scans } = useRecentScans()
  const { counts: candidates } = useCompetitorCandidates(ref('pending'))
  const { counts: terms } = useTermCandidates(ref('pending'))
  return {
    counts: computed<Record<string, number>>(() => ({
      '/manage/prompt-test': (results.value?.runs ?? []).filter((r) => r.status === 'running').length,
      '/manage/scans': scans.value.filter((s) => isInProgress(s.status)).length,
      '/manage/competitors': candidates.value?.pending ?? 0,
      '/manage/terms': terms.value?.pending ?? 0,
    })),
  }
}
