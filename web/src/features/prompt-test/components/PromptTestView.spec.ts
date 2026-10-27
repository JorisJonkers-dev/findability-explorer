import type { VueWrapper } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { PromptAnswer } from '../../../infrastructure/api/types.gen'
import { competitor, me, promptAnswer, promptResults, promptRun, RUN_ID } from '../../../testing/fixtures'
import { type Handler, json, mountApp, problem, routes, settle } from '../../../testing/mountApp'

const OLDER = '1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d'

function answers(): PromptAnswer[] {
  const web = promptAnswer({ provider: 'openai', mode: 'web', citations: ['https://www.tribelt.nl/'] }, ['Twentebelt:twentebelt', 'Ashworth', 'Tribelt:tribelt'], {
    addresses: ['https://www.tribelt.nl/', 'https://www.europages.nl/'],
    citesTribelt: true,
    correctness: 'wrong',
    wrongClaims: ['Tribelt is in Hengelo.'],
  })
  return [
    promptAnswer({ provider: 'openai', mode: 'knowledge' }, ['Tribelt:tribelt', 'Twentebelt:twentebelt']),
    web,
    promptAnswer({ provider: 'google', mode: 'knowledge', failure: 'The model is rate limited.', extraction: undefined }),
    promptAnswer({ promptId: 'eyelink-application-german', scope: 'german', language: 'de', provider: 'google', mode: 'web', text: 'x '.repeat(200) }, ['Twentebelt:twentebelt']),
  ]
}

function api(admin = false, extra: Record<string, Handler> = {}) {
  return routes({
    'GET /api/v1/me': () => json(me({ isAdmin: admin })),
    'GET /api/v1/prompt-test': () => json(promptResults({ runs: [promptRun(), promptRun({ id: OLDER, startedAt: '2026-10-25T06:00:00Z', status: 'stopped' })] })),
    [`GET /api/v1/prompt-runs/${RUN_ID}/answers`]: () => json({ run: promptRun(), items: answers() }),
    [`GET /api/v1/prompt-runs/${OLDER}/answers`]: () => json({ run: promptRun({ id: OLDER }), items: [promptAnswer({}, ['Ashworth:ashworth'])] }),
    'GET /api/v1/competitors': () =>
      json({ items: [competitor('Twentebelt', { websiteUrl: 'https://www.twentebelt.com/' }), competitor('Tribelt B.V.', { id: 'tribelt', role: 'tribelt' })] }),
    ...extra,
  })
}

const squares = (wrapper: VueWrapper) => wrapper.findAll('table button')
const detail = (wrapper: VueWrapper) => wrapper.get('[aria-live="polite"]')

afterEach(() => {
  vi.restoreAllMocks()
})

describe('the Prompt test grid', () => {
  it('shows every question against every provider and mode, coloured by Tribelt’s place', async () => {
    const { wrapper } = await mountApp(api(), '/ai-search/prompt-test')
    expect(wrapper.get('h1').text()).toBe('Twente buyers hear of Tribelt. Everyone else hears of Twentebelt.')
    expect(wrapper.findAll('tbody th').map((th) => th.text())).toEqual(['EyelinkTwente (Dutch)', 'EyelinkGerman, no region'])
    expect(squares(wrapper).map((b) => b.text())).toEqual(['1', '3', '✕', '✕', '✕', '✕', '✕', '–'])
    expect(squares(wrapper)[1]?.attributes('aria-label')).toBe('Eyelink, Twente (Dutch), OpenAI with web search: Tribelt named 3')
    expect(squares(wrapper)[2]?.attributes('aria-label')).toBe('Eyelink, Twente (Dutch), Google from its own knowledge: No answer')
    expect(wrapper.get('ul[aria-label="Legend"]').text()).toContain('Tribelt named first')
    expect(wrapper.findAll('dl > div').map((d) => `${d.get('dd').text()} ${d.get('dt').text()}`)).toEqual([
      '67% of answers name Tribelt',
      '100% when the buyer asks for Twente',
      '1 of 2 web answers cite tribelt.nl',
    ])
    expect(wrapper.text()).toContain('360 of 360 answers, 2 without an answer. Done. Cost $0.42.')
  })

  it('narrows to a mode and a scope, and keeps them in the address', async () => {
    const { wrapper, router } = await mountApp(api(), '/ai-search/prompt-test')
    await wrapper.get('[aria-label="Search mode"] button:nth-child(3)').trigger('click')
    await settle()
    expect(router.currentRoute.value.query).toEqual({ mode: 'web' })
    expect(squares(wrapper)).toHaveLength(4)
    await wrapper.findAll('[aria-label="Where the buyer asks"] button').find((b) => b.text() === 'German')?.trigger('click')
    await settle()
    expect(router.currentRoute.value.query).toEqual({ mode: 'web', scope: 'german' })
    expect(wrapper.findAll('tbody tr')).toHaveLength(1)
    await wrapper.findAll('[aria-label="Search mode"] button')[0]?.trigger('click')
    await settle()
    expect(router.currentRoute.value.query).toEqual({ scope: 'german' })
  })

  it('opens an answer: the companies it names with their websites, its sources, model, tokens and cost', async () => {
    const { wrapper, router } = await mountApp(api(), '/ai-search/prompt-test')
    expect(detail(wrapper).text()).toBe('Pick a square to read its answer.')
    await squares(wrapper)[1]?.trigger('click')
    await settle()
    expect(router.currentRoute.value.query.answer).toBe('eyelink-application-twente|openai|web')
    expect(squares(wrapper)[1]?.attributes('aria-pressed')).toBe('true')
    const d = detail(wrapper)
    expect(d.text()).toContain('OpenAI with web search. Eyelink, Twente (Dutch)')
    const names = d.findAll('ol[aria-label="Companies named"] li')
    expect(names.map((n) => n.text())).toEqual(['1Twentebeltits page', '2Ashworthnot a competitor yet', '3Tribelt B.V.'])
    expect(names[0]?.get('a[target="_blank"]').attributes('href')).toBe('https://www.twentebelt.com/')
    expect(names[2]?.classes()).toContain('text-orange-ink')
    expect(d.findAll('ul[aria-label="Sources cited"] li').map((l) => l.text())).toEqual(['tribelt.nl', 'europages.nl'])
    expect(d.text()).toMatch(/Wrong about Tribelt\.\s+“Tribelt is in Hengelo\.”/)
    expect(d.text()).toContain('openai/gpt-oss-20b. 1,200 tokens. $0.0012.')
  })

  it('shows the whole of a long answer on demand, and says when there is none', async () => {
    const { wrapper } = await mountApp(api(), '/ai-search/prompt-test?answer=eyelink-application-german|google|web')
    expect(detail(wrapper).text()).toContain('…”')
    await detail(wrapper).get('button').trigger('click')
    expect(detail(wrapper).text()).not.toContain('…”')
    expect(detail(wrapper).get('button').text()).toBe('Show less')
    await squares(wrapper)[2]?.trigger('click')
    await settle()
    expect(detail(wrapper).text()).toContain('No answer to read. The model is rate limited.')
  })

  it('opens an earlier run from the picker', async () => {
    const { wrapper, router } = await mountApp(api(), '/ai-search/prompt-test?answer=x')
    const select = wrapper.get('select')
    expect(select.findAll('option').map((o) => o.text())).toEqual(['1 Nov 2026 (done)', '25 Oct 2026 (stopped at the budget)'])
    await select.setValue(OLDER)
    await settle()
    expect(router.currentRoute.value.query).toEqual({ run: OLDER })
    expect(wrapper.get('h1').text()).toBe('AI answers name ashworth most.')
  })

  it("downloads the run's answers as CSV", async () => {
    const created = vi.fn<(blob: Blob) => string>(() => 'blob:csv')
    vi.stubGlobal('URL', Object.assign(URL, { createObjectURL: created, revokeObjectURL: vi.fn() }))
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => undefined)
    const { wrapper } = await mountApp(api(), '/ai-search/prompt-test')
    await wrapper.findAll('button').find((b) => b.text() === 'Download as CSV')?.trigger('click')
    expect(click).toHaveBeenCalledOnce()
    const csv = await (created.mock.calls[0]?.[0] as Blob).text()
    expect(csv.split('\r\n')).toHaveLength(5)
    expect(csv).toContain('eyelink-application-german')
  })

  it('lets an administrator start a run', async () => {
    let started = false
    const { wrapper } = await mountApp(
      api(true, { 'POST /api/v1/admin/prompt-runs': () => ((started = true), json(promptRun({ status: 'running' }), 202)) }),
      '/ai-search/prompt-test',
    )
    await wrapper.findAll('button').find((b) => b.text() === 'Run now')?.trigger('click')
    await settle()
    expect(started).toBe(true)
    expect(wrapper.text()).toContain('A Prompt run started: 360 answers to ask.')
  })

  it('offers no run button to a viewer', async () => {
    const { wrapper } = await mountApp(api(), '/ai-search/prompt-test')
    expect(wrapper.findAll('button').some((b) => b.text() === 'Run now')).toBe(false)
  })
})

describe('asking again', () => {
  const retried = (body: unknown) => ({ run: promptRun({ status: 'running' }), tasks: (body as { promptId?: string }).promptId ? 1 : 2, estimatedCostUsd: 0.0024 })

  it('asks every missing answer again, with the cost shown first', async () => {
    let sent: unknown = null
    const { wrapper } = await mountApp(
      api(true, {
        [`POST /api/v1/admin/prompt-runs/${RUN_ID}/retry`]: async (r) => {
          sent = await r.json()
          return json(retried(sent), 202)
        },
      }),
      '/ai-search/prompt-test',
    )
    const all = wrapper.findAll('button').find((b) => b.text().startsWith('Ask the'))
    // 360 tasks over two providers: 180 each; OpenAI gave 2 answers and Google 1.
    expect(all?.text()).toBe('Ask the 357 missing answers again, about $0.43')
    await all?.trigger('click')
    await settle()
    expect(sent).toEqual({})
    expect(wrapper.text()).toContain('Asking 2 answers again, for about $0.00.')
  })

  it('asks one failed answer again from its detail', async () => {
    let sent: unknown = null
    const { wrapper } = await mountApp(
      api(true, {
        [`POST /api/v1/admin/prompt-runs/${RUN_ID}/retry`]: async (r) => {
          sent = await r.json()
          return json(retried(sent), 202)
        },
      }),
      '/ai-search/prompt-test?answer=eyelink-application-twente|google|knowledge',
    )
    const one = detail(wrapper).findAll('button').find((b) => b.text().startsWith('Ask Google again'))
    expect(one?.text()).toBe('Ask Google again, about $0.0012')
    await one?.trigger('click')
    await settle()
    expect(sent).toEqual({ promptId: 'eyelink-application-twente', provider: 'google', mode: 'knowledge' })
    expect(wrapper.text()).toContain('Asking one answer again')
  })

  it('says why it cannot, and offers nothing to a viewer', async () => {
    const { wrapper } = await mountApp(
      api(true, { [`POST /api/v1/admin/prompt-runs/${RUN_ID}/retry`]: () => problem(409, 'Conflict', "The month's budget for Prompt runs is reached.") }),
      '/ai-search/prompt-test',
    )
    await wrapper.findAll('button').find((b) => b.text().startsWith('Ask the'))?.trigger('click')
    await settle()
    expect(wrapper.get('[role="alert"]').text()).toBe("The month's budget for Prompt runs is reached.")
    wrapper.unmount()
    const viewer = await mountApp(api(), '/ai-search/prompt-test?answer=eyelink-application-twente|google|knowledge')
    expect(viewer.wrapper.findAll('button').some((b) => b.text().startsWith('Ask'))).toBe(false)
  })
})

describe('before and around the grid', () => {
  it('says when no run has happened yet', async () => {
    const { wrapper } = await mountApp(api(true, { 'GET /api/v1/prompt-test': () => json(promptResults({ runs: [] })) }), '/ai-search/prompt-test')
    expect(wrapper.get('h1').text()).toBe('Do AI assistants name Tribelt?')
    expect(wrapper.text()).toContain('No Prompt run yet.')
    expect(wrapper.find('a[href="/settings/prompt-test"]').exists()).toBe(true)
  })

  it('says when the results or the answers cannot be loaded', async () => {
    const broken = await mountApp(api(false, { 'GET /api/v1/prompt-test': () => problem(500, 'Internal Server Error', 'Down.') }), '/ai-search/prompt-test')
    expect(broken.wrapper.get('[role="alert"]').text()).toContain('Down.')
    broken.wrapper.unmount()
    const answersDown = await mountApp(api(false, { [`GET /api/v1/prompt-runs/${RUN_ID}/answers`]: () => problem(500, 'Internal Server Error', 'Gone.') }), '/ai-search/prompt-test')
    expect(answersDown.wrapper.get('[role="alert"]').text()).toContain('Gone.')
  })
})

describe('a company an answer names', () => {
  it('can be added as a competitor by an administrator, from the answer', async () => {
    let sent: unknown = null
    const named = promptAnswer({ provider: 'openai', mode: 'web' }, ['Twentebelt:twentebelt', 'Intralox'])
    const withUrl = { ...named, extraction: { ...(named.extraction as NonNullable<PromptAnswer['extraction']>), companies: [{ name: 'Intralox', url: 'https://www.intralox.com', position: 1 }, { name: 'Habasit', position: 2 }] } }
    const { wrapper } = await mountApp(
      api(true, {
        [`GET /api/v1/prompt-runs/${RUN_ID}/answers`]: () => json({ run: promptRun(), items: [withUrl] }),
        'POST /api/v1/admin/competitors/batch': async (r) => {
          sent = await r.json()
          return json({ items: [{ name: 'Intralox', outcome: 'added', competitorId: 'intralox' }], scans: 0 })
        },
      }),
      '/ai-search/prompt-test?answer=eyelink-application-twente|openai|web',
    )
    const names = wrapper.findAll('ol[aria-label="Companies named"] li')
    expect(names[1]?.find('a[href="/competitors/all"]').exists()).toBe(true)
    await names[0]?.get('button').trigger('click')
    await settle()
    expect(sent).toEqual({ items: [{ name: 'Intralox', url: 'https://www.intralox.com' }], scan: false })
    expect(wrapper.text()).toContain('Added 1 competitor.')
  })
})
