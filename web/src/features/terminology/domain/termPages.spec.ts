import { describe, expect, it } from 'vitest'
import type { CompanyTermPages } from '../../../infrastructure/api/types.gen'
import { pageLabel, splitTermPages, unrecordedLine } from './termPages'

const tribelt: CompanyTermPages = { competitorId: 'tribelt', name: 'Tribelt B.V.', role: 'tribelt', pages: [{ url: 'https://www.tribelt.nl/en/eyelink', title: 'Eyelink belts' }] }
const esfo: CompanyTermPages = { competitorId: 'esfo', name: 'ESFO', role: 'competitor', pages: [] }
const group: CompanyTermPages = { competitorId: 'lesjofors', name: 'Lesjöfors', role: 'group', pages: [{ url: 'https://lesjofors.test/', title: '' }] }

describe('the pages that use a term', () => {
  it('puts tribelt.nl apart from the other companies', () => {
    expect(splitTermPages([tribelt, esfo, group])).toEqual({ tribelt, others: [esfo, group] })
    expect(splitTermPages([esfo])).toEqual({ tribelt: undefined, others: [esfo] })
    expect(splitTermPages(undefined)).toEqual({ tribelt: undefined, others: [] })
  })

  it('names a page by its title, or its address without the scheme', () => {
    expect(pageLabel({ url: 'https://www.tribelt.nl/en/eyelink', title: 'Eyelink belts' })).toBe('Eyelink belts')
    expect(pageLabel({ url: 'https://www.tribelt.nl/en/eyelink/', title: ' ' })).toBe('www.tribelt.nl/en/eyelink')
  })

  it('says why a company that uses the term lists no page', () => {
    expect(unrecordedLine).toBe('Its newest scan found the term before scans recorded the pages. The next scan lists them.')
  })
})
