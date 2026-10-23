import { describe, expect, it } from 'vitest'
import { section, SECTIONS, sectionOfPath } from './sections'

describe('the sections', () => {
  it('numbers the five in the poster order', () => {
    expect(SECTIONS.map((s) => `${s.number} ${s.label}`)).toEqual(['01 Buyers', '02 AI search', '03 Competitors', '04 Search terms', '05 The plan'])
  })

  it('finds the section a path shows, a scan in the competitors section', () => {
    expect(sectionOfPath('/terms')?.id).toBe('terms')
    expect(sectionOfPath('/scan/5b0c')?.id).toBe('competitors')
    expect(sectionOfPath('/')).toBeUndefined()
    expect(sectionOfPath('/settings')).toBeUndefined()
    expect(section('plan').path).toBe('/plan')
  })
})
