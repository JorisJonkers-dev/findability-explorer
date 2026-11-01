import { describe, expect, it } from 'vitest'
import { promptAnswer, promptRun } from '../../../testing/fixtures'
import {
  checkedBy,
  companiesNamed,
  FAMILY_OPTIONS,
  filtersFrom,
  gridOf,
  intentOf,
  isTribeltCompany,
  keyOf,
  leftOutOf,
  longDate,
  modelLabel,
  placeSentence,
  questionsOf,
  responsePath,
  responsesCsv,
  responsesFile,
  responseTitle,
  runDataLine,
  runLabel,
  scopeOptions,
  shown,
  sourcesOf,
  stateOf,
  tribeltPlace,
  wrongClaimsOf,
} from './responses'

const named = (place: string[]) => promptAnswer({}, place)

describe('Tribelt in a response', () => {
  it('is Tribelt by its id or by its name', () => {
    expect(isTribeltCompany({ name: 'Somebody', competitorId: 'tribelt', position: 1 })).toBe(true)
    expect(isTribeltCompany({ name: 'Tribelt B.V.', competitorId: 'twentebelt', position: 1 })).toBe(true)
    expect(isTribeltCompany({ name: 'tribelt.nl', position: 1 })).toBe(true)
    expect(isTribeltCompany({ name: 'Twentebelt', competitorId: 'twentebelt', position: 1 })).toBe(false)
  })

  it('has its place even when the reading files it under another company', () => {
    expect(tribeltPlace(named(['Twentebelt:twentebelt', 'Tribelt B.V.:ashworth']))).toBe(2)
    expect(tribeltPlace(named(['Twentebelt:twentebelt']))).toBeUndefined()
    expect(tribeltPlace(promptAnswer({ extraction: undefined }))).toBeUndefined()
  })

  it('gives each response one state', () => {
    expect(stateOf(named(['Tribelt:tribelt']))).toBe('first')
    expect(stateOf(named(['Twentebelt:twentebelt', 'Tribelt B.V.:twentebelt']))).toBe('named')
    expect(stateOf(promptAnswer({}, ['Twentebelt:twentebelt'], { namesTribelt: true }))).toBe('named')
    expect(stateOf(named(['Twentebelt:twentebelt']))).toBe('not')
    expect(stateOf(promptAnswer({ failure: 'Rate limited.', extraction: undefined }))).toBe('failed')
    expect(stateOf(promptAnswer({ extraction: undefined }))).toBe('unread')
    expect(stateOf(promptAnswer({}, [], { failed: true }))).toBe('unread')
  })

  it('says where Tribelt is', () => {
    expect(placeSentence(named(['Tribelt:tribelt']))).toBe('Tribelt is named first')
    expect(placeSentence(named(['Twentebelt:twentebelt', 'Tribelt B.V.:ashworth']))).toBe('Tribelt is named, number 2')
    expect(placeSentence(promptAnswer({}, [], { namesTribelt: true }))).toBe('Tribelt is named')
    expect(placeSentence(named([]))).toBe('Tribelt is not named')
    expect(placeSentence(promptAnswer({ failure: 'x', extraction: undefined }))).toBe('The response failed')
    expect(placeSentence(promptAnswer({ extraction: undefined }))).toBe('No model has read this response yet')
  })
})

describe('the words for a response', () => {
  it('names a model by its provider and the model', () => {
    expect(modelLabel('openai', 'openai/gpt-oss-20b')).toBe('OpenAI, gpt-oss-20b')
    expect(modelLabel('somebody', 'plain')).toBe('somebody, plain')
  })

  it('titles a response by its model and mode', () => {
    expect(responseTitle(promptAnswer({ mode: 'web' }))).toBe('OpenAI, gpt-oss-20b, with web search')
    expect(responseTitle(promptAnswer({ mode: 'knowledge' }))).toBe('OpenAI, gpt-oss-20b, from its own knowledge')
  })

  it('names the buyer intent and explains it', () => {
    expect(intentOf('application')).toEqual({ name: 'application', help: 'Buyer intent: application. The buyer describes what the belt is for, such as a bakery oven.' })
    expect(intentOf('supplier').name).toBe('product')
    expect(intentOf('problem').help).toBe('Buyer intent: problem. The buyer describes a problem to solve.')
  })

  it('writes dates in full, in Dutch time', () => {
    expect(longDate('2026-10-27T23:30:00Z')).toBe('28 October 2026')
  })

  it('leads to a response with its run', () => {
    expect(responsePath(promptAnswer({ id: 'a1' }), 'r1')).toEqual({ path: '/ai-responses/a1', query: { run: 'r1' } })
  })
})

describe('a run', () => {
  it('is labelled by its date, time, version, and how it ended', () => {
    const at = { startedAt: '2026-10-27T07:36:00Z', promptSetVersion: 2 }
    expect(runLabel(promptRun({ ...at, tasks: 200, answered: 200 }))).toBe('27 October 2026, 08:36, prompt set version 2. 200 responses')
    expect(runLabel(promptRun({ ...at, status: 'stopped', tasks: 1260, answered: 62 }))).toBe('27 October 2026, 08:36, prompt set version 2. Stopped after 62 of 1,260 responses')
    expect(runLabel(promptRun({ ...at, status: 'running', tasks: 200, answered: 3 }))).toBe('27 October 2026, 08:36, prompt set version 2. Running, 3 of 200 responses')
    expect(runLabel(promptRun({ ...at, status: 'failed', tasks: 200, answered: 0 }))).toBe('27 October 2026, 08:36, prompt set version 2. Failed, no responses')
    expect(runLabel(promptRun({ ...at, status: 'failed', tasks: 200, answered: 5 }))).toBe('27 October 2026, 08:36, prompt set version 2. Failed after 5 of 200 responses')
  })

  it('has a data line', () => {
    const run = promptRun({ startedAt: '2026-10-27T07:36:00Z', promptSetVersion: 2, tasks: 200, answered: 200 })
    expect(runDataLine(run)).toBe(
      'From the Prompt run of 27 October 2026: 50 questions to 2 models, each from the model’s own knowledge and with web search, 200 responses in all. Prompt set version 2.',
    )
    const one = promptRun({ models: [{ provider: 'openai', model: 'openai/gpt-oss-20b', free: false }], tasks: 2, answered: 1 })
    expect(runDataLine(one)).toBe(
      'From the Prompt run of 1 November 2026: 1 question to 1 model, each from the model’s own knowledge and with web search, 1 response in all. Prompt set version 1.',
    )
    expect(runDataLine(promptRun({ models: [] }))).toContain('0 questions to 0 models')
  })
})

describe('the filters of Every response', () => {
  it('reads the filters from the address, ignoring what it does not know', () => {
    expect(filtersFrom({})).toEqual({ mode: 'all', scope: 'all', family: 'all', model: 'all', names: 'all' })
    expect(filtersFrom({ mode: 'web', scope: 'twente', family: 'eyelink', model: 'openai', names: 'yes' })).toEqual({
      mode: 'web',
      scope: 'twente',
      family: 'eyelink',
      model: 'openai',
      names: 'yes',
    })
    expect(filtersFrom({ mode: 'radio', scope: ['twente'], family: 'belts', model: null, names: 'maybe' })).toEqual({
      mode: 'all',
      scope: 'all',
      family: 'all',
      model: 'all',
      names: 'all',
    })
  })

  it('keeps the responses each filter allows', () => {
    const list = [
      promptAnswer({ id: 'a', mode: 'web', scope: 'twente', family: 'eyelink', provider: 'openai' }, ['Tribelt:tribelt']),
      promptAnswer({ id: 'b', mode: 'knowledge', scope: 'german', family: 'open_spiral', provider: 'google' }, ['Twentebelt:twentebelt']),
      promptAnswer({ id: 'c', failure: 'x', extraction: undefined }),
    ]
    const all = filtersFrom({})
    const ids = (f: Partial<typeof all>) => shown(list, { ...all, ...f }).map((a) => a.id)
    expect(ids({})).toEqual(['a', 'b', 'c'])
    expect(ids({ mode: 'web' })).toEqual(['a'])
    expect(ids({ scope: 'german' })).toEqual(['b'])
    expect(ids({ family: 'eyelink' })).toEqual(['a', 'c'])
    expect(ids({ model: 'google' })).toEqual(['b'])
    expect(ids({ names: 'yes' })).toEqual(['a'])
    expect(ids({ names: 'no' })).toEqual(['b'])
  })

  it('offers the scopes the run asked, and every belt family', () => {
    expect(scopeOptions([promptAnswer({ scope: 'english' }), promptAnswer({ scope: 'twente' }), promptAnswer({ scope: 'twente' })])).toEqual([
      { value: 'all', label: 'All scopes' },
      { value: 'twente', label: 'Twente, in Dutch' },
      { value: 'english', label: 'English, no region' },
    ])
    expect(FAMILY_OPTIONS.map((f) => f.label)).toEqual(['All belt families', 'Eyelink belts', 'Open spiral belts', 'Closed spiral belts', 'Tri-Flex curve and spiral belts', 'Other metal belts'])
  })
})

describe('the grid', () => {
  it('has a column group per model with the modes under it, and a row per question', () => {
    const run = promptRun()
    const list = [
      promptAnswer({ promptId: 'q2', family: 'open_spiral', scope: 'twente', provider: 'google', mode: 'web' }),
      promptAnswer({ promptId: 'q1', family: 'eyelink', scope: 'german', provider: 'openai', mode: 'knowledge' }),
      promptAnswer({ promptId: 'q1', family: 'eyelink', scope: 'german', provider: 'openai', mode: 'web' }),
      promptAnswer({ promptId: 'q0', family: 'eyelink', scope: 'twente', kind: 'problem', provider: 'mistralai', model: 'mistralai/nemo', mode: 'web' }),
    ]
    const grid = gridOf(run, list)
    expect(grid.groups).toEqual([
      { provider: 'openai', label: 'OpenAI, gpt-oss-20b', modes: ['knowledge', 'web'] },
      { provider: 'google', label: 'Google, gemma:free', modes: ['web'] },
      { provider: 'mistralai', label: 'Mistral, nemo', modes: ['web'] },
    ])
    expect(grid.columns).toEqual([
      { provider: 'openai', mode: 'knowledge' },
      { provider: 'openai', mode: 'web' },
      { provider: 'google', mode: 'web' },
      { provider: 'mistralai', mode: 'web' },
    ])
    expect(grid.rows.map((r) => `${r.place}, ${r.intent.name}`)).toEqual(['Eyelink, Twente, problem', 'Eyelink, German, application', 'Open spiral, Twente, application'])
    expect(grid.rows[1]?.cells.map((c) => c?.mode)).toEqual(['knowledge', 'web', undefined, undefined])
  })

  it('orders questions of one family and scope by their buyer intent', () => {
    const list = [promptAnswer({ promptId: 'b', kind: 'problem' }), promptAnswer({ promptId: 'a', kind: 'supplier' })]
    expect(gridOf(promptRun(), list).rows.map((r) => r.promptId)).toEqual(['a', 'b'])
  })

  it('keys a response by its question, model, and mode', () => {
    expect(keyOf(promptAnswer({ promptId: 'q', provider: 'openai', mode: 'web' }))).toBe('q|openai|web')
  })
})

describe('one response', () => {
  it('lists the companies named in order, with Tribelt marked by its name', () => {
    const a = promptAnswer({}, ['Tribelt B.V.:ashworth', 'Twentebelt:twentebelt'])
    a.extraction?.companies.reverse()
    expect(companiesNamed(a)).toEqual([
      { position: 1, name: 'Tribelt B.V.', competitorId: undefined, tribelt: true },
      { position: 2, name: 'Twentebelt', competitorId: 'twentebelt', tribelt: false },
    ])
    expect(companiesNamed(promptAnswer({ extraction: undefined }))).toEqual([])
  })

  it('lists the sources, with tribelt.nl marked', () => {
    expect(sourcesOf(promptAnswer({ citations: ['https://x.test/'] }, [], { addresses: ['https://www.tribelt.nl/belts', 'https://europages.nl/'] }))).toEqual([
      { address: 'https://www.tribelt.nl/belts', tribelt: true },
      { address: 'https://europages.nl/', tribelt: false },
    ])
    expect(sourcesOf(promptAnswer({ citations: ['https://shop.tribelt.nl/'], extraction: undefined }))).toEqual([{ address: 'https://shop.tribelt.nl/', tribelt: true }])
  })

  it('has the claims the fact sheet contradicts and the facts it left out', () => {
    const read = promptAnswer({}, [], { wrongClaims: ['Tribelt is in Hengelo.'], leftOut: ['Tribelt makes eyelink belts to measure.'] })
    expect(wrongClaimsOf(read)).toEqual(['Tribelt is in Hengelo.'])
    expect(leftOutOf(read)).toEqual(['Tribelt makes eyelink belts to measure.'])
    expect(wrongClaimsOf(promptAnswer({ extraction: undefined }))).toEqual([])
    expect(leftOutOf(promptAnswer({ extraction: undefined }))).toEqual([])
  })

  it('says who checked it, or that nobody has', () => {
    expect(checkedBy(named(['Tribelt:tribelt']))).toBe('A model read this response to find the companies it names and to check it against the fact sheet.')
    expect(checkedBy(promptAnswer({ extraction: undefined }))).toBe('No model has checked this response yet.')
    expect(checkedBy(promptAnswer({}, [], { failed: true }))).toBe('No model has checked this response yet.')
  })
})

describe('the questions of a run', () => {
  const set = [
    { id: 'q2', family: 'open_spiral' as const, intent: 'problem' as const, scope: 'german', scopeName: 'German', language: 'de', text: 'Wer?', english: 'Who?' },
    { id: 'q1', family: 'eyelink' as const, intent: 'application' as const, scope: 'twente', scopeName: 'Twente (Dutch)', language: 'nl', region: 'Twente', text: 'Wie?', english: 'Who?' },
    { id: 'q3', family: 'eyelink' as const, intent: 'application' as const, scope: 'english', scopeName: 'English', language: 'en', text: 'Who?' },
  ]

  it('lists the published questions per belt family, each in its own language with its translation', () => {
    expect(questionsOf(set, [])).toEqual([
      {
        family: 'eyelink',
        label: 'Eyelink belts',
        questions: [
          { id: 'q1', text: 'Wie?', language: 'nl', where: 'Twente (Dutch)', english: 'Who?' },
          { id: 'q3', text: 'Who?', language: 'en', where: 'English', english: undefined },
        ],
      },
      { family: 'open_spiral', label: 'Open spiral belts', questions: [{ id: 'q2', text: 'Wer?', language: 'de', where: 'German', english: 'Who?' }] },
    ])
  })

  it('falls back on the questions the responses answer, once each, without a published set', () => {
    const list = [
      promptAnswer({ promptId: 'q1', family: 'eyelink', scope: 'twente', language: 'nl', prompt: 'Wie?' }),
      promptAnswer({ promptId: 'q1', family: 'eyelink', scope: 'twente', language: 'nl', prompt: 'Wie?', mode: 'web' }),
    ]
    expect(questionsOf(undefined, list)).toEqual([
      { family: 'eyelink', label: 'Eyelink belts', questions: [{ id: 'q1', text: 'Wie?', language: 'nl', where: 'Twente, in Dutch', english: undefined }] },
    ])
  })
})

describe('the CSV', () => {
  it('has a row per response with a header in the words of the page', () => {
    const csv = responsesCsv([
      promptAnswer({ prompt: 'Wie, "echt"?', mode: 'web', text: 'Line one\nline two' }, ['Twentebelt:twentebelt', 'Tribelt B.V.:x'], {
        addresses: ['https://www.tribelt.nl/'],
        citesTribelt: true,
      }),
      promptAnswer({ failure: 'Rate limited.', extraction: undefined, citations: ['https://a.test/'] }),
    ])
    const [header, first, second] = csv.split('\r\n')
    expect(header).toBe(
      "question id,belt family,scope,buyer intent,language,question,model,search mode,Tribelt's place,companies named,cites tribelt.nl,sources,input tokens,output tokens,cost in dollars,failure,response",
    )
    expect(first).toContain('"Wie, ""echt""?"')
    expect(first).toContain(',with web search,"Tribelt is named, number 2",Twentebelt; Tribelt B.V.,yes,https://www.tribelt.nl/,')
    expect(csv).toContain('"Line one\nline two"')
    expect(second).toContain(',The response failed,,no,https://a.test/,900,300,0.0012,Rate limited.,')
  })

  it('is named for the run’s day', () => {
    expect(responsesFile(promptRun({ startedAt: '2026-10-27T07:36:00Z' }))).toBe('responses-2026-10-27.csv')
  })
})
