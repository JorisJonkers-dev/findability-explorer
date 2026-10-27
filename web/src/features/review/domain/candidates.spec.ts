import { describe, expect, it } from 'vitest'
import type { TermCandidate } from '../../../infrastructure/api/types.gen'
import { term } from '../../../testing/fixtures'
import { checkDraft, decisionLine, draftOf, evidenceLine, parseStatus, reviewLine, reviewPoll, reviewProblem, shouldRetry, STATUSES, statusTab, suggestionLine, verdictLine } from './candidates'

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
  it('starts from the candidate: its first known language, else English, and other belts without a family', () => {
    expect(draftOf(candidate())).toEqual({ term: 'spiral freezer belt', language: 'de', family: 'other', tier: 'broad', usedBy: 'Twentebelt' })
    expect(draftOf(candidate({ languages: ['und'], suggestedFamily: 'eyelink' }))).toMatchObject({ language: 'en', family: 'eyelink' })
  })

  it('is checked before it is sent', () => {
    const draft = draftOf(candidate())
    expect(checkDraft({ ...draft, term: '  ' })).toEqual({ kind: 'invalid', message: 'Enter the term as the map should show it.' })
    expect(checkDraft({ ...draft, term: 'x'.repeat(201) })).toEqual({ kind: 'invalid', message: 'A term has at most 200 characters.' })
    const many = Array.from({ length: 21 }, (_, i) => `C${String(i)}`).join(',')
    expect(checkDraft({ ...draft, usedBy: many })).toEqual({ kind: 'invalid', message: 'Name at most 20 companies.' })
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
    expect(reviewPoll({ running: true, waiting: 3 })).toBe(3000)
    expect(reviewPoll({ running: false, waiting: 3 })).toBe(false)
    expect(reviewPoll(undefined)).toBe(false)
  })

  it('says what it made of a phrase', () => {
    expect(verdictLine(candidate())).toBeNull()
    expect(verdictLine(candidate({ verdict }))).toBe('The language model was unsure. Could go either way.')
    expect(verdictLine(candidate({ verdict: { ...verdict, sure: true, reason: 'A belt buyers search for.' } }))).toBe('A belt buyers search for.')
  })

  it('names itself as who decided', () => {
    expect(decisionLine(candidate({ decision: { status: 'accepted', by: 'language model', at: '2026-10-27T09:00:00Z' } }))).toBe(
      'Accepted by the language model, 27 Oct 2026, 09:00 UTC',
    )
  })

  it('starts a person\'s draft from the term it suggests', () => {
    const suggested = candidate({ verdict: { ...verdict, term: { term: 'spiral freezer belt', language: 'en', family: 'tri_flex_turn_curve', tier: 'application' } } })
    expect(draftOf(suggested)).toEqual({ term: 'spiral freezer belt', language: 'en', family: 'tri_flex_turn_curve', tier: 'application', usedBy: 'Twentebelt' })
  })
})
