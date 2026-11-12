import { PRODUCT_FAMILIES } from '../../../shared/vocabulary'

/** Every section the top bar links to. */
export type SectionId = 'overview' | 'ai-responses' | 'ai-readiness' | 'terms' | 'competitors' | 'recommendations' | 'help'

export type Section = {
  id: SectionId
  /** Its number in the eyebrow, "01". */
  number: string
  label: string
  path: string
}

/** The Results sections in the top bar's order. Manage sits apart, on the right, for administrators. */
export const NAV_SECTIONS: readonly Section[] = [
  { id: 'overview', number: '01', label: 'Overview', path: '/' },
  { id: 'ai-responses', number: '02', label: 'AI responses', path: '/ai-responses' },
  { id: 'ai-readiness', number: '03', label: 'AI readiness', path: '/ai-readiness' },
  { id: 'terms', number: '04', label: 'Search terms', path: '/terms' },
  { id: 'competitors', number: '05', label: 'Competitors', path: '/competitors' },
  { id: 'recommendations', number: '06', label: 'Recommendations', path: '/recommendations' },
  { id: 'help', number: '07', label: 'Help', path: '/help' },
]

/** The numbered sections, in order. */
export const SECTIONS: readonly Section[] = NAV_SECTIONS

/** Pages outside a section's own path that still belong to it. */
const EXTRA_PREFIXES: readonly (readonly [string, SectionId])[] = [['/goals', 'overview']]

/** The section a path shows, or undefined: a goal and the Overview are both in Overview, and Manage is in none. */
export function sectionOfPath(path: string): Section | undefined {
  if (path === '/') return NAV_SECTIONS[0]
  const extra = EXTRA_PREFIXES.find(([prefix]) => path === prefix || path.startsWith(`${prefix}/`))
  if (extra) return NAV_SECTIONS.find((s) => s.id === extra[1])
  return NAV_SECTIONS.slice(1).find((s) => s.path === path || path.startsWith(`${s.path}/`))
}

/** The section with this id. */
export function section(id: SectionId): Section {
  return NAV_SECTIONS.find((s) => s.id === id) as Section
}

/** A competitor's latest scan page. */
export function companyPath(id: string): string {
  return `/competitors/${encodeURIComponent(id)}`
}

/** A belt family as an address: "open-spiral". */
export function familySlug(family: string): string {
  return family.replaceAll('_', '-')
}

/** The family an address names, or undefined. */
export function familyOfSlug(slug: string): (typeof PRODUCT_FAMILIES)[number] | undefined {
  return PRODUCT_FAMILIES.find((f) => familySlug(f) === slug)
}

/** A belt family's page. */
export function familyPath(family: string): string {
  return `/terms/families/${familySlug(family)}`
}
