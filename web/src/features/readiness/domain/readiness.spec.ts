import { describe, expect, it } from 'vitest'
import { bots, readiness } from '../../../testing/fixtures'
import { changeLabel, markedLines, policySummary } from './readiness'

describe('the crawler policy', () => {
  it('sums up the crawler policy in a line', () => {
    expect(policySummary(undefined)).toBe('Not checked yet. The next scan will check it.')
    expect(policySummary(readiness({ robots: 'unreachable', bots: [] }))).toBe('Its robots.txt could not be read, so its crawler policy is unknown.')
    expect(policySummary(readiness())).toBe('Lets every search and AI crawler in.')
    expect(policySummary(readiness({ bots: bots({ CCBot: 'partly' }) }))).toBe('Lets every search and AI crawler in, but closes some pages to 1 crawler.')
    expect(policySummary(readiness({ bots: bots({ GPTBot: 'blocked' }) }))).toBe('Blocks GPTBot.')
    expect(policySummary(readiness({ bots: bots({ GPTBot: 'blocked', ClaudeBot: 'blocked' }) }))).toBe('Blocks GPTBot and ClaudeBot.')
    expect(policySummary(readiness({ bots: bots({ GPTBot: 'blocked', ClaudeBot: 'blocked', CCBot: 'blocked', Bingbot: 'partly' }) }))).toBe(
      'Blocks GPTBot, ClaudeBot, and CCBot, and closes some pages to 1 more crawler.',
    )
    expect(policySummary(readiness({ bots: bots({ GPTBot: 'blocked', Bingbot: 'partly', CCBot: 'partly' }) }))).toBe('Blocks GPTBot, and closes some pages to 2 more crawlers.')
    expect(policySummary(readiness({ bots: bots({ Bingbot: 'partly', CCBot: 'partly' }) }))).toBe('Lets every search and AI crawler in, but closes some pages to 2 crawlers.')
  })
})

describe('changes and raw files', () => {
  it('reads a readiness change', () => {
    expect(changeLabel({ field: 'bot:GPTBot', from: 'allowed', to: 'blocked' })).toBe('GPTBot: allowed → blocked')
    expect(changeLabel({ field: 'llmsTxt', from: 'no', to: 'yes' })).toBe('llms.txt: no → yes')
    expect(changeLabel({ field: 'somethingNew', from: 'a', to: 'b' })).toBe('somethingNew: a → b')
  })

  it('marks the lines of a raw file that name a search or AI crawler', () => {
    expect(markedLines('')).toEqual([])
    expect(markedLines('User-agent: GPTBot\nDisallow: /\nuser-agent: meta-externalagent')).toEqual([
      { text: 'User-agent: GPTBot', ai: true },
      { text: 'Disallow: /', ai: false },
      { text: 'user-agent: meta-externalagent', ai: true },
    ])
  })
})
