import { describe, expect, it } from 'vitest'
import type { Product } from '../../../infrastructure/api/types.gen'
import { product } from '../../../testing/fixtures'
import { aiShare, languagesOf, productHeadline, shareText, standoutText, topTerm } from './products'

describe('a belt family', () => {
  it('has a share of answers naming Tribelt, once a run asked about it', () => {
    expect(aiShare(product())).toBe(0.3)
    expect(aiShare(product({ answers: 0, named: 0 }))).toBeUndefined()
    expect(shareText(0.3)).toBe('30%')
    expect(shareText(undefined)).toBe('–')
  })

  it('has a most used term and languages, Dutch, German and English first', () => {
    expect(topTerm(product())).toBe('draadogenband')
    expect(topTerm(product({ terms: [] }))).toBeUndefined()
    expect(languagesOf(product())).toEqual(['nl', 'de', 'en', 'fr'])
    expect(languagesOf(product({ terms: [{ term: 'b', language: 'sv', tier: 'product', usedBy: 1, usedByTribelt: false }, { term: 'a', language: 'fr', tier: 'product', usedBy: 1, usedByTribelt: false }] }))).toEqual(['fr', 'sv'])
  })

  it('heads its page with who sells it and where buyers search', () => {
    expect(productHeadline(product())).toBe('Eyelink. One competitor sells it, and buyers search for it in four languages.')
    expect(productHeadline(product({ sellers: [...product().sellers, ...product().sellers], terms: [product().terms[0] as Product['terms'][number]] }))).toBe(
      'Eyelink. Two competitors sell it, and buyers search for it in one language.',
    )
    expect(productHeadline(product({ terms: [] }))).toBe('Eyelink. One competitor sells it.')
  })
})

describe('what stands out', () => {
  const products = [product(), product({ family: 'other', termsUsed: 0, sellers: [] })]

  it('is written from the data', () => {
    expect(standoutText({ finding: 'crowded', family: 'eyelink' }, products)).toEqual({
      lead: 'Most competitors sell Eyelink.',
      rest: '1 competitor sells it. Tribelt uses 1 of its 4 search terms.',
    })
    expect(standoutText({ finding: 'fewestTerms', family: 'other' }, products)).toEqual({
      lead: 'Tribelt uses the fewest search terms for Other belts.',
      rest: 'Tribelt uses none of its 4 search terms.',
    })
    expect(standoutText({ finding: 'bestInAI', family: 'eyelink' }, products)).toEqual({ lead: 'AI answers name Tribelt most for Eyelink.', rest: '30% of the answers about it name Tribelt.' })
    expect(standoutText({ finding: 'bestInAI', family: 'closed_spiral' }, products).rest).toBe('– of the answers about it name Tribelt.')
    expect(standoutText({ finding: 'crowded', family: 'closed_spiral' }, products).rest).toBe('0 competitors sell it.')
  })
})
