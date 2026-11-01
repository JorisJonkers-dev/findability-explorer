import { describe, expect, it } from 'vitest'
import { competitor, promptAnswer, term } from '../../../testing/fixtures'
import type { Measure } from './compare'
import { aheadLines, competitorPurpose, detailDataLine, modelShares, theirTerms } from './detail'

describe('one competitor', () => {
  it('says where it is on the web and what it sells', () => {
    expect(competitorPurpose(competitor('Twentebelt', { beltNames: [{ name: 'A', family: 'eyelink' }, { name: 'B', family: 'tri_flex_turn_curve' }, { name: 'C', family: 'all' }] }))).toBe(
      'twentebelt.com, on the competitor list. It sells eyelink and Tri-Flex belts.',
    )
    expect(competitorPurpose(competitor('Ashworth', { host: undefined, website: 'https://ashworth.com', beltNames: [] }))).toBe(
      'https://ashworth.com, on the competitor list. No belt names are recorded for it yet.',
    )
  })

  it('says where its figures come from', () => {
    expect(detailDataLine('Twentebelt', '2026-10-26T06:00:00Z', '2026-10-20T09:00:00Z')).toBe('From the Prompt run of 26 October 2026 and the scan of Twentebelt on 20 October 2026.')
    expect(detailDataLine('Twentebelt', undefined, undefined)).toBe('No Prompt run has finished yet, and Twentebelt has not been scanned yet.')
  })

  it('lists the measures on which it is ahead of Tribelt', () => {
    const m = (id: Measure['id'], them: number | undefined, us: number | undefined): Measure => ({
      id,
      label: `Measure ${id}`,
      them,
      us,
      themText: String(them),
      usText: String(us),
      scale: 1,
      text: '',
      path: '/',
      pathLabel: '',
    })
    expect(aheadLines([m('answers', 0.5, 0.2), m('ready', 3, 5), m('terms', undefined, 1), m('sitemap', 2, undefined)])).toEqual([
      'Measure answers: 0.5, against Tribelt’s 0.2.',
      'Measure sitemap: 2, against Tribelt’s undefined.',
    ])
  })

  it('lists the terms it uses that tribelt.nl does not, the most used first', () => {
    const terms = [
      term({ term: 'aa', usedBy: ['Twentebelt'] }),
      term({ term: 'a', usedBy: ['Twentebelt'] }),
      term({ term: 'b', usedBy: ['Twentebelt', 'Ashworth'] }),
      term({ term: 'c', usedBy: ['Twentebelt'], tribeltUses: true }),
      term({ term: 'd', usedBy: ['Ashworth'] }),
      { ...term({ term: 'e' }), makers: undefined },
    ]
    expect(theirTerms(terms, 'twentebelt', 1)).toEqual({ shown: [terms[2]], total: 3 })
    expect(theirTerms(terms, 'twentebelt').shown.map((t) => t.term)).toEqual(['b', 'a', 'aa'])
  })

  it('bars how often each model names it in the newest run', () => {
    const named = (model: string, ids: string[]) => promptAnswer({ model }, ids.map((id) => `${id}:${id}`))
    const answers = [
      named('openai/gpt', ['twentebelt']),
      named('openai/gpt', []),
      named('google/gemma', ['ashworth']),
      promptAnswer({ model: 'google/gemma', extraction: undefined }),
      promptAnswer({ model: 'google/gemma', failure: 'empty' }),
      promptAnswer({ model: 'google/gemma' }, [], { failed: true }),
    ]
    expect(modelShares(answers, 'twentebelt')).toEqual([
      { label: 'google/gemma', value: 0, display: '0%', tone: 'blue-grey', highlight: false },
      { label: 'openai/gpt', value: 0.5, display: '50%', tone: 'blue-grey', highlight: false },
    ])
  })
})
