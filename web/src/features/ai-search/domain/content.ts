import type { GoalRow, Recommendation } from '../../../shared/ui/types'

/**
 * Our mirror of tribelt.nl, with Tribelt's permission, and its access log of 30 September to
 * 23 October 2026 (export of 23 October). Day 0 of the timeline is 30 September, day 23 is
 * 23 October.
 */

export const TITLE = 'Google never came. AI crawlers read everything.'
export const SOURCE = ["Mirror of tribelt.nl, with Tribelt's permission", 'Log of 30 September – 23 October 2026'] as const

/** The last day of the log, counted from 30 September. */
export const LOG_DAYS = 23

export const TICKS: readonly { label: string; day: number }[] = [
  { label: '30 Sep', day: 0 },
  { label: '7 Oct', day: 7 },
  { label: '14 Oct', day: 14 },
  { label: '23 Oct', day: 23 },
]

/** The four figures above the timeline. A zero is a gap, shown in orange ink. */
export const STATS: readonly { value: string; label: string; gap: boolean }[] = [
  { value: '124', label: 'pages in Dutch, English and German', gap: false },
  { value: '248', label: 'pages ClaudeBot took on 10 October', gap: false },
  { value: '0', label: 'visits by Googlebot', gap: true },
  { value: '0', label: 'visits by Bingbot', gap: true },
]

/** The mirror itself, in a line. */
export const MIRROR_FACTS = '124 pages (45 Dutch, 44 English, 35 German), 292 words on average. 8,630 requests logged.'

export type CrawlerId = 'claudebot' | 'claude-user' | 'oai-searchbot' | 'gptbot' | 'googlebot' | 'bingbot'

export type CrawlerEvent = {
  /** Days after 30 September. */
  day: number
  /** A whole-site crawl, drawn large; anything else is a small dot. */
  big: boolean
}

export type Crawler = {
  id: CrawlerId
  name: string
  owner: string
  purpose: string
  /** Whether the visits were verified as the crawler's own; null when there were none to verify. */
  verified: boolean | null
  /** Content pages it fetched. */
  pages: number
  events: readonly CrawlerEvent[]
  /** What it did. */
  did: string
  /** What that means for Tribelt. */
  means: string
}

export const CRAWLERS: readonly Crawler[] = [
  {
    id: 'claudebot',
    name: 'ClaudeBot',
    owner: 'Anthropic, training',
    purpose: 'Model training',
    verified: true,
    pages: 248,
    events: [{ day: 10, big: true }],
    did: 'On 10 October it fetched all 124 pages as HTML and all 124 Markdown versions, plus 19 images and llms.txt. Then it read robots.txt and sitemap.xml about ten times a day, every day from 10 to 23 October: 142 reads of robots.txt.',
    means: 'AI crawlers take whatever format they are offered. But training is not search: being read is not being recommended.',
  },
  {
    id: 'claude-user',
    name: 'Claude-User',
    owner: 'Anthropic, for a user',
    purpose: 'User-requested fetch',
    verified: true,
    pages: 1,
    events: [{ day: 4, big: false }],
    did: 'On 4 October it read robots.txt and the home page once, when a user asked Claude about the site.',
    means: 'The full crawl came six days later. One case cannot prove that one caused the other.',
  },
  {
    id: 'oai-searchbot',
    name: 'OAI-SearchBot',
    owner: 'OpenAI, ChatGPT search',
    purpose: 'ChatGPT search',
    verified: true,
    pages: 0,
    events: [20, 21, 22, 23].map((day) => ({ day, big: false })),
    did: 'Read robots.txt five times between 20 and 23 October. No content yet.',
    means: 'ChatGPT search is starting to look; the 30 October export will show more.',
  },
  {
    id: 'gptbot',
    name: 'GPTBot',
    owner: 'OpenAI, training',
    purpose: 'Unverified',
    verified: false,
    pages: 0,
    events: [{ day: 1, big: false }],
    did: 'One request for llms.txt on 1 October, from an unverified user agent one second after one of our own test requests. It may not be GPTBot at all.',
    means: 'llms.txt drew one verified request in three weeks, from ClaudeBot during its full crawl.',
  },
  {
    id: 'googlebot',
    name: 'Googlebot',
    owner: 'Google, search index',
    purpose: 'Search index',
    verified: null,
    pages: 0,
    events: [],
    did: 'No visits at all in more than three weeks, without inbound links or a Search Console submission.',
    means: 'AI Overviews, AI Mode and Gemini draw on the Google index. A page Google has not indexed cannot appear in any of them.',
  },
  {
    id: 'bingbot',
    name: 'Bingbot',
    owner: 'Microsoft, search index',
    purpose: 'Search index',
    verified: null,
    pages: 0,
    events: [],
    did: 'No visits at all.',
    means: 'Bing feeds Copilot and is one input of ChatGPT search. Submitting to Bing Webmaster Tools is part of phase 1.',
  },
]

/** The crawler the timeline opens on: the one that never came. */
export const DEFAULT_CRAWLER: CrawlerId = 'googlebot'

/** The poster's AI search tile: content pages each crawler fetched. */
export const CRAWLER_PAGES: readonly { name: string; pages: number }[] = CRAWLERS.map((c) => ({ name: c.name, pages: c.pages }))

/** The prompt-test pilot: did the assistant name Tribelt? */
export const ASSISTANTS = ['Gemini', 'ChatGPT', 'ChatGPT thinking'] as const

export type PromptCell = { named: boolean; note: string }

export const PROMPTS: readonly { search: string; cells: readonly [PromptCell, PromptCell, PromptCell] }[] = [
  {
    search: 'Supplier search',
    cells: [
      { named: false, note: '' },
      { named: false, note: '' },
      { named: true, note: 'In both searches' },
    ],
  },
  {
    search: 'Technical and quotation',
    cells: [
      { named: false, note: '' },
      { named: false, note: 'FAQ cited once' },
      { named: false, note: '' },
    ],
  },
]

/** Who the assistants named instead of Tribelt. */
export const NAMED_INSTEAD = ['Twentebelt', 'ESFO', 'Ashworth', 'Unibelt', 'Wire Belt', 'Märtens'] as const

/** Whether buyers in the Netherlands and Germany can be reached on a surface: no, not yet, or yes. */
export type AdState = 'no' | 'notYet' | 'yes'

export const AD_SURFACES: readonly { surface: string; state: AdState; text: string }[] = [
  { surface: 'Inside Gemini', state: 'no', text: 'No ad product' },
  { surface: 'Inside AI Overviews', state: 'no', text: 'English only, 12 countries' },
  { surface: 'Around AI Overviews', state: 'yes', text: 'Available' },
  { surface: 'AI Mode', state: 'notYet', text: 'Not yet in Dutch or German' },
  { surface: 'ChatGPT', state: 'yes', text: 'Available; small test in phase 2' },
]

export const ADS_CHECKED = 'Checked 22 October 2026. Rechecked every quarter.'

export const RECOMMENDATIONS: readonly Recommendation[] = [
  {
    mark: '4',
    lead: 'Get indexed, in every language.',
    text: 'Search Console and Bing Webmaster Tools, a sitemap per language, crawlers allowed, structured data.',
  },
  { mark: '3', lead: 'Be the most citable source.', text: 'Exact figures and standards early on every belt page.' },
  { mark: '→', lead: 'Be ready, not early,', text: 'for ads in AI: an account eligible the day they open in NL and DE.' },
]

export const GOAL = {
  goal: 'G4',
  text: 'Test prompts in which Gemini names Tribelt',
  rows: [
    { label: 'Today', value: '0%', fraction: 0, fill: 'today' },
    { label: 'Target', value: '25%', fraction: 0.25, fill: 'target' },
  ] satisfies GoalRow[],
} as const
