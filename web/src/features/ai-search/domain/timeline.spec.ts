import { describe, expect, it } from 'vitest'
import { CRAWLER_PAGES, CRAWLERS, DEFAULT_CRAWLER, STATS, TICKS } from './content'
import { crawler, dayPosition, ownerLine, rowLabel } from './timeline'

describe('the crawler timeline', () => {
  it('places a day between 30 September and 23 October', () => {
    expect(dayPosition(0)).toBe(0)
    expect(dayPosition(23)).toBe(1)
    expect(dayPosition(10)).toBeCloseTo(10 / 23)
    expect(dayPosition(-2)).toBe(0)
    expect(dayPosition(30)).toBe(1)
    expect(TICKS.map((t) => dayPosition(t.day))).toEqual([0, 7 / 23, 14 / 23, 1])
  })

  it('opens on Googlebot, which never came', () => {
    expect(crawler(DEFAULT_CRAWLER).events).toEqual([])
    expect(rowLabel(crawler('googlebot'))).toBe('Googlebot: no visits.')
  })

  it('says when each crawler came', () => {
    expect(rowLabel(crawler('claudebot'))).toBe('ClaudeBot: 1 day with visits, on 10 October.')
    expect(rowLabel(crawler('oai-searchbot'))).toBe('OAI-SearchBot: 4 days with visits, from 20 October to 23 October.')
    expect(rowLabel(crawler('gptbot'))).toBe('GPTBot: 1 day with visits, unverified, on 1 October.')
    expect(rowLabel({ name: 'Early', verified: true, events: [{ day: 0, big: false }] })).toBe('Early: 1 day with visits, on 30 September.')
  })

  it('marks a visit that could not be verified', () => {
    expect(ownerLine(crawler('gptbot'))).toBe('OpenAI, training · unverified')
    expect(ownerLine(crawler('claudebot'))).toBe('Anthropic, training')
  })

  it("gives the poster each crawler's pages", () => {
    expect(CRAWLER_PAGES).toEqual([
      { name: 'ClaudeBot', pages: 248 },
      { name: 'Claude-User', pages: 1 },
      { name: 'OAI-SearchBot', pages: 0 },
      { name: 'GPTBot', pages: 0 },
      { name: 'Googlebot', pages: 0 },
      { name: 'Bingbot', pages: 0 },
    ])
    expect(CRAWLERS).toHaveLength(6)
    expect(STATS.filter((s) => s.gap).map((s) => s.label)).toEqual(['visits by Googlebot', 'visits by Bingbot'])
  })
})
