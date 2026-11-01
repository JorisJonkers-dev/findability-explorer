import type { VueWrapper } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { PromptAnswer } from '../../../infrastructure/api/types.gen'
import { me, promptAnswer, promptResults, promptRun, RUN_ID } from '../../../testing/fixtures'
import { type Handler, json, mountApp, problem, routes, settle } from '../../../testing/mountApp'
import { words } from '../../../testing/words'

const OLDER = '1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d'
const FIRST = '00000000-0000-4000-8000-0000000000b1'

function answers(): PromptAnswer[] {
  return [
    promptAnswer({ id: FIRST, provider: 'openai', mode: 'knowledge' }, ['Tribelt:tribelt', 'Twentebelt:twentebelt']),
    promptAnswer({ provider: 'openai', mode: 'web' }, ['Twentebelt:twentebelt', 'Ashworth', 'Tribelt B.V.:ashworth']),
    promptAnswer({ provider: 'google', model: 'google/gemma:free', mode: 'knowledge', failure: 'The model is rate limited.', extraction: undefined }),
    promptAnswer({ promptId: 'eyelink-application-german', scope: 'german', language: 'de', provider: 'google', model: 'google/gemma:free', mode: 'web' }, [
      'Twentebelt:twentebelt',
    ]),
  ]
}

function api(extra: Record<string, Handler> = {}) {
  return routes({
    'GET /api/v1/prompt-test': () =>
      json(promptResults({ runs: [promptRun({ startedAt: '2026-11-01T07:36:00Z' }), promptRun({ id: OLDER, startedAt: '2026-10-25T06:00:00Z', status: 'stopped', answered: 62 })] })),
    [`GET /api/v1/prompt-runs/${RUN_ID}/answers`]: () => json({ run: promptRun(), items: answers() }),
    [`GET /api/v1/prompt-runs/${OLDER}/answers`]: () => json({ run: promptRun({ id: OLDER }), items: [promptAnswer({}, ['Ashworth:ashworth'])] }),
    ...extra,
  })
}

const squares = (wrapper: VueWrapper) => wrapper.findAll('tbody a')

afterEach(() => {
  vi.restoreAllMocks()
})

describe('Every response', () => {
  it('has the frame, with the way back to AI responses', async () => {
    const { wrapper } = await mountApp(api(), '/ai-responses/all')
    expect(wrapper.get('h1').text()).toBe('Every response')
    expect(wrapper.get('[data-part="purpose"]').text()).toBe(
      'Each square is one model’s response to one question. Its colour shows whether the response names Tribelt, and where.',
    )
    expect(wrapper.get('[data-part="data-line"]').text()).toContain('358 responses in all. Prompt set version 1.')
    expect(wrapper.findAll('a[href="/ai-responses"]').map((a) => a.text())).toContain('Back to AI responses')
    expect(wrapper.text()).not.toMatch(/\banswers?\b/i)
  })

  it('shows models as column groups with the two modes under each, and a row per question', async () => {
    const { wrapper } = await mountApp(api(), '/ai-responses/all')
    const groups = wrapper.findAll('thead th[scope="colgroup"]')
    expect(groups.map((g) => [g.text(), g.attributes('colspan')])).toEqual([
      ['OpenAI, gpt-oss-20b', '2'],
      ['Google, gemma:free', '2'],
    ])
    expect(wrapper.findAll('thead th[scope="col"]').map((th) => th.text())).toEqual(['Question', 'Own knowledge', 'Web search', 'Own knowledge', 'Web search'])
    expect(wrapper.findAll('tbody th').map(words)).toEqual([
      'Eyelink, Twente, application . Buyer intent: application. The buyer describes what the belt is for, such as a bakery oven.',
      'Eyelink, German, application . Buyer intent: application. The buyer describes what the belt is for, such as a bakery oven.',
    ])
    expect(squares(wrapper).map((a) => a.attributes('aria-label'))).toEqual([
      'OpenAI, gpt-oss-20b, from its own knowledge. Tribelt is named first.',
      'OpenAI, gpt-oss-20b, with web search. Tribelt is named, number 3.',
      'Google, gemma:free, from its own knowledge. The response failed.',
      'Google, gemma:free, with web search. Tribelt is not named.',
    ])
    expect(squares(wrapper)[0]?.attributes('href')).toBe(`/ai-responses/${FIRST}?run=${RUN_ID}`)
  })

  it('has the key inside the grid’s panel, with how many responses are shown', async () => {
    const { wrapper } = await mountApp(api(), '/ai-responses/all')
    const panel = wrapper.get('section[aria-label="Every response"]')
    expect(panel.get('ul[aria-label="What the squares mean"]').findAll('li').map(words)).toEqual([
      'Tribelt is named first.',
      'Tribelt is named, but not first.',
      'Tribelt is not named.',
      'The response failed.',
    ])
    expect(panel.get('[data-part="shown"]').text()).toMatch(/^4 responses shown\. A square opens the response\./)
  })

  it('filters by search mode, scope, belt family, model, and whether Tribelt is named, in the address', async () => {
    const { wrapper, router } = await mountApp(api(), '/ai-responses/all')
    const select = (label: string) => {
      const found = wrapper.findAll('label').find((l) => l.text().startsWith(label))
      if (!found) throw new Error(label)
      return found.get('select')
    }
    await select('Search mode').setValue('web')
    await settle()
    expect(router.currentRoute.value.query).toEqual({ mode: 'web' })
    expect(squares(wrapper)).toHaveLength(2)
    await select('Model').setValue('google')
    await settle()
    expect(router.currentRoute.value.query).toEqual({ mode: 'web', model: 'google' })
    expect(wrapper.get('[data-part="shown"]').text()).toMatch(/^1 response shown/)
    await select('Search mode').setValue('all')
    await settle()
    await select('Model').setValue('all')
    await settle()
    await select('Scope').setValue('german')
    await settle()
    expect(router.currentRoute.value.query).toEqual({ scope: 'german' })
    expect(squares(wrapper)).toHaveLength(1)
    await select('Belt family').setValue('open_spiral')
    await settle()
    expect(wrapper.text()).toContain('No response matches these filters. Widen a filter to see more.')
  })

  it('narrows to the responses that name Tribelt, with the toggle pressed', async () => {
    const { wrapper, router } = await mountApp(api(), '/ai-responses/all?names=yes')
    const yes = wrapper.findAll('[role="group"] button').find((b) => b.text() === 'Yes')
    expect(yes?.attributes('aria-pressed')).toBe('true')
    expect(squares(wrapper)).toHaveLength(2)
    await wrapper.findAll('[role="group"] button').find((b) => b.text() === 'No')?.trigger('click')
    await settle()
    expect(router.currentRoute.value.query).toEqual({ names: 'no' })
    expect(squares(wrapper)).toHaveLength(1)
  })

  it('picks a run by date and Prompt set version, with a stopped run marked', async () => {
    const { wrapper, router, requests } = await mountApp(api(), '/ai-responses/all')
    const runs = wrapper.findAll('label')[0]?.findAll('option').map((o) => o.text())
    expect(runs).toEqual(['1 November 2026, 08:36, prompt set version 1. 358 responses', '25 October 2026, 07:00, prompt set version 1. Stopped after 62 of 360 responses'])
    await wrapper.findAll('label')[0]?.get('select').setValue(OLDER)
    await settle()
    expect(router.currentRoute.value.query).toEqual({ run: OLDER })
    expect(requests()).toContain(`GET /api/v1/prompt-runs/${OLDER}/answers`)
  })

  it('downloads the shown responses as CSV, named for responses', async () => {
    const created = vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:x')
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => undefined)
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => undefined)
    const { wrapper } = await mountApp(api(), '/ai-responses/all?mode=web')
    await wrapper.findAll('button').find((b) => b.text() === 'Download as CSV')?.trigger('click')
    expect(click).toHaveBeenCalledOnce()
    expect((click.mock.contexts[0] as HTMLAnchorElement).download).toBe('responses-2026-11-01.csv')
    const blob = created.mock.calls[0]?.[0] as Blob
    const text = await blob.text()
    expect(text.split('\r\n')[0]).toContain(',response')
    expect(text.split('\r\n')).toHaveLength(3)
  })

  it('says so before the first run, and reports failures', async () => {
    const none = await mountApp(api({ 'GET /api/v1/prompt-test': () => json(promptResults({ runs: [] })), 'GET /api/v1/me': () => json(me({ isAdmin: true })) }), '/ai-responses/all')
    expect(none.wrapper.text()).toContain('No Prompt run has finished yet.')
    expect(none.wrapper.find('a[href="/manage/prompt-test"]').exists()).toBe(true)
    const down = await mountApp(api({ 'GET /api/v1/prompt-test': () => problem(500, 'Internal Server Error', 'Down.') }), '/ai-responses/all')
    expect(down.wrapper.get('[role="alert"]').text()).toContain('Down.')
    const gone = await mountApp(api({ [`GET /api/v1/prompt-runs/${RUN_ID}/answers`]: () => problem(500, 'Internal Server Error', 'Gone.') }), '/ai-responses/all')
    expect(gone.wrapper.get('[role="alert"]').text()).toContain('Gone.')
  })

  it('shows a response no model has read in the key', async () => {
    const { wrapper } = await mountApp(
      api({ [`GET /api/v1/prompt-runs/${RUN_ID}/answers`]: () => json({ run: promptRun(), items: [promptAnswer({ extraction: undefined })] }) }),
      '/ai-responses/all',
    )
    expect(wrapper.get('ul[aria-label="What the squares mean"]').text()).toContain('No model has read the response yet.')
    expect(wrapper.get('[data-part="shown"]').text()).toMatch(/^1 response shown/)
  })
})
