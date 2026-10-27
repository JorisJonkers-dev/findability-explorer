import { describe, expect, it } from 'vitest'
import type { CompetitorCandidate } from '../../../infrastructure/api/types.gen'
import { candidateDecisionLine, candidateDraft, candidateEvidence, checkCandidateDraft } from './candidates'

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

  it('starts an addition from its name and the first website the answers gave, and checks it', () => {
    expect(candidateDraft(candidate())).toEqual({ name: 'Ashworth', website: 'https://www.ashworth.com/' })
    expect(candidateDraft(candidate({ addresses: [] }))).toEqual({ name: 'Ashworth', website: '' })
    expect(checkCandidateDraft({ name: ' Ashworth ', website: ' https://www.ashworth.com/ ' })).toEqual({ kind: 'ok', name: 'Ashworth', website: 'https://www.ashworth.com/' })
    expect(checkCandidateDraft({ name: ' ', website: 'https://x.com' })).toEqual({ kind: 'invalid', message: 'Enter the company’s name.' })
    expect(checkCandidateDraft({ name: 'X', website: '' })).toEqual({ kind: 'invalid', message: 'Enter its website, starting with https://.' })
    expect(checkCandidateDraft({ name: 'X', website: 'www.x.com' })).toEqual({ kind: 'invalid', message: 'Enter its website, starting with https://.' })
  })
})
