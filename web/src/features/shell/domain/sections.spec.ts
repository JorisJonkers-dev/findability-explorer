import { describe, expect, it } from 'vitest'
import { AI_SEARCH_TABS, companyTabs, familyOfSlug, familySlug, NAV_SECTIONS, PRODUCT_TABS, section, SECTIONS, sectionOfPath } from './sections'

describe('the sections', () => {
  it('numbers the five the poster shows, in its order', () => {
    expect(SECTIONS.map((s) => `${s.number} ${s.label}`)).toEqual(['01 Catching up', '02 AI search', '03 Competitors', '04 Search terms', '05 The plan'])
  })

  it('lists every section in the top bar, Products and Buyers too', () => {
    expect(NAV_SECTIONS.map((s) => s.label)).toEqual(['Catching up', 'AI search', 'Competitors', 'Products', 'Search terms', 'The plan', 'Buyers'])
  })

  it('finds the section a path shows, a scan and a competitor in the competitors section', () => {
    expect(sectionOfPath('/terms')?.id).toBe('terms')
    expect(sectionOfPath('/scan/5b0c')?.id).toBe('competitors')
    expect(sectionOfPath('/competitors/twentebelt/history')?.id).toBe('competitors')
    expect(sectionOfPath('/competitors/all')?.id).toBe('competitors')
    expect(sectionOfPath('/ai-search/readiness')?.id).toBe('ai-search')
    expect(sectionOfPath('/products/open-spiral')?.id).toBe('products')
    expect(sectionOfPath('/catching-up')?.id).toBe('catching-up')
    expect(sectionOfPath('/')).toBeUndefined()
    expect(sectionOfPath('/settings')).toBeUndefined()
    expect(section('plan').path).toBe('/plan')
  })
})

describe('the tabs', () => {
  it('give AI search its summary and two pages', () => {
    expect(AI_SEARCH_TABS.map((t) => t.path)).toEqual(['/ai-search', '/ai-search/prompt-test', '/ai-search/readiness'])
  })

  it("give a competitor the list, its latest scan and its history", () => {
    expect(companyTabs('märtens')).toEqual([
      { label: 'All competitors', path: '/competitors/all' },
      { label: 'Latest scan', path: '/competitors/m%C3%A4rtens' },
      { label: 'Over time', path: '/competitors/m%C3%A4rtens/history' },
    ])
  })

  it('give products every family, each at an address of its own', () => {
    expect(PRODUCT_TABS.map((t) => t.label)).toEqual(['All products', 'Eyelink', 'Open spiral', 'Closed spiral', 'Tri-Flex', 'Other belts'])
    expect(familySlug('tri_flex_turn_curve')).toBe('tri-flex-turn-curve')
    expect(familyOfSlug('open-spiral')).toBe('open_spiral')
    expect(familyOfSlug('nothing')).toBeUndefined()
  })
})
