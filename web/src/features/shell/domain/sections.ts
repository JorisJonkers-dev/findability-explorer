/** The poster's five sections, in its order. */
export type SectionId = 'buyers' | 'ai-search' | 'competitors' | 'terms' | 'plan'

export type Section = {
  id: SectionId
  /** Its number on the poster: "01". */
  number: string
  label: string
  path: string
}

export const SECTIONS: readonly Section[] = [
  { id: 'buyers', number: '01', label: 'Buyers', path: '/buyers' },
  { id: 'ai-search', number: '02', label: 'AI search', path: '/ai-search' },
  { id: 'competitors', number: '03', label: 'Competitors', path: '/competitors' },
  { id: 'terms', number: '04', label: 'Search terms', path: '/terms' },
  { id: 'plan', number: '05', label: 'The plan', path: '/plan' },
]

/** The section a path shows, or undefined: a scan is shown in the competitors section. */
export function sectionOfPath(path: string): Section | undefined {
  const scan = path.startsWith('/scan/') || path.startsWith('/competitors/') ? '/competitors' : path
  return SECTIONS.find((s) => s.path === scan)
}

/** The section with this id. */
export function section(id: SectionId): Section {
  return SECTIONS.find((s) => s.id === id) as Section
}
