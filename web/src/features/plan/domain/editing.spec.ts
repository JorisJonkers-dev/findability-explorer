import { describe, expect, it } from 'vitest'
import type { PlanImprovement } from '../../../infrastructure/api/types.gen'
import { details } from '../../../testing/fixtures'
import { draftOf, editable, editedLine, editProblem, toEdit } from './editing'

const written = (over: Partial<PlanImprovement> = {}): PlanImprovement => ({
  ...details, id: 'g1', title: 'Use it', moves: 'm', measure: 'termsUsed', auto: true, done: false, generated: true, kind: 'term',
  where: 'https://www.tribelt.nl/spiraalbanden', ...over,
})

const problemOf = (status: number, detail?: string) => ({ type: 'about:blank', title: 'Problem', status, ...(detail ? { detail } : {}) })

describe('editing a recommendation a run wrote', () => {
  it('lists only the ones a run wrote, best value first as the server sends them', () => {
    expect(editable([written(), { ...written({ id: 'hand' }), generated: false }]).map((i) => i.id)).toEqual(['g1'])
  })

  it('fills the form from the recommendation, and turns it back into an edit', () => {
    const draft = draftOf(written())
    expect(draft).toEqual({
      title: 'Use it', where: 'https://www.tribelt.nl/spiraalbanden', why: 'It helps.', steps: 'Do the first thing.\nDo the second thing.', doneWhen: 'It is in place.',
      benefit: 3, effort: 2,
    })
    expect(draftOf(written({ where: undefined })).where).toBe('')
    expect(toEdit({ ...draft, steps: ' One.\n\n Two. \n', where: ' ' })).toEqual({
      title: 'Use it', why: 'It helps.', steps: ['One.', 'Two.'], doneWhen: 'It is in place.', benefit: 3, effort: 2,
    })
    expect(toEdit(draft).where).toBe('https://www.tribelt.nl/spiraalbanden')
  })

  it('says when one was edited', () => {
    expect(editedLine(written({ editedAt: '2026-11-01T09:00:00Z' }))).toBe('Edited on 1 November 2026. A new Recommendation run keeps it.')
    expect(editedLine(written())).toBe('Not edited yet. A new Recommendation run replaces it unless it is edited or done.')
  })

  it('says why an edit was refused', () => {
    expect(editProblem(problemOf(403))).toBe('Only administrators can edit recommendations.')
    expect(editProblem(problemOf(422, 'A title has 1 to 200 characters, and each score is from 1 to 5.'))).toBe(
      'A title has 1 to 200 characters, and each score is from 1 to 5.',
    )
    expect(editProblem(new Error('offline'))).toBe('Something went wrong. Try again.')
  })
})
