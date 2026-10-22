import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { Analysis, AnalysisSummary, Keyword, KnownTerm } from '../../../infrastructure/api/types.gen'
import { ANALYSIS_ID, analysis, competitor, summary, term, termSearch } from '../../../testing/fixtures'
import { type Handler, json, mountApp, problem, routes, settle } from '../../../testing/mountApp'

const NEW_ID = '6c1d2b0f-3e4a-4f6b-9c7d-8e9f0a1b2c3d'

/** The server, with the scan on screen answering as given. */
function api(scan: Analysis = analysis(), extra: Record<string, Handler> = {}) {
  return routes({
    'GET /api/v1/analyses': () => json({ items: [summary()] }),
    'GET /api/v1/terms': () => json(termSearch([term()])),
    'GET /api/v1/competitors': () =>
      json({ items: [competitor('Twentebelt', { website: 'https://www.twentebelt.com' })] }),
    [`GET /api/v1/analyses/${ANALYSIS_ID}`]: () => json(scan),
    'POST /api/v1/analyses': () => json(analysis({}, { id: NEW_ID, status: 'queued' }), 202),
    [`GET /api/v1/analyses/${NEW_ID}`]: () => json(analysis({}, { id: NEW_ID, status: 'queued' })),
    ...extra,
  })
}

function scanWith(overrides: Partial<Analysis>, summaryOverrides: Partial<AnalysisSummary> = {}): Analysis {
  return analysis(overrides, summaryOverrides)
}

async function fillAndSubmit(wrapper: VueWrapper, url: string, pages = '15') {
  await wrapper.get('input[type="url"]').setValue(url)
  await wrapper.get('select').setValue(pages)
  await wrapper.get('form[aria-label="Scan a website"]').trigger('submit')
  await settle()
}

const known = (name: string, occurrences: number, usedByTribelt = false): KnownTerm => ({
  term: name,
  language: 'en',
  family: 'spec_all',
  tier: 'spec',
  occurrences,
  usedByTribelt,
})
const keyword = (phrase: string, score: number, language = 'en'): Keyword => ({ phrase, language, score, pages: 1 })

describe('the scan form', () => {
  it('offers four page budgets, fifteen to start with', async () => {
    const { wrapper } = await mountApp(api(), '/scan')
    expect(wrapper.get('h1').text()).toBe('Scan a competitor')
    expect(wrapper.findAll('option').map((o) => o.text())).toEqual(['5 pages', '10 pages', '15 pages', '30 pages'])
    expect((wrapper.get('select').element as HTMLSelectElement).value).toBe('15')
  })

  it('checks the address before sending it, and clears the message on typing', async () => {
    const { wrapper, requests } = await mountApp(api(), '/scan')
    await fillAndSubmit(wrapper, '')
    expect(wrapper.get('[role="alert"]').text()).toBe('Enter the address of a website.')
    expect(wrapper.get('input[type="url"]').attributes('aria-invalid')).toBe('true')
    expect(wrapper.get('input[type="url"]').attributes('aria-describedby')).toBe('scan-url-error')
    expect(requests().filter((r) => r.startsWith('POST'))).toEqual([])
    await wrapper.get('input[type="url"]').setValue('https://www.')
    await wrapper.get('input[type="url"]').trigger('input')
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })

  it('starts a scan and goes to it', async () => {
    const { wrapper, router, requests } = await mountApp(api(), '/scan')
    await fillAndSubmit(wrapper, 'www.twentebelt.com', '10')
    expect(requests()).toContain('POST /api/v1/analyses')
    expect(router.currentRoute.value.path).toBe(`/scan/${NEW_ID}`)
    expect(wrapper.get('[role="progressbar"]').attributes('aria-valuetext')).toBe('Waiting to start')
    // What was typed stays in the form.
    expect((wrapper.get('input[type="url"]').element as HTMLInputElement).value).toBe('www.twentebelt.com')
  })

  it("puts the server's refusal under the address or under the form", async () => {
    let status = 422
    const refuse = { 'POST /api/v1/analyses': () => problem(status, 'Refused', status === 422 ? 'The host is not public.' : 'Too many scans. Wait a minute.') }
    const { wrapper } = await mountApp(api(analysis(), refuse), '/scan')
    await fillAndSubmit(wrapper, 'https://www.example.com')
    expect(wrapper.get('#scan-url-error').text()).toBe('The host is not public.')
    status = 429
    await wrapper.get('select').setValue('5')
    await wrapper.get('select').trigger('change')
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    await wrapper.get('form[aria-label="Scan a website"]').trigger('submit')
    await settle()
    expect(wrapper.find('#scan-url-error').exists()).toBe(false)
    expect(wrapper.get('[role="alert"]').text()).toBe('Too many scans. Wait a minute.')
  })

  it("starts the overview's quick scan once, and drops it from the address", async () => {
    const { router, requests } = await mountApp(api(), '/scan?url=https%3A%2F%2Fwww.twentebelt.com')
    expect(requests().filter((r) => r.startsWith('POST'))).toEqual(['POST /api/v1/analyses'])
    expect(router.currentRoute.value.path).toBe(`/scan/${NEW_ID}`)
  })

  it("shows the scan's own address and budget until something is typed", async () => {
    const { wrapper } = await mountApp(api(scanWith({}, { maxPages: 7 })), `/scan/${ANALYSIS_ID}`)
    expect((wrapper.get('input[type="url"]').element as HTMLInputElement).value).toBe('https://www.twentebelt.com')
    // A budget the form does not offer shows as the default.
    expect((wrapper.get('select').element as HTMLSelectElement).value).toBe('15')
  })
})

describe('a finished scan', () => {
  it('heads the result with the site, the date and what was read', async () => {
    const { wrapper } = await mountApp(api(), `/scan/${ANALYSIS_ID}`)
    expect(wrapper.get('#result-title').text()).toBe('twentebelt.com')
    expect(wrapper.text()).toMatch(/Scanned 22 October 2026 at \d\d:\d\d\. Read 1 of 15 pages, all in English\./)
    expect(wrapper.find('[role="progressbar"]').exists()).toBe(false)
  })

  it('shows the temper spectrum with every tier in the legend', async () => {
    const { wrapper } = await mountApp(api(), `/scan/${ANALYSIS_ID}`)
    expect(wrapper.text()).toContain('The site uses 2 terms from the terminology map.')
    expect(wrapper.get('[role="img"]').attributes('aria-label')).toBe('2 terms: 2 product')
    expect(wrapper.findAll('[role="img"] > div')).toHaveLength(1)
    expect(wrapper.text()).toContain('Specification 0')
  })

  it('lists the terms Tribelt is missing, each linked to who else uses it', async () => {
    const { wrapper } = await mountApp(api(), `/scan/${ANALYSIS_ID}`)
    const gaps = wrapper.get('section[aria-labelledby="gap-title"]')
    expect(gaps.text()).toContain("Twentebelt uses these on its pages. Tribelt's own website does not")
    expect(gaps.findAll('li')).toHaveLength(1)
    expect(gaps.get('li').text()).toContain('balanced weave belt')
    expect(gaps.get('li').text()).toContain('9×')
    expect(gaps.get('li').text()).toContain('Product, Open spiral')
    expect(gaps.get('a').attributes('href')).toBe('/terminology?q=balanced+weave+belt')
  })

  it('opens a long list of missing terms on request', async () => {
    const terms = Array.from({ length: 12 }, (_, i) => known(`term ${String(i)}`, 20 - i))
    const { wrapper } = await mountApp(api(scanWith({ knownTerms: terms })), `/scan/${ANALYSIS_ID}`)
    const gaps = wrapper.get('section[aria-labelledby="gap-title"]')
    expect(gaps.findAll('li')).toHaveLength(10)
    await gaps.get('button').trigger('click')
    expect(gaps.findAll('li')).toHaveLength(12)
    expect(gaps.get('button').text()).toBe('Show fewer')
  })

  it('says so when Tribelt uses every term, or the site none', async () => {
    const all = await mountApp(api(scanWith({ knownTerms: [known('pitch', 3, true)] })), `/scan/${ANALYSIS_ID}`)
    expect(all.wrapper.text()).toContain("Tribelt's own website uses every term from the map that Twentebelt uses.")
    all.wrapper.unmount()
    const none = await mountApp(api(scanWith({ knownTerms: [] })), `/scan/${ANALYSIS_ID}`)
    expect(none.wrapper.text()).toContain('The site uses none of the terms in the terminology map.')
  })

  it('lists the phrases, hiding single words until asked, in one language on request', async () => {
    const { wrapper } = await mountApp(api(), `/scan/${ANALYSIS_ID}`)
    const list = wrapper.get('section[aria-labelledby="phrase-title"]')
    const phrases = () => list.findAll('li').map((li) => li.get('span').text())
    expect(phrases()).toEqual(['metal conveyor belts', 'spiral belts'])
    expect(list.get('li div[aria-hidden] div').attributes('style')).toBe('width: 100%;')
    expect(list.text()).toContain('Single words are hidden.')
    const toggle = list.get('button[aria-pressed]')
    await toggle.trigger('click')
    expect(toggle.attributes('aria-pressed')).toBe('true')
    expect(list.text()).toContain('Single words are shown.')
    expect(phrases()).toEqual(['metal conveyor belts', 'förderbänder', 'spiral belts'])
    await list.get('select').setValue('de')
    expect(phrases()).toEqual(['förderbänder'])
    await toggle.trigger('click')
    expect(list.text()).toContain('No phrases to show.')
  })

  it('opens a long list of phrases on request, and has no language filter for one language', async () => {
    const keywords = Array.from({ length: 14 }, (_, i) => keyword(`phrase ${String(i)}`, 100 - i))
    const { wrapper } = await mountApp(api(scanWith({ keywords })), `/scan/${ANALYSIS_ID}`)
    const list = wrapper.get('section[aria-labelledby="phrase-title"]')
    expect(list.find('select').exists()).toBe(false)
    expect(list.findAll('li')).toHaveLength(10)
    await list.get('button[aria-expanded]').trigger('click')
    expect(list.findAll('li')).toHaveLength(14)
  })

  it("pairs the site's products with Tribelt's", async () => {
    const { wrapper } = await mountApp(api(), `/scan/${ANALYSIS_ID}`)
    const table = wrapper.get('section[aria-labelledby="eq-title"] table')
    expect(table.findAll('th').map((th) => th.text())).toEqual(['Twentebelt calls it', 'Tribelt makes', 'Words on their page'])
    expect(table.findAll('tbody td').map((td) => td.text())).toEqual([
      'TwenteFlex spiral belts',
      'Tri-Flex',
      'spiral belt, Spiralband',
    ])
  })

  it('says in plain words why there is no product table', async () => {
    const off = await mountApp(api(scanWith({ matching: 'disabled', equivalents: [] })), `/scan/${ANALYSIS_ID}`)
    expect(off.wrapper.text()).toContain('Product matching is off: no language model is configured on this server.')
    off.wrapper.unmount()
    const empty = await mountApp(api(scanWith({ equivalents: [] })), `/scan/${ANALYSIS_ID}`)
    expect(empty.wrapper.text()).toContain("Claude found no products on these pages to match with Tribelt's.")
  })

  it('lists the pages read, and the link at the top opens the list', async () => {
    const { wrapper } = await mountApp(api(), `/scan/${ANALYSIS_ID}`)
    const details = wrapper.get('details#pages')
    expect(details.get('summary').text()).toBe('The page read')
    expect(details.get('li').text()).toContain('/conveyor-belts')
    expect(details.get('li').text()).toContain('412 words')
    const element = details.element as HTMLDetailsElement
    element.scrollIntoView = () => undefined
    expect(element.open).toBe(false)
    const link = wrapper.get('a[href="#pages"]')
    expect(link.text()).toBe('See the page read')
    await link.trigger('click')
    await settle()
    expect(element.open).toBe(true)
  })

  it('names the site by its host when it is no company in the map', async () => {
    const { wrapper } = await mountApp(api(scanWith({}, { url: 'https://www.example.com/' })), `/scan/${ANALYSIS_ID}`)
    expect(wrapper.text()).toContain('example.com uses these on its pages.')
    expect(wrapper.text()).toContain('example.com calls it')
  })
})

describe('a scan in progress or failed', () => {
  it('shows a running scan as a moving belt, without results yet', async () => {
    const scan = scanWith({ keywords: [], knownTerms: [], equivalents: [], matching: 'pending' }, { status: 'running', pagesCrawled: 3, maxPages: 10, finishedAt: undefined })
    const { wrapper } = await mountApp(api(scan), `/scan/${ANALYSIS_ID}`)
    const bar = wrapper.get('[role="progressbar"]')
    expect(bar.attributes('aria-valuenow')).toBe('3')
    expect(bar.attributes('aria-valuemax')).toBe('10')
    expect(bar.get('.belt').attributes('style')).toBe('width: 30%;')
    expect(wrapper.text()).toContain('Reading page 4 of 10…')
    expect(wrapper.text()).toMatch(/Started 22 October 2026 at \d\d:\d\d\./)
    expect(wrapper.find('section[aria-labelledby="gap-title"]').exists()).toBe(false)
  })

  it('runs the whole belt, dimmed, while the scan waits its turn', async () => {
    const scan = scanWith({ pages: [] }, { status: 'queued', pagesCrawled: 0 })
    const { wrapper } = await mountApp(api(scan), `/scan/${ANALYSIS_ID}`)
    const belt = wrapper.get('[role="progressbar"] .belt')
    expect(belt.attributes('style')).toBe('width: 100%;')
    expect(belt.classes()).toContain('opacity-40')
    expect(wrapper.get('[role="progressbar"]').attributes('aria-valuenow')).toBeUndefined()
    expect(wrapper.find('a[href="#pages"]').exists()).toBe(false)
  })

  it('says why a scan failed, and what to do', async () => {
    const failed = await mountApp(
      api(scanWith({}, { status: 'failed', failure: 'The site did not answer within 10 seconds.' })),
      `/scan/${ANALYSIS_ID}`,
    )
    expect(failed.wrapper.text()).toContain('The scan failed')
    expect(failed.wrapper.text()).toContain('The site did not answer within 10 seconds.')
    expect(failed.wrapper.text()).toContain('Check the address and scan again.')
    failed.wrapper.unmount()
    const unexplained = await mountApp(api(scanWith({}, { status: 'failed' })), `/scan/${ANALYSIS_ID}`)
    expect(unexplained.wrapper.text()).toContain('The site could not be read.')
  })

  it('says so when the scan cannot be loaded', async () => {
    const missing = { [`GET /api/v1/analyses/${ANALYSIS_ID}`]: () => problem(404, 'Not Found', 'No such analysis.') }
    const { wrapper } = await mountApp(api(analysis(), missing), `/scan/${ANALYSIS_ID}`)
    expect(wrapper.get('[role="alert"]').text()).toBe('No such analysis.')
  })

  it('says it is loading before the scan arrives', async () => {
    const slow = { [`GET /api/v1/analyses/${ANALYSIS_ID}`]: () => new Promise<Response>(() => undefined) }
    const { wrapper } = await mountApp(api(analysis(), slow), `/scan/${ANALYSIS_ID}`)
    expect(wrapper.text()).toContain('Loading the scan…')
  })
})
