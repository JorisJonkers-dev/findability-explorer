import { type VueWrapper } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { AnalysisStatus } from '../../../infrastructure/api/types.gen'
import { ANALYSIS_ID, analysis, competitor, me, overview, promptResults, readinessRow, readinessTable, summary, term, termSearch } from '../../../testing/fixtures'
import { type Handler, json, mountApp, problem, routes, settle } from '../../../testing/mountApp'

const NEW_ID = '6c1d2b0f-3e4a-4f6b-9c7d-8e9f0a1b2c3d'

const twentebelt = competitor('Twentebelt', { scans: 2 })
const martens = competitor('Martens', { country: 'DE', websiteUrl: 'https://www.martens-belts.com/', host: 'martens-belts.com', website: 'https://www.martens-belts.com' })
const ashworth = competitor('Ashworth', { country: 'US', websiteUrl: undefined, host: undefined, website: '', beltNames: [] })
const tribelt = competitor('Tribelt', { role: 'tribelt', groupCompany: true, websiteUrl: 'https://www.tribelt.nl/', host: 'tribelt.nl', website: 'https://www.tribelt.nl' })
const sister = competitor('Tribelt Group', { role: 'group', groupCompany: true, websiteUrl: 'https://group.example/', host: 'group.example', website: 'https://group.example' })

/** The server, signed in as an administrator; the scan started answers with each status in line, the last one for good. */
function api(extra: Record<string, Handler> = {}, statuses: AnalysisStatus[] = ['queued', 'done']) {
  return routes({
    'GET /api/v1/me': () => json(me({ isAdmin: true })),
    'GET /api/v1/competitors': () => json({ items: [twentebelt, martens, ashworth, tribelt, sister] }),
    'GET /api/v1/terms': () => json(termSearch([term({ term: 'TwenteFlex', tier: 'product', usedBy: ['Twentebelt (TwenteFlex)'] })], { tribeltFrom: 'scan' })),
    'GET /api/v1/overview': () => json(overview({ competitors: { makers: 3, families: [{ family: 'eyelink', makers: 2 }, { family: 'open_spiral', makers: 1 }] } })),
    'GET /api/v1/prompt-test': () => json(promptResults()),
    'GET /api/v1/ai-readiness': () => json(readinessTable([readinessRow('Twentebelt'), readinessRow('Tribelt', { role: 'tribelt' })])),
    'GET /api/v1/competitors/twentebelt': () => json({ competitor: twentebelt, scans: [summary()] }),
    'GET /api/v1/analyses': () => json({ items: [] }),
    'POST /api/v1/analyses': () => json(analysis({}, { id: NEW_ID, status: 'queued', pagesCrawled: 0 }), 202),
    [`GET /api/v1/analyses/${NEW_ID}`]: () => {
      const status = statuses.length > 1 ? statuses.shift() : statuses[0]
      return json(analysis({}, { id: NEW_ID, status: status ?? 'done', pagesCrawled: status === 'queued' ? 0 : 15 }))
    },
    [`GET /api/v1/analyses/${ANALYSIS_ID}`]: () => json(analysis()),
    ...extra,
  })
}

const combobox = (wrapper: VueWrapper) => wrapper.get('input[role="combobox"]')
const list = (wrapper: VueWrapper) => wrapper.get('section[aria-labelledby="list-title"]')

async function type(wrapper: VueWrapper, text: string) {
  await combobox(wrapper).setValue(text)
  await combobox(wrapper).trigger('input')
}

async function submit(wrapper: VueWrapper) {
  await wrapper.get('form[aria-label="Scan a website"]').trigger('submit')
  await settle()
}

afterEach(() => {
  vi.useRealTimers()
})

describe('the competitors page', () => {
  it('says how many competitors sell the belts, with the key figures and the list', async () => {
    const { wrapper } = await mountApp(api({ 'GET /api/v1/me': () => json(me()) }), '/competitors')
    expect(wrapper.get('h1').text()).toBe('3 competitors sell metal belts that Tribelt’s buyers could buy instead.')
    const figures = wrapper.get('section[aria-label="Key figures"]').text()
    expect(figures).toContain('3competitors on the list')
    expect(figures).toContain('1is named in more AI answers than TribeltTwentebelt leads.')
    expect(figures).toContain('1shares Tribelt’s AI readiness score of 8 of 8Among them Twentebelt.')
    expect(list(wrapper).findAll('li a').map((a) => a.attributes('href'))).toEqual(['/competitors/twentebelt', '/competitors/ashworth', '/competitors/martens'])
    expect(list(wrapper).text()).toContain('Tribelt Group is left out')
    expect(wrapper.get('section[aria-labelledby="family-title"]').text()).toContain('Eyelink')
  })

  it('leaves the tools to administrators', async () => {
    const viewer = await mountApp(api({ 'GET /api/v1/me': () => json(me()) }), '/competitors')
    expect(viewer.wrapper.find('#admin-tools-title').exists()).toBe(false)
    expect(viewer.wrapper.find('form[aria-label="Scan a website"]').exists()).toBe(false)
    viewer.wrapper.unmount()
    const admin = await mountApp(api(), '/competitors')
    expect(admin.wrapper.get('#admin-tools-title').text()).toBe('Manage the list')
    expect(admin.wrapper.find('form[aria-label="Scan a website"]').exists()).toBe(true)
    expect(admin.wrapper.find('#upkeep-title').exists()).toBe(true)
  })

  it('leaves out the chart when the server counts no families', async () => {
    const { wrapper } = await mountApp(api({ 'GET /api/v1/overview': () => json(overview({ competitors: { makers: 3, families: [] } })) }), '/competitors')
    expect(wrapper.find('section[aria-labelledby="family-title"]').exists()).toBe(false)
  })

  it('says while it loads, and when the companies cannot be loaded', async () => {
    let answer: (response: Response) => void = () => undefined
    const pending = new Promise<Response>((resolve) => {
      answer = resolve
    })
    const loading = await mountApp(api({ 'GET /api/v1/competitors': () => pending }), '/competitors')
    expect(loading.wrapper.text()).toContain('Loading the competitors…')
    answer(json({ items: [twentebelt] }))
    await settle()
    expect(loading.wrapper.text()).not.toContain('Loading the competitors…')
    loading.wrapper.unmount()
    const failing = await mountApp(api({ 'GET /api/v1/competitors': () => problem(500, 'Internal Server Error') }), '/competitors')
    expect(failing.wrapper.get('[role="alert"]').text()).toBe('Internal Server Error')
    expect(failing.wrapper.find('section[aria-label="Key figures"]').exists()).toBe(false)
  })

  it('opens a started scan on its own page', async () => {
    const { wrapper, router } = await mountApp(api(), '/competitors')
    await type(wrapper, 'https://www.example-belts.de')
    await submit(wrapper)
    expect(router.currentRoute.value.fullPath).toBe(`/scan/${NEW_ID}`)
    expect(wrapper.get('h1').text()).toBe('What the scan found.')
  })

  it('opens an added company on its own page', async () => {
    const added = competitor('New Belts', { source: 'added' })
    const { wrapper, router } = await mountApp(
      api({
        'POST /api/v1/competitors': () => json(added, 201),
        'GET /api/v1/competitors/new-belts': () => json({ competitor: added, scans: [] }),
      }),
      '/competitors',
    )
    await wrapper.findAll('button').find((b) => b.text() === 'Add a company')?.trigger('click')
    const form = wrapper.get('form[aria-label="Add a company"]')
    await form.findAll('input')[0]?.setValue('New Belts')
    await form.findAll('input')[1]?.setValue('new-belts.com')
    await form.trigger('submit')
    await settle()
    expect(router.currentRoute.value.fullPath).toBe('/competitors/new-belts')
  })
})

describe('the scan form', () => {
  it('checks the address before sending it, and clears the message on typing', async () => {
    const { wrapper, requests } = await mountApp(api(), '/competitors')
    await type(wrapper, 'https://intranet')
    await submit(wrapper)
    expect(wrapper.get('#scan-url-error').text()).toBe('Enter an address with a full host name, like https://www.example.com.')
    expect(combobox(wrapper).attributes('aria-invalid')).toBe('true')
    expect(combobox(wrapper).attributes('aria-describedby')).toBe('scan-url-error')
    expect(requests().filter((r) => r.startsWith('POST'))).toEqual([])
    await type(wrapper, 'https://www.')
    expect(wrapper.find('#scan-url-error').exists()).toBe(false)
  })

  it("puts the server's refusal under the address or under the form", async () => {
    let status = 422
    const refuse = { 'POST /api/v1/analyses': () => problem(status, 'Refused', status === 422 ? 'The host is not public.' : 'A scan of this company is still queued or running.') }
    const { wrapper } = await mountApp(api(refuse), '/competitors')
    await type(wrapper, 'https://www.example.com')
    await submit(wrapper)
    expect(wrapper.get('#scan-url-error').text()).toBe('The host is not public.')
    status = 409
    await submit(wrapper)
    expect(wrapper.find('#scan-url-error').exists()).toBe(false)
    expect(wrapper.get('[role="alert"]').text()).toBe('A scan of this company is still queued or running.')
  })

  it('suggests known companies as you type, and picks one with the keyboard', async () => {
    const { wrapper } = await mountApp(api(), '/competitors')
    expect(combobox(wrapper).attributes('aria-expanded')).toBe('false')
    await type(wrapper, 'twente')
    expect(combobox(wrapper).attributes('aria-expanded')).toBe('true')
    expect(wrapper.findAll('[role="option"]').map((o) => o.text())).toEqual(['Twentebelttwentebelt.com'])
    await combobox(wrapper).trigger('keydown', { key: 'ArrowDown' })
    expect(combobox(wrapper).attributes('aria-activedescendant')).toBe('scan-url-option-0')
    await combobox(wrapper).trigger('keydown', { key: 'ArrowUp' })
    await combobox(wrapper).trigger('keydown', { key: 'ArrowUp' })
    expect(combobox(wrapper).attributes('aria-activedescendant')).toBe('scan-url-option-0')
    await combobox(wrapper).trigger('keydown', { key: 'Enter' })
    expect((combobox(wrapper).element as HTMLInputElement).value).toBe('https://www.twentebelt.com/')
    expect(combobox(wrapper).attributes('aria-expanded')).toBe('false')
    expect(wrapper.get('#scan-url-company').text()).toBe('Twentebelt, a known competitor. Scanned 2 times.')
  })

  it('closes the suggestions on Escape or blur, picks one with the mouse, and ignores keys with nothing to suggest', async () => {
    const { wrapper } = await mountApp(api(), '/competitors')
    await type(wrapper, 'zzz')
    await combobox(wrapper).trigger('keydown', { key: 'ArrowDown' })
    await combobox(wrapper).trigger('keydown', { key: 'ArrowUp' })
    await combobox(wrapper).trigger('keydown', { key: 'Enter' })
    await combobox(wrapper).trigger('keydown', { key: 'Escape' })
    expect(combobox(wrapper).attributes('aria-expanded')).toBe('false')
    await type(wrapper, 'tribelt')
    await combobox(wrapper).trigger('keydown', { key: 'Escape' })
    expect(combobox(wrapper).attributes('aria-expanded')).toBe('false')
    await combobox(wrapper).trigger('keydown', { key: 'ArrowDown' })
    expect(combobox(wrapper).attributes('aria-expanded')).toBe('true')
    await combobox(wrapper).trigger('blur')
    expect(combobox(wrapper).attributes('aria-expanded')).toBe('false')
    await type(wrapper, 'märtens')
    await wrapper.get('[role="option"]').trigger('mousedown')
    expect((combobox(wrapper).element as HTMLInputElement).value).toBe('https://www.martens-belts.com/')
    await type(wrapper, 'https://www.tribelt.nl')
    expect(wrapper.get('#scan-url-company').text()).toContain("Tribelt, in Tribelt's own group.")
  })

  it('names an unknown company after its host, lets the name be changed, and sends it', async () => {
    const sentBodies: unknown[] = []
    const { wrapper } = await mountApp(
      api({
        'POST /api/v1/analyses': async (request) => {
          sentBodies.push(await request.json())
          return json(analysis({}, { id: NEW_ID, status: 'queued' }), 202)
        },
      }),
      '/competitors',
    )
    await type(wrapper, 'www.example-belts.de')
    const name = wrapper.get('#scan-url-company input')
    expect((name.element as HTMLInputElement).value).toBe('Example Belts')
    expect(wrapper.text()).toContain('The scan adds it as a new competitor.')
    await name.setValue('Example Belts GmbH')
    await name.trigger('input')
    await type(wrapper, 'www.example-belts.de/')
    expect((name.element as HTMLInputElement).value).toBe('Example Belts GmbH')
    await submit(wrapper)
    expect(sentBodies).toEqual([{ url: 'https://www.example-belts.de/', maxPages: 60, companyName: 'Example Belts GmbH' }])
  })
})
