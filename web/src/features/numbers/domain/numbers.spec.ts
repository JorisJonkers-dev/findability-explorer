import { describe, expect, it } from 'vitest'
import type { Plan, PlanTarget, Standings } from '../../../infrastructure/api/types.gen'
import { planFixture, standingsFixture } from '../../../testing/numbersFixtures'
import { improvementRanks, numberDetail, numberRows, oxford, pathOfTarget, targetOfSlug, valueText } from './numbers'

const facts = (change: (plan: Plan, standings: Standings) => void = () => undefined) => {
  const plan = planFixture()
  const standings = standingsFixture()
  change(plan, standings)
  return { plan, standings }
}

const TARGETS: PlanTarget[] = ['aiShareOfVoice', 'twenteShareOfVoice', 'citationShare', 'readinessChecks', 'termsUsed']

describe('oxford', () => {
  it('writes a list as a sentence does', () => {
    expect(oxford([])).toBe('')
    expect(oxford(['1'])).toBe('1')
    expect(oxford(['1', '5'])).toBe('1 and 5')
    expect(oxford(['1', '2', '5'])).toBe('1, 2, and 5')
  })
})

describe('addresses', () => {
  it('maps every number to an address and back', () => {
    for (const t of TARGETS) expect(targetOfSlug(pathOfTarget(t).replace('/numbers/', ''))).toBe(t)
    expect(pathOfTarget('aiShareOfVoice')).toBe('/numbers/aiShareOfVoice')
    expect(targetOfSlug('nothing')).toBeUndefined()
  })
})

describe('valueText', () => {
  it('writes shares, counts and readiness checks', () => {
    expect(valueText('aiShareOfVoice', 0.24, 5)).toBe('24%')
    expect(valueText('citationShare', 0.36, 8)).toBe('36%')
    expect(valueText('readinessChecks', 3, 5)).toBe('3 of 5')
    expect(valueText('termsUsed', 1234, 5)).toBe('1,234')
  })
})

describe('improvementRanks', () => {
  it('lists the places in the plan of the improvements for a number', () => {
    const { plan } = facts()
    expect(improvementRanks(plan, 'aiShareOfVoice')).toEqual([1, 4])
    expect(improvementRanks(plan, 'twenteShareOfVoice')).toEqual([1, 4])
    expect(improvementRanks(plan, 'readinessChecks')).toEqual([2])
    expect(improvementRanks(plan, 'citationShare')).toEqual([3])
    expect(improvementRanks(plan, 'termsUsed')).toEqual([])
  })
})

describe('numberRows', () => {
  it('draws every number in the server order, with its bar, target and leader', () => {
    const rows = numberRows(facts())
    expect(rows.map((r) => r.target)).toEqual(TARGETS)
    const ai = rows[0]
    expect(ai).toMatchObject({ path: '/numbers/aiShareOfVoice', now: '24%', goal: '40%', measured: true, improvedBy: 'Improved by recommendations 1 and 4' })
    expect(ai?.progress).toBe('60% of the way to the target. Ashworth is at 57%.')
    expect(ai?.fill).toBeCloseTo((0.24 / (0.57 * 1.08)) * 100)
    expect(ai?.leaderAt).toBeCloseTo(100 / 1.08)
    expect(ai?.aria).toBe('AI answers that name Tribelt: 24%, target 40%. 60% of the way to the target. Ashworth is at 57%. Open its page.')
  })

  it('looks at the competitor the plan names for Twente', () => {
    const twente = numberRows(facts())[1]
    expect(twente?.progress).toBe('40% of the way to the target. Twentebelt is at 60%.')
    expect(twente?.leaderAt).toBeDefined()
  })

  it('sets the citation share against the leader, like the other shares', () => {
    const citations = numberRows(facts())[2]
    expect(citations).toMatchObject({ now: '10%', goal: '50%', improvedBy: 'Improved by recommendation 3', progress: '20% of the way to the target. Märtens is at 30%.' })
    expect(citations?.leaderAt).toBeDefined()
  })

  it('writes the readiness checks out of the total', () => {
    expect(numberRows(facts())[3]).toMatchObject({ now: '3 of 5', goal: '5 of 5' })
  })

  it('says that a target is met, and that Tribelt leads', () => {
    const terms = numberRows(facts())[4]
    expect(terms?.progress).toBe('The target is met. Tribelt leads the field.')
    expect(terms?.improvedBy).toBe('')
  })

  it('says when a number is not measured, and when there is no leader', () => {
    const rows = numberRows(
      facts((plan, standings) => {
        plan.watched = [{ target: 'aiShareOfVoice', goal: 0.4 }, { target: 'termsUsed', now: 5, goal: 10 }]
        standings.measures = standings.measures.filter((m) => m.measure !== 'termsUsed')
      }),
    )
    expect(rows[0]).toMatchObject({ now: 'Not measured', measured: false, fill: 0, progress: 'Not measured yet.' })
    expect(rows[1]).toMatchObject({ leaderAt: undefined, progress: '50% of the way to the target.' })
  })

  it('treats a target of zero as reached', () => {
    const rows = numberRows(facts((plan) => (plan.watched = [{ target: 'citationShare', now: 0, goal: 0 }])))
    expect(rows[0]?.progress).toBe('100% of the way to the target. Märtens is at 30%.')
  })
})

describe('numberDetail', () => {
  it('compares Tribelt, the typical competitor, the leader and the target on a share', () => {
    const d = numberDetail(facts(), 'aiShareOfVoice')
    expect(d?.headline).toBe("Ashworth is ahead with 57%, against Tribelt's 24%.")
    expect(d?.counts).toBe("This number is the share of the newest Prompt run's answers that name Tribelt at least once.")
    expect(d?.bars.map((b) => [b.label, b.display, b.tone])).toEqual([
      ['Tribelt', '24%', 'orange'],
      ['Typical competitor', '10%', 'blue-grey'],
      ['Ashworth', '57%', 'ink'],
      ['Target', '40%', 'silver'],
    ])
    expect(d?.barsMax).toBeCloseTo(0.57 * 1.08)
    expect(d?.figures.map((f) => f.big)).toEqual(['24%', '57%', '40%'])
    expect(d?.figures[0]?.note).toBe('Tribelt is 60% of the way there.')
    expect(d?.figures[1]).toMatchObject({ title: 'Ashworth, the leader', note: 'The competitor with the best value now.' })
    expect(d?.figures[2]?.note).toBe('The goal for this number by the end of the year.')
    expect(d?.comparison).toBe('The typical competitor is the middle value over the four competitors with data.')
    expect(d?.improvements.map((i) => i.rank)).toEqual([1, 4])
  })

  it('takes the competitor for Twente from the plan', () => {
    const d = numberDetail(facts(), 'twenteShareOfVoice')
    expect(d?.bars.map((b) => b.label)).toEqual(['Tribelt', 'Twentebelt', 'Target'])
    expect(d?.comparison).toBe('Twentebelt is named most when buyers ask for Twente. The other competitors are not compared on this number.')
    const none = numberDetail(facts((plan) => (plan.eyeOn = [])), 'twenteShareOfVoice')
    expect(none?.comparison).toBe('No competitor is named for Twente yet.')
    expect(none?.figures[1]).toMatchObject({ big: 'None', title: 'No leader yet' })
    expect(none?.headline).toBe('Tribelt is at 20%, against a target of 50%.')
  })

  it('sets the citation share against the leader', () => {
    const d = numberDetail(facts(), 'citationShare')
    expect(d?.headline).toBe("Märtens is ahead with 30%, against Tribelt's 10%.")
    expect(d?.bars.map((b) => [b.label, b.display])).toEqual([
      ['Tribelt', '10%'],
      ['Typical competitor', '5%'],
      ['Märtens', '30%'],
      ['Target', '50%'],
    ])
    expect(d?.figures[1]).toMatchObject({ big: '30%', title: 'Märtens, the leader' })
  })

  it('writes the readiness checks as "3 of 5"', () => {
    const d = numberDetail(facts(), 'readinessChecks')
    expect(d?.figures[0]?.big).toBe('3 of 5')
    expect(d?.bars.find((b) => b.label === 'Typical competitor')?.display).toBe('2 of 5')
  })

  it('says that a target is met and that Tribelt leads', () => {
    const d = numberDetail(facts(), 'termsUsed')
    expect(d?.headline).toBe('Tribelt leads on this number, with 45.')
    expect(d?.figures[0]?.note).toBe('Tribelt has met the target.')
    expect(d?.figures[2]?.note).toBe('The target is met.')
    expect(d?.improvements).toEqual([])
  })

  it('says what is missing when nothing has been measured', () => {
    const d = numberDetail(
      facts((plan, standings) => {
        plan.watched = [{ target: 'aiShareOfVoice', goal: 0.4 }]
        standings.measures = [{ measure: 'aiShareOfVoice', competitors: 0, trails: false, leads: false }]
      }),
      'aiShareOfVoice',
    )
    expect(d?.headline).toBe('AI answers that name Tribelt is not measured yet.')
    expect(d?.figures[0]).toMatchObject({ big: 'No data', note: 'The first scan or Prompt run fills this in.' })
    expect(d?.figures[1]).toMatchObject({ big: 'None', title: 'No leader yet' })
    expect(d?.bars[1]).toMatchObject({ label: 'Typical competitor', value: 0, display: 'No data yet' })
    expect(d?.comparison).toBe('No competitor has data on this number yet.')
    expect(d?.leader).toBeUndefined()
  })

  it('has nothing for a number the plan does not watch', () => {
    expect(numberDetail(facts((plan) => (plan.watched = [])), 'citationShare')).toBeUndefined()
  })
})
