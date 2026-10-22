import type { BeltFamily, BeltName, Competitor } from '../../../infrastructure/api/types.gen'
import { PRODUCT_FAMILIES } from '../../../shared/vocabulary'
import { fold } from '../../terminology/domain/fold'
import { companyName } from '../../terminology/domain/filters'

type Cut = { at: number; length: number }

/** Where separator occurs in text outside brackets: in "Flat Wire, Z-Belts (Z, CB)" one ", " counts. */
function cutsOutsideBrackets(text: string, separator: RegExp): Cut[] {
  const cuts: Cut[] = []
  let depth = 0
  for (let at = 0; at < text.length; at++) {
    const char = text.charAt(at)
    if (char === '(') depth++
    else if (char === ')') depth = Math.max(0, depth - 1)
    else if (depth === 0) {
      const match = separator.exec(text.slice(at))
      if (match) {
        cuts.push({ at, length: match[0].length })
        at += match[0].length - 1
      }
    }
  }
  return cuts
}

/** Splits text at a separator outside brackets. */
function splitOutsideBrackets(text: string, separator: RegExp): string[] {
  const parts: string[] = []
  let start = 0
  for (const cut of cutsOutsideBrackets(text, separator)) {
    parts.push(text.slice(start, cut.at))
    start = cut.at + cut.length
  }
  parts.push(text.slice(start))
  return parts
}

/** The text before the first separator outside brackets, or all of it. */
function firstOutsideBrackets(text: string, separator: RegExp): string {
  return text.slice(0, cutsOutsideBrackets(text, separator)[0]?.at ?? text.length)
}

/** A name without the note in brackets at its end: "Eye-Link belts (OGBL)" is "Eye-Link belts". */
function withoutNote(name: string): string {
  return name.replace(/\s*\([^()]*\)\s*$/, '').trim()
}

/**
 * The separate product names in one recorded belt name. A list ("Flat Wire, Z-Baking Belts;
 * Cleatrac") is several products; translations of one ("Drahtösengliederband / Loop-joined wire
 * belt") are one product, named by the first.
 */
export function productNames(beltName: string): string[] {
  return splitOutsideBrackets(beltName, /^\s*[;,]\s*/)
    .map((product) => withoutNote(firstOutsideBrackets(product, /^\s+\/\s+/)))
    .filter((name) => name !== '')
}

/** One cell of the matrix: the company's first name for a family, and how many more it has. */
export type FamilyCell = { family: BeltFamily; first: string | null; more: number }

/** A company's names for one family, as a matrix cell. */
export function familyCell(beltNames: readonly BeltName[], family: BeltFamily): FamilyCell {
  const names = beltNames.filter((b) => b.family === family).flatMap((b) => productNames(b.name))
  return { family, first: names[0] ?? null, more: Math.max(0, names.length - 1) }
}

/** "and 2 more", or '' for none. */
export function moreLabel(more: number): string {
  return more > 0 ? `and ${String(more)} more` : ''
}

// The map writes the United Kingdom as UK; ISO 3166 calls it GB.
const REGION_ALIASES: Readonly<Record<string, string>> = { UK: 'GB' }
const REGIONS = new Intl.DisplayNames(['en'], { type: 'region', fallback: 'code' })

/** Where a company is based, for people: "NL (Panheel)" is "Netherlands". */
export function countryName(country: string): string {
  const code = /^([A-Z]{2})\b/.exec(country)?.[1]
  if (code !== undefined) {
    // A code the browser does not know comes back as itself.
    const region = String(REGIONS.of(REGION_ALIASES[code] ?? code))
    if (region !== code) return region
  }
  return withoutNote(country)
}

/**
 * How many languages a company's website is in. The map notes a language it could not check as
 * "unverified" or "others unverified", which is not one.
 */
export function languageCount(languages: readonly string[]): number {
  return languages.filter((l) => /^[a-z]{2,3}(?:-[A-Za-z]{2,4})?(?![\p{L}-])/u.test(l)).length
}

/** The line under a company's name: "Netherlands, 2 languages". */
export function whereLabel(competitor: Pick<Competitor, 'country' | 'languages'>): string {
  const country = countryName(competitor.country)
  const count = languageCount(competitor.languages)
  if (count === 0) return country
  return `${country}, ${String(count)} ${count === 1 ? 'language' : 'languages'}`
}

const LEGAL_FORM = /^(?:gmbh|b\.?v\.?|s\.?a\.?|s\.?p\.?a\.?|s\.?l\.?|ltd\.?|inc\.?|ag|co\.?|company|corp\.?)$/i

/** A company's registered name without its legal form: "Belt Technologies, Inc." is "Belt Technologies". */
export function tidyName(name: string): string {
  const words = companyName(name).split(/\s+/)
  while (words.length > 1 && LEGAL_FORM.test(String(words.at(-1)))) words.pop()
  return words.join(' ').replace(/,$/, '')
}

function words(text: string): string[] {
  return fold(text)
    .split(/[^\p{L}\p{N}]+/u)
    .filter((w) => w !== '')
}

/**
 * A company's short name, the one the terminology map credits its terms to: "Märtens" for
 * "Märtens Transportbänder GmbH (MARTENS)". It is the longest name among users whose every word
 * begins a word of the company's registered name, so "Wire Belt Osterloh" wins over "Wire Belt"
 * for "Wire Belt Company Osterloh GmbH". users comes most-credited first, which breaks a tie.
 * A company no term credits keeps its registered name, tidied.
 */
export function shortName(name: string, users: readonly string[]): string {
  const registered = words(name)
  let best: { name: string; length: number } | null = null
  for (const user of users) {
    const own = words(user)
    const fits = own.length > 0 && own.every((w) => registered.some((r) => r.startsWith(w)))
    if (fits && (best === null || own.length > best.length)) best = { name: user, length: own.length }
  }
  return best ? best.name : tidyName(name)
}

/** The companies credited with the map's terms, most-credited first, each once. */
export function creditedCompanies(terms: readonly { usedBy: readonly string[] }[]): string[] {
  const counts = new Map<string, number>()
  for (const term of terms) {
    for (const user of term.usedBy) {
      const company = companyName(user)
      counts.set(company, (counts.get(company) ?? 0) + 1)
    }
  }
  return [...counts.entries()].sort(([, m], [, n]) => n - m).map(([company]) => company)
}

/** One company's row of the matrix. */
export type MatrixRow = {
  /** The registered name, which is unique. */
  key: string
  /** The short name, shown and searched for in the terminology. */
  name: string
  where: string
  country: string
  cells: FamilyCell[]
  groupCompany: boolean
}

/** The matrix: one row per company, one cell per Tribelt belt family. */
export function matrixRows(competitors: readonly Competitor[], users: readonly string[]): MatrixRow[] {
  return competitors.map((c) => ({
    key: c.name,
    name: shortName(c.name, users),
    where: whereLabel(c),
    country: c.country,
    cells: PRODUCT_FAMILIES.map((family) => familyCell(c.beltNames, family)),
    groupCompany: c.groupCompany,
  }))
}

/** The rows whose name or country contains text, folded as the server folds. */
export function filterRows(rows: readonly MatrixRow[], text: string): MatrixRow[] {
  const needle = fold(text)
  if (needle === '') return [...rows]
  return rows.filter((row) =>
    [row.name, row.key, row.country, row.where].some((field) => fold(field).includes(needle)),
  )
}

/** How many companies are shown of how many there are: "3 of 25 competitors". */
export function rowSummary(shown: number, total: number, noun: string): string {
  return shown === total ? `${String(total)} ${noun}` : `${String(shown)} of ${String(total)} ${noun}`
}
