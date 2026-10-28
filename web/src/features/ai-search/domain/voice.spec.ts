import { describe, expect, it } from 'vitest'
import type { AiSearchSummary, Plan, PromptCell } from '../../../infrastructure/api/types.gen'
import { promptResults, promptRun } from '../../../testing/fixtures'
import { figures, newestCells, providerTallies, readinessCard, readinessScore, slice, voiceHeadline, voiceLede, voiceRows, type VoiceRow } from './voice'

const followed: AiSearchSummary['followed'] = [
  { competitorId: 'tribelt', name: 'Tribelt B.V.', role: 'tribelt' },
  { competitorId: 'ashworth', name: 'Ashworth', role: 'competitor' },
  { competitorId: 'lesjofors', name: 'Lesjöfors', role: 'group' },
  { competitorId: 'tribelt-mirror', name: 'Mirror of tribelt.nl', role: 'mirror' },
  { competitorId: 'twentebelt', name: 'Twentebelt', role: 'competitor' },
]

const summary = (change: Partial<AiSearchSummary> = {}): AiSearchSummary => ({
  goal: 0.4,
  followed,
  trend: [{ runStartedAt: '2026-11-02T06:00:00Z', shares: [0.24, 0.57, 0.1, 0.24, 0.24] }],
  crawlers: [],
  keepingOut: [],
  ranking: [
    { competitorId: 'martens', name: 'Märtens', role: 'competitor', score: 5 },
    { competitorId: 'tribelt', name: 'Tribelt B.V.', role: 'tribelt', score: 3.5 },
  ],
  sites: 41,
  questions: 20,
  providers: 5,
  readinessChecks: 8,
  checks: [
    { check: 'crawlers', scored: true, passing: 30, tribelt: true },
    { check: 'sitemap', scored: true, passing: 30, tribelt: false },
    { check: 'product', scored: false, passing: 3, tribelt: true },
  ],
  ...change,
})

const plan = (watched: Plan['watched']): Plan => ({ catalogueVersion: 1, improvements: [], watched, eyeOn: [], readinessChecks: 8 })

const cells = promptRun().cells

describe('who the answers name', () => {
  it('lists the followed companies by share, keeping the followed order on a tie', () => {
    const rows = voiceRows(summary())
    expect(rows.map((r) => [r.rank, r.name, r.share])).toEqual([
      [1, 'Ashworth', 0.57],
      [2, 'Tribelt B.V.', 0.24],
      [3, 'Mirror of tribelt.nl', 0.24],
      [4, 'Twentebelt', 0.24],
      [5, 'Lesjöfors', 0.1],
    ])
    expect(rows.map((r) => r.to)).toEqual(['/competitors/ashworth', undefined, undefined, '/competitors/twentebelt', undefined])
    expect(rows.map((r) => r.note)).toEqual(['', 'Target 40%', 'The mirror of tribelt.nl, not a competitor', '', 'A group company of Tribelt, not a competitor'])
    expect(rows.map((r) => r.tribelt)).toEqual([false, true, false, false, false])
  })

  it('counts a company missing from the newest run as named in none', () => {
    const rows = voiceRows(summary({ trend: [{ runStartedAt: '2026-11-02T06:00:00Z', shares: [0.5] }] }))
    expect(rows.map((r) => r.share)).toEqual([0.5, 0, 0, 0, 0])
  })

  it('has no rows before a run', () => {
    expect(voiceRows(summary({ trend: [] }))).toEqual([])
  })
})

describe('the headline', () => {
  const row = (rank: number, tribelt = false): VoiceRow => ({ id: `c${String(rank)}`, name: `C${String(rank)}`, role: tribelt ? 'tribelt' : 'competitor', share: 0, rank, to: undefined, note: '', tribelt })
  const list = (n: number, tribeltAt?: number) => Array.from({ length: n }, (_, i) => row(i + 1, i + 1 === tribeltAt))

  it('says where Tribelt ranks among the companies the answers name', () => {
    expect(voiceHeadline(list(5), 8)).toBe('AI assistants have not been asked about Tribelt yet.')
    expect(voiceHeadline(list(5, 1), 8)).toBe('Tribelt is the company AI answers name most.')
    expect(voiceHeadline(list(5, 3), 8)).toBe('Tribelt ranks third among the companies AI answers name.')
  })

  it('says Tribelt is not among the most named when it is last on a full list', () => {
    expect(voiceHeadline(list(5, 5), 5)).toBe('Tribelt is not among the four companies AI answers name most.')
  })

  it('does not name a place it has no word for', () => {
    expect(voiceHeadline(list(12, 11), 20)).toBe('Tribelt is not among the 11 companies AI answers name most.')
  })
})

describe('the lede', () => {
  it('names how many answers the newest run holds, and which provider names Tribelt most', () => {
    expect(voiceLede(cells)).toBe("In the newest Prompt run, AI assistants from two providers gave 12 answers to buyers' questions. OpenAI names Tribelt most often, in 3 of its 6 answers.")
  })

  it('says so before any run', () => {
    expect(voiceLede([])).toBe('No Prompt run has finished yet. The first one starts on the schedule.')
  })

  it('says when no provider names Tribelt', () => {
    const none = cells.map((c): PromptCell => ({ ...c, named: 0 }))
    expect(voiceLede(none)).toContain('No provider named Tribelt in any answer.')
  })

  it('names providers that tie, and a single provider', () => {
    const tie = cells.map((c): PromptCell => ({ ...c, named: 1 }))
    expect(voiceLede(tie)).toContain('OpenAI and Google name Tribelt most often, in 2 answers each.')
    expect(voiceLede(cells.slice(0, 2))).toContain('from one provider gave 6 answers')
  })

  it('counts the answers read per provider', () => {
    expect(providerTallies(cells)).toEqual([
      { provider: 'openai', answers: 6, named: 3, cited: 1 },
      { provider: 'google', answers: 6, named: 0, cited: 0 },
    ])
  })
})

describe('the key figures', () => {
  const goals: Plan['watched'] = [
    { target: 'aiShareOfVoice', goal: 0.5 },
    { target: 'citationShare', goal: 0.5 },
  ]

  it('write the share of answers naming Tribelt, the web answers citing it, and the checks it passes', () => {
    expect(figures(summary(), cells, plan(goals))).toEqual([
      { to: '/numbers/aiShareOfVoice', big: '25%', text: 'of AI answers name Tribelt', note: 'Target 50%.' },
      { to: '/numbers/citationShare', big: '17%', text: 'of web answers cite tribelt.nl', note: 'Target 50%.' },
      { to: '/ai-search/readiness', big: '3½ of 8', text: 'AI readiness checks passed', note: 'See how AI readiness is checked.' },
    ])
  })

  it('fall back on the summary’s goal, and write a dash before a run or a scan', () => {
    const [named, cited, ready] = figures(summary({ ranking: [], checks: [] }), [], undefined)
    expect([named?.big, named?.note, cited?.big, cited?.note, ready?.big]).toEqual(['–', 'Target 40%.', '–', '', '–'])
  })

  it('score readiness by the ranking, else by the checks Tribelt passes', () => {
    expect(readinessScore(summary())).toBe(3.5)
    expect(readinessScore(summary({ ranking: [] }))).toBe(1)
    expect(readinessScore(summary({ ranking: [], checks: [{ check: 'crawlers', scored: true, passing: 1 }] }))).toBeUndefined()
    expect(readinessScore(summary({ ranking: [], checks: [{ check: 'product', scored: false, passing: 1, tribelt: true }] }))).toBeUndefined()
  })
})

describe('the donut', () => {
  it('shows all answers, with the year’s target', () => {
    expect(slice('all', cells, summary(), plan([{ target: 'aiShareOfVoice', goal: 0.5 }]))).toEqual({
      label: 'All answers',
      named: 3,
      total: 12,
      namedText: '3 answers name Tribelt',
      restText: '9 answers do not',
      goal: 'The target for the year is 50% of all answers.',
      centre: '25%',
      sub: '3 of 12',
      aria: 'All answers: 3 of 12 answers, 25%, name Tribelt.',
    })
  })

  it('shows the answers about Twente, and singular and empty counts', () => {
    const twente = slice('twente', cells, summary(), plan([{ target: 'twenteShareOfVoice', goal: 0.6 }]))
    expect([twente.label, twente.named, twente.total, twente.goal]).toEqual(['Answers about Twente', 3, 6, 'The target for the year is 60% of the answers about Twente.'])
    const one = slice('all', [{ ...cells[0], answers: 1, named: 1 } as PromptCell], summary(), undefined)
    expect([one.namedText, one.restText, one.goal]).toEqual(['1 answer names Tribelt', '0 answers do not', 'The target for the year is 40% of all answers.'])
    const rest = slice('all', [{ ...cells[0], answers: 2, named: 1 } as PromptCell], summary(), undefined)
    expect(rest.restText).toBe('1 answer does not')
    const empty = slice('twente', [], summary(), undefined)
    expect([empty.centre, empty.goal]).toEqual(['–', ''])
  })
})

describe('the card to AI readiness', () => {
  it('says how Tribelt fares on the checks', () => {
    expect(readinessCard(summary({ checks: [] }))).toBe('See how each site is set up for AI.')
    expect(readinessCard(summary())).toBe('Tribelt fails one of the eight checks. See which ones.')
    expect(readinessCard(summary({ checks: [{ check: 'crawlers', scored: true, passing: 3, tribelt: true }] }))).toBe('Tribelt passes all eight checks. See how the other sites compare.')
  })
})

describe('the newest run', () => {
  it('gives its cells, or none before a run', () => {
    expect(newestCells(promptResults())).toEqual(cells)
    expect(newestCells(undefined)).toEqual([])
  })
})
