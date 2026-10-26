import type { BeltFamily, IntentTier, Term } from '../../../infrastructure/api/types.gen'
import type { SortValue } from '../../../shared/sort'
import { familyLabel, languageLabel, tierLabel } from '../../../shared/vocabulary'
import { chartLanguage, companyName, type TermGroup, type TermView } from './filters'
import { fold } from './fold'

/** The name the map credits Tribelt's own website under. */
export const TRIBELT = 'Tribelt'

/** The kinds of search in the charts' order: marketing first, broad to narrow, then technical. */
export const CHART_TIERS = ['broad', 'product', 'application', 'spec', 'problem'] as const satisfies readonly IntentTier[]

const TECHNICAL: ReadonlySet<IntentTier> = new Set(['spec', 'problem'])

/** Whether a kind of search is technical (specifications, problems) or marketing (the rest). */
export function groupOf(tier: IntentTier): TermGroup {
  return TECHNICAL.has(tier) ? 'technical' : 'marketing'
}

export const GROUP_LABELS: Record<TermGroup, string> = { marketing: 'Marketing', technical: 'Technical' }

/** Whether Tribelt's own website uses the term. */
export function tribeltUses(term: Pick<Term, 'usedBy'>): boolean {
  return term.usedBy.some((u) => companyName(u) === TRIBELT)
}

/** The other companies that use the term, each once, in the map's order. */
export function makersOf(term: Pick<Term, 'usedBy'>): string[] {
  return [...new Set(term.usedBy.map(companyName))].filter((c) => c !== TRIBELT)
}

/** One bar of a chart: how many terms, how many of them Tribelt uses, as shares of the longest bar. */
export type ChartRow = {
  value: string
  label: string
  count: number
  tribelt: number
  /** Tribelt's part of the bar, from 0 to 1 of the longest bar. */
  tribeltShare: number
  /** The others' part of the bar. */
  otherShare: number
  /** The bar in words, for its button. */
  description: string
}

function chartRows(groups: { value: string; label: string; terms: Term[] }[]): ChartRow[] {
  const longest = Math.max(1, ...groups.map((g) => g.terms.length))
  return groups.map(({ value, label, terms }) => {
    const tribelt = terms.filter(tribeltUses).length
    return {
      value,
      label,
      count: terms.length,
      tribelt,
      tribeltShare: tribelt / longest,
      otherShare: (terms.length - tribelt) / longest,
      description: `${label}: ${String(terms.length)} ${terms.length === 1 ? 'term' : 'terms'}, Tribelt uses ${String(tribelt)}`,
    }
  })
}

/** The terms per language, the most first; American English counts as English. */
export function languageRows(terms: readonly Term[]): ChartRow[] {
  const byLanguage = new Map<string, Term[]>()
  for (const term of terms) {
    const code = chartLanguage(term.language)
    byLanguage.set(code, [...(byLanguage.get(code) ?? []), term])
  }
  const groups = [...byLanguage.entries()].map(([value, list]) => ({ value, label: languageLabel(value), terms: list }))
  groups.sort((a, b) => b.terms.length - a.terms.length || a.label.localeCompare(b.label))
  return chartRows(groups)
}

/** The terms per kind of search, in the charts' order. */
export function tierRows(terms: readonly Term[]): ChartRow[] {
  return chartRows(CHART_TIERS.map((tier) => ({ value: tier, label: tierLabel(tier), terms: terms.filter((t) => t.tier === tier) })))
}

/** One cell of the heatmap: a company's terms of one kind. */
export type HeatCell = { tier: IntentTier; count: number; technical: boolean; intensity: number }

/** One company's row of the heatmap. */
export type HeatRow = {
  company: string
  cells: HeatCell[]
  marketing: number
  technical: number
  total: number
  description: string
}

/** The terms a company uses: Tribelt's own, or those crediting the company. */
export function termsOf(terms: readonly Term[], company: string): Term[] {
  return company === TRIBELT ? terms.filter(tribeltUses) : terms.filter((t) => makersOf(t).includes(company))
}

/**
 * Every company the map credits, Tribelt included, with its terms per kind of search, the most
 * terms first. A cell's intensity is its count against the fullest cell of the whole map, from
 * a quarter (one term) up to one, or 0 for none.
 */
export function heatmap(terms: readonly Term[]): HeatRow[] {
  const companies = [...new Set(terms.flatMap(makersOf)), TRIBELT]
  const counts = companies.map((company) => {
    const own = termsOf(terms, company)
    return { company, perTier: CHART_TIERS.map((tier) => ({ tier, count: own.filter((t) => t.tier === tier).length })) }
  })
  const fullest = Math.max(1, ...counts.flatMap((c) => c.perTier.map((p) => p.count)))
  return counts
    .map(({ company, perTier }) => {
      const cells = perTier.map(({ tier, count }) => ({
        tier,
        count,
        technical: groupOf(tier) === 'technical',
        intensity: count > 0 ? 0.25 + (0.75 * count) / fullest : 0,
      }))
      const marketing = cells.filter((c) => !c.technical).reduce((sum, c) => sum + c.count, 0)
      const technical = cells.filter((c) => c.technical).reduce((sum, c) => sum + c.count, 0)
      return {
        company,
        cells,
        marketing,
        technical,
        total: marketing + technical,
        description: `${company}: ${String(marketing)} marketing terms, ${String(technical)} technical terms`,
      }
    })
    .sort((a, b) => b.total - a.total || a.company.localeCompare(b.company))
}

/** A heatmap cell's background: blue-grey for marketing, orange for technical, as strong as its count. */
export function cellBackground(cell: Pick<HeatCell, 'intensity' | 'technical'>): string | undefined {
  if (cell.intensity === 0) return undefined
  const rgb = cell.technical ? '245, 117, 3' : '141, 170, 191'
  return `rgba(${rgb}, ${cell.intensity.toFixed(2)})`
}

/** The terms the view lets through, in the map's order. */
export function filterTerms(terms: readonly Term[], view: TermView): Term[] {
  const needle = fold(view.q)
  return terms.filter(
    (t) =>
      (view.group === null || groupOf(t.tier) === view.group) &&
      (view.tier === null || t.tier === view.tier) &&
      (view.language === null || chartLanguage(t.language) === view.language) &&
      (view.user === '' || termsOf([t], view.user).length > 0) &&
      (!view.gap || !tribeltUses(t)) &&
      (needle === '' || fold(t.term).includes(needle)),
  )
}

function familyText(family: BeltFamily): string {
  return family === 'spec_all' ? 'Any belt' : familyLabel(family)
}

/** One line of the ranked list. */
export type RankedTerm = {
  rank: number
  term: string
  /** "English · Tri-Flex · Twentebelt, Ashworth". */
  meta: string
  tier: IntentTier
  kind: string
  makers: number
  /** The makers' bar, from 0 to 1 of the most-used term in the whole map. */
  share: number
  tribelt: boolean
  /** What Tribelt's own site calls the same thing: "Uses it", its term, or "Lacks it". */
  tribeltTerm: string
}

/**
 * The terms ranked by how many makers use them, the most first; a tie goes to the more technical
 * kind of search, then the alphabet. The bars are against the most-used term of the whole map,
 * so they keep their length as filters narrow the list.
 */
export function rankTerms(shown: readonly (Term & { scanMakers?: readonly string[] })[], all: readonly Term[]): RankedTerm[] {
  const top = Math.max(1, ...all.map((t) => makersOf(t).length))
  return [...shown]
    .map((t) => ({ t, makers: makersOf(t) }))
    .sort(
      (a, b) =>
        b.makers.length - a.makers.length ||
        CHART_TIERS.indexOf(b.t.tier) - CHART_TIERS.indexOf(a.t.tier) ||
        a.t.term.localeCompare(b.t.term),
    )
    .map(({ t, makers }, index) => ({
      rank: index + 1,
      term: t.term,
      meta: [
        languageLabel(chartLanguage(t.language)),
        familyText(t.family),
        makers.map((m) => (t.scanMakers?.includes(m) === true ? `${m} (scan)` : m)).join(', '),
      ]
        .filter((p) => p !== '')
        .join(' · '),
      tier: t.tier,
      kind: tierLabel(t.tier),
      makers: makers.length,
      share: makers.length / top,
      tribelt: tribeltUses(t),
      tribeltTerm: tribeltTerm(t, all),
    }))
}

/** The kinds of search a term can stand in for another in: they name a belt or its use. */
const NAMING_TIERS: readonly string[] = ['broad', 'product', 'application']

/**
 * What Tribelt's own site calls the same thing: "Uses it" for a term Tribelt uses, else
 * Tribelt's term for the same belt family that names a belt or its use too (the same kind of
 * search first), in the term's language if Tribelt has one there, else naming its language; or
 * "Lacks it". A specification or a problem is its own thing: another one is no stand-in for it.
 */
export function tribeltTerm(term: Term, all: readonly Term[]): string {
  if (tribeltUses(term)) return 'Uses it'
  if (!NAMING_TIERS.includes(term.tier)) return 'Lacks it'
  const own = all.filter((t) => t !== term && tribeltUses(t) && t.family === term.family && NAMING_TIERS.includes(t.tier))
  const sameTier = own.filter((t) => t.tier === term.tier)
  const pool = sameTier.length > 0 ? sameTier : own
  const language = chartLanguage(term.language)
  const match = pool.find((t) => chartLanguage(t.language) === language) ?? pool[0]
  if (match === undefined) return 'Lacks it'
  const other = chartLanguage(match.language) === language ? '' : ` (${languageLabel(chartLanguage(match.language))})`
  return `${match.term}${other}`
}

/** A filter in use that a chip can remove. */
export type Chip = { key: 'language' | 'tier' | 'user'; label: string }

/** The chart and heatmap filters in use, as removable chips. */
export function chips(view: TermView): Chip[] {
  const out: Chip[] = []
  if (view.language !== null) out.push({ key: 'language', label: languageLabel(view.language) })
  if (view.tier !== null) out.push({ key: 'tier', label: tierLabel(view.tier) })
  if (view.user !== '') out.push({ key: 'user', label: view.user })
  return out
}

/** How many terms are shown of the map: "35 of 146 terms". */
export function summary(shown: number, total: number): string {
  return `${String(shown)} of ${String(total)} ${total === 1 ? 'term' : 'terms'}`
}

/** A column of the per-company table to sort by: the company, a kind of search, or the total. */
export type HeatColumn = 'company' | (typeof CHART_TIERS)[number] | 'total'

/** What a row of the per-company table sorts by in a column. */
export function heatSortValue(row: HeatRow, column: HeatColumn): SortValue {
  if (column === 'company') return row.company
  if (column === 'total') return row.total
  return row.cells.find((c) => c.tier === column)?.count ?? 0
}

/** A column of the ranking to sort by. */
export type RankColumn = 'rank' | 'term' | 'kind' | 'makers' | 'tribelt'

/** What a ranked term sorts by in a column; Tribelt's own terms first when descending. */
export function rankSortValue(row: RankedTerm, column: RankColumn): SortValue {
  return row[column]
}
