import { describe, expect, it } from 'vitest'
import type { Overview, Plan, PlanTarget, PromptCell, ReadinessCheckSummary, Standings } from '../../../infrastructure/api/types.gen'
import { promptRun } from '../../../testing/fixtures'
import { overviewFixture, planFixture, standingsFixture } from '../../../testing/numbersFixtures'
import { everyResponse, GOAL_ORDER, goalPage, goalRows, goalTabTitle, historyMeasure, isPromptGoal, overviewSummary, pathOfTarget, promptBreakdown, promptDataLine, readinessBreakdown, targetOfSlug, termsBreakdown, valueText } from './goals'

const facts = (change: (plan: Plan, standings: Standings) => void = () => undefined) => {
  const plan = planFixture()
  const standings = standingsFixture()
  change(plan, standings)
  return { plan, standings }
}

describe('addresses', () => {
  it('maps every goal to an address and back, in the order of the spec', () => {
    expect(GOAL_ORDER).toEqual(['aiShareOfVoice', 'twenteShareOfVoice', 'citationShare', 'readinessChecks', 'termsUsed'])
    for (const t of GOAL_ORDER) expect(targetOfSlug(pathOfTarget(t).replace('/goals/', ''))).toBe(t)
    expect(pathOfTarget('aiShareOfVoice')).toBe('/goals/aiShareOfVoice')
    expect(targetOfSlug('nothing')).toBeUndefined()
  })
})

describe('valueText', () => {
  it('writes shares, counts, and readiness checks', () => {
    expect(valueText('aiShareOfVoice', 0.24, 5)).toBe('24%')
    expect(valueText('citationShare', 0.36, 8)).toBe('36%')
    expect(valueText('readinessChecks', 3, 8)).toBe('3 of 8')
    expect(valueText('termsUsed', 1234, 5)).toBe('1,234')
  })
})

describe('the Goals table', () => {
  it('has a row per goal, named as in the spec, with Tribelt, the goal, and the best competitor', () => {
    const rows = goalRows(facts())
    expect(rows.map((r) => [r.name, r.now, r.goal, r.best, r.path])).toEqual([
      ['AI responses that name Tribelt', '24%', '40%', 'Ashworth, 57%', '/goals/aiShareOfVoice'],
      ['AI responses about the Twente region name Tribelt', '20%', '50%', 'Twentebelt, 60%', '/goals/twenteShareOfVoice'],
      ['Web responses that cite tribelt.nl', '10%', '50%', 'Märtens, 30%', '/goals/citationShare'],
      ['AI readiness checks passed', '3 of 8', '8 of 8', 'Märtens, 8 of 8', '/goals/readinessChecks'],
      ['Search terms tribelt.nl uses', '45', '35', 'Ashworth, 40', '/goals/termsUsed'],
    ])
  })

  it('places Tribelt, the best competitor, and the goal on a bar, shares out of 100%, checks out of all of them', () => {
    const [ai, , , checks, terms] = goalRows(facts())
    expect(ai?.bar).toEqual({ now: 24, best: 57, goal: 40 })
    expect(checks?.bar).toEqual({ now: 37.5, best: 100, goal: 100 })
    // A count runs to the largest of a fifth past the goal, Tribelt, and the best competitor.
    expect(terms?.bar.goal).toBeCloseTo((35 / 45) * 100)
    expect(terms?.bar.now).toBeCloseTo(100)
    expect(ai?.barLabel).toBe('Tribelt now 24%, the best competitor 57%, the goal 40%.')
    expect(ai?.aria).toBe('AI responses that name Tribelt: Tribelt 24%, the goal 40%, the best competitor Ashworth with 57%. Open its page.')
  })

  it('says when a goal is not measured, and when no competitor has a value', () => {
    const rows = goalRows(
      facts((plan, standings) => {
        plan.watched = [{ target: 'termsUsed', goal: 10 }]
        standings.measures = []
      }),
    )
    expect(rows).toHaveLength(1)
    expect(rows[0]).toMatchObject({ now: 'Not measured', measured: false, best: 'None yet', bar: { now: 0, goal: 83.33, best: undefined } })
    expect(rows[0]?.barLabel).toBe('Tribelt now not measured, the goal 10.')
    expect(rows[0]?.aria).toBe('Search terms tribelt.nl uses: Tribelt not measured, the goal 10, no best competitor yet. Open its page.')
  })
})

describe('the Overview summary', () => {
  const overview = (change: (o: Overview) => void = () => undefined): Overview => {
    const o = overviewFixture()
    change(o)
    return o
  }
  it('says how many competitors AI responses name more often, the readiness checks, and the search terms', () => {
    const o = overview((x) => {
      x.aiSearch.companies.push({ competitorId: 'ashworth', name: 'Ashworth', role: 'competitor', answers: 20, share: 0.4 })
    })
    expect(overviewSummary(facts(), o)).toBe(
      'AI responses name one competitor more often than Tribelt, tribelt.nl passes 3 of the 8 AI readiness checks, and tribelt.nl uses 30 of the 387 search terms buyers use.',
    )
    o.aiSearch.companies.push(
      { competitorId: 'martens', name: 'Märtens', role: 'competitor', answers: 20, share: 0.3 },
      { competitorId: 'lesjofors', name: 'Lesjöfors', role: 'group', answers: 30, share: 0.6 },
      { competitorId: 'quiet', name: 'Quiet', role: 'competitor', answers: 0 },
    )
    expect(overviewSummary(facts(), o)).toContain('AI responses name 2 competitors more often than Tribelt,')
  })
  it('says when Tribelt is named most, or not yet asked about, and when tribelt.nl is not checked', () => {
    expect(overviewSummary(facts(), overview())).toMatch(/^AI responses name Tribelt more often than any competitor,/)
    const none = overview((x) => (x.aiSearch.companies = []))
    const unchecked = facts((plan) => (plan.watched = plan.watched.filter((w) => w.target !== 'readinessChecks')))
    expect(overviewSummary(unchecked, none)).toBe(
      'No Prompt run has finished yet, tribelt.nl has not been checked for AI readiness yet, and tribelt.nl uses 30 of the 387 search terms buyers use.',
    )
  })
})

describe('a goal page', () => {
  it('names the goal, says what it measures, and who is ahead', () => {
    const page = goalPage(facts(), 'aiShareOfVoice')
    expect(page).toMatchObject({ name: 'AI responses that name Tribelt', goal: 0.4, purpose: "The share of a Prompt run's responses that name Tribelt." })
    expect(page?.summary).toBe('Ashworth is ahead of Tribelt, with 57% against 24%. The goal is 40%.')
    expect(page?.bars.map((b) => [b.label, b.display, b.tone, b.highlight ?? false])).toEqual([
      ['Tribelt', '24%', 'orange', true],
      ['The typical competitor', '10%', 'blue-grey', false],
      ['Ashworth, the best competitor', '57%', 'ink', false],
      ['The goal', '40%', 'silver', false],
    ])
    expect(page?.barsMax).toBe(1)
    expect(page?.comparison).toBe('The typical competitor is the middle figure across all competitors on the list.')
    expect(page?.improvements.map((i) => i.id)).toEqual(['submit-search-engines', 'answer-questions'])
    expect(page?.leader?.name).toBe('Ashworth')
  })

  it('says so when Tribelt leads, and when no competitor has a value', () => {
    expect(goalPage(facts(), 'termsUsed')?.summary).toBe("Tribelt leads every competitor, with 45 against Ashworth's 40. The goal is 35.")
    const alone = goalPage(
      facts((_, standings) => (standings.measures = [])),
      'termsUsed',
    )
    expect(alone?.summary).toBe('Tribelt leads every competitor, with 45. The goal is 35.')
    expect(alone?.bars.map((b) => [b.label, b.display])).toEqual([
      ['Tribelt', '45'],
      ['The typical competitor', 'No data yet'],
      ['The best competitor', 'None yet'],
      ['The goal', '35'],
    ])
    expect(alone?.barsMax).toBeCloseTo(45 * 1.1)
    expect(alone?.comparison).toBe('No competitor has data on this goal yet.')
  })

  it('compares the Twente goal with the competitor named most for Twente only', () => {
    const page = goalPage(facts(), 'twenteShareOfVoice')
    expect(page?.purpose).toBe("The share of a Prompt run's responses about the Twente region that name Tribelt.")
    expect(page?.bars.map((b) => b.label)).toEqual(['Tribelt', 'Twentebelt, the best competitor', 'The goal'])
    expect(page?.comparison).toBe('Twentebelt is named most when buyers ask about Twente. The other competitors are not compared on this goal.')
    const none = goalPage(
      facts((plan) => (plan.eyeOn = [])),
      'twenteShareOfVoice',
    )
    expect(none?.comparison).toBe('No competitor is named for Twente yet.')
    expect(none?.bars.map((b) => b.label)).toEqual(['Tribelt', 'The best competitor', 'The goal'])
  })

  it('counts the readiness checks out of all of them', () => {
    const page = goalPage(facts(), 'readinessChecks')
    expect(page?.purpose).toBe('How many of the eight AI readiness checks tribelt.nl passes.')
    expect(page?.summary).toBe('Märtens is ahead of Tribelt, with 8 of 8 against 3 of 8. The goal is 8 of 8.')
    expect(page?.barsMax).toBe(8)
    expect(page?.improvements).toEqual([])
  })

  it('says when Tribelt is not measured yet, and has nothing for a goal the plan does not hold', () => {
    const page = goalPage(
      facts((plan) => (plan.watched = [{ target: 'citationShare', goal: 0.5 }])),
      'citationShare',
    )
    expect(page?.summary).toBe('Tribelt is not measured on this goal yet. The goal is 50%.')
    expect(page?.bars[0]).toMatchObject({ value: 0, display: 'Not measured' })
    expect(
      goalPage(
        facts((plan) => (plan.watched = [])),
        'citationShare',
      ),
    ).toBeUndefined()
  })

  it('knows which goals the Prompt test measures, and their line over time', () => {
    expect(GOAL_ORDER.filter(isPromptGoal)).toEqual(['aiShareOfVoice', 'twenteShareOfVoice', 'citationShare'])
    const run = { runId: 'r', startedAt: '2026-10-27T08:36:32Z', promptSetVersion: 2, tasks: 1, answered: 1, read: 1, partial: false, all: 0.25, twente: 0.4, citation: 0.1 }
    const picks = (['aiShareOfVoice', 'twenteShareOfVoice', 'citationShare'] as PlanTarget[]).map((t) => historyMeasure(t)?.pick(run))
    expect(picks).toEqual([0.25, 0.4, 0.1])
    expect(historyMeasure('aiShareOfVoice')).toMatchObject({ label: 'Tribelt', tone: 'orange', strong: true })
    expect(historyMeasure('termsUsed')).toBeUndefined()
  })
})

describe('the way to the responses behind a goal', () => {
  it('opens Every response on the responses the goal counts', () => {
    expect(everyResponse('aiShareOfVoice')).toEqual({ path: '/ai-responses/all' })
    expect(everyResponse('twenteShareOfVoice')).toEqual({ path: '/ai-responses/all', query: { scope: 'twente' } })
    expect(everyResponse('citationShare')).toEqual({ path: '/ai-responses/all', query: { mode: 'web' } })
    expect(everyResponse('termsUsed')).toBeUndefined()
  })
})

describe('the data line of a Prompt test goal', () => {
  it('says which run, how many questions, models, and responses, and the Prompt set version', () => {
    expect(promptDataLine(promptRun({ promptSetVersion: 2, tasks: 80, answered: 78 }))).toBe(
      'From the Prompt run of 1 November 2026: 20 questions to 2 models, each from the model’s own knowledge and with web search, 78 responses in all. Prompt set version 2.',
    )
    expect(promptDataLine(promptRun({ models: [], answered: 1 }))).toBe('From the Prompt run of 1 November 2026: 1 response in all. Prompt set version 1.')
    expect(promptDataLine(undefined)).toBe('No Prompt run has finished yet.')
  })
})

describe('Broken down', () => {
  const cells: PromptCell[] = [
    { provider: 'openai', scope: 'twente', family: 'eyelink', mode: 'knowledge', answers: 4, named: 1, cited: 0 },
    { provider: 'openai', scope: 'twente', family: 'eyelink', mode: 'web', answers: 4, named: 3, cited: 2 },
    { provider: 'google', scope: 'english', family: 'open_spiral', mode: 'web', answers: 4, named: 0, cited: 1 },
  ]

  it('breaks the share down by model, scope, and belt family, each with its total', () => {
    const groups = promptBreakdown('aiShareOfVoice', cells)
    expect(groups.map((g) => g.title)).toEqual(['By model', 'By scope', 'By belt family'])
    expect(groups[0]?.rows.map((r) => [r.label, r.display, r.highlight ?? false])).toEqual([
      ['OpenAI', '50%', false],
      ['Google', '0%', false],
      ['All models', '33%', true],
    ])
    expect(groups[1]?.rows.map((r) => [r.label, r.display])).toEqual([
      ['Twente (Dutch)', '50%'],
      ['English, no region', '0%'],
      ['All scopes', '33%'],
    ])
    expect(groups[2]?.rows.map((r) => r.label)).toEqual(['Eyelink', 'Open spiral', 'All belt families'])
  })

  it('counts only the Twente responses for the Twente goal, without a scope list', () => {
    const groups = promptBreakdown('twenteShareOfVoice', cells)
    expect(groups.map((g) => g.title)).toEqual(['By model', 'By belt family'])
    expect(groups[0]?.rows.map((r) => [r.label, r.display])).toEqual([
      ['OpenAI', '50%'],
      ['All models', '50%'],
    ])
  })

  it('counts the citations of web responses for the citation goal', () => {
    const groups = promptBreakdown('citationShare', cells)
    expect(groups[0]?.rows.map((r) => [r.label, r.display])).toEqual([
      ['OpenAI', '50%'],
      ['Google', '25%'],
      ['All models', '38%'],
    ])
    expect(promptBreakdown('citationShare', [])).toEqual([])
    expect(promptBreakdown('termsUsed', cells)).toEqual([])
  })

  it('lists each scored readiness check, passed or not', () => {
    const checks: ReadinessCheckSummary[] = [
      { check: 'crawlers', scored: true, passing: 30, tribelt: true },
      { check: 'llmsTxt', scored: true, passing: 12, tribelt: false },
      { check: 'markdown', scored: true, passing: 2 },
      { check: 'product', scored: false, passing: 3, tribelt: false },
    ]
    expect(readinessBreakdown(checks).map((c) => [c.label, c.passed])).toEqual([
      ['AI crawlers let in', true],
      ['llms.txt', false],
      ['Markdown pages', null],
    ])
    expect(readinessBreakdown(checks)[1]?.what).toBe('llms.txt is a short guide to the site for language models, with links to the pages that matter. Having it helps.')
  })

  it('counts the search terms tribelt.nl uses per language', () => {
    const { rows, max } = termsBreakdown([
      { language: 'nl', terms: 58, tribelt: 13 },
      { language: 'de', terms: 221, tribelt: 15 },
    ])
    expect(max).toBeCloseTo(15 * 1.1)
    expect(termsBreakdown([]).max).toBe(1)
    expect(rows.map((r) => [r.label, r.display])).toEqual([
      ['Dutch, of 58', '13'],
      ['German, of 221', '15'],
    ])
  })
})

describe('a goal page in the browser tab', () => {
  it('says Goal and the goal’s name, or Goal alone for an address no goal has', () => {
    expect(goalTabTitle('citationShare')).toBe('Goal, Web responses that cite tribelt.nl')
    expect(goalTabTitle('nothing')).toBe('Goal')
  })
})
