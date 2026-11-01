import { describe, expect, it } from 'vitest'
import type { PromptSetQuestion } from '../../../infrastructure/api/types.gen'
import { questionGroups, versionLine } from './questions'

function q(id: string, family: PromptSetQuestion['family'], overrides: Partial<PromptSetQuestion> = {}): PromptSetQuestion {
  return { id, family, intent: 'product', scope: 'twente', scopeName: 'Dutch, Twente', language: 'nl', text: `Vraag ${id}?`, ...overrides }
}

describe('questionGroups', () => {
  it('groups the questions by belt family, in the families’ order, leaving out an empty family', () => {
    const groups = questionGroups([q('a', 'other'), q('b', 'eyelink', { english: 'Question b?' }), q('c', 'other', { scopeName: 'English' })])
    expect(groups).toEqual([
      { family: 'eyelink', label: 'Eyelink', questions: [{ id: 'b', scope: 'Dutch, Twente', text: 'Vraag b?', english: 'Question b?' }] },
      {
        family: 'other',
        label: 'Other metal belts',
        questions: [
          { id: 'a', scope: 'Dutch, Twente', text: 'Vraag a?' },
          { id: 'c', scope: 'English', text: 'Vraag c?' },
        ],
      },
    ])
  })
})

describe('versionLine', () => {
  it('says when a version was published and how many questions it asks', () => {
    expect(versionLine({ publishedAt: '2026-10-27T15:00:00Z', questions: 1 })).toBe('Published on 27 October 2026, 1 question')
    expect(versionLine({ publishedAt: '2026-10-27T15:00:00Z', questions: 200 })).toBe('Published on 27 October 2026, 200 questions')
  })
})
