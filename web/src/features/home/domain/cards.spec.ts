import { describe, expect, it } from 'vitest'
import { overviewFixture, planFixture } from '../../../testing/numbersFixtures'
import { sectionTiles } from './cards'

describe('the numbered tiles', () => {
  it('give each part of the app its key figure and the way in', () => {
    const tiles = sectionTiles(overviewFixture(), planFixture())
    expect(tiles.map((t) => [t.number, t.label, t.big, t.text, t.path])).toEqual([
      ['02', 'AI responses', '24%', 'of AI responses name Tribelt.', '/ai-responses'],
      ['03', 'AI readiness', '3 of 8', 'AI readiness checks passed by tribelt.nl.', '/ai-readiness'],
      ['04', 'Search terms', '30', 'of the 387 search terms appear on tribelt.nl.', '/terms'],
      ['05', 'Competitors', '24', 'competitors make metal belts buyers could choose instead.', '/competitors'],
      ['06', 'Recommendations', '3', 'changes to make on tribelt.nl, ranked by value.', '/recommendations'],
    ])
  })

  it('says so before a Prompt run or a readiness check', () => {
    const overview = overviewFixture()
    overview.aiSearch.companies = []
    const plan = planFixture()
    plan.watched = []
    const tiles = sectionTiles(overview, plan)
    expect(tiles[0]).toMatchObject({ big: '–', text: 'No Prompt run has finished yet.' })
    expect(tiles[1]).toMatchObject({ big: '–', text: 'tribelt.nl has not been checked for AI readiness yet.' })
  })
})
