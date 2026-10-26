import { describe, expect, it } from 'vitest'
import {
  FAMILIES,
  familyLabel,
  isFamily,
  isLanguage,
  isTier,
  LANGUAGES,
  languageLabel,
  PRODUCT_FAMILIES,
  scopeLabel,
  TIERS,
  tierLabel,
  tribeltProduct,
} from './vocabulary'

describe('the vocabulary', () => {
  it("lists every value in the contract's order", () => {
    expect(LANGUAGES).toEqual(['nl', 'de', 'en', 'en-US'])
    expect(FAMILIES).toEqual(['eyelink', 'open_spiral', 'closed_spiral', 'tri_flex_turn_curve', 'other', 'all', 'spec_all'])
    expect(TIERS).toEqual(['spec', 'product', 'application', 'problem', 'broad'])
  })

  it('recognises its own values only', () => {
    expect(isLanguage('en-US')).toBe(true)
    expect(isLanguage('toString')).toBe(false)
    expect(isFamily('open_spiral')).toBe(true)
    expect(isFamily('spiral')).toBe(false)
    expect(isTier('broad')).toBe(true)
    expect(isTier('narrow')).toBe(false)
  })

  it('names each value for people', () => {
    expect(languageLabel('de')).toBe('German')
    expect(languageLabel('en-US')).toBe('American English')
    expect(languageLabel('pt')).toBe('Portuguese')
    expect(languageLabel('en-GB')).toBe('British English')
    expect(languageLabel('not a code')).toBe('not a code')
    expect(familyLabel('tri_flex_turn_curve')).toBe('Tri-Flex')
    expect(familyLabel('spec_all')).toBe('Specification')
    expect(tierLabel('spec')).toBe('Specification')
  })

  it('says which belts a term is about, and what Tribelt makes for a family', () => {
    expect(scopeLabel('tri_flex_turn_curve')).toBe('Tri-Flex')
    expect(scopeLabel('other')).toBe('other belts')
    expect(scopeLabel('spec_all')).toBe('all belts')
    expect(tribeltProduct('open_spiral')).toBe('Open spiral belt')
    expect(tribeltProduct('all')).toBe('Any belt')
    expect(PRODUCT_FAMILIES.map(familyLabel)).toEqual(['Eyelink', 'Open spiral', 'Closed spiral', 'Tri-Flex', 'Other belts'])
  })
})
