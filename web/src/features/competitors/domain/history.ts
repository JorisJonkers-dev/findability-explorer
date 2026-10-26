import type { SortValue } from '../../../shared/sort'
import type {
  AnalysisSummary,
  ComparisonGroup,
  ProfileGroup,
  ProfileGroupTerms,
  ProfileTerm,
  TermChange,
} from '../../../infrastructure/api/types.gen'
import { languageLabel, tierLabel } from '../../../shared/vocabulary'
import { isInProgress } from '../../scan/domain/status'

const GROUP_LABELS: Record<ProfileGroup, string> = {
  eyelink: 'Eyelink',
  open_spiral: 'Open spiral',
  closed_spiral: 'Closed spiral',
  tri_flex_turn_curve: 'Tri-Flex',
  other: 'Other belts',
  broad: 'Broad',
  spec: 'Specification',
}

/** The profile's groups in their fixed order: Tribelt's families, other belts, broad, specification. */
export const PROFILE_GROUPS = Object.keys(GROUP_LABELS) as ProfileGroup[]

/** A group's name: "Tri-Flex", "Broad". */
export function groupLabel(group: ProfileGroup): string {
  return GROUP_LABELS[group]
}

/** A tab's label: the group with how many terms it holds, "Eyelink (12)". */
export function groupTabLabel(group: ProfileGroupTerms): string {
  return `${groupLabel(group.group)} (${String(group.terms.length)})`
}

/** The group a profile opens on: the first with terms, else the first. */
export function firstGroupWithTerms(groups: readonly ProfileGroupTerms[]): ProfileGroup | undefined {
  return (groups.find((g) => g.terms.length > 0) ?? groups[0])?.group
}

export type LanguageTerms = { language: string; label: string; terms: ProfileTerm[] }

/** A group's terms by language, the language with most terms first; each keeps the score order. */
export function byLanguage(terms: readonly ProfileTerm[]): LanguageTerms[] {
  const groups = new Map<string, ProfileTerm[]>()
  for (const t of terms) groups.set(t.language, [...(groups.get(t.language) ?? []), t])
  return [...groups.entries()]
    .sort(([a, m], [b, n]) => n.length - m.length || a.localeCompare(b))
    .map(([language, list]) => ({ language, label: languageLabel(language), terms: list }))
}

/** When the term was first found: "new in this scan", or "since scan 2". */
export function sinceLabel(firstScan: number, latest: number | undefined): string {
  return firstScan === latest ? 'new in this scan' : `since scan ${String(firstScan)}`
}

/** What the map makes of a term: "in the map", or "not in the map, closest: eyelink belt". */
export function mapLabel(t: Pick<ProfileTerm, 'inMap' | 'closestTerm'>): string {
  if (t.inMap) return 'in the map'
  return t.closestTerm ? `not in the map, closest: ${t.closestTerm}` : 'not in the map'
}

/** A scan's name: "Scan 3", or "Scan" for one without a number. */
export function scanName(scan: Pick<AnalysisSummary, 'number'>): string {
  return scan.number === undefined ? 'Scan' : `Scan ${String(scan.number)}`
}

const DAY = new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'short', year: 'numeric', timeZone: 'UTC' })

/** When a scan was started: "22 Oct 2026". */
export function scanDate(scan: Pick<AnalysisSummary, 'createdAt'>): string {
  return DAY.format(new Date(scan.createdAt))
}

/** What a scan read: "48 of 60 pages", or how far it got while it runs. */
export function scanPages(scan: Pick<AnalysisSummary, 'status' | 'pagesCrawled' | 'maxPages'>): string {
  const pages = `${String(scan.pagesCrawled)} of ${String(scan.maxPages)} pages`
  return isInProgress(scan.status) ? `${pages} so far` : pages
}

/** How a scan ended, as the timeline shows it. */
export function scanState(scan: Pick<AnalysisSummary, 'status'>): string {
  switch (scan.status) {
    case 'queued':
      return 'Queued'
    case 'running':
      return 'Scanning'
    case 'failed':
      return 'Failed'
    case 'done':
      return 'Finished'
  }
}

/** The model that matched the scan's products, or that none did. */
export function scanModel(scan: Pick<AnalysisSummary, 'matchingModel'>): string {
  return scan.matchingModel ?? 'no model'
}

/** The scans that finished, newest first, which the compare view may pick between. */
export function finishedScans<T extends Pick<AnalysisSummary, 'status'>>(scans: readonly T[]): T[] {
  return scans.filter((s) => s.status === 'done')
}

/** The pair compared at first: the second newest finished scan against the newest. */
export function defaultPair(scans: readonly Pick<AnalysisSummary, 'id' | 'status'>[]): { from: string; to: string } | null {
  const [to, from] = finishedScans(scans)
  return to && from ? { from: from.id, to: to.id } : null
}

export type ChangeKind = 'added' | 'removed' | 'moved'

const KIND_LABELS: Record<ChangeKind, string> = { added: 'Added', removed: 'Removed', moved: 'Moved' }

/** A kind of change's name: "Added". */
export function kindLabel(kind: ChangeKind): string {
  return KIND_LABELS[kind]
}

function pageCount(n: number | undefined): string {
  return n === 1 ? '1 page' : `${String(n ?? 0)} pages`
}

function score(value: number | undefined): string {
  return value === undefined ? '–' : String(Math.round(value * 10) / 10)
}

/** What changed about a term, in words: "score 30.5, 5 pages", "was 12, 2 pages", "score 12 → 30.5, pages 2 → 5". */
export function changeDetail(change: TermChange, kind: ChangeKind): string {
  switch (kind) {
    case 'added':
      return `score ${score(change.toScore)}, ${pageCount(change.toPages)}`
    case 'removed':
      return `was ${score(change.fromScore)}, ${pageCount(change.fromPages)}`
    case 'moved':
      return `score ${score(change.fromScore)} → ${score(change.toScore)}, pages ${String(change.fromPages ?? 0)} → ${String(change.toPages ?? 0)}`
  }
}

/** How many terms a comparison added, removed and moved over every group. */
export function changeCounts(groups: readonly ComparisonGroup[]): Record<ChangeKind, number> {
  const counts = { added: 0, removed: 0, moved: 0 }
  for (const g of groups) {
    counts.added += g.added.length
    counts.removed += g.removed.length
    counts.moved += g.moved.length
  }
  return counts
}

/** The groups with any change, which the compare view lists. */
export function changedGroups(groups: readonly ComparisonGroup[]): ComparisonGroup[] {
  return groups.filter((g) => g.added.length + g.removed.length + g.moved.length > 0)
}

/** A column of a company's term tables to sort by. */
export type ProfileColumn = 'phrase' | 'score' | 'pages' | 'tier' | 'map' | 'found'

/** What a term sorts by in a column; in the map first, and the newest first, when descending. */
export function profileSortValue(t: Pick<ProfileTerm, 'phrase' | 'score' | 'pages' | 'tier' | 'inMap' | 'firstScan'>, column: ProfileColumn): SortValue {
  switch (column) {
    case 'phrase':
      return t.phrase
    case 'score':
      return t.score
    case 'pages':
      return t.pages
    case 'tier':
      return tierLabel(t.tier)
    case 'map':
      return t.inMap
    case 'found':
      return t.firstScan
  }
}
