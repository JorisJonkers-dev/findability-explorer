import { describe, expect, it } from 'vitest'
import type { CompetitorCandidate } from '../../../infrastructure/api/types.gen'
import { CHECK_FILTERS, type CheckFilter, MODEL_DECIDER, candidateDecisionLine, decidedByHaiku, candidateDraft, candidateEvidence, checkCandidateDraft, checkLine, decidedLine, idsOf, keptBy, matchesSearch, otherNames, reviewLine } from './candidates'

const candidate = (overrides: Partial<CompetitorCandidate> = {}): CompetitorCandidate => ({
  id: '3a7bd3e2360a9f01',
  name: 'Ashworth',
  status: 'pending',
  answers: 37,
  providers: ['openai', 'x-ai', 'google'],
  models: ['openai/gpt-oss-20b'],
  addresses: ['https://www.ashworth.com/products', 'ashworth.com'],
  lastSeenAt: '2026-11-01T06:10:00Z',
  ...overrides,
})

const check = (over: Partial<NonNullable<CompetitorCandidate['check']>> = {}) => ({
  competitor: true, sure: true, reason: 'Makes wire mesh belts.', model: 'anthropic: claude-haiku-5-5', at: '2026-10-27T09:00:00Z', ...over,
})

describe('a company named in AI answers', () => {
  it('says how often and by whom it was named', () => {
    expect(candidateEvidence(candidate())).toBe('Named in 37 answers by OpenAI, xAI and Google, last on 1 Nov 2026, 06:10 UTC')
    expect(candidateEvidence(candidate({ answers: 1, providers: ['deepseek'] }))).toBe('Named in 1 answer by DeepSeek, last on 1 Nov 2026, 06:10 UTC')
  })

  it('says who decided and when', () => {
    expect(candidateDecisionLine(candidate())).toBeNull()
    expect(candidateDecisionLine(candidate({ decision: { status: 'pending', by: 'joris', at: '2026-11-02T09:00:00Z' } }))).toBeNull()
    expect(candidateDecisionLine(candidate({ status: 'added', decision: { status: 'added', competitorId: 'ashworth', by: 'joris', at: '2026-11-02T09:00:00Z' } }))).toBe(
      'Added by joris, 2 Nov 2026, 09:00 UTC',
    )
    expect(candidateDecisionLine(candidate({ status: 'dismissed', decision: { status: 'dismissed', by: 'joris', at: '2026-11-02T09:00:00Z' } }))).toBe(
      'Dismissed by joris, 2 Nov 2026, 09:00 UTC',
    )
  })

  it('says why Claude Haiku added or dismissed it', () => {
    const byHaiku = { by: MODEL_DECIDER, at: '2026-11-02T09:00:00Z' }
    const added = candidate({ status: 'added', decision: { status: 'added', competitorId: 'ashworth', ...byHaiku }, check: check({ reason: 'Makes eyelink belts for bakery ovens.' }) })
    expect(candidateDecisionLine(added)).toBe('Added by Claude Haiku: Makes eyelink belts for bakery ovens.')
    expect(checkLine(added)).toBeNull()
    expect(decidedByHaiku(added)).toBe(true)
    const dismissed = candidate({ status: 'dismissed', decision: { status: 'dismissed', ...byHaiku } })
    expect(candidateDecisionLine(dismissed)).toBe('Dismissed by Claude Haiku, 2 Nov 2026, 09:00 UTC')
    expect(decidedByHaiku(candidate({ decision: { status: 'pending', ...byHaiku } }))).toBe(false)
    expect(decidedByHaiku(candidate())).toBe(false)
  })

  it('starts an addition from its name and the first website the answers gave, and checks it', () => {
    expect(candidateDraft(candidate())).toEqual({ name: 'Ashworth', website: 'https://www.ashworth.com/' })
    expect(candidateDraft(candidate({ addresses: [] }))).toEqual({ name: 'Ashworth', website: '' })
    expect(checkCandidateDraft({ name: ' Ashworth ', website: ' https://www.ashworth.com/ ' })).toEqual({ kind: 'ok', name: 'Ashworth', website: 'https://www.ashworth.com/' })
    expect(checkCandidateDraft({ name: ' ', website: 'https://x.com' })).toEqual({ kind: 'invalid', message: 'Enter the company’s name.' })
    expect(checkCandidateDraft({ name: 'X', website: '' })).toEqual({ kind: 'invalid', message: 'Enter its website, starting with https://.' })
    expect(checkCandidateDraft({ name: 'X', website: 'www.x.com' })).toEqual({ kind: 'invalid', message: 'Enter its website, starting with https://.' })
  })
})

describe('finding and acting on many companies', () => {
  const metiz = candidate({
    name: 'Taiwan Metiz Alliance',
    sites: ['metiz.tw'],
    website: 'https://metiz.tw/',
    names: [
      { id: '3a7bd3e2360a9f01', name: 'Taiwan Metiz Alliance', answers: 4, joined: 'lead' },
      { id: '3a7bd3e2360a9f02', name: 'Metiz', answers: 1, joined: 'site' },
    ],
  })

  it('searches every name and website, and lists the other names', () => {
    expect(matchesSearch(metiz, '  ')).toBe(true)
    expect(matchesSearch(metiz, 'metiz')).toBe(true)
    expect(matchesSearch(metiz, 'METIZ.TW')).toBe(true)
    expect(matchesSearch(metiz, 'ashworth')).toBe(false)
    expect(matchesSearch(candidate(), 'ashw')).toBe(true)
    expect(otherNames(metiz).map((n) => n.name)).toEqual(['Metiz'])
    expect(otherNames(candidate())).toEqual([])
    expect(candidateDraft(metiz)).toEqual({ name: 'Taiwan Metiz Alliance', website: 'https://metiz.tw/' })
    expect(idsOf([metiz, candidate()])).toEqual(['3a7bd3e2360a9f01', '3a7bd3e2360a9f01'])
  })

  it('filters on what Haiku made of each, and says it', () => {
    const yes = candidate({ check: check() })
    const no = candidate({ check: check({ competitor: false, reason: 'Makes plastic belts.' }) })
    const unsure = candidate({ check: check({ sure: false, reason: 'Could be either.' }) })
    const silent = candidate({ check: check({ sure: false, reason: '' }) })
    const none = candidate()
    const haiku = candidate({ status: 'added', decision: { status: 'added', competitorId: 'ashworth', by: MODEL_DECIDER, at: '2026-11-02T09:00:00Z' }, check: check() })
    const kept = (f: CheckFilter) => [yes, no, unsure, none, haiku].filter((c) => keptBy(c, f)).length
    expect(CHECK_FILTERS.map((f) => [f.value, kept(f.value)])).toEqual([['all', 5], ['competitor', 2], ['not', 1], ['unsure', 1], ['unchecked', 1], ['haiku', 1]])
    expect(checkLine(none)).toBeNull()
    expect(checkLine(yes)).toBe('Haiku says a competitor. Makes wire mesh belts.')
    expect(checkLine(no)).toBe('Haiku says not a competitor. Makes plastic belts.')
    expect(checkLine(unsure)).toBe('Haiku is unsure. Could be either.')
    expect(checkLine(silent)).toBe('Haiku gave no answer for it.')
  })

  it('says how far Haiku is, and what acting on several did', () => {
    expect(reviewLine({ running: true }, 3)).toBe('Haiku is merging the duplicates and checking the companies.')
    expect(reviewLine({ running: false, problem: 'noKey' }, 3)).toBe('No Anthropic key is set up, so Haiku cannot check the companies.')
    expect(reviewLine({ running: false, problem: 'failed' }, 3)).toBe('Haiku failed on its last try. Try again.')
    expect(reviewLine({ running: false }, 1)).toBe('Haiku has not checked 1 company yet.')
    expect(reviewLine({ running: false }, 12)).toBe('Haiku has not checked 12 companies yet.')
    expect(reviewLine({ running: false }, 0)).toBe('Haiku has checked every company.')
    const done = { id: '3a7bd3e2360a9f01', outcome: 'done' as const }
    const skipped = { id: '3a7bd3e2360a9f02', outcome: 'skipped' as const, reason: 'No website.' }
    expect(decidedLine('add', [done, done, skipped])).toBe('Added 2 companies. Skipped 1.')
    expect(decidedLine('dismiss', [done])).toBe('Dismissed 1 company.')
  })
})
