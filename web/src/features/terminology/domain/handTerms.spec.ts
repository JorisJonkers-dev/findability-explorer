import { describe, expect, it } from 'vitest'
import { term } from '../../../testing/fixtures'
import { draftError, FAMILY_CHOICES, handLine, LANGUAGE_CHOICES, ORIGIN_WORDS, TIER_CHOICES, termName } from './handTerms'

describe('adding and removing search terms by hand', () => {
  it('offers Dutch, German, and English, every belt family, and every kind of buyer intent', () => {
    expect(LANGUAGE_CHOICES).toEqual([
      { value: 'nl', label: 'Dutch' },
      { value: 'de', label: 'German' },
      { value: 'en', label: 'English' },
    ])
    expect(FAMILY_CHOICES.map((f) => f.value)).toEqual(['eyelink', 'open_spiral', 'closed_spiral', 'tri_flex_turn_curve', 'other', 'all', 'spec_all'])
    expect(FAMILY_CHOICES[0]?.label).toBe('Eyelink')
    expect(TIER_CHOICES.map((t) => t.value)).toEqual(['spec', 'product', 'application', 'problem', 'broad'])
    expect(TIER_CHOICES[2]?.help).toBe('The buyer describes what the belt is for, such as a bakery oven.')
  })

  it('checks the term before it is sent', () => {
    expect(draftError('  ')).toBe('Write the term.')
    expect(draftError('x'.repeat(201))).toBe('A term has at most 200 characters.')
    expect(draftError(' oven belt ')).toBe('')
  })

  it('names a term with its language, and says who added or removed it and when', () => {
    expect(termName(term({ term: 'oven belt', language: 'en' }))).toBe('oven belt (English)')
    expect(handLine('Added', { by: 'Joris Jonkers', at: '2026-11-01T09:00:00Z' })).toBe('Added by Joris Jonkers on 1 November 2026.')
    expect(ORIGIN_WORDS).toEqual({ research: 'The desk research', review: 'Accepted from a scan', hand: 'Added by hand' })
  })
})
