import { describe, expect, it } from 'vitest'
import type { WatchedNumber } from '../../../infrastructure/api/types.gen'
import { checkGoals, goalFields, goalLabel, tickedOff, toTickOff } from './marking'

const item = (id: string, auto: boolean, done: boolean) => ({ id, auto, done })

describe('ticking off yourself', () => {
  it('lists the open ones no scan can check, and the ones marked as done', () => {
    const items = [item('a', false, false), item('b', true, false), item('c', false, true), item('d', true, true)]
    expect(toTickOff(items).map((i) => i.id)).toEqual(['a'])
    expect(tickedOff(items).map((i) => i.id)).toEqual(['c'])
  })
})

describe('the goals', () => {
  const watched: WatchedNumber[] = [
    { target: 'termsUsed', goal: 40, max: 663 },
    { target: 'readinessChecks', goal: 6 },
    { target: 'aiShareOfVoice', goal: 0.25, max: 1 },
    { target: 'citationShare', goal: 0.125, max: 1 },
    { target: 'twenteShareOfVoice', goal: 0.4, max: 1 },
  ]

  it('shows the five in order, shares in percent', () => {
    expect(goalFields(watched, 8).map((f) => [f.label, f.value, f.max, f.step])).toEqual([
      ['AI responses that name Tribelt (percent)', '25', 100, 0.1],
      ['AI responses about the Twente region that name Tribelt (percent)', '40', 100, 0.1],
      ['Web responses that cite tribelt.nl (percent)', '12.5', 100, 0.1],
      ['AI readiness checks passed (0 to 8)', '6', 8, 0.5],
      ['Search terms tribelt.nl uses', '40', 663, 1],
    ])
    expect(goalFields([], 8)).toEqual([])
    expect(goalLabel('readinessChecks', 7)).toBe('AI readiness checks passed (0 to 7)')
  })

  it('turns the typed values back into fractions for the shares', () => {
    const fields = goalFields(watched, 8).map((f) => (f.target === 'aiShareOfVoice' ? { ...f, value: '30,5' } : f))
    expect(checkGoals(fields)).toEqual({
      kind: 'ok',
      values: [
        { target: 'aiShareOfVoice', value: 0.305 },
        { target: 'twenteShareOfVoice', value: 0.4 },
        { target: 'citationShare', value: 0.125 },
        { target: 'readinessChecks', value: 6 },
        { target: 'termsUsed', value: 40 },
      ],
    })
  })

  it('refuses a value that is empty, not a number, or out of range', () => {
    const fields = goalFields(watched, 8)
    const withValue = (target: string, value: string) => fields.map((f) => (f.target === target ? { ...f, value } : f))
    expect(checkGoals(withValue('aiShareOfVoice', '120'))).toEqual({ kind: 'invalid', target: 'aiShareOfVoice', message: 'Type a number from 0 to 100 for AI responses that name Tribelt.' })
    expect(checkGoals(withValue('readinessChecks', ''))).toMatchObject({ kind: 'invalid', message: 'Type a number from 0 to 8 for AI readiness checks passed.' })
    expect(checkGoals(withValue('termsUsed', 'many'))).toMatchObject({ kind: 'invalid', target: 'termsUsed' })
    const open = goalFields([{ target: 'termsUsed', goal: 40 }], 8)
    expect(checkGoals(open.map((f) => ({ ...f, value: '-1' })))).toEqual({ kind: 'invalid', target: 'termsUsed', message: 'Type a number of 0 or more for Search terms tribelt.nl uses.' })
  })
})
