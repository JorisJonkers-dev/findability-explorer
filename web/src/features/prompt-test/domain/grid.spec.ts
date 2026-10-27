import { describe, expect, it } from 'vitest'
import { promptAnswer, promptRun } from '../../../testing/fixtures'
import {
  answerCost,
  byKey,
  cellLabel,
  columnsOf,
  counts,
  excerpt,
  gridHeadline,
  headlineNumbers,
  hostOf,
  isTribeltSite,
  keyOf,
  LEGEND,
  markOf,
  MODE_COLUMNS,
  MODE_FILTERS,
  modeCost,
  mostNamed,
  ordinal,
  parseFilter,
  PLACE_ORDER,
  PLACES,
  placeOf,
  rowsOf,
  SCOPE_FILTERS,
  saysOf,
  type Square,
  squareOf,
  toCsv,
  tribeltPosition,
  unknownLine,
  unread,
} from './grid'

/** An answer that puts Tribelt at a place in its list, from 1. */
const at = (position: number) => promptAnswer({}, [...Array.from({ length: position - 1 }, (_, i) => `C${String(i)}`), 'Tribelt:tribelt'])

describe("Tribelt's place in an answer", () => {
  it('is first, second or third, later, named without a place, not named, or there is no answer', () => {
    expect(placeOf(at(1))).toBe('first')
    expect(placeOf(at(2))).toBe('top3')
    expect(placeOf(at(4))).toBe('later')
    expect(placeOf(promptAnswer({}, ['A:a'], { namesTribelt: true }))).toBe('unplaced')
    expect(placeOf(promptAnswer({}, ['A:a']))).toBe('none')
    expect(placeOf(promptAnswer({ failure: 'Rate limited.', extraction: undefined }))).toBe('missing')
    expect(placeOf(undefined)).toBe('missing')
    expect(placeOf(promptAnswer({}, [], { failed: true }))).toBe('missing')
    expect(counts(undefined)).toBe(false)
  })

  it('marks a named square with its place in the list, and the others with a symbol', () => {
    expect([1, 2, 3, 4, 7, 30].map((p) => markOf(squareOf(at(p))))).toEqual(['1', '2', '3', '4', '7', '30'])
    const named = promptAnswer({}, ['A:a'], { namesTribelt: true })
    expect(tribeltPosition(named)).toBeUndefined()
    expect(markOf(squareOf(named))).toBe('?')
    expect(markOf({ place: 'none' })).toBe('–')
    expect(markOf({ place: 'missing' })).toBe('✕')
  })

  it('says each square in words', () => {
    const squares: Square[] = [{ place: 'first', position: 1 }, { place: 'top3', position: 2 }, { place: 'later', position: 11 }, { place: 'unplaced' }, { place: 'none' }, { place: 'missing' }]
    expect(squares.map(saysOf)).toEqual([
      'Tribelt named first',
      'Tribelt named 2nd',
      'Tribelt named 11th',
      'Tribelt named, place unknown',
      'Tribelt not named',
      'No answer, the model gave none or it could not be read',
    ])
    expect([1, 2, 3, 4, 11, 12, 13, 21, 22, 23, 30].map(ordinal)).toEqual(['1st', '2nd', '3rd', '4th', '11th', '12th', '13th', '21st', '22nd', '23rd', '30th'])
    const row = { family: 'eyelink' as const, scope: 'twente' as const }
    expect(cellLabel(row, { name: 'OpenAI', mode: 'knowledge' }, squareOf(at(4)))).toBe('OpenAI, own knowledge, Eyelink, Twente (Dutch): Tribelt named 4th')
    expect(cellLabel(row, { name: 'Google', mode: 'web' }, { place: 'none' })).toBe('Google, with web search, Eyelink, Twente (Dutch): Tribelt not named')
  })

  it('has a legend entry for every mark a square can show, with the same text and look', () => {
    const answers = [
      ...Array.from({ length: 30 }, (_, i) => at(i + 1)),
      promptAnswer({}, ['A:a'], { namesTribelt: true }),
      promptAnswer({}, ['A:a']),
      promptAnswer({ failure: 'Rate limited.', extraction: undefined }),
      promptAnswer({}, [], { failed: true }),
      undefined,
    ]
    expect(new Set(answers.map(placeOf))).toEqual(new Set(PLACE_ORDER))
    for (const a of answers) {
      const square = squareOf(a)
      const entry = LEGEND.find((s) => s.place === square.place) as Square
      expect(PLACES[entry.place].cell).toBe(PLACES[square.place].cell)
      if ('position' in square) expect(markOf(entry)).toMatch(/^\d+$/)
      else expect(markOf(entry)).toBe(markOf(square))
    }
    // Each numbered example sits in its own tier: the legend's "4" is a real 4th place.
    for (const s of LEGEND) if ('position' in s) expect(squareOf(at(s.position))).toEqual(s)
    expect(PLACE_ORDER).toEqual(['first', 'top3', 'later', 'unplaced', 'none', 'missing'])
    expect(LEGEND.map(markOf)).toEqual(['1', '2', '4', '?', '–', '✕'])
    expect(new Set(PLACE_ORDER.map((p) => PLACES[p].cell)).size).toBe(PLACE_ORDER.length)
  })

  it('names both columns of a provider in full', () => {
    expect(MODE_COLUMNS).toEqual({ knowledge: { short: 'Own', long: 'Own knowledge' }, web: { short: 'Web', long: 'With web search' } })
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

  it("count Tribelt in its bar as the headline does: by the reading, also with no name matched to it", () => {
    const read = [promptAnswer({}, ['Twentebelt:twentebelt', 'Tribelt B.V.'], { namesTribelt: true }), promptAnswer({}, ['Twentebelt:twentebelt'], { namesTribelt: false })]
    const tribelt = mostNamed(read).find((n) => n.tribelt)
    expect(tribelt?.share).toBe(headlineNumbers(read).named / headlineNumbers(read).counted)
    expect(tribelt?.share).toBe(0.5)
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
  it('reads what one answer costs per provider and mode from the plan', () => {
    const plan = { answerCosts: [{ provider: 'openai', mode: 'web' as const, costUsd: 0.021 }] }
    expect(modeCost(plan, 'openai', 'web')).toBe(0.021)
    expect(modeCost(plan, 'openai', 'knowledge')).toBe(0)
    expect(modeCost(undefined, 'openai', 'web')).toBe(0)
  })

  it('says when missing answers cannot be asked again', () => {
    expect(unknownLine(0)).toBeNull()
    expect(unknownLine(1)).toBe('1 missing answer cannot be asked again. This run asked an earlier set of questions, and no answer kept them.')
    expect(unknownLine(70)).toBe('70 missing answers cannot be asked again. This run asked an earlier set of questions, and no answer kept them.')
  })

  it('knows an answer that was given but not read', () => {
    expect(unread(promptAnswer({}, [], { failed: true }))).toBe(true)
    expect(unread(promptAnswer())).toBe(false)
    expect(unread(promptAnswer({ failure: 'Rate limited.', extraction: undefined }))).toBe(false)
    expect(unread(undefined)).toBe(false)
  })
})
