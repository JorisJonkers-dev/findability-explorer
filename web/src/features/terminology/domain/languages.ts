import { languageLabel } from '../../../shared/vocabulary'
import { chartLanguage } from './filters'

/** One language a filter offers: its code, and its name for people. */
export type LanguageChoice = { value: string; label: string }

/** The languages Tribelt's buyers search in, which every language filter lists first, in this order. */
const FIRST: readonly string[] = ['nl', 'de', 'en']

function rank(code: string): number {
  const i = FIRST.indexOf(code)
  return i < 0 ? FIRST.length : i
}

/**
 * The languages a filter offers for rows in codes: only those that appear, each once (American
 * English is English), Dutch, German and English first and any other after them by name.
 */
export function languageChoices(codes: readonly string[]): LanguageChoice[] {
  return [...new Set(codes.map(chartLanguage))]
    .map((value) => ({ value, label: languageLabel(value) }))
    .sort((a, b) => rank(a.value) - rank(b.value) || a.label.localeCompare(b.label))
}

/** Whether a row in language code is shown when the filter is set to choice; null shows every row. */
export function inLanguage(code: string, choice: string | null): boolean {
  return choice === null || chartLanguage(code) === choice
}

/** The rows in the chosen language, in their order; every row when none is chosen. */
export function ofLanguage<T extends { language: string }>(rows: readonly T[], choice: string | null): T[] {
  return rows.filter((r) => inLanguage(r.language, choice))
}

/**
 * A language read from the address (?language=de): a language code, with American English as
 * English, or null for every language when it is missing or no code.
 */
export function parseLanguage(raw: unknown): string | null {
  const value = Array.isArray(raw) ? (raw as unknown[])[0] : raw
  return typeof value === 'string' && /^[a-z]{2,3}(?:-[A-Z]{2})?$/.test(value) ? chartLanguage(value) : null
}

/** The address's query with the language set, or without it for every language. */
export function withLanguage(query: Record<string, string>, language: string | null): Record<string, string> {
  const rest = { ...query }
  delete rest.language
  return language === null ? rest : { ...rest, language }
}
