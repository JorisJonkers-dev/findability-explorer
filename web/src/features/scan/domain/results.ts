import type {
  AnalysisSummary,
  CrawledPage,
  Equivalent,
  IntentTier,
  Keyword,
  KnownTerm,
} from '../../../infrastructure/api/types.gen'
import { languageLabel, TIERS, tierLabel, tribeltProduct } from '../../../shared/vocabulary'

/** A site's host without www.: "twentebelt.com". An address that does not parse shows as itself. */
export function siteHost(url: string): string {
  try {
    return new URL(url).hostname.replace(/^www\./, '')
  } catch {
    return url
  }
}

/** A company the scanned site may belong to: its name for people, and its website. */
export type KnownSite = { name: string; website: string }

/**
 * Who the scanned site is, for sentences such as "Twentebelt uses these": the company from the
 * terminology map whose website is on the same host, or else the host itself.
 */
export function siteName(url: string, sites: readonly KnownSite[]): string {
  const host = siteHost(url)
  // A website may carry a note after the address: "https://www.cambridge-es.com (blocked …)".
  const site = sites.find((s) => siteHost(s.website.split(' ', 1).join('')) === host)
  return site ? site.name : host
}

const DAY = new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'long', year: 'numeric' })
const TIME = new Intl.DateTimeFormat('en-GB', { hour: '2-digit', minute: '2-digit' })

/** When the scan was made: "22 October 2026 at 14:35". */
export function scannedAt(iso: string): string {
  const date = new Date(iso)
  return `${DAY.format(date)} at ${TIME.format(date)}`
}

/** A list for a sentence: "English", "English and German", "English, German and Dutch". */
export function joinWords(words: readonly string[]): string {
  if (words.length <= 1) return words.join('')
  return `${words.slice(0, -1).join(', ')} and ${String(words.at(-1))}`
}

/** The pages' languages, the most used first: ["English", "German"]. */
export function pageLanguages(pages: readonly Pick<CrawledPage, 'language'>[]): string[] {
  const counts = new Map<string, number>()
  for (const { language } of pages) {
    if (language !== '') counts.set(language, (counts.get(language) ?? 0) + 1)
  }
  return [...counts.entries()]
    .sort(([a, m], [b, n]) => n - m || a.localeCompare(b))
    .map(([code]) => languageLabel(code))
}

/** What the scan read: "Read 10 of 10 pages, all in English." or "…, in English and German." */
export function readSummary(
  summary: Pick<AnalysisSummary, 'maxPages'>,
  pages: readonly Pick<CrawledPage, 'language'>[],
): string {
  const read = `Read ${String(pages.length)} of ${String(summary.maxPages)} ${summary.maxPages === 1 ? 'page' : 'pages'}`
  const languages = pageLanguages(pages)
  if (languages.length === 0) return `${read}.`
  if (languages.length === 1) return `${read}, all in ${joinWords(languages)}.`
  return `${read}, in ${joinWords(languages)}.`
}

export type SpectrumBand = { tier: IntentTier; label: string; count: number }

export type Spectrum = {
  /** How many known terms the site uses. */
  total: number
  /** One band per tier, in the tiers' order, empty ones included for the legend. */
  bands: SpectrumBand[]
  /** The bar in words: "14 terms: 8 specification, 4 product, 2 broad". */
  description: string
}

function terms(n: number): string {
  return `${String(n)} ${n === 1 ? 'term' : 'terms'}`
}

/** The temper spectrum: how many of the site's known terms are of each kind of search. */
export function spectrum(knownTerms: readonly Pick<KnownTerm, 'tier'>[]): Spectrum {
  const bands = TIERS.map((tier) => ({
    tier,
    label: tierLabel(tier),
    count: knownTerms.filter((t) => t.tier === tier).length,
  }))
  const parts = bands.filter((b) => b.count > 0).map((b) => `${String(b.count)} ${b.label.toLowerCase()}`)
  const total = knownTerms.length
  return { total, bands, description: parts.length > 0 ? `${terms(total)}: ${parts.join(', ')}` : terms(total) }
}

/** The terms the site uses and Tribelt's own website does not, most frequent first. */
export function missingTerms(knownTerms: readonly KnownTerm[]): KnownTerm[] {
  return knownTerms.filter((t) => !t.usedByTribelt).sort((a, b) => b.occurrences - a.occurrences)
}

/** The languages the phrases are in, each once, alphabetically by code. */
export function keywordLanguages(keywords: readonly Keyword[]): string[] {
  return [...new Set(keywords.map((k) => k.language))].sort()
}

/** Whether a phrase is one word, which says little about what a site is about. */
export function isSingleWord(phrase: string): boolean {
  return !/\s/.test(phrase.trim())
}

export type Phrase = Keyword & {
  /** The score as a share of the top one shown, from 0 to 1, for its bar. */
  share: number
}

export type PhraseFilter = {
  /** A language code, or '' for every language. */
  language: string
  singleWords: boolean
}

/** The phrases to show, highest score first, each with its bar's length against the top score. */
export function phrases(keywords: readonly Keyword[], filter: PhraseFilter): Phrase[] {
  const shown = keywords
    .filter((k) => filter.language === '' || k.language === filter.language)
    .filter((k) => filter.singleWords || !isSingleWord(k.phrase))
    .sort((a, b) => b.score - a.score)
  const top = shown[0]?.score ?? 0
  return shown.map((k) => ({ ...k, share: top > 0 ? k.score / top : 0 }))
}

/** A score as the list shows it: whole points. */
export function scoreLabel(score: number): string {
  return Math.round(score).toLocaleString('en')
}

export type EquivalentRow = {
  /** The site's own name for the product. */
  theirs: string
  /** The Tribelt product that answers to it. */
  ours: string
  /** The site's words for it, each once, in the order they were found. */
  words: string
  reason: string
}

/** The matched products, as rows of the equivalents table. */
export function equivalentRows(equivalents: readonly Equivalent[]): EquivalentRow[] {
  return equivalents.map((e) => ({
    theirs: e.product,
    ours: tribeltProduct(e.family),
    words: [...new Set(e.searchTerms.map((t) => t.term))].join(', '),
    reason: e.reason,
  }))
}

/** A crawled page's path, as the pages list shows it: "/conveyor-belts". */
export function pagePath(url: string): string {
  try {
    const { pathname, search } = new URL(url)
    const path = pathname.length > 1 ? pathname.replace(/\/+$/, '') : pathname
    return `${path}${search}`
  } catch {
    return url
  }
}

/** A word count: "2,620 words". */
export function wordsLabel(count: number): string {
  return `${count.toLocaleString('en')} ${count === 1 ? 'word' : 'words'}`
}

/** How many pages were read: "The 10 pages read", "The page read". */
export function pagesReadTitle(count: number): string {
  return count === 1 ? 'The page read' : `The ${String(count)} pages read`
}
