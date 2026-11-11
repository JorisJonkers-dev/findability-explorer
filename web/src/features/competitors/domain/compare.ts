import type { AiReadiness } from '../../../infrastructure/api/types.gen'
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
      label: 'AI responses that name it',
      them: input.share,
      us: input.tribeltShare,
      themText: pct(input.share),
      usText: pct(input.tribeltShare),
      scale: scaled(input.share, input.tribeltShare, 0.01),
      text:
        input.share === undefined || input.tribeltShare === undefined
          ? `No Prompt run has counted the responses that name ${name} or Tribelt yet.`
          : `${name} is named in ${shareText(input.share)} of the responses in the newest Prompt run, and Tribelt in ${shareText(input.tribeltShare)}.`,
      path: '/ai-responses/all',
      pathLabel: 'Open every response',
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
      path: '/ai-readiness',
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
      path: '/ai-readiness',
      pathLabel: 'Open the AI readiness table',
    },
    {
      id: 'terms',
      label: 'Search terms from the list on its site',
      them: input.terms,
      us: input.tribeltTerms,
      themText: count(input.terms),
      usText: count(input.tribeltTerms),
      scale: scaled(input.terms, input.tribeltTerms, 1),
      text: `tribelt.nl uses ${count(input.tribeltTerms)} of the ${count(input.mapTerms)} terms on the search term list. ${name}’s site uses ${count(input.terms)}.`,
      path: '/terms',
      pathLabel: 'Open the search terms',
    },
  ]
}

/** The length of a measure's bar, from 0 to 1; none when the value is not known. */
export function barLength(value: number | undefined, scale: number): number | undefined {
  return value === undefined || scale <= 0 ? undefined : Math.min(1, value / scale)
}
