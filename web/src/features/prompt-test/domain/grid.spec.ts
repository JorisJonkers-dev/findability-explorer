import { describe, expect, it } from 'vitest'
import { promptAnswer, promptRun } from '../../../testing/fixtures'
import {
  answerCost,
  averageCost,
  retryPlan,
  byKey,
  columnsOf,
  counts,
  excerpt,
  gridHeadline,
  headlineNumbers,
  hostOf,
  isTribeltSite,
  keyOf,
  MODE_FILTERS,
  mostNamed,
  parseFilter,
  PLACE_ORDER,
  PLACES,
  placeOf,
  rowsOf,
  SCOPE_FILTERS,
  toCsv,
  tribeltPosition,
} from './grid'

describe("Tribelt's place in an answer", () => {
  it('is first, second or third, later, not named, or there is no answer', () => {
    expect(placeOf(promptAnswer({}, ['Tribelt:tribelt']))).toBe('first')
    expect(placeOf(promptAnswer({}, ['A:a', 'Tribelt:tribelt']))).toBe('top3')
    expect(placeOf(promptAnswer({}, ['A:a', 'B', 'C', 'Tribelt:tribelt']))).toBe('later')
    expect(placeOf(promptAnswer({}, ['A:a']))).toBe('none')
    expect(placeOf(promptAnswer({ failure: 'Rate limited.', extraction: undefined }))).toBe('missing')
    expect(placeOf(undefined)).toBe('missing')
    expect(placeOf(promptAnswer({}, [], { failed: true }))).toBe('missing')
  })

  it('counts an answer that names Tribelt without a place as later', () => {
    const named = promptAnswer({}, ['A:a'], { namesTribelt: true })
    expect(tribeltPosition(named)).toBeUndefined()
    expect(placeOf(named)).toBe('later')
    expect(PLACES.later.mark(undefined)).toBe('+')
    expect(PLACES.later.mark(5)).toBe('5')
    expect(PLACES.top3.mark(undefined)).toBe('')
  })

  it('marks every place differently, in the legend order', () => {
    expect(PLACE_ORDER.map((p) => PLACES[p].mark(2))).toEqual(['1', '2', '2', '–', '✕'])
    expect(counts(undefined)).toBe(false)
  })
})

describe('the grid', () => {
  it('has a column per provider and mode, narrowed to one mode when asked', () => {
    const run = promptRun()
    expect(columnsOf(run).map((c) => `${c.name} ${c.mode}`)).toEqual(['OpenAI knowledge', 'OpenAI web', 'Google knowledge', 'Google web'])
    expect(columnsOf(run, 'web').map((c) => c.provider)).toEqual(['openai', 'google'])
  })

  it('has a row per question, by family then scope, narrowed to one scope when asked', () => {
    const answers = [
      promptAnswer({ promptId: 'open-dutch', family: 'open_spiral', scope: 'dutch' }),
      promptAnswer({ promptId: 'eye-english', family: 'eyelink', scope: 'english' }),
      promptAnswer({ promptId: 'eye-twente', family: 'eyelink', scope: 'twente', provider: 'google' }),
      promptAnswer({ promptId: 'eye-twente', family: 'eyelink', scope: 'twente' }),
    ]
    expect(rowsOf(answers).map((r) => r.promptId)).toEqual(['eye-twente', 'eye-english', 'open-dutch'])
    expect(rowsOf(answers, 'dutch').map((r) => r.promptId)).toEqual(['open-dutch'])
    const a = promptAnswer({ promptId: 'q', provider: 'google', mode: 'web' })
    expect(keyOf(a)).toBe('q|google|web')
    expect(byKey([a]).get('q|google|web')).toBe(a)
  })

  it('reads its filters from the address', () => {
    expect(parseFilter('web', MODE_FILTERS)).toBe('web')
    expect(parseFilter('mars', MODE_FILTERS)).toBe('all')
    expect(parseFilter(undefined, SCOPE_FILTERS)).toBe('all')
    expect(parseFilter('twente', SCOPE_FILTERS)).toBe('twente')
  })
})

describe('the numbers', () => {
  const answers = [
    promptAnswer({ scope: 'twente' }, ['Tribelt:tribelt', 'Twentebelt:twentebelt']),
    promptAnswer({ scope: 'dutch', mode: 'web' }, ['Twentebelt:twentebelt', 'Unknown']),
    promptAnswer({ scope: 'dutch', mode: 'web', provider: 'google' }, ['Ashworth:ashworth', 'Tribelt:tribelt'], { citesTribelt: true }),
    promptAnswer({ failure: 'No answer.', extraction: undefined }),
  ]

  it('count the answers naming Tribelt, in Twente, and the web answers citing tribelt.nl', () => {
    expect(headlineNumbers(answers)).toEqual({ named: 2, counted: 3, twente: 1, twenteCounted: 1, cited: 1, web: 2 })
  })

  it('rank who the answers name, Tribelt always among them', () => {
    expect(mostNamed(answers).map((n) => [n.id, Math.round(n.share * 100)])).toEqual([
      ['tribelt', 67],
      ['twentebelt', 67],
      ['ashworth', 33],
    ])
    expect(mostNamed(answers, 1)[0]?.id).toBe('tribelt')
    expect(mostNamed([promptAnswer({}, ['A:a', 'B:b'])], 2).map((n) => n.id)).toEqual(['a', 'tribelt'])
    expect(mostNamed([])).toEqual([{ id: 'tribelt', share: 0, tribelt: true }])
  })

  it('head the page with who the answers name most', () => {
    const name = (id: string) => id.toUpperCase()
    expect(gridHeadline([], name)).toBe('Do AI assistants name Tribelt?')
    expect(gridHeadline(answers, name)).toBe('AI answers name Tribelt more than any competitor.')
    const twente = [promptAnswer({ scope: 'twente' }, ['Tribelt:tribelt']), promptAnswer({ scope: 'dutch' }, ['Ashworth:ashworth']), promptAnswer({ scope: 'german' }, ['Ashworth:ashworth'])]
    expect(gridHeadline(twente, name)).toBe('Twente buyers hear of Tribelt. Everyone else hears of ASHWORTH.')
    expect(gridHeadline([promptAnswer({ scope: 'dutch' }, ['Ashworth:ashworth'])], name)).toBe('AI answers name ASHWORTH most.')
  })
})

describe('an answer in detail', () => {
  it('starts with an excerpt cut at a word', () => {
    expect(excerpt('Short  answer.')).toBe('Short answer.')
    expect(excerpt('one two three four', 9)).toBe('one two…')
    expect(excerpt('abcdefghij', 5)).toBe('abcde…')
  })

  it('reads hosts and spots tribelt.nl', () => {
    expect(hostOf('https://www.tribelt.nl/producten')).toBe('tribelt.nl')
    expect(hostOf('ashworth.com/belts')).toBe('ashworth.com')
    expect(hostOf('http://[bad')).toBe('http://[bad')
    expect(isTribeltSite('https://shop.tribelt.nl')).toBe(true)
    expect(isTribeltSite('https://nottribelt.nl')).toBe(false)
  })
})

describe('the CSV', () => {
  it('has a header and a row per answer, quoting what needs it', () => {
    const cited = promptAnswer({ text: 'Says "hi", then\nleaves.' }, ['Tribelt:tribelt'], { citesTribelt: true })
    const csv = toCsv([
      cited,
      promptAnswer({ failure: 'Rate limited.', extraction: undefined, citations: ['https://a.example'] }),
    ]).split('\r\n')
    expect(csv[0]).toBe(
      "question id,family,scope,language,question,provider,model,mode,Tribelt's place,companies named,cites tribelt.nl,sources,input tokens,output tokens,cost in dollars,failure,answer",
    )
    expect(csv[1]).toContain(',first,Tribelt,yes,,900,300,0.0012,,"Says ""hi"", then')
    expect(csv[2]).toContain(',no answer,,no,https://a.example,900,300,0.0012,Rate limited.,')
  })
})

describe('a cost', () => {
  it('shows a fraction of a cent', () => {
    expect(answerCost(0.00143)).toBe('$0.0014')
    expect(answerCost(0.42)).toBe('$0.42')
    expect(answerCost(0)).toBe('$0')
    expect(answerCost(0.05)).toBe('$0.05')
  })
})

describe('asking again', () => {
  it('counts the failed and missing answers per provider and prices them by its average answer', () => {
    const run = promptRun({ tasks: 4 })
    const answers = [
      promptAnswer({ provider: 'openai', costUsd: 0.002 }),
      promptAnswer({ provider: 'openai', mode: 'web', costUsd: 0.004 }),
      promptAnswer({ provider: 'google', failure: 'Rate limited.', extraction: undefined, costUsd: 0.0001 }),
    ]
    expect(averageCost(answers, 'openai')).toBeCloseTo(0.003)
    // Google has no answer to go by: the average of every answer.
    expect(averageCost(answers, 'google')).toBeCloseTo(0.003)
    expect(averageCost([], 'google')).toBe(0)
    const plan = retryPlan(run, answers)
    expect(plan.count).toBe(2)
    expect(plan.estimate).toBeCloseTo(0.006)
    expect(retryPlan(promptRun({ models: [] }), answers)).toEqual({ count: 0, estimate: 0 })
  })
})
