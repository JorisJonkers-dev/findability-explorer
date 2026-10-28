import { describe, expect, it } from 'vitest'
import type { Product } from '../../../infrastructure/api/types.gen'
import { product } from '../../../testing/fixtures'
import {
  aiShare,
  familyHeadline,
  familyLede,
  familyTitle,
  SELLERS_SHOWN,
  shareText,
  termLanguages,
  termsOnlyOthers,
  tribeltCalls,
  visibleSellers,
} from './products'

const sellers = (n: number): Product['sellers'] => Array.from({ length: n }, (_, i) => ({ competitorId: `c${String(i)}`, name: `C${String(i)}`, names: ['x'] }))

describe('a belt family', () => {
  it('has a share of answers naming Tribelt, once a run asked about it', () => {
    expect(aiShare(product())).toBe(0.3)
    expect(aiShare(product({ answers: 0, named: 0 }))).toBeUndefined()
    expect(shareText(0.3)).toBe('30%')
    expect(shareText(undefined)).toBe('–')
  })

  it('has a title in the list of families', () => {
    expect(familyTitle('eyelink')).toBe('Eyelink')
    expect(familyTitle('tri_flex_turn_curve')).toBe('Tri-Flex, the curve-going belt')
  })

  it('says what Tribelt calls it, if anything', () => {
    expect(tribeltCalls(product())).toBe('Tribelt calls it Draadogenbanden')
    expect(tribeltCalls(product({ tribeltNames: ['A', 'B', 'C'] }))).toBe('Tribelt calls it A, B, and C')
    expect(tribeltCalls(product({ tribeltNames: [] }))).toBe('Tribelt names none for it yet')
    expect(familyLede(product())).toBe('Tribelt calls it Draadogenbanden.')
    expect(familyLede(product({ tribeltNames: [] }))).toBe('Tribelt names none for it yet.')
  })

  it('heads its page with how many competitors sell it', () => {
    expect(familyHeadline(product())).toBe('One competitor sells eyelink belts.')
    expect(familyHeadline(product({ family: 'open_spiral', sellers: sellers(3) }))).toBe('Three competitors sell open spiral belts.')
    expect(familyHeadline(product({ sellers: sellers(12) }))).toBe('12 competitors sell eyelink belts.')
    expect(familyHeadline(product({ family: 'other', sellers: [] }))).toBe('No competitor sells other belts.')
  })

  it('counts the terms only competitors use', () => {
    expect(termsOnlyOthers(product())).toBe(3)
    expect(termsOnlyOthers(product({ termsUsed: 9 }))).toBe(0)
  })

  it('shows the first sellers, or all of them when asked', () => {
    const many = sellers(SELLERS_SHOWN + 2)
    expect(visibleSellers(many, false)).toHaveLength(SELLERS_SHOWN)
    expect(visibleSellers(many, true)).toEqual(many)
    expect(visibleSellers(many, true)).not.toBe(many)
  })

  it('names the languages of its terms, Dutch, German, and English first', () => {
    expect(termLanguages(product())).toBe('Terms in Dutch, German, English, and French.')
    expect(termLanguages(product({ terms: [] }))).toBe('')
    const t = (language: string) => ({ term: language, language, tier: 'product' as const, usedBy: 1, usedByTribelt: false })
    expect(termLanguages(product({ terms: [t('sv'), t('fr'), t('en')] }))).toBe('Terms in English, French, and Swedish.')
  })
})
