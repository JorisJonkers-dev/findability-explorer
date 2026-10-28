import { computed } from 'vue'
import { useStandings } from '../../catching-up/composables/useStandings'
import { useOverview } from '../../home/composables/useOverview'
import { usePromptResults } from '../../prompt-test/composables/usePromptTest'
import { useReadinessTable } from '../../readiness/composables/useReadiness'
import { familyBars, figuresOf, headline, type Row } from '../domain/compare'
import { readinessScore } from '../domain/compare'
import { shareOf } from '../domain/roster'
import { useCompanies } from './useCompanies'

/** How many checks the readiness score counts until the standings say; the five that the chips show. */
const FIVE_CHECKS = 5

/**
 * What the Competitors page says: the competitors on the list, each with its share of AI answers
 * and its readiness score, the figures and the belt-family chart. Everything is read from the
 * server's lists, counts and the newest Prompt run.
 */
export function useCompetitorBoard() {
  const { companies, isLoading, error } = useCompanies()
  const { results } = usePromptResults()
  const { rowOf } = useReadinessTable()
  const { readinessChecks } = useStandings()
  const { overview } = useOverview()

  const outOf = computed(() => (readinessChecks.value > 0 ? readinessChecks.value : FIVE_CHECKS))
  const active = computed(() => companies.value.filter((c) => !c.archived))
  const tribelt = computed(() => active.value.find((c) => c.role === 'tribelt'))
  /** The group's own companies, which the list leaves out. */
  const group = computed(() => active.value.filter((c) => c.role === 'group'))
  const rows = computed<Row[]>(() =>
    active.value
      .filter((c) => c.role === 'competitor')
      .map((c) => ({ id: c.id, name: c.shortName, share: shareOf(results.value, c.id), ready: readinessScore(rowOf(c.id)?.readiness) })),
  )
  // The server counts the competitors on the list, as every page shows them.
  const count = computed(() => overview.value?.competitors.makers ?? rows.value.length)
  const tribeltShare = computed(() => (tribelt.value ? shareOf(results.value, tribelt.value.id) : undefined))
  const tribeltReady = computed(() => (tribelt.value ? readinessScore(rowOf(tribelt.value.id)?.readiness) : undefined))
  return {
    rows,
    count,
    outOf,
    tribelt,
    tribeltShare,
    tribeltReady,
    group,
    title: computed(() => headline(count.value)),
    figures: computed(() => figuresOf({ count: count.value, rows: rows.value, tribeltShare: tribeltShare.value, tribeltReady: tribeltReady.value, outOf: outOf.value })),
    bars: computed(() => familyBars(overview.value?.competitors.families ?? [], tribelt.value)),
    isLoading,
    error,
  }
}
