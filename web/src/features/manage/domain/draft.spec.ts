import { describe, expect, it } from 'vitest'
import type { PromptSetScope, PublishedPromptSet } from '../../../infrastructure/api/types.gen'
import {
  addQuestion,
  costSentence,
  diffOf,
  draftOf,
  editQuestion,
  familiesWithout,
  familyRows,
  fillScope,
  firstGap,
  intentFor,
  isUnchanged,
  makeScope,
  publishLine,
  removeQuestion,
  requestOf,
  restoreOf,
  scopeID,
  scopeNameOf,
  scopesOf,
} from './draft'

const SCOPES: PromptSetScope[] = [
  { id: 'twente', name: 'Twente (Dutch)', language: 'nl', region: 'Twente' },
  { id: 'english', name: 'English, no region', language: 'en' },
]

const PUBLISHED: PublishedPromptSet = {
  version: 2,
  publishedAt: '2026-10-27T06:35:33Z',
  note: 'Four scopes.',
  questions: [
    { id: 'eyelink-application-english', family: 'eyelink', intent: 'application', scope: 'english', scopeName: 'English, no region', language: 'en', text: 'Which belt?' },
    { id: 'eyelink-application-twente', family: 'eyelink', intent: 'application', scope: 'twente', scopeName: 'Twente (Dutch)', language: 'nl', text: 'Welke band?', english: 'Which belt, in Twente?' },
    { id: 'other-problem-twente', family: 'other', intent: 'problem', scope: 'twente', scopeName: 'Twente (Dutch)', language: 'nl', text: 'Waarom?' },
  ],
}

describe('a draft from a published version', () => {
  it('starts as the published set, unchanged, and a restore remembers its version', () => {
    const draft = draftOf(PUBLISHED)
    expect(draft.questions.map((q) => q.key)).toEqual(['eyelink-application-english', 'eyelink-application-twente', 'other-problem-twente'])
    expect(draft.questions[1]?.english).toBe('Which belt, in Twente?')
    expect(draft.questions[0]).not.toHaveProperty('english')
    expect(isUnchanged(diffOf(PUBLISHED, draft))).toBe(true)
    expect(restoreOf({ ...PUBLISHED, version: 1 }).restoredFrom).toBe(1)
  })
})

describe('editing the draft', () => {
  it('adds a question with the intent its family has most, and fresh keys', () => {
    let draft = addQuestion(draftOf(PUBLISHED), 'other', 'english', '  Why does it break?  ', 'Why does it break?')
    draft = addQuestion(draft, 'closed_spiral', 'english', 'Which closed belt?')
    expect(draft.questions.slice(3)).toEqual([
      { key: 'new-1', family: 'other', intent: 'problem', scope: 'english', text: 'Why does it break?', english: 'Why does it break?' },
      { key: 'new-2', family: 'closed_spiral', intent: 'application', scope: 'english', text: 'Which closed belt?' },
    ])
    expect(intentFor(draft, 'eyelink')).toBe('application')
    // The other family asks two problem questions and one application question.
    const mixed = { ...draft, questions: [...draft.questions, { key: 'x', family: 'other' as const, intent: 'application' as const, scope: 'twente', text: 'Welke?' }] }
    expect(intentFor(mixed, 'other')).toBe('problem')
  })

  it('removes a question, and a new scope with its last question', () => {
    const made = makeScope(draftOf(PUBLISHED), SCOPES, 'fr', 'Lyon')
    const draft = addQuestion(made.draft, 'eyelink', 'fr-lyon', 'Quelle bande ?')
    expect(draft.scopes).toHaveLength(1)
    const without = removeQuestion(draft, 'new-1')
    expect(without.questions).toHaveLength(3)
    expect(without.scopes).toEqual([])
    expect(removeQuestion(draft, 'other-problem-twente').scopes).toHaveLength(1)
  })

  it('edits a question’s text, which drops its English', () => {
    const draft = editQuestion(draftOf(PUBLISHED), 'eyelink-application-twente', 'Welke band nu?')
    expect(draft.questions[1]).toEqual({ key: 'eyelink-application-twente', id: 'eyelink-application-twente', family: 'eyelink', intent: 'application', scope: 'twente', text: 'Welke band nu?' })
    expect(draft.questions[0]?.text).toBe('Which belt?')
    expect(diffOf(PUBLISHED, draft).changed).toHaveLength(1)
  })
})

describe('scopes', () => {
  it('lists the known scopes, then the new ones, and names a scope', () => {
    const made = makeScope(draftOf(PUBLISHED), SCOPES, 'fr', '')
    expect(scopesOf(SCOPES, made.draft).map((s) => s.id)).toEqual(['twente', 'english', 'fr'])
    expect(scopeNameOf(SCOPES, 'twente')).toBe('Twente (Dutch)')
    expect(scopeNameOf(SCOPES, 'mars')).toBe('mars')
  })

  it('makes an id from the language and region', () => {
    expect(scopeID('fr', 'Lyon')).toBe('fr-lyon')
    expect(scopeID('fr', ' Île-de-France ')).toBe('fr-ile-de-france')
    expect(scopeID('pl', '')).toBe('pl')
    expect(scopeID('de', 'x'.repeat(50))).toHaveLength(40)
  })

  it('makes a new scope once, and finds an existing one by id or by name', () => {
    const made = makeScope(draftOf(PUBLISHED), SCOPES, 'fr', ' Lyon ')
    expect(made).toMatchObject({ made: true, scope: { id: 'fr-lyon', name: 'French, Lyon', language: 'fr', region: 'Lyon' } })
    expect(makeScope(made.draft, SCOPES, 'fr', 'Lyon').made).toBe(false)
    expect(makeScope(draftOf(PUBLISHED), SCOPES, 'en', '')).toMatchObject({ made: false, scope: { id: 'english' } })
    expect(makeScope(draftOf(PUBLISHED), SCOPES, 'fr', '').scope).toEqual({ id: 'fr', name: 'French, no region', language: 'fr' })
  })

  it('starts the add row on the first scope a family lacks', () => {
    const draft = draftOf(PUBLISHED)
    expect(firstGap(draft, SCOPES, 'other')).toBe('english')
    expect(firstGap(draft, SCOPES, 'eyelink')).toBe('twente')
    expect(firstGap(draft, [], 'eyelink')).toBe('')
  })

  it('fills a new scope with a question for every family without one, from its English', () => {
    const draft = addQuestion(makeScope(draftOf(PUBLISHED), SCOPES, 'fr', 'Lyon').draft, 'eyelink', 'fr-lyon', 'Quelle bande ?')
    expect(familiesWithout(draft, 'fr-lyon')).toEqual(['open_spiral', 'closed_spiral', 'tri_flex_turn_curve', 'other'])
    const filled = fillScope(draft, 'fr-lyon')
    expect(filled.added).toBe(4)
    expect(filled.draft.questions.filter((q) => q.scope === 'fr-lyon').map((q) => [q.family, q.text])).toEqual([
      ['eyelink', 'Quelle bande ?'],
      ['open_spiral', 'Write the question as a buyer would ask it.'],
      ['closed_spiral', 'Write the question as a buyer would ask it.'],
      ['tri_flex_turn_curve', 'Write the question as a buyer would ask it.'],
      ['other', 'Waarom?'],
    ])
    const english = fillScope(draftOf(PUBLISHED), 'english').draft.questions.find((q) => q.family === 'other' && q.scope === 'english')
    expect(english?.text).toBe('Waarom?')
    const translated = fillScope(draftOf({ ...PUBLISHED, questions: PUBLISHED.questions.slice(1, 2) }), 'english')
    expect(translated.draft.questions.find((q) => q.key === 'new-1')).toMatchObject({ family: 'eyelink', text: 'Which belt, in Twente?' })
  })
})

describe('the rows by family', () => {
  it('lists every family, its questions in the scopes’ order, an unknown scope last', () => {
    const draft = addQuestion(draftOf(PUBLISHED), 'eyelink', 'mars', 'Hmm?')
    const rows = familyRows(draft, SCOPES)
    expect(rows.map((r) => r.label)).toEqual(['Eyelink', 'Open spiral', 'Closed spiral', 'Curve-going (Tri-Flex)', 'Other metal belts'])
    expect(rows[0]?.questions.map((q) => q.scope)).toEqual(['twente', 'english', 'mars'])
    expect(rows[1]?.questions).toEqual([])
  })
})

describe('what publishing changes', () => {
  it('counts what the draft adds, changes, and removes, and says so with the new scopes', () => {
    const made = makeScope(draftOf(PUBLISHED), SCOPES, 'fr', 'Lyon')
    let draft = addQuestion(made.draft, 'eyelink', 'fr-lyon', 'Quelle bande ?')
    draft = addQuestion(draft, 'other', 'fr-lyon', 'Pourquoi ?')
    draft = addQuestion(draft, 'other', 'english', 'Why?')
    draft = removeQuestion(editQuestion(draft, 'eyelink-application-english', 'Which belt now?'), 'other-problem-twente')
    const d = diffOf(PUBLISHED, draft)
    expect([d.added.length, d.changed.length, d.removed, d.scopes.length]).toEqual([3, 1, 1, 1])
    expect(isUnchanged(d)).toBe(false)
    const scopes = scopesOf(SCOPES, draft)
    expect(publishLine(d, draft, scopes)).toBe('Adds 3 questions (2 in French, 1 in English), changes 1, and removes 1. Adds 1 scope (French, Lyon).')
    const restored = { ...draftOf(PUBLISHED), restoredFrom: 1 }
    expect(publishLine(diffOf(PUBLISHED, restored), restored, SCOPES)).toBe('Restores version 1. Adds no questions, changes none, and removes none.')
    const one = addQuestion(draftOf(PUBLISHED), 'eyelink', 'mars', 'Hmm?')
    expect(publishLine(diffOf(PUBLISHED, one), one, [])).toContain('Adds 1 question (1 in ')
    const two = { ...one, scopes: SCOPES }
    expect(publishLine(diffOf(PUBLISHED, two), two, SCOPES)).toContain('Adds 2 scopes (Twente (Dutch); English, no region).')
  })

  it('sees a new order as a change', () => {
    const draft = draftOf(PUBLISHED)
    const reordered = { ...draft, questions: [draft.questions[1]!, draft.questions[0]!, draft.questions[2]!] }
    expect(diffOf(PUBLISHED, reordered).reordered).toBe(true)
    expect(isUnchanged(diffOf(PUBLISHED, reordered))).toBe(false)
  })

  it('says what a run and a year cost more or less', () => {
    expect(costSentence(1.2, 20, 24, 'monthly')).toBe('A run will cost about $0.24 more, or $2.88 more a year on a monthly schedule.')
    expect(costSentence(1.2, 20, 10, 'weekly')).toBe('A run will cost about $0.60 less, or $31.20 less a year on a weekly schedule.')
    expect(costSentence(1.2, 20, 20, 'monthly')).toBe('A run will cost about the same as now.')
    expect(costSentence(0, 0, 5, 'monthly')).toBe('A run will cost about the same as now.')
  })
})

describe('the request', () => {
  it('sends the questions without their keys, and only the new scopes a question asks in', () => {
    const made = makeScope(makeScope(draftOf(PUBLISHED), SCOPES, 'fr', 'Lyon').draft, SCOPES, 'pl', '')
    const draft = { ...addQuestion(made.draft, 'eyelink', 'fr-lyon', ' Quelle bande ? '), restoredFrom: 1 }
    const request = requestOf(draft)
    expect(request.questions[1]).toEqual({ id: 'eyelink-application-twente', family: 'eyelink', intent: 'application', scope: 'twente', text: 'Welke band?', english: 'Which belt, in Twente?' })
    expect(request.questions[3]).toEqual({ family: 'eyelink', intent: 'application', scope: 'fr-lyon', text: 'Quelle bande ?' })
    expect(request.scopes?.map((s) => s.id)).toEqual(['fr-lyon'])
    expect(request.restoredFrom).toBe(1)
    expect(requestOf(draftOf(PUBLISHED))).not.toHaveProperty('restoredFrom')
  })
})
