import type { Standing, StandingMeasure, TermsSource } from '../../../infrastructure/api/types.gen'

/** What each measure is called on the page, and how its values read. */
export const MEASURES: Record<StandingMeasure, { label: string; short: string; unit: 'share' | 'score' | 'count' }> = {
  aiShareOfVoice: { label: 'AI answers naming the company, in the newest Prompt run', short: 'AI answers', unit: 'share' },
  citationShare: { label: 'Web answers citing its website, in the newest Prompt run', short: 'Citations', unit: 'share' },
  readinessChecks: { label: 'AI readiness checks passed, out of five', short: 'AI readiness', unit: 'score' },
  sitemapPages: { label: 'Pages in the sitemap', short: 'Sitemap', unit: 'count' },
  termsUsed: { label: 'Search terms from the map found on the site', short: 'Search terms', unit: 'count' },
}

/** A note under the search terms when tribelt.nl has no scan yet and the desk research counts Tribelt's terms; empty otherwise. */
export function termsSourceLine(from: TermsSource | undefined): string {
  return from === 'research' ? "tribelt.nl has not been scanned yet. Tribelt's bar counts the terms our research found for it." : ''
}

/** A value as the page writes it: "24%", "3½", "2,140". */
export function formatValue(measure: StandingMeasure, value: number): string {
  switch (MEASURES[measure].unit) {
    case 'share':
      return `${String(Math.round(value * 100))}%`
    case 'score': {
      const whole = Math.floor(value)
      return value - whole >= 0.5 ? `${String(whole)}½` : String(whole)
    }
    case 'count':
      return Math.round(value).toLocaleString('en-US')
  }
}

/** One labelled bar: who, the value written out, its length from 0 to 1, and whose colour it has. */
export type Bar = { who: string; value: string; length: number; tone: 'tribelt' | 'typical' | 'leader' }

/** A measure's three bars: Tribelt, the typical competitor and the leader by name, against the largest. */
export function barsOf(s: Standing): Bar[] {
  const max = Math.max(s.tribelt ?? 0, s.median ?? 0, s.leader?.value ?? 0)
  const bar = (who: string, value: number | undefined, tone: Bar['tone']): Bar => ({
    who,
    value: value === undefined ? 'No data yet' : formatValue(s.measure, value),
    length: value === undefined || max === 0 ? 0 : value / max,
    tone,
  })
  return [bar('Tribelt', s.tribelt, 'tribelt'), bar('Typical competitor', s.median, 'typical'), bar(s.leader?.name ?? 'The leader', s.leader?.value, 'leader')]
}

/** A short line under a measure: where Tribelt leads, or how far behind it is. */
export function noteOf(s: Standing): string {
  if (s.leads) return 'Tribelt leads here.'
  if (s.tribelt === undefined) return 'Tribelt has no data here yet.'
  if (!s.leader) return 'No competitor has data here yet.'
  return `${s.leader.name} is ahead.`
}

const WORDS = ['no', 'one', 'two', 'three', 'four', 'five', 'six', 'seven', 'eight', 'nine', 'ten']

/** A small count in words: "four"; larger ones as digits. */
export function inWords(n: number): string {
  return WORDS[n] ?? String(n)
}

/** The page's headline, from the count of measures Tribelt trails on. */
export function headline(measures: readonly Standing[], trailing: number): string {
  const measured = measures.filter((m) => m.tribelt !== undefined && m.leader !== undefined).length
  if (measured === 0) return 'Not measured yet. The first scans and Prompt run fill this page.'
  if (trailing === 0) return 'Tribelt leads the field on every measure.'
  return `Tribelt trails the leaders on ${inWords(trailing)} of ${inWords(measured)} measures.`
}
