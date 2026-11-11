import type { CompanyTermPages, TermPage } from '../../../infrastructure/api/types.gen'

/** The pages that use a term: tribelt.nl's, and every other company's, in the server's order. */
export function splitTermPages(items: readonly CompanyTermPages[] | undefined): { tribelt: CompanyTermPages | undefined; others: CompanyTermPages[] } {
  const all = items ?? []
  return { tribelt: all.find((c) => c.role === 'tribelt'), others: all.filter((c) => c.role !== 'tribelt') }
}

/** A page by its title, or by its address without the scheme and the closing slash. */
export function pageLabel(page: TermPage): string {
  return page.title.trim() || page.url.replace(/^https?:\/\//, '').replace(/\/$/, '')
}

/** What is said of a company that uses the term on a scan from before scans recorded the pages. */
export const unrecordedLine = 'Its newest scan found the term before scans recorded the pages. The next scan lists them.'
