import { describe, expect, it } from 'vitest'
import { overviewFixture, planFixture } from '../../../testing/numbersFixtures'
import { lowerFirst, sectionCards } from './cards'

describe('lowerFirst', () => {
  it('starts a title in lower case', () => {
    expect(lowerFirst('Submit tribelt.nl')).toBe('submit tribelt.nl')
    expect(lowerFirst('')).toBe('')
  })
})

describe('sectionCards', () => {
  it('shows a figure from each section, with the way in', () => {
    const cards = sectionCards(overviewFixture(), planFixture())
    expect(cards.map((c) => [c.id, c.number, c.label, c.big, c.cta, c.path])).toEqual([
      ['ai-search', '02', 'AI SEARCH', '24%', 'Open AI search', '/ai-search'],
      ['competitors', '03', 'COMPETITORS', '24', 'Open Competitors', '/competitors'],
      ['terms', '04', 'SEARCH TERMS', '30', 'Open Search terms', '/terms'],
      ['recommendations', '05', 'RECOMMENDATIONS', '4', 'Open Recommendations', '/recommendations'],
    ])
    expect(cards[2]?.text).toBe('of the 387 terms buyers search with appear on tribelt.nl.')
    expect(cards[3]?.text).toBe('changes to tribelt.nl, ranked. The first is to submit tribelt.nl to Google and Bing.')
  })

  it('says so when no Prompt run has finished or the plan is empty', () => {
    const overview = overviewFixture()
    overview.aiSearch.companies = []
    const plan = planFixture()
    plan.improvements = []
    const cards = sectionCards(overview, plan)
    expect(cards[0]?.big).toBe('–')
    expect(cards[0]?.text).toContain('No Prompt run has finished yet.')
    expect(cards[3]).toMatchObject({ big: '0', text: 'changes to tribelt.nl, ranked.' })
  })
})
