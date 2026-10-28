import type { AiReadiness, Competitor, OverviewFamily, PromptFamily } from '../../../infrastructure/api/types.gen'
import { familyLabel } from '../../../shared/vocabulary'
import { inWords } from '../../catching-up/domain/standings'
import { familySlug } from '../../shell/domain/sections'
import { familyCell } from './matrix'
import { chipsOf } from './roster'

/** A list of names as a sentence writes it, with the Oxford comma: "a", "a and b", "a, b, and c". */
export function oxford(names: readonly string[]): string {
  const last = names.at(-1) ?? ''
  if (names.length <= 1) return last
  if (names.length === 2) return `${String(names[0])} and ${last}`
  return `${names.slice(0, -1).join(', ')}, and ${last}`
}

/** How many readiness checks a site passes, the crawlers counting half when some are kept out; none before a scan read it. */
export function readinessScore(readiness: AiReadiness | undefined): number | undefined {
  if (!readiness) return undefined
  return chipsOf(readiness).reduce((sum, c) => sum + (c.state === 'yes' ? 1 : c.state === 'partly' ? 0.5 : 0), 0)
}

/** A score as the page writes it: "3 of 5", "3½ of 5"; "Not scanned" without one. */
export function scoreText(score: number | undefined, outOf: number): string {
  if (score === undefined) return 'Not scanned'
  const whole = Math.floor(score)
  return `${score - whole >= 0.5 ? `${String(whole)}½` : String(whole)} of ${String(outOf)}`
}

/** A share from 0 to 1 as the page writes it: "16%"; "No run yet" without one. */
export function shareText(share: number | undefined): string {
  return share === undefined ? 'No run yet' : `${String(Math.round(share * 100))}%`
}

/** What the competitors list knows of one competitor. */
export type Row = { id: string; name: string; share: number | undefined; ready: number | undefined }

/** The ways to rank the list. */
export type RankBy = 'answers' | 'ready' | 'name'

export const RANKINGS: readonly { value: RankBy; label: string }[] = [
  { value: 'answers', label: 'AI answers' },
  { value: 'ready', label: 'AI readiness' },
  { value: 'name', label: 'Name' },
]

/** The rows ranked: those with a value first, the highest first, then by name. */
export function rankRows(rows: readonly Row[], by: RankBy): Row[] {
  const value = (r: Row): number | undefined => (by === 'ready' ? r.ready : r.share)
  const byName = (a: Row, b: Row) => a.name.localeCompare(b.name)
  if (by === 'name') return [...rows].sort(byName)
  return [...rows].sort((a, b) => Number(value(b) !== undefined) - Number(value(a) !== undefined) || (value(b) ?? 0) - (value(a) ?? 0) || byName(a, b))
}

/** How a row reads aloud, as the link to the competitor's page. */
export function rowLabel(r: Row, outOf: number): string {
  return `${r.name}. AI answers that name it: ${shareText(r.share)}. AI readiness checks passed: ${scoreText(r.ready, outOf)}. Open its page.`
}

/** One of the page's key figures. */
export type Figure = { big: string; text: string; note: string }

/** The first three names, as a sentence lists them. */
function leaders(rows: readonly Row[]): string {
  return oxford(rows.slice(0, 3).map((r) => r.name))
}

/** The page's figures: how many competitors, how many are named more than Tribelt, how many share its readiness score. */
export function figuresOf(input: { count: number; rows: readonly Row[]; tribeltShare: number | undefined; tribeltReady: number | undefined; outOf: number }): Figure[] {
  const figures: Figure[] = [{ big: String(input.count), text: input.count === 1 ? 'competitor on the list' : 'competitors on the list', note: 'Archived companies are not counted.' }]
  const { tribeltShare, tribeltReady } = input
  if (tribeltShare !== undefined) {
    const ahead = rankRows(input.rows.filter((r) => (r.share ?? 0) > tribeltShare), 'answers')
    figures.push({
      big: String(ahead.length),
      text: ahead.length === 1 ? 'is named in more AI answers than Tribelt' : 'are named in more AI answers than Tribelt',
      note: ahead.length === 0 ? 'Tribelt is named in as many AI answers as any competitor, or more.' : `${leaders(ahead)} ${Math.min(ahead.length, 3) === 1 ? 'leads' : 'lead'}.`,
    })
  }
  if (tribeltReady !== undefined) {
    const level = rankRows(input.rows.filter((r) => r.ready === tribeltReady), 'name')
    figures.push({
      big: String(level.length),
      text: `${level.length === 1 ? 'shares' : 'share'} Tribelt’s AI readiness score of ${scoreText(tribeltReady, input.outOf)}`,
      note: level.length === 0 ? 'No other competitor passes as many checks.' : `Among them ${leaders(level)}.`,
    })
  }
  return figures
}

/** The page's headline, from the competitors the server counts. */
export function headline(count: number): string {
  if (count === 0) return 'The competitors that sell metal belts Tribelt’s buyers could buy instead.'
  return `${String(count)} ${count === 1 ? 'competitor sells' : 'competitors sell'} metal belts that Tribelt’s buyers could buy instead.`
}

/** One bar of the chart: how many competitors sell a belt family, and what Tribelt calls it. */
export type FamilyBar = { family: PromptFamily; label: string; makers: number; tribelt: string | null; path: string }

/** The chart's bars, in the families' order, with Tribelt's own name for each family. */
export function familyBars(families: readonly OverviewFamily[], tribelt: Pick<Competitor, 'beltNames'> | undefined): FamilyBar[] {
  return families.map((f) => ({
    family: f.family,
    label: familyLabel(f.family),
    makers: f.makers,
    tribelt: tribelt ? familyCell(tribelt.beltNames, f.family).first : null,
    path: `/terms/families/${familySlug(f.family)}`,
  }))
}

/** What the picked family says: how Tribelt sells it. */
export function familyLine(bar: FamilyBar): string {
  const sold = bar.tribelt === null ? 'Tribelt does not sell this family.' : `Tribelt sells it as ${bar.tribelt}.`
  return `${String(bar.makers)} of the competitors sell it. ${sold} How buyers search for it is on its page.`
}

/** The chart's text alternative. */
export function familyChartLabel(bars: readonly FamilyBar[]): string {
  return `Competitors that sell each belt family. ${bars.map((b) => `${b.label}: ${String(b.makers)}`).join('. ')}.`
}

/** How a competitor's share of AI answers reads against Tribelt's, as the detail page's headline. */
export function detailHeadline(name: string, share: number | undefined, tribeltShare: number | undefined): string {
  if (share === undefined) return `${name}, as its newest scan finds it.`
  if (tribeltShare === undefined) return `${name} is named in ${shareText(share)} of AI answers.`
  if (tribeltShare === 0) return share === 0 ? `No AI answer names ${name} or Tribelt.` : `${name} is named in ${shareText(share)} of AI answers, and no answer names Tribelt.`
  const ratio = share / tribeltShare
  const times = Math.round(ratio)
  if (ratio >= 1.5) {
    const word = times === 2 ? 'twice' : `${inWords(times)} times`
    return `${name} is named in ${Math.abs(ratio - times) > 0.1 ? 'about ' : ''}${word} as many AI answers as Tribelt.`
  }
  if (ratio > 1.15) return `${name} is named in more AI answers than Tribelt.`
  if (ratio >= 0.87) return `${name} is named in about as many AI answers as Tribelt.`
  return `${name} is named in fewer AI answers than Tribelt.`
}

/** One measure the detail page sets the competitor against Tribelt on. */
export type Measure = {
  id: 'answers' | 'ready' | 'sitemap' | 'terms'
  label: string
  /** The competitor's value, undefined when not known. */
  them: number | undefined
  us: number | undefined
  themText: string
  usText: string
  /** The value a full bar stands for. */
  scale: number
  text: string
  /** The page that says more about the measure. */
  path: string
  pathLabel: string
}

type Known = { name: string; share: number | undefined; tribeltShare: number | undefined; ready: number | undefined; tribeltReady: number | undefined; outOf: number }
type Coverage = { sitemap: number | undefined; tribeltSitemap: number | undefined }
type Terms = { terms: number | undefined; tribeltTerms: number | undefined; mapTerms: number }

const NOT_KNOWN = 'Not known'

function scaled(them: number | undefined, us: number | undefined, floor: number): number {
  return Math.max(them ?? 0, us ?? 0, floor)
}

/** The four measures the competitor is set against Tribelt on, each with a sentence saying what it means. */
export function measuresOf(input: Known & Coverage & Terms): Measure[] {
  const { name } = input
  const count = (n: number | undefined) => (n === undefined ? NOT_KNOWN : n.toLocaleString('en-US'))
  const pct = (n: number | undefined) => (n === undefined ? NOT_KNOWN : shareText(n))
  const score = (n: number | undefined) => (n === undefined ? NOT_KNOWN : scoreText(n, input.outOf))
  return [
    {
      id: 'answers',
      label: 'AI answers that name it',
      them: input.share,
      us: input.tribeltShare,
      themText: pct(input.share),
      usText: pct(input.tribeltShare),
      scale: scaled(input.share, input.tribeltShare, 0.01),
      text:
        input.share === undefined || input.tribeltShare === undefined
          ? `No Prompt run has counted the answers that name ${name} or Tribelt yet.`
          : `${name} is named in ${shareText(input.share)} of the answers in the newest Prompt run, and Tribelt in ${shareText(input.tribeltShare)}.`,
      path: '/ai-search/answers',
      pathLabel: 'Open every answer',
    },
    {
      id: 'ready',
      label: 'AI readiness checks passed',
      them: input.ready,
      us: input.tribeltReady,
      themText: score(input.ready),
      usText: score(input.tribeltReady),
      scale: input.outOf,
      text:
        input.ready === undefined
          ? `${name}’s site has not been scanned yet, so its AI readiness is not known.`
          : input.ready === input.tribeltReady
            ? `Both sites pass ${scoreText(input.ready, input.outOf)} of the AI readiness checks.`
            : `${name}’s site passes ${scoreText(input.ready, input.outOf)} of the AI readiness checks, and tribelt.nl ${input.tribeltReady === undefined ? 'has not been scanned yet' : `passes ${scoreText(input.tribeltReady, input.outOf)}`}.`,
      path: '/ai-search/readiness',
      pathLabel: 'Open the AI readiness table',
    },
    {
      id: 'sitemap',
      label: 'Pages its sitemap lists',
      them: input.sitemap,
      us: input.tribeltSitemap,
      themText: pct(input.sitemap),
      usText: pct(input.tribeltSitemap),
      scale: 1,
      text:
        input.sitemap === undefined
          ? `How many of ${name}’s pages its sitemap lists is not known. Scan its site again to measure it.`
          : `${name}’s sitemap lists ${shareText(input.sitemap)} of the pages its newest scan read, against ${pct(input.tribeltSitemap)} on tribelt.nl.`,
      path: '/ai-search/readiness',
      pathLabel: 'Open the AI readiness table',
    },
    {
      id: 'terms',
      label: 'Search terms from the map on its site',
      them: input.terms,
      us: input.tribeltTerms,
      themText: count(input.terms),
      usText: count(input.tribeltTerms),
      scale: scaled(input.terms, input.tribeltTerms, 1),
      text: `tribelt.nl uses ${count(input.tribeltTerms)} of the ${count(input.mapTerms)} terms in the map. ${name}’s site uses ${count(input.terms)}.`,
      path: '/terms',
      pathLabel: 'Open the search terms',
    },
  ]
}

/** The length of a measure's bar, from 0 to 1; none when the value is not known. */
export function barLength(value: number | undefined, scale: number): number | undefined {
  return value === undefined || scale <= 0 ? undefined : Math.min(1, value / scale)
}

/** The detail page's three figures: its AI answers, its readiness checks and its search terms, each with Tribelt's value beside. */
export function detailFigures(measures: readonly Measure[]): Figure[] {
  const texts: Partial<Record<Measure['id'], string>> = {
    answers: 'of AI answers name it',
    ready: 'AI readiness checks passed',
    terms: 'search terms from the map on its site',
  }
  return measures.flatMap((m) => {
    const text = texts[m.id]
    return text === undefined ? [] : [{ big: m.themText, text, note: `Tribelt: ${m.usText}.` }]
  })
}
