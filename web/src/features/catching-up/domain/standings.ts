import type { StandingMeasure } from '../../../infrastructure/api/types.gen'

/** What each measure is called on the page, how its values read, and what a bar without a value says. */
export const MEASURES: Record<StandingMeasure, { label: string; short: string; unit: 'share' | 'score' | 'count'; missing?: string }> = {
  aiShareOfVoice: { label: 'AI responses naming the company, in the newest Prompt run', short: 'AI responses', unit: 'share' },
  citationShare: { label: 'Web responses citing its website, in the newest Prompt run', short: 'Citations', unit: 'share' },
  readinessChecks: { label: 'AI readiness checks passed', short: 'AI readiness', unit: 'score' },
  sitemapCoverage: { label: "Share of the site's pages its sitemap lists", short: 'Sitemap', unit: 'share', missing: 'Not measured yet. Scan the site again.' },
  termsUsed: { label: 'Search terms from the map found on the site', short: 'Search terms', unit: 'count' },
}

/** A value as the page writes it: "24%", "3½", "45". */
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

const WORDS = ['no', 'one', 'two', 'three', 'four', 'five', 'six', 'seven', 'eight', 'nine', 'ten']

/** A small count in words: "four"; larger ones as digits. */
export function inWords(n: number): string {
  return WORDS[n] ?? String(n)
}
