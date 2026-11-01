import { computed } from 'vue'
import { useStandings } from '../../catching-up/composables/useStandings'
import { useOverview } from '../../home/composables/useOverview'
import { usePromptResults } from '../../prompt-test/composables/usePromptTest'
import { useReadinessTable } from '../../readiness/composables/useReadiness'
import { useTermMap } from '../../terminology/composables/useTermSearch'
import { readinessScore } from '../domain/compare'
import { CHIP_COUNT, shareOf } from '../domain/roster'
import { competitorRows } from '../domain/table'
import { useCompanies } from './useCompanies'

/**
 * What the Competitors page says: Tribelt and every competitor, archived ones too, each with its
 * share of AI responses, its readiness score, and the search terms it uses; how many competitors
 * the server counts; and the group's own companies, which the list leaves out.
 */
export function useCompetitorBoard() {
  const { companies, isLoading, error } = useCompanies()
  const { results } = usePromptResults()
  const { rowOf } = useReadinessTable()
  const { readinessChecks } = useStandings()
  const { overview } = useOverview()
  const { terms } = useTermMap()

  const outOf = computed(() => (readinessChecks.value > 0 ? readinessChecks.value : CHIP_COUNT))
  const active = computed(() => companies.value.filter((c) => !c.archived))
  const tribelt = computed(() => active.value.find((c) => c.role === 'tribelt'))
  /** The group's own companies, which the list leaves out. */
  const group = computed(() => active.value.filter((c) => c.role === 'group'))
  const termsOf = (id: string) =>
    id === tribelt.value?.id ? terms.value.filter((t) => t.tribeltUses === true).length : terms.value.filter((t) => t.makers?.some((m) => m.competitorId === id) === true).length
  const tableRows = computed(() =>
    competitorRows(companies.value, { share: (id) => shareOf(results.value, id), ready: (id) => readinessScore(rowOf(id)?.readiness), terms: termsOf }),
  )
  // The server counts the competitors on the list, as every page shows them.
  const count = computed(() => overview.value?.competitors.makers ?? active.value.filter((c) => c.role === 'competitor').length)
  const newestScan = computed(() =>
    tableRows.value
      .map((r) => r.scannedAt)
      .filter((at) => at !== undefined)
      .sort()
      .at(-1),
  )
  return { tableRows, newestScan, count, outOf, tribelt, group, isLoading, error }
}
