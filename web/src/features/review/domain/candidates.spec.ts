import { describe, expect, it } from 'vitest'
import type { TermCandidate } from '../../../infrastructure/api/types.gen'
import { term } from '../../../testing/fixtures'
import {
  candidatesIn,
  checkDraft,
  decisionLine,
  draftLanguages,
  draftOf,
  evidenceLine,
  parseStatus,
  reviewLine,
  reviewPoll,
  reviewProblem,
  shouldRetry,
  STATUSES,
  statusTab,
  suggestionLine,
  verdictLine,
} from './candidates'

describe('the language of a candidate', () => {
  const of = (id: string, languages: string[]): TermCandidate => candidate({ id, languages })

  it('is the only one it can join the map in, English also as American English', () => {
    expect(draftLanguages(of('a', ['de']))).toEqual(['de'])
    expect(draftLanguages(of('a', ['en']))).toEqual(['en', 'en-US'])
    expect(draftLanguages(of('a', []))).toEqual(['nl', 'de', 'en', 'en-US'])
  })

  it('filters the list', () => {
    const list = [of('a', ['nl']), of('b', ['en']), of('c', [])]
    expect(candidatesIn(list, 'en').map((c) => c.id)).toEqual(['b'])
    expect(candidatesIn(list, null).map((c) => c.id)).toEqual(['a', 'b', 'c'])
  })
})

const candidate = (overrides: Partial<TermCandidate> = {}): TermCandidate => ({
  id: '3a7bd3e2360a9f01',
  phrase: 'spiral freezer belt',
  status: 'pending',
  languages: ['pt', 'de'],
  companies: [{ competitorId: 'twentebelt', name: 'Twentebelt' }],
  scans: 1,
  bestScore: 4,
  maxPages: 1,
  suggestedTier: 'broad',
  lastSeenAt: '2026-10-23T09:30:00Z',
  ...overrides,
})

describe('the review tabs', () => {
  it('label each status with its count, and read the status from the address', () => {
    expect(STATUSES).toEqual(['pending', 'accepted', 'rejected'])
    expect(statusTab('pending', { pending: 4, accepted: 0, rejected: 1 })).toBe('To review (4)')
    expect(statusTab('rejected', undefined)).toBe('Rejected')
    expect(parseStatus('accepted')).toBe('accepted')
    expect(parseStatus('nonsense')).toBe('pending')
    expect(parseStatus(['accepted'])).toBe('pending')
  })
})

describe('a draft', () => {
  it('starts from the candidate: its language, none for one not told, and other belts without a family', () => {
    expect(draftOf(candidate())).toEqual({ term: 'spiral freezer belt', language: 'de', family: 'other', tier: 'broad', usedBy: 'Twentebelt' })
    expect(draftOf(candidate({ languages: ['und'], suggestedFamily: 'eyelink' }))).toMatchObject({ language: '', family: 'eyelink' })
  })

  it('is checked before it is sent', () => {
    const draft = draftOf(candidate())
    expect(checkDraft({ ...draft, term: '  ' })).toEqual({ kind: 'invalid', message: 'Enter the term as the map should show it.' })
    expect(checkDraft({ ...draft, term: 'x'.repeat(201) })).toEqual({ kind: 'invalid', message: 'A term has at most 200 characters.' })
    const many = Array.from({ length: 21 }, (_, i) => `C${String(i)}`).join(',')
    expect(checkDraft({ ...draft, usedBy: many })).toEqual({ kind: 'invalid', message: 'Name at most 20 companies.' })
    expect(checkDraft({ ...draft, language: '' })).toEqual({ kind: 'invalid', message: 'Choose the language the phrase is in.' })
    expect(checkDraft({ ...draft, usedBy: ' A, ,B,A ' })).toEqual({
      kind: 'ok',
      body: { term: 'spiral freezer belt', language: 'de', family: 'other', tier: 'broad', usedBy: ['A', 'B'] },
    })
  })
})

describe('the evidence', () => {
  it('says how much there is', () => {
    expect(evidenceLine(candidate())).toBe('1 company, 1 scan, in Portuguese and German')
    expect(evidenceLine(candidate({ languages: ['en'], scans: 2, companies: [] }))).toBe('0 companies, 2 scans, in English')
    expect(evidenceLine(candidate({ languages: [] }))).toBe('1 company, 1 scan')
    expect(evidenceLine(candidate({ languages: ['und'] }))).toBe('1 company, 1 scan, language not told')
    expect(evidenceLine(candidate({ languages: ['en', 'de', 'nl'] }))).toBe('1 company, 1 scan, in English, German and Dutch')
  })

  it('says what the classifier suggests', () => {
    expect(suggestionLine(candidate())).toBe('Broad')
    expect(suggestionLine(candidate({ suggestedFamily: 'eyelink', closestTerm: 'eyelink belt', suggestedTier: 'product' }))).toBe(
      'Product, Eyelink; closest map term: eyelink belt',
    )
  })

  it('says who decided, if anyone', () => {
    expect(decisionLine(candidate())).toBeNull()
    expect(decisionLine(candidate({ decision: { status: 'rejected', by: 'joris', at: '2026-10-23T10:00:00Z', term: term() } }))).toBe(
      'Rejected by joris, 23 Oct 2026, 10:00 UTC',
    )
  })

  it('says why an action failed', () => {
    expect(reviewProblem({ type: 'about:blank', title: 'Forbidden', status: 403 })).toBe('Only administrators can review terms.')
    expect(reviewProblem({ type: 'about:blank', title: 'Not Found', status: 404 })).toBe('This candidate is gone. Reload the page.')
    expect(reviewProblem({ type: 'about:blank', title: 'Conflict', status: 409, detail: 'Decided already.' })).toBe('Decided already.')
  })
})

describe('asking again', () => {
  it('never after a refusal, and at most three times after a failure', () => {
    expect(shouldRetry(0, { type: 'about:blank', title: 'Forbidden', status: 403 })).toBe(false)
    expect(shouldRetry(0, { type: 'about:blank', title: 'Bad Gateway', status: 502 })).toBe(true)
    expect(shouldRetry(3, new Error('offline'))).toBe(false)
  })
})

describe('the language model\'s review', () => {
  const verdict = { accept: true, sure: false, reason: 'Could go either way.', model: 'claude-haiku-5-5', at: '2026-10-27T09:00:00Z' }

  it('says how far it is', () => {
    expect(reviewLine({ running: true, waiting: 41 })).toBe('The language model is reviewing 41 phrases.')
    expect(reviewLine({ running: true, waiting: 1 })).toBe('The language model is reviewing 1 phrase.')
    expect(reviewLine({ running: true, waiting: 0 })).toBe('The language model is reviewing the phrases.')
    expect(reviewLine({ running: false, waiting: 3, problem: 'noModel' })).toBe('No Anthropic key is set up, so every phrase waits for you.')
    expect(reviewLine({ running: false, waiting: 3, problem: 'failed' })).toBe('The language model failed on its last try. It tries again within ten minutes.')
    expect(reviewLine({ running: false, waiting: 3 })).toBe('3 phrases wait for the language model.')
    expect(reviewLine({ running: false, waiting: 1 })).toBe('1 phrase waits for the language model.')
    expect(reviewLine({ running: false, waiting: 0 })).toBe('The language model has reviewed every phrase.')
  })

  it('keeps the lists fresh while it reviews', () => {
    expect(reviewPoll({ running: true })).toBe(3000)
    expect(reviewPoll({ running: false })).toBe(false)
    expect(reviewPoll(undefined)).toBe(false)
  })

  it('says what it made of a phrase', () => {
    expect(verdictLine(candidate())).toBeNull()
    expect(verdictLine(candidate({ verdict }))).toBe('The language model was unsure. Could go either way.')
    expect(verdictLine(candidate({ verdict: { ...verdict, reason: '' } }))).toBe('The language model gave no verdict on it.')
    const sure = { ...verdict, sure: true, reason: 'A belt buyers search for.' }
    expect(verdictLine(candidate({ status: 'accepted', verdict: sure }))).toBe('A belt buyers search for.')
    // Sure, but waiting: its decision was undone, or the map could not take its term.
    expect(verdictLine(candidate({ verdict: sure }))).toBe('The language model would accept it. A belt buyers search for.')
    expect(verdictLine(candidate({ verdict: { ...sure, accept: false, reason: 'Navigation.' } }))).toBe('The language model would reject it. Navigation.')
  })

  it('names itself as who decided', () => {
    expect(decisionLine(candidate({ decision: { status: 'accepted', by: 'language model', at: '2026-10-27T09:00:00Z' } }))).toBe(
      'Accepted by the language model, 27 Oct 2026, 09:00 UTC',
    )
  })

  it('starts a person\'s draft from the term it suggests', () => {
    const suggested = candidate({ verdict: { ...verdict, term: { term: 'spiral freezer belt', language: 'en', family: 'tri_flex_turn_curve', tier: 'application' } } })
    expect(draftOf(suggested)).toEqual({ term: 'spiral freezer belt', language: 'en', family: 'tri_flex_turn_curve', tier: 'application', usedBy: 'Twentebelt' })
    // The map names at most twenty companies.
    const many = Array.from({ length: 25 }, (_, i) => ({ competitorId: `c${String(i)}`, name: `C${String(i)}` }))
    expect(draftOf(candidate({ companies: many })).usedBy.split(', ')).toHaveLength(20)
  })
})
