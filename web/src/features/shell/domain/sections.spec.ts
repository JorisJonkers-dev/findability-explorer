import { describe, expect, it } from 'vitest'
import { companyPath, familyOfSlug, familyPath, familySlug, NAV_SECTIONS, section, SECTIONS, sectionOfPath } from './sections'

describe('sections', () => {
  it('numbers the seven sections in the top bar order', () => {
    expect(SECTIONS.map((s) => `${s.number} ${s.label}`)).toEqual([
      '01 Overview',
      '02 AI responses',
      '03 AI readiness',
      '04 Search terms',
      '05 Competitors',
      '06 Recommendations',
      '07 Help',
    ])
    expect(NAV_SECTIONS.map((s) => s.path)).toEqual(['/', '/ai-responses', '/ai-readiness', '/terms', '/competitors', '/recommendations', '/help'])
  })

  it('finds the section a path is in', () => {
    expect(sectionOfPath('/')?.id).toBe('overview')
    expect(sectionOfPath('/goals/aiShareOfVoice')?.id).toBe('overview')
    expect(sectionOfPath('/ai-responses/all')?.id).toBe('ai-responses')
    expect(sectionOfPath('/ai-readiness')?.id).toBe('ai-readiness')
    expect(sectionOfPath('/terms/families/open-spiral')?.id).toBe('terms')
    expect(sectionOfPath('/competitors/twentebelt/history')?.id).toBe('competitors')
    expect(sectionOfPath('/recommendations/llms')?.id).toBe('recommendations')
    expect(sectionOfPath('/help/prompt-test')?.id).toBe('help')
    expect(sectionOfPath('/manage/keys')).toBeUndefined()
    expect(sectionOfPath('/ai-search')).toBeUndefined()
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
