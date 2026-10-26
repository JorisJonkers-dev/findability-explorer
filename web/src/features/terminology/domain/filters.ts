import type { IntentTier } from '../../../infrastructure/api/types.gen'
import { isFocusLanguage, isTier } from '../../../shared/vocabulary'

/** The two halves of the map: what a belt is called or used for, and what engineers type. */
export type TermGroup = 'marketing' | 'technical'

/**
 * What the search terms view shows. The URL's query string holds it, so a view can be shared and
 * old /terminology links keep working.
 */
export type TermView = {
  /** The search text as typed; matching trims and folds it. */
  q: string
  /** A chart language: en-US counts as English, so it is never en-US. */
  language: string | null
  tier: IntentTier | null
  /** A company, by the name the map credits it under; '' for every company. */
  user: string
  group: TermGroup | null
  /** Only the terms Tribelt's own website does not use. */
  gap: boolean
}

/** The view's filters a pill or chart row toggles. */
export type ToggleKey = 'language' | 'tier' | 'user' | 'group'

/** A query string as the router hands it over. */
export type RawQuery = Record<string, string | null | (string | null)[] | undefined>

/** The longest search text and company name the view keeps. */
export const MAX_TEXT_LENGTH = 100

export const NO_VIEW: TermView = { q: '', language: null, tier: null, user: '', group: null, gap: false }

function first(value: RawQuery[string]): string {
  const raw = Array.isArray(value) ? value[0] : value
  return typeof raw === 'string' ? raw.slice(0, MAX_TEXT_LENGTH) : ''
}

/**
 * The company behind a used-by entry, which may name a product line or division in brackets:
 * "Twentebelt (TwenteFlex)" is Twentebelt. Filtering by the company finds all of its terms.
 */
export function companyName(usedBy: string): string {
  const name = usedBy.replace(/\s*\([^()]*\)\s*$/, '').trim()
  return name === '' ? usedBy.trim() : name
}

/** The language a chart counts a term under: American English is English. */
export function chartLanguage(code: string): string {
  return code === 'en-US' ? 'en' : code
}

function isGroup(value: string): value is TermGroup {
  return value === 'marketing' || value === 'technical'
}

/** Reads the view from a query string, dropping any value it does not know. */
export function parseView(query: RawQuery): TermView {
  const language = first(query.language)
  const tier = first(query.tier)
  const group = first(query.group)
  const user = first(query.user)
  return {
    q: first(query.q),
    language: isFocusLanguage(language) ? chartLanguage(language) : null,
    tier: isTier(tier) ? tier : null,
    user: user.trim() === '' ? '' : companyName(user),
    group: isGroup(group) ? group : null,
    gap: first(query.gap) === '1',
  }
}

/** The view as a query string, without the filters not in use. */
export function toViewQuery(view: TermView): Record<string, string> {
  const query: Record<string, string> = {}
  if (view.q !== '') query.q = view.q
  if (view.language !== null) query.language = view.language
  if (view.tier !== null) query.tier = view.tier
  if (view.user !== '') query.user = view.user
  if (view.group !== null) query.group = view.group
  if (view.gap) query.gap = '1'
  return query
}

/** The view with one filter toggled: set to value, or unset when it already has it. */
export function toggled(view: TermView, key: ToggleKey, value: string): TermView {
  const raw = { ...toViewQuery(view), [key]: view[key] === value ? '' : value }
  return parseView(raw)
}
