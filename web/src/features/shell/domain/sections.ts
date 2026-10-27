import { familyLabel, PRODUCT_FAMILIES } from '../../../shared/vocabulary'

/** Every section the top bar links to. */
export type SectionId = 'catching-up' | 'ai-search' | 'competitors' | 'products' | 'terms' | 'plan' | 'buyers'

export type Section = {
  id: SectionId
  /** Its number on the poster, "01"; empty for a section the poster does not show. */
  number: string
  label: string
  path: string
}

/** The sections in the top bar's order. */
export const NAV_SECTIONS: readonly Section[] = [
  { id: 'catching-up', number: '01', label: 'Catching up', path: '/catching-up' },
  { id: 'ai-search', number: '02', label: 'AI search', path: '/ai-search' },
  { id: 'competitors', number: '03', label: 'Competitors', path: '/competitors' },
  { id: 'products', number: '', label: 'Products', path: '/products' },
  { id: 'terms', number: '04', label: 'Search terms', path: '/terms' },
  { id: 'plan', number: '05', label: 'The plan', path: '/plan' },
  { id: 'buyers', number: '', label: 'Buyers', path: '/buyers' },
]

/** The poster's five sections, in its order. */
export const SECTIONS: readonly Section[] = NAV_SECTIONS.filter((s) => s.number !== '')

/** The section a path shows, or undefined: a scan and a competitor's pages are in the competitors section. */
export function sectionOfPath(path: string): Section | undefined {
  const scan = path.startsWith('/scan/') ? '/competitors' : path
  return NAV_SECTIONS.find((s) => s.path === scan || scan.startsWith(`${s.path}/`))
}

/** The section with this id. */
export function section(id: SectionId): Section {
  return NAV_SECTIONS.find((s) => s.id === id) as Section
}

/** A tab inside a section: a page of it. */
export type Tab = { label: string; path: string; /** A count beside the label, such as the terms waiting. */ count?: number }

/** The AI search section's pages. */
export const AI_SEARCH_TABS: readonly Tab[] = [
  { label: 'Summary', path: '/ai-search' },
  { label: 'Prompt test', path: '/ai-search/prompt-test' },
  { label: 'AI readiness', path: '/ai-search/readiness' },
]

/** Where the list of every competitor lives. */
export const ALL_COMPETITORS = '/competitors/all'

/** The competitors section's pages. */
export const COMPETITOR_TABS: readonly Tab[] = [
  { label: 'Summary', path: '/competitors' },
  { label: 'All competitors', path: ALL_COMPETITORS },
]

/** A competitor's latest scan page. */
export function companyPath(id: string): string {
  return `/competitors/${encodeURIComponent(id)}`
}

/** A competitor's pages: back to the list, its latest scan, and how it changed over time. */
export function companyTabs(id: string): Tab[] {
  return [
    { label: 'All competitors', path: ALL_COMPETITORS },
    { label: 'Latest scan', path: companyPath(id) },
    { label: 'Over time', path: `${companyPath(id)}/history` },
  ]
}

/** A belt family as an address: "open-spiral". */
export function familySlug(family: string): string {
  return family.replaceAll('_', '-')
}

/** The family an address names, or undefined. */
export function familyOfSlug(slug: string): (typeof PRODUCT_FAMILIES)[number] | undefined {
  return PRODUCT_FAMILIES.find((f) => familySlug(f) === slug)
}

/** The products section's pages: every family, then one per family. */
export const PRODUCT_TABS: readonly Tab[] = [
  { label: 'All products', path: '/products' },
  ...PRODUCT_FAMILIES.map((f) => ({ label: familyLabel(f), path: `/products/${familySlug(f)}` })),
]
