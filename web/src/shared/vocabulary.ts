import type { BeltFamily, IntentTier, Language } from '../infrastructure/api/types.gen'

// Each record is keyed by the contract's own enum, so a value the contract adds or drops is a
// compile error here rather than a missing label on screen.

const LANGUAGE_LABELS: Record<Language, string> = {
  de: 'German',
  en: 'English',
  'en-US': 'American English',
  nl: 'Dutch',
  fr: 'French',
  pl: 'Polish',
  it: 'Italian',
  es: 'Spanish',
  sv: 'Swedish',
  da: 'Danish',
}

const FAMILY_LABELS: Record<BeltFamily, string> = {
  eyelink: 'Eyelink',
  open_spiral: 'Open spiral',
  closed_spiral: 'Closed spiral',
  tri_flex_turn_curve: 'Tri-Flex',
  other: 'Other belts',
  all: 'All belts',
  spec_all: 'Specification',
}

/** Which of Tribelt's belts a term is about, as it reads after the tier: "Product, Tri-Flex". */
const SCOPE_LABELS: Record<BeltFamily, string> = {
  eyelink: 'Eyelink',
  open_spiral: 'Open spiral',
  closed_spiral: 'Closed spiral',
  tri_flex_turn_curve: 'Tri-Flex',
  other: 'other belts',
  all: 'all belts',
  spec_all: 'all belts',
}

/** The Tribelt product that answers to a family. */
const TRIBELT_PRODUCTS: Record<BeltFamily, string> = {
  eyelink: 'Eyelink belt',
  open_spiral: 'Open spiral belt',
  closed_spiral: 'Closed spiral belt',
  tri_flex_turn_curve: 'Tri-Flex',
  other: 'Other belts',
  all: 'Any belt',
  spec_all: 'Any belt',
}

const TIER_LABELS: Record<IntentTier, string> = {
  spec: 'Specification',
  product: 'Product',
  application: 'Application',
  problem: 'Problem',
  broad: 'Broad',
}

/** Every language, in the contract's order. */
export const LANGUAGES = Object.keys(LANGUAGE_LABELS) as Language[]
/** Every belt family, in the contract's order. */
export const FAMILIES = Object.keys(FAMILY_LABELS) as BeltFamily[]
/** Every intent tier, in the contract's order. */
export const TIERS = Object.keys(TIER_LABELS) as IntentTier[]

export function isLanguage(value: string): value is Language {
  return Object.hasOwn(LANGUAGE_LABELS, value)
}

export function isFamily(value: string): value is BeltFamily {
  return Object.hasOwn(FAMILY_LABELS, value)
}

export function isTier(value: string): value is IntentTier {
  return Object.hasOwn(TIER_LABELS, value)
}

const LANGUAGE_NAMES = new Intl.DisplayNames(['en'], { type: 'language', fallback: 'code' })

/**
 * The language's English name. A code the map does not know (a crawled page's, such as pt or
 * en-GB) is named by the browser; one that is not a language code at all shows as itself.
 */
export function languageLabel(code: string): string {
  if (isLanguage(code)) return LANGUAGE_LABELS[code]
  try {
    return LANGUAGE_NAMES.of(code) ?? code
  } catch {
    return code
  }
}

/** The family's short name, as the competitor matrix heads its columns. */
export function familyLabel(family: BeltFamily): string {
  return FAMILY_LABELS[family]
}

/** The belts a term is about: a family's name, or "all belts" for a broad or specification term. */
export function scopeLabel(family: BeltFamily): string {
  return SCOPE_LABELS[family]
}

/** The Tribelt product that answers to a family. */
export function tribeltProduct(family: BeltFamily): string {
  return TRIBELT_PRODUCTS[family]
}

/** Tribelt's five belt families, the columns of the competitor matrix. */
export const PRODUCT_FAMILIES = ['eyelink', 'open_spiral', 'closed_spiral', 'tri_flex_turn_curve', 'other'] as const satisfies readonly BeltFamily[]

export function tierLabel(tier: IntentTier): string {
  return TIER_LABELS[tier]
}
