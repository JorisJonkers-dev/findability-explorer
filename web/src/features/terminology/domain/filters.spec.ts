import { describe, expect, it } from 'vitest'
import { chartLanguage, companyName, MAX_TEXT_LENGTH, NO_VIEW, parseView, toggled, toViewQuery } from './filters'

describe('parseView', () => {
  it('reads every filter from the query string', () => {
    expect(
      parseView({ q: 'spiral ', language: 'de', tier: 'product', user: 'Twentebelt', group: 'technical', gap: '1' }),
    ).toEqual({ q: 'spiral ', language: 'de', tier: 'product', user: 'Twentebelt', group: 'technical', gap: true })
  })

  it('has no filters for an empty query string', () => {
    expect(parseView({})).toEqual(NO_VIEW)
  })

  it('drops values it does not know', () => {
    expect(parseView({ language: 'xx', tier: 'narrow', group: 'both', gap: 'yes', q: null, user: '  ' })).toEqual(NO_VIEW)
    // Only Dutch, German and English are languages to filter by.
    expect(parseView({ language: 'fr' }).language).toBeNull()
    expect(parseView({ language: 'en-US' }).language).toBe('en')
  })

  it('reads American English as English, and a used-by entry as its company', () => {
    expect(parseView({ language: 'en-US', user: 'Twentebelt (TwenteFlex)' })).toEqual({ ...NO_VIEW, language: 'en', user: 'Twentebelt' })
  })

  it('takes the first of a repeated value, and cuts text to the longest it keeps', () => {
    expect(parseView({ q: ['a', 'b'], user: [null], language: ['nl'] })).toEqual({ ...NO_VIEW, q: 'a', language: 'nl' })
    expect(parseView({ q: 'x'.repeat(150) }).q).toHaveLength(MAX_TEXT_LENGTH)
  })
})

describe('toViewQuery', () => {
  it('keeps only the filters in use, as typed', () => {
    expect(toViewQuery(NO_VIEW)).toEqual({})
    expect(toViewQuery({ q: 'spiral ', language: 'en', tier: 'spec', user: 'Ashworth', group: 'marketing', gap: true })).toEqual({
      q: 'spiral ',
      language: 'en',
      tier: 'spec',
      user: 'Ashworth',
      group: 'marketing',
      gap: '1',
    })
  })

  it('round-trips through parseView', () => {
    const view = { ...NO_VIEW, q: 'belt', tier: 'problem' as const, gap: true }
    expect(parseView(toViewQuery(view))).toEqual(view)
  })
})

describe('toggled', () => {
  it('sets a filter, keeping the others, and unsets it with the same value', () => {
    const on = toggled({ ...NO_VIEW, q: 'belt' }, 'tier', 'spec')
    expect(on).toEqual({ ...NO_VIEW, q: 'belt', tier: 'spec' })
    expect(toggled(on, 'tier', 'spec')).toEqual({ ...NO_VIEW, q: 'belt' })
    expect(toggled(on, 'user', 'Tribelt')).toEqual({ ...on, user: 'Tribelt' })
    expect(toggled(on, 'group', 'technical').group).toBe('technical')
  })
})

describe('chartLanguage', () => {
  it('counts American English as English', () => {
    expect(chartLanguage('en-US')).toBe('en')
    expect(chartLanguage('de')).toBe('de')
  })
})

describe('companyName', () => {
  it('drops a trailing bracketed product line or division', () => {
    expect(companyName('Twentebelt (TwenteFlex)')).toBe('Twentebelt')
    expect(companyName('Campbelt (CMG) ')).toBe('Campbelt')
  })

  it('keeps a name without one, and a name that is only brackets', () => {
    expect(companyName(' Märtens ')).toBe('Märtens')
    expect(companyName('Wire Belt (UK) Ltd')).toBe('Wire Belt (UK) Ltd')
    expect(companyName('(unknown)')).toBe('(unknown)')
  })
})
