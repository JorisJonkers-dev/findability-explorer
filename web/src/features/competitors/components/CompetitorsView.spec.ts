import { flushPromises, type VueWrapper } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Analysis, AnalysisStatus } from '../../../infrastructure/api/types.gen'
import { ANALYSIS_ID, analysis, competitor, summary, term, termSearch } from '../../../testing/fixtures'
import { type Handler, json, mountApp, problem, routes, settle } from '../../../testing/mountApp'
import { POLL_INTERVAL_MS } from '../../scan/domain/status'

const NEW_ID = '6c1d2b0f-3e4a-4f6b-9c7d-8e9f0a1b2c3d'

const twentebelt = competitor('Twentebelt', {
  scans: 2,
  beltNames: [
    { name: 'TwenteFlex spiral belts', family: 'tri_flex_turn_curve' },
    { name: 'TwenteSideFlex', family: 'tri_flex_turn_curve' },
  ],
})
const martens = competitor('Märtens Transportbänder GmbH', {
  id: 'martens',
  country: 'DE',
  websiteUrl: 'https://www.martens-belts.com/',
  host: 'martens-belts.com',
  website: 'https://www.martens-belts.com',
  languages: ['de', 'en'],
  beltNames: [{ name: 'Drahtösengliederband', family: 'eyelink' }],
})
const ashworth = competitor('Ashworth', { id: 'ashworth', country: 'US', websiteUrl: undefined, host: undefined, website: '', beltNames: [] })
const tribelt = competitor('Tribelt B.V.', { id: 'tribelt-b-v', groupCompany: true, websiteUrl: 'https://www.tribelt.nl/', host: 'tribelt.nl', website: 'https://www.tribelt.nl' })

/** The server; the scan started answers with each status in line, the last one for good. */
function api(extra: Record<string, Handler> = {}, statuses: AnalysisStatus[] = ['queued', 'done']) {
  return routes({
    'GET /api/v1/competitors': () => json({ items: [twentebelt, martens, ashworth, tribelt] }),
    'GET /api/v1/terms': () =>
      json(
        termSearch([
          term({ term: 'TwenteFlex', tier: 'product', usedBy: ['Twentebelt (TwenteFlex)'] }),
          term({ term: 'Drahtösengliederband', language: 'de', tier: 'product', usedBy: ['Märtens', 'Tribelt'] }),
        ]),
      ),
    'GET /api/v1/competitors/twentebelt': () => json({ competitor: twentebelt, scans: [summary()] }),
    'GET /api/v1/competitors/martens': () => json({ competitor: martens, scans: [] }),
    'GET /api/v1/competitors/ashworth': () => json({ competitor: ashworth, scans: [] }),
    'GET /api/v1/analyses': () => json({ items: [summary({ createdAt: new Date().toISOString() }), summary({ id: NEW_ID, url: 'https://www.example-belts.de', competitorId: undefined, status: 'running' })] }),
    [`GET /api/v1/analyses/${ANALYSIS_ID}`]: () => json(analysis()),
    'POST /api/v1/analyses': () => json(analysis({}, { id: NEW_ID, status: 'queued', pagesCrawled: 0 }), 202),
    [`GET /api/v1/analyses/${NEW_ID}`]: () => {
      const status = statuses.length > 1 ? statuses.shift() : statuses[0]
      return json(analysis({}, { id: NEW_ID, status: status ?? 'done', pagesCrawled: status === 'queued' ? 0 : 15 }))
    },
    ...extra,
  })
}

const rows = (wrapper: VueWrapper) => wrapper.findAll('ul[aria-label="Companies"] button')
const pill = (wrapper: VueWrapper, group: string, label: string) => {
  const button = wrapper.get(`[role="group"][aria-label="${group}"]`).findAll('button').find((b) => b.text() === label)
  if (!button) throw new Error(`no pill ${label}`)
  return button
}
const combobox = (wrapper: VueWrapper) => wrapper.get('input[role="combobox"]')

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

describe('the company list', () => {
  it('lists every company with the families it offers, counting what is shown', async () => {
    const { wrapper } = await mountApp(api(), '/competitors')
    expect(wrapper.get('h1').text()).toBe('3 makers, the same belts, other names.')
    expect(rows(wrapper).map((r) => r.attributes('aria-label'))).toEqual([
      'Twentebelt, Netherlands: Tri-Flex',
      'Märtens, Germany: Eyelink',
      'Ashworth, United States: no families recorded',
      'Tribelt, Netherlands: Tri-Flex',
    ])
    expect(wrapper.text()).toContain('4 of 4 companies')
  })

  it('filters by region and by name or country', async () => {
    const { wrapper } = await mountApp(api(), '/competitors')
    await pill(wrapper, 'Show', 'Germany').trigger('click')
    expect(rows(wrapper).map((r) => r.text())).toEqual([expect.stringContaining('Märtens')])
    expect(pill(wrapper, 'Show', 'Germany').attributes('aria-pressed')).toBe('true')
    await pill(wrapper, 'Show', "Tribelt's group").trigger('click')
    expect(rows(wrapper)).toHaveLength(1)
    await pill(wrapper, 'Show', 'All').trigger('click')
    await wrapper.get('#company-q').setValue('united')
    expect(rows(wrapper).map((r) => r.attributes('aria-label'))).toEqual(['Ashworth, United States: no families recorded'])
    expect(wrapper.text()).toContain('1 of 4 companies')
  })

  it('says when the companies cannot be loaded', async () => {
    const { wrapper } = await mountApp(api({ 'GET /api/v1/competitors': () => problem(500, 'Internal Server Error') }), '/competitors')
    expect(wrapper.get('[role="alert"]').text()).toBe('Internal Server Error')
    expect(wrapper.get('h1').text()).toBe('The same belts, other names.')
  })
})

describe('the company card', () => {
  it('shows the first company, its names per family, its terms and its scans', async () => {
    const { wrapper } = await mountApp(api(), '/competitors')
    expect(rows(wrapper)[0]?.attributes('aria-pressed')).toBe('true')
    const card = wrapper.get('section[aria-labelledby="company-name"]')
    expect(card.get('h2').text()).toBe('Twentebelt')
    expect(card.text()).toContain('Competitor')
    expect(card.text()).toContain('Netherlands · website in 2 languages')
    expect(card.findAll('tbody tr').map((r) => r.text())).toEqual([
      'EyelinkNot offered',
      'Open spiralNot offered',
      'Closed spiralNot offered',
      'Tri-FlexTwenteFlex spiral belts (and 1 more)',
      'Other beltsNot offered',
    ])
    expect(card.text()).toContain('1 term in the terminology map')
    expect(card.text()).toContain('TwenteFlex, Product')
    expect(card.text()).toContain('Scanned 2×')
    expect(card.get(`a[href="/scan/${ANALYSIS_ID}?company=twentebelt"]`).text()).toContain('15 pages read')
    const visit = card.get('a[target="_blank"]')
    expect(visit.attributes('href')).toBe('https://www.twentebelt.com/')
    expect(visit.attributes('rel')).toBe('noopener noreferrer')
  })

  it('shows the company picked, and keeps it in the address', async () => {
    const { wrapper, router } = await mountApp(api(), '/competitors')
    await rows(wrapper)[1]?.trigger('click')
    await settle()
    expect(router.currentRoute.value.fullPath).toBe('/competitors?company=martens')
    const card = wrapper.get('section[aria-labelledby="company-name"]')
    expect(card.get('h2').text()).toBe('Märtens')
    expect(card.text()).toContain('Showing Märtens.')
    expect(card.text()).toContain('Not scanned yet')
  })

  it('opens on the company in the address, and says when it has no website', async () => {
    const { wrapper } = await mountApp(api(), '/competitors?company=ashworth')
    const card = wrapper.get('section[aria-labelledby="company-name"]')
    expect(card.get('h2').text()).toBe('Ashworth')
    expect(card.text()).toContain('The research found no website for this company.')
    expect(card.find('a[target="_blank"]').exists()).toBe(false)
  })

  it("names a group company as Tribelt's own, with Tribelt's own terms", async () => {
    const { wrapper } = await mountApp(api(), '/competitors?company=tribelt-b-v')
    const card = wrapper.get('section[aria-labelledby="company-name"]')
    expect(card.text()).toContain("Tribelt's own group")
    expect(card.text()).toContain('Drahtösengliederband')
  })
})

describe('the scan', () => {
  it('lists the recent scans until one is open', async () => {
    const { wrapper } = await mountApp(api(), '/competitors?company=martens')
    const recent = wrapper.get('section[aria-labelledby="recent-title"]')
    expect(recent.findAll('a').map((a) => [a.text(), a.attributes('href')])).toEqual([
      ['Twentebelt, today', `/scan/${ANALYSIS_ID}?company=martens`],
      ['example-belts.de, scanning', `/scan/${NEW_ID}?company=martens`],
    ])
  })

  it('says when there are no scans yet, or they cannot be loaded', async () => {
    const empty = await mountApp(api({ 'GET /api/v1/analyses': () => json({ items: [] }) }), '/competitors')
    expect(empty.wrapper.get('section[aria-labelledby="recent-title"]').text()).toContain('No scans yet.')
    empty.wrapper.unmount()
    const failing = await mountApp(api({ 'GET /api/v1/analyses': () => problem(500, 'Internal Server Error') }), '/competitors')
    expect(failing.wrapper.get('section[aria-labelledby="recent-title"] [role="alert"]').text()).toBe('Internal Server Error')
  })

  it("scans a company's website from its card: queued, then the result", async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout', 'setInterval', 'clearInterval'] })
    const sentBodies: unknown[] = []
    const { wrapper, router } = await mountApp(
      api({
        'POST /api/v1/analyses': async (request) => {
          sentBodies.push(await request.json())
          return json(analysis({}, { id: NEW_ID, status: 'queued', pagesCrawled: 0 }), 202)
        },
      }),
      '/competitors?company=martens',
    )
    await wrapper.findAll('button').find((b) => b.text() === 'Scan this website')?.trigger('click')
    await settle()
    expect(sentBodies).toEqual([{ url: 'https://www.martens-belts.com/', maxPages: 30 }])
    expect(router.currentRoute.value.fullPath).toBe(`/scan/${NEW_ID}?company=martens`)
    expect((combobox(wrapper).element as HTMLInputElement).value).toBe('https://www.martens-belts.com/')
    const progress = wrapper.get('[role="progressbar"]')
    expect(progress.attributes('aria-valuetext')).toBe('Waiting to start')
    expect(wrapper.text()).toContain('The scan reads up to 15 pages')
    // The company stays on screen beside its scan.
    expect(wrapper.get('section[aria-labelledby="company-name"] h2').text()).toBe('Märtens')

    await vi.advanceTimersByTimeAsync(POLL_INTERVAL_MS)
    await flushPromises()
    expect(wrapper.find('[role="progressbar"]').exists()).toBe(false)
    expect(wrapper.get('h3').text()).toBe('twentebelt.com')
    expect(wrapper.text()).toContain('1 page read · 2 new terms')
  })

  it('shows a finished scan: what Tribelt is missing, the ranked phrases and the matched products', async () => {
    const { wrapper } = await mountApp(api(), `/scan/${ANALYSIS_ID}`)
    expect(wrapper.text()).toContain('Tribelt is missing:')
    expect(wrapper.text()).toContain('balanced weave belt, Product')
    const phrases = () => wrapper.findAll('ol[aria-label="Ranked phrases"] li').map((li) => li.text())
    expect(phrases()).toEqual([
      expect.stringMatching(/^1metal conveyor belts, Broad.*In the map$/),
      expect.stringMatching(/^2förderbänder, Broad.*New term$/),
      expect.stringMatching(/^3spiral belts, Product.*New term$/),
    ])
    await pill(wrapper, 'Which phrases', 'In the map').trigger('click')
    expect(phrases()).toHaveLength(1)
    await pill(wrapper, 'Which phrases', 'New terms').trigger('click')
    expect(phrases()).toHaveLength(2)
    const matched = wrapper.get('details')
    expect(matched.get('summary').text()).toBe("1 product matched to Tribelt's belts")
    expect(matched.text()).toContain("TwenteFlex spiral belts is Tribelt's Tri-Flex. Its words: spiral belt, Spiralband.")
    expect(matched.text()).not.toContain('Matched by')
  })

  it('names the model that matched, says what became of matching, and handles a scan with no phrases', async () => {
    const scan = (overrides: Partial<Analysis>) => api({ [`GET /api/v1/analyses/${ANALYSIS_ID}`]: () => json(analysis(overrides)) })
    const withModel = await mountApp(scan({ matchingModel: 'openrouter/anthropic/claude-haiku' }), `/scan/${ANALYSIS_ID}`)
    expect(withModel.wrapper.get('details').text()).toContain('Matched by openrouter/anthropic/claude-haiku')
    withModel.wrapper.unmount()
    const disabled = await mountApp(scan({ matching: 'disabled', equivalents: [], keywords: [], knownTerms: [] }), `/scan/${ANALYSIS_ID}`)
    expect(disabled.wrapper.text()).toContain('Product matching is off')
    expect(disabled.wrapper.text()).toContain('Tribelt uses every map term this site uses.')
    expect(disabled.wrapper.text()).toContain('No phrases of this kind.')
    expect(disabled.wrapper.find('details').exists()).toBe(false)
  })

  it('says why a scan failed, or could not be loaded', async () => {
    const failed = await mountApp(
      api({ [`GET /api/v1/analyses/${ANALYSIS_ID}`]: () => json(analysis({}, { status: 'failed', failure: 'robots.txt forbids it.' })) }),
      `/scan/${ANALYSIS_ID}`,
    )
    expect(failed.wrapper.text()).toContain('Scan failed')
    expect(failed.wrapper.get('[role="alert"]').text()).toBe('robots.txt forbids it.')
    failed.wrapper.unmount()
    const silent = await mountApp(api({ [`GET /api/v1/analyses/${ANALYSIS_ID}`]: () => json(analysis({}, { status: 'failed' })) }), `/scan/${ANALYSIS_ID}`)
    expect(silent.wrapper.get('[role="alert"]').text()).toBe('The scan stopped before it read a page.')
    silent.wrapper.unmount()
    const missing = await mountApp(api({ [`GET /api/v1/analyses/${ANALYSIS_ID}`]: () => problem(404, 'Not Found', 'No such scan.') }), `/scan/${ANALYSIS_ID}`)
    expect(missing.wrapper.get('[role="alert"]').text()).toBe('No such scan.')
  })

  it('shows a scan reading pages, and goes back to the recent scans', async () => {
    const { wrapper, router } = await mountApp(
      api({ [`GET /api/v1/analyses/${ANALYSIS_ID}`]: () => json(analysis({}, { status: 'running', pagesCrawled: 3, maxPages: 30 })) }),
      `/scan/${ANALYSIS_ID}?company=martens`,
    )
    expect(wrapper.get('[role="progressbar"]').attributes('aria-valuetext')).toBe('Reading page 4 of 30')
    await wrapper.findAll('a').find((a) => a.text() === 'Show the recent scans')?.trigger('click')
    await settle()
    expect(router.currentRoute.value.fullPath).toBe('/competitors?company=martens')
    expect(wrapper.find('section[aria-labelledby="recent-title"]').exists()).toBe(true)
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
    const refuse = { 'POST /api/v1/analyses': () => problem(status, 'Refused', status === 422 ? 'The host is not public.' : 'Too many scans. Wait a minute.') }
    const { wrapper } = await mountApp(api(refuse), '/competitors')
    await type(wrapper, 'https://www.example.com')
    await submit(wrapper)
    expect(wrapper.get('#scan-url-error').text()).toBe('The host is not public.')
    status = 429
    await submit(wrapper)
    expect(wrapper.find('#scan-url-error').exists()).toBe(false)
    expect(wrapper.get('[role="alert"]').text()).toBe('Too many scans. Wait a minute.')
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
    expect(wrapper.get('#scan-url-company').text()).toBe('Twentebelt, a known competitor. Scanned 2×.')
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
    expect(sentBodies).toEqual([{ url: 'https://www.example-belts.de/', maxPages: 30, companyName: 'Example Belts GmbH' }])
  })
})
