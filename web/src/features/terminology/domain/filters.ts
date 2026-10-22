import type { BeltFamily, Facets, IntentTier, Language, SearchTermsData } from '../../../infrastructure/api/types.gen'
import {
  FAMILIES,
  familyLabel,
  isFamily,
  isLanguage,
  isTier,
  LANGUAGES,
  languageLabel,
  TIERS,
  tierLabel,
} from '../../../shared/vocabulary'

/** What the terminology page searches for. The URL's query string holds it, so a view can be shared. */
export type TermFilters = {
  /** The text as typed; the request trims it. */
  q: string
  language: Language | null
  tier: IntentTier | null
  family: BeltFamily | null
  user: string
}

export type FilterKey = keyof TermFilters

/** A query string as the router hands it over. */
export type RawQuery = Record<string, string | null | (string | null)[] | undefined>

/** The longest q and user the contract accepts. */
export const MAX_TEXT_LENGTH = 100

function first(value: RawQuery[string]): string {
  const raw = Array.isArray(value) ? value[0] : value
  return typeof raw === 'string' ? raw.slice(0, MAX_TEXT_LENGTH) : ''
}

/** Reads filters from a query string, dropping any value the contract would refuse. */
export function parseFilters(query: RawQuery): TermFilters {
  const language = first(query.language)
  const tier = first(query.tier)
  const family = first(query.family)
  return {
    q: first(query.q),
    language: isLanguage(language) ? language : null,
    tier: isTier(tier) ? tier : null,
    family: isFamily(family) ? family : null,
    user: first(query.user),
  }
}

/** The filters as a query string, without the empty ones. */
export function toRouteQuery(filters: TermFilters): Record<string, string> {
  const query: Record<string, string> = {}
  for (const [key, value] of Object.entries(filters)) {
    if (typeof value === 'string' && value !== '') query[key] = value
  }
  return query
}

/** Whether any filter is set, so there is something to clear. */
export function hasFilters(filters: TermFilters): boolean {
  return Object.keys(toRouteQuery(filters)).length > 0
}

/** The filters as a searchTerms request: trimmed, without the empty ones. */
export function toSearchQuery(filters: TermFilters, limit: number): NonNullable<SearchTermsData['query']> {
  const query: NonNullable<SearchTermsData['query']> = {}
  const q = filters.q.trim()
  const user = filters.user.trim()
  if (q !== '') query.q = q
  if (user !== '') query.user = user
  if (filters.language) query.language = filters.language
  if (filters.tier) query.tier = filters.tier
  if (filters.family) query.family = filters.family
  query.limit = limit
  return query
}

/** One value of a facet, with how many matching terms have it. */
export type FacetOption<T extends string> = { value: T; label: string; count: number }

function counter(counts: readonly { value: string; count: number }[]): (value: string) => number {
  const byValue = new Map(counts.map((c) => [c.value, c.count]))
  return (value) => byValue.get(value) ?? 0
}

/** Every tier, in the tiers' order, for the kind-of-search pills. */
export function tierOptions(facets: Facets): FacetOption<IntentTier>[] {
  const count = counter(facets.tiers)
  return TIERS.map((value) => ({ value, label: tierLabel(value), count: count(value) }))
}

/**
 * The languages worth a chip: those some matching term is in, and the one picked even when
 * nothing matches it any more, so it can still be unpicked.
 */
export function languageOptions(facets: Facets, picked: Language | null): FacetOption<Language>[] {
  const count = counter(facets.languages)
  return LANGUAGES.filter((value) => count(value) > 0 || value === picked).map((value) => ({
    value,
    label: languageLabel(value),
    count: count(value),
  }))
}

/** How many terms match in any language: every term has one, so the language counts add up. */
export function anyLanguageCount(facets: Facets): number {
  return facets.languages.reduce((sum, c) => sum + c.count, 0)
}

/** Every belt family, in the families' order, for the family select. */
export function familyOptions(facets: Facets): FacetOption<BeltFamily>[] {
  const count = counter(facets.families)
  return FAMILIES.map((value) => ({ value, label: familyLabel(value), count: count(value) }))
}

function terms(n: number): string {
  return n === 1 ? 'term' : 'terms'
}

/** How many terms match of how many the map holds: "35 of 146 terms". */
export function resultSummary(matching: number, inMap: number): string {
  if (matching === 0) return 'No terms match'
  if (inMap === 0 || matching >= inMap) return `${String(matching)} ${terms(matching)}`
  return `${String(matching)} of ${String(inMap)} ${terms(inMap)}`
}

/** A used-by entry split in two: the company, and the product line or note in brackets after it. */
export type UsedBy = { company: string; note: string }

/**
 * The company behind a used-by entry, which may name a product line or division in brackets:
 * "Twentebelt (TwenteFlex)" is Twentebelt. Filtering by the company finds all of its terms.
 */
export function companyName(usedBy: string): string {
  const name = usedBy.replace(/\s*\([^()]*\)\s*$/, '').trim()
  return name === '' ? usedBy.trim() : name
}

/** A used-by entry as its button shows it: the company, then its note. */
export function splitUsedBy(usedBy: string): UsedBy {
  const company = companyName(usedBy)
  const note = /\(([^()]*)\)\s*$/.exec(usedBy)?.[1]?.trim() ?? ''
  return { company, note: company === usedBy.trim() ? '' : note }
}
