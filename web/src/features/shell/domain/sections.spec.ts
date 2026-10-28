import { describe, expect, it } from 'vitest'
import { companyPath, familyOfSlug, familyPath, familySlug, NAV_SECTIONS, section, SECTIONS, sectionOfPath } from './sections'

describe('sections', () => {
  it('numbers the five sections in the top bar order', () => {
    expect(SECTIONS.map((s) => `${s.number} ${s.label}`)).toEqual(['01 Overview', '02 AI search', '03 Competitors', '04 Search terms', '05 Recommendations'])
    expect(NAV_SECTIONS.map((s) => s.path)).toEqual(['/', '/ai-search', '/competitors', '/terms', '/recommendations'])
  })

  it('finds the section a path is in', () => {
    expect(sectionOfPath('/')?.id).toBe('overview')
    expect(sectionOfPath('/numbers/aiShareOfVoice')?.id).toBe('overview')
    expect(sectionOfPath('/buyers')?.id).toBe('overview')
    expect(sectionOfPath('/terms/families/open-spiral')?.id).toBe('terms')
    expect(sectionOfPath('/scan/5b0c')?.id).toBe('competitors')
    expect(sectionOfPath('/competitors/twentebelt/history')?.id).toBe('competitors')
    expect(sectionOfPath('/ai-search/readiness')?.id).toBe('ai-search')
    expect(sectionOfPath('/recommendations/llms')?.id).toBe('recommendations')
    expect(sectionOfPath('/settings')).toBeUndefined()
    expect(section('recommendations').path).toBe('/recommendations')
  })

  it('writes addresses for a company and a belt family', () => {
    expect(companyPath('märtens')).toBe('/competitors/m%C3%A4rtens')
    expect(familySlug('tri_flex_turn_curve')).toBe('tri-flex-turn-curve')
    expect(familyPath('open_spiral')).toBe('/terms/families/open-spiral')
    expect(familyOfSlug('open-spiral')).toBe('open_spiral')
    expect(familyOfSlug('nothing')).toBeUndefined()
  })
})
