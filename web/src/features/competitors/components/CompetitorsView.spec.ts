import { flushPromises, type VueWrapper } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Analysis, AnalysisStatus } from '../../../infrastructure/api/types.gen'
import { ANALYSIS_ID, analysis, bots, competitor, me, promptResults, readiness, readinessRow, summary, term, termSearch, readinessTable } from '../../../testing/fixtures'
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

/** The server, signed in as an administrator; the scan started answers with each status in line, the last one for good. */
function api(extra: Record<string, Handler> = {}, statuses: AnalysisStatus[] = ['queued', 'done']) {
  return routes({
    'GET /api/v1/me': () => json(me({ isAdmin: true })),
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

describe('the columns', () => {
  it('widens, minimises and restores a column, keeps one open, and remembers the layout', async () => {
    localStorage.clear()
    const first = await mountApp(api(), '/competitors')
    const grid = () => first.wrapper.get('[style*="grid-template-columns"]').attributes('style') ?? ''
    const control = (name: string) => first.wrapper.get(`button[aria-label="${name}"]`)
    expect(grid()).toContain('minmax(0,1.3fr) minmax(0,0.95fr) minmax(0,1fr)')
    await control('Widen Who makes which belt').trigger('click')
    expect(grid()).toContain('minmax(0,3.25fr)')
    expect(control('Put Who makes which belt back to its width').attributes('aria-pressed')).toBe('true')
    await control('Minimise Company').trigger('click')
    expect(grid()).toContain('150px')
    expect(first.wrapper.find('button[aria-label="Restore Company"]').exists()).toBe(true)
    await control('Minimise Scan a competitor').trigger('click')
    // The last open column cannot be minimised.
    expect(control('Minimise Who makes which belt').attributes('disabled')).toBeDefined()
    first.wrapper.unmount()

    const again = await mountApp(api(), '/competitors')
    expect(again.wrapper.get('[style*="grid-template-columns"]').attributes('style')).toContain('minmax(0,3.25fr) 150px 150px')
    await again.wrapper.get('button[aria-label="Restore Company"]').trigger('click')
    expect(again.wrapper.get('[style*="grid-template-columns"]').attributes('style')).toContain('minmax(0,3.25fr) minmax(0,0.95fr) 150px')
    localStorage.clear()
  })
})

describe('the company list', () => {
  it('lists every company with the families it offers, counting what is shown', async () => {
    const { wrapper } = await mountApp(api(), '/competitors')
    expect(wrapper.get('h1').text()).toBe('3 competitors, the same belts, other names.')
    expect(rows(wrapper).map((r) => r.attributes('aria-label'))).toEqual([
      'Twentebelt, Netherlands: Tri-Flex; 2 scans',
      'Märtens, Germany: Eyelink; No scans',
      'Ashworth, United States: no families recorded; No scans',
      'Tribelt, Netherlands: Tri-Flex; No scans',
    ])
    expect(wrapper.text()).toContain('4 of 4 companies')
  })

  it('sorts by a column, then the other way, then back to the list order', async () => {
    const { wrapper } = await mountApp(api(), '/competitors')
    const order = () => rows(wrapper).map((r) => r.attributes('aria-label')?.split(',')[0])
    const header = wrapper.get('[role="group"][aria-label="Sort the companies"]')
    const eyelink = () => header.get('button[aria-label^="Eyelink"]')
    expect(order()).toEqual(['Twentebelt', 'Märtens', 'Ashworth', 'Tribelt'])
    await eyelink().trigger('click')
    // The eyelink maker first.
    expect(eyelink().attributes('aria-label')).toBe('Eyelink, sorted descending')
    expect(order()[0]).toBe('Märtens')
    await eyelink().trigger('click')
    expect(eyelink().attributes('aria-label')).toBe('Eyelink, sorted ascending')
    expect(order()[3]).toBe('Märtens')
    await eyelink().trigger('click')
    expect(eyelink().attributes('aria-label')).toBe('Eyelink, not sorted')
    expect(order()).toEqual(['Twentebelt', 'Märtens', 'Ashworth', 'Tribelt'])
    await header.get('button[aria-label^="Country"]').trigger('click')
    expect(order()).toEqual(['Märtens', 'Twentebelt', 'Tribelt', 'Ashworth'])
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
    expect(rows(wrapper).map((r) => r.attributes('aria-label'))).toEqual(['Ashworth, United States: no families recorded; No scans'])
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
    expect(card.text()).toContain('Netherlands. Its website is in 2 languages.')
    expect(card.findAll('tbody tr').map((r) => r.text())).toEqual([
      'EyelinkNot offered',
      'Open spiralNot offered',
      'Closed spiralNot offered',
      'Tri-FlexTwenteFlex spiral belts (and 1 more)',
      'Other beltsNot offered',
    ])
    expect(card.text()).toContain('1 term in the terminology map')
    expect(card.text()).toContain('TwenteFlex, Product')
    const stats = card.get('[data-testid="scan-stats"]').text()
    expect(stats).toContain('Scanned 2 times')
    // The list has no last scan here, so the card reads it from the company's own scans.
    expect(stats).toMatch(/Last scan: 22 October 2026 at \d\d:\d\d, 15 pages read/)
    expect(card.get(`a[href="/scan/${ANALYSIS_ID}?company=twentebelt"]`).text()).toContain('15 pages read')
    const visit = card.get('a[target="_blank"]')
    expect(visit.attributes('href')).toBe('https://www.twentebelt.com/')
    expect(visit.attributes('rel')).toBe('noopener noreferrer')
  })

  it('shows a company being scanned, live, in its row and on its card', async () => {
    const scanning = { ...twentebelt, scans: 3, lastScan: summary({ status: 'running' as const, pagesCrawled: 4, maxPages: 15 }) }
    const { wrapper } = await mountApp(
      api({
        'GET /api/v1/competitors': () => json({ items: [scanning, martens, ashworth, tribelt] }),
        'GET /api/v1/competitors/twentebelt': () => json({ competitor: scanning, scans: [scanning.lastScan] }),
      }),
      '/competitors',
    )
    expect(rows(wrapper)[0]?.attributes('aria-label')).toBe('Twentebelt, Netherlands: Tri-Flex; 3 scans, reading page 5 of 15')
    expect(rows(wrapper)[0]?.text()).toContain('3 scans, reading page 5 of 15')
    const stats = wrapper.get('[data-testid="scan-stats"]')
    expect(stats.text()).toMatch(/Last scan: .* at \d\d:\d\d, reading page 5 of 15/)
    const bar = stats.get('[role="progressbar"]')
    expect(bar.attributes('aria-label')).toBe('Scan of Twentebelt')
    expect(bar.attributes('aria-valuenow')).toBe('27')
  })

  it('explains a company whose site turned the scan away', async () => {
    const blocked = { ...twentebelt, scans: 1, lastScan: summary({ status: 'failed' as const, blocked: true, pagesCrawled: 0, failure: 'Blocked by the site: …' }) }
    const { wrapper } = await mountApp(
      api({
        'GET /api/v1/competitors': () => json({ items: [blocked, martens, ashworth, tribelt] }),
        'GET /api/v1/competitors/twentebelt': () => json({ competitor: blocked, scans: [blocked.lastScan] }),
      }),
      '/competitors',
    )
    expect(rows(wrapper)[0]?.text()).toContain('1 scan, blocked')
    const stats = wrapper.get('[data-testid="scan-stats"]')
    expect(stats.text()).toMatch(/Last scan: .*, blocked by its site/)
    expect(stats.get('[data-testid="blocked-note"]').text()).toContain('turns automated readers away')
  })

  it('shows the company picked, and keeps it in the address', async () => {
    const { wrapper, router } = await mountApp(api(), '/competitors')
    await rows(wrapper)[1]?.trigger('click')
    await settle()
    expect(router.currentRoute.value.fullPath).toBe('/competitors?company=martens')
    const card = wrapper.get('section[aria-labelledby="company-name"]')
    expect(card.get('h2').text()).toBe('Märtens')
    expect(card.text()).toContain('Showing Märtens.')
    expect(card.get('[data-testid="scan-stats"]').text()).toBe('Never scannedNo scan of its website has run yet.')
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
    expect(sentBodies).toEqual([{ url: 'https://www.martens-belts.com/', maxPages: 60 }])
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
    expect(wrapper.findAll('h3').map((h) => h.text())).toContain('twentebelt.com')
    expect(wrapper.text()).toContain('1 page read, 2 new terms')
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

describe('for an account that is no administrator', () => {
  it('shows the companies and the scans, but no way to scan or change them', async () => {
    const { wrapper } = await mountApp(api({ 'GET /api/v1/me': () => json(me()) }), '/competitors?company=martens')
    expect(wrapper.get('section[aria-labelledby="company-name"] h2').text()).toBe('Märtens')
    expect(wrapper.get('#scan-title').text()).toBe('Scans')
    expect(wrapper.find('form[aria-label="Scan a website"]').exists()).toBe(false)
    expect(wrapper.find('[aria-label="Manage this company"]').exists()).toBe(false)
    expect(wrapper.find('#admin-title').exists()).toBe(false)
    expect(wrapper.findAll('button').some((b) => b.text() === 'Scan this website')).toBe(false)
    expect(wrapper.get('a[href="/competitors/martens/history"]').text()).toBe('Over time')
    expect(wrapper.get('a[href="/competitors/martens"]').text()).toBe('Latest scan')
    expect(wrapper.get('section[aria-labelledby="recent-title"]').text()).toContain('Twentebelt')
  })

  it('hides an archived company, which an administrator sees apart', async () => {
    const archived = competitor('Old Belts', { id: 'old-belts', archived: true, host: 'old-belts.com', websiteUrl: 'https://old-belts.com/' })
    const list = { 'GET /api/v1/competitors': () => json({ items: [twentebelt, archived] }), 'GET /api/v1/competitors/old-belts': () => json({ competitor: archived, scans: [] }) }
    const viewer = await mountApp(api({ ...list, 'GET /api/v1/me': () => json(me()) }), '/competitors?company=old-belts')
    expect(viewer.wrapper.text()).not.toContain('Old Belts')
    expect(viewer.wrapper.get('section[aria-labelledby="company-name"] h2').text()).toBe('Twentebelt')
    viewer.wrapper.unmount()

    const admin = await mountApp(api(list), '/competitors')
    const button = admin.wrapper.findAll('button').find((b) => b.text() === 'Old Belts')
    await button?.trigger('click')
    await settle()
    const card = admin.wrapper.get('section[aria-labelledby="company-name"]')
    expect(card.get('h2').text()).toBe('Old Belts')
    expect(card.text()).toContain('Archived: hidden from the list and the re-scans.')
    expect(card.findAll('button').map((b) => b.text())).toContain('Restore')
  })
})

describe('the AI readiness on a company card', () => {
  it('sums up the crawler policy and the other signals, and leaves the mirror out of the list', async () => {
    const mirror = competitor('Mirror of tribelt.nl', { id: 'tribelt-mirror', role: 'mirror', host: 'tribelt.jorisjonkers.dev' })
    const { wrapper } = await mountApp(
      api({
        'GET /api/v1/competitors': () => json({ items: [twentebelt, martens, mirror] }),
        'GET /api/v1/ai-readiness': () => json(readinessTable([readinessRow('Twentebelt', { readiness: readiness({ bots: bots({ GPTBot: 'blocked', ClaudeBot: 'blocked' }), gaps: 2 }) })])),
      }),
      '/competitors',
    )
    const block = wrapper.get('[data-testid="company-readiness"]')
    expect(block.text()).toContain('Blocks GPTBot and ClaudeBot.')
    expect(block.text()).toContain('Follows 15 of 17 recommendations.')
    expect(block.text()).toContain('llms.txtYes, 124 links')
    expect(block.get('a').attributes('href')).toBe('/ai-search/readiness')
    expect(rows(wrapper).map((r) => r.text())).not.toContain(expect.stringContaining('Mirror'))
  })

  it('says when no scan has checked the company yet', async () => {
    const { wrapper } = await mountApp(api({ 'GET /api/v1/ai-readiness': () => json(readinessTable([])) }), '/competitors')
    expect(wrapper.get('[data-testid="company-readiness"]').text()).toContain('Not checked yet: the next scan records it.')
  })
})

describe('the AI answers on a company card', () => {
  it('says how often the latest Prompt run named the company', async () => {
    const { wrapper } = await mountApp(api({ 'GET /api/v1/prompt-test': () => json(promptResults()) }), '/competitors')
    expect(wrapper.get('[data-testid="prompt-mentions"]').text()).toBe(
      'AI answers: Named in 9 of 12 answers of the Prompt run of 1 Nov 2026, first in 4. The Prompt test',
    )
  })
})
