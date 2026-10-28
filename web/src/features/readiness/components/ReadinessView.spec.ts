import type { DOMWrapper, VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { AiReadiness } from '../../../infrastructure/api/types.gen'
import { ANALYSIS_ID, bots, me, readiness, readinessRow, readinessTable } from '../../../testing/fixtures'
import { type Handler, json, mountApp, problem, routes, settle } from '../../../testing/mountApp'

const gappy: AiReadiness = readiness({
  bots: bots({ GPTBot: 'blocked' }),
  llmsTxt: { present: false, bytes: 0, links: 0 },
  advice: readiness().advice.map((a) => (a.check === 'llmsTxt' ? { ...a, ok: false } : a)),
  gaps: 2,
})

const table = [
  readinessRow('Twentebelt', { readiness: gappy }),
  readinessRow('ESFO'),
  readinessRow('Unibelt', { readiness: undefined, scan: undefined }),
  readinessRow('Lesjöfors', { role: 'group', readiness: gappy }),
  readinessRow('Mirror of tribelt.nl', { competitorId: 'tribelt-mirror', role: 'mirror' }),
  readinessRow('Tribelt B.V.', { competitorId: 'tribelt', role: 'tribelt', readiness: gappy }),
]

const api = (extra: Record<string, Handler> = {}) =>
  routes({
    'GET /api/v1/ai-readiness': () => json(readinessTable(table)),
    'GET /api/v1/ai-search': () =>
      json({
        goal: 0.4,
        followed: [],
        trend: [],
        crawlers: [],
        keepingOut: [],
        ranking: [
          { competitorId: 'esfo', name: 'ESFO', role: 'competitor', score: 5 },
          { competitorId: 'tribelt', name: 'Tribelt B.V.', role: 'tribelt', score: 3.5 },
        ],
        sites: 5,
        questions: 0,
        providers: 0,
      }),
    [`GET /api/v1/analyses/${ANALYSIS_ID}/readiness-files`]: () =>
      json({ robotsTxt: 'User-agent: GPTBot\nDisallow: /', robotsTruncated: false, llmsTxt: '', llmsTruncated: false, llmsFullTxt: '', llmsFullTruncated: false }),
    ...extra,
  })

/** Each band's company names, top to bottom. */
const bands = (wrapper: VueWrapper) =>
  wrapper
    .findAll('section[aria-labelledby="sites-title"] tbody')
    .map((b) => `${b.attributes('aria-label') ?? ''}: ${b.findAll('th[scope="row"] button').map((n) => n.element.firstChild?.textContent?.trim() ?? '').join(', ')}`)

describe('the AI readiness view', () => {
  it('pins Tribelt and the mirror above the group and the competitors', async () => {
    const { wrapper } = await mountApp(api(), '/ai-search/readiness')
    expect(document.title).toBe('AI readiness · Findability Explorer')
    expect(bands(wrapper)).toEqual([
      'Tribelt today, and the target: Tribelt B.V., Mirror of tribelt.nl',
      "Tribelt's group: Lesjöfors",
      'Competitors: Twentebelt, ESFO, Unibelt',
    ])
  })

  it('sorts a band by any column, and back, keeping Tribelt and the mirror on top', async () => {
    const { wrapper } = await mountApp(api(), '/ai-search/readiness')
    const gptbot = wrapper.get('button[aria-label^="GPTBot (OpenAI training)"]')
    await gptbot.trigger('click')
    expect(gptbot.attributes('aria-label')).toBe('GPTBot (OpenAI training), sorted descending')
    expect(bands(wrapper)[2]).toBe('Competitors: ESFO, Twentebelt, Unibelt')
    await gptbot.trigger('click')
    expect(bands(wrapper)[2]).toBe('Competitors: Unibelt, Twentebelt, ESFO')
    expect(bands(wrapper)[0]).toBe('Tribelt today, and the target: Tribelt B.V., Mirror of tribelt.nl')
    await gptbot.trigger('click')
    expect(gptbot.attributes('aria-label')).toBe('GPTBot (OpenAI training), not sorted')
  })

  it('filters the group and the competitors by a gap', async () => {
    const { wrapper } = await mountApp(api(), '/ai-search/readiness')
    await wrapper.get('select').setValue('bot:GPTBot')
    expect(bands(wrapper)).toEqual([
      'Tribelt today, and the target: Tribelt B.V., Mirror of tribelt.nl',
      "Tribelt's group: Lesjöfors",
      'Competitors: Twentebelt',
    ])
    await wrapper.get('select').setValue('')
    expect(bands(wrapper)[2]).toBe('Competitors: Twentebelt, ESFO, Unibelt')
  })

  it('heads the page with the leader and counts the sites passing each check', async () => {
    const { wrapper } = await mountApp(api(), '/ai-search/readiness')
    expect(wrapper.get('h1').text()).toBe('ESFO leads with 5 of five checks. Tribelt passes 3½.')
    expect(wrapper.text()).toContain('The five checks ask whether a site lets every AI crawler in, has a sitemap, has llms.txt, has llms-full.txt and has Markdown pages.')
    expect(wrapper.text()).toContain('5 sites. Scanned on 22 October 2026.')
    const part = (label: string) =>
      wrapper.findAll(`section[aria-labelledby="sum-title"] tbody[aria-label="${label}"] tr`).map((r) => r.findAll('th, td').map((c) => c.text()))
    const counted = part('The five checks the ranking counts')
    expect(counted.map((r) => r[0])).toEqual([
      'The five checks the ranking counts',
      'Lets every AI crawler in',
      'Has a sitemap',
      'Has llms.txt',
      'Has llms-full.txt',
      'Has Markdown pages',
    ])
    expect(counted[1]).toEqual(['Lets every AI crawler in', '2', '✕ No'])
    expect(counted[2]).toEqual(['Has a sitemap', '5', '✓ Yes'])
    expect(counted[3]).toEqual(['Has llms.txt', '2', '✕ No'])
    expect(part('Also checked').map((r) => r[0])).toEqual(['Also checked', 'Sitemap in Dutch, German and English', 'Organization structured data', 'Product structured data'])
  })

  it('marks each crawler by shape, and opens the site in a new tab', async () => {
    const { wrapper } = await mountApp(api(), '/ai-search/readiness')
    const row = wrapper.findAll('section[aria-labelledby="sites-title"] tbody tr').find((r) => r.text().startsWith('Twentebelt'))
    const marks = (cells: DOMWrapper<Element>[] | undefined) => cells?.map((c) => `${c.find('[data-mark]').attributes('data-mark') ?? ''} ${c.text()}`)
    expect(marks(row?.findAll('td').slice(4, 6))).toEqual(['✓ ChatGPT-User may read every page.', '✕ GPTBot may not read any page.'])
    expect(row?.findAll('td')[5]?.attributes('title')).toBe('GPTBot may not read any page.')
    // The file columns tick and cross with the same symbols, after the sitemap.
    expect(marks(row?.findAll('td').slice(11, 14))).toEqual(['✕ No llms.txt.', '✓ Has llms-full.txt.', '✓ Has Markdown pages.'])
    const legend = wrapper.get('ul[aria-label="What the crawler columns say"]')
    expect(marks(legend.findAll('li'))?.map((l) => l.slice(0, 1))).toEqual(['✓', '~', '✕', '–'])
    expect(row?.get('a[target="_blank"]').attributes('href')).toBe('https://www.twentebelt.com')
    expect(wrapper.text()).toContain('Not scanned yet')
  })

  it("shows Tribelt's robots.txt and what it means, and another site's when picked", async () => {
    const { wrapper, router } = await mountApp(api(), '/ai-search/readiness')
    const panel = wrapper.get('section[aria-labelledby="sites-title"] [aria-live="polite"]')
    expect(panel.text()).toContain("Tribelt B.V.'s robots.txt")
    expect(panel.get('pre').text()).toContain('Disallow: /')
    expect(panel.text()).toContain('Blocks GPTBot.')
    const esfo = wrapper.findAll('th[scope="row"] button').find((b) => b.text().startsWith('ESFO'))
    await esfo?.trigger('click')
    await settle()
    expect(router.currentRoute.value.query.site).toBe('esfo')
    expect(esfo?.attributes('aria-pressed')).toBe('true')
    expect(wrapper.get('section[aria-labelledby="sites-title"] [aria-live="polite"]').text()).toContain('Lets every search and AI crawler in.')
    const unibelt = wrapper.findAll('th[scope="row"] button').find((b) => b.text().startsWith('Unibelt'))
    await unibelt?.trigger('click')
    await settle()
    expect(wrapper.get('section[aria-labelledby="sites-title"] [aria-live="polite"]').text()).toContain('No scan has recorded its robots.txt yet.')
  })

  it('lets an administrator check every site again without a scan, and shows how far it has come', async () => {
    let started = false
    const { wrapper } = await mountApp(
      api({
        'GET /api/v1/me': () => json(me({ isAdmin: true })),
        'GET /api/v1/ai-readiness': () =>
          json(readinessTable(table, started ? { running: true, checked: 2, total: 5 } : {})),
        'POST /api/v1/admin/readiness-checks': () => {
          started = true
          return json({ running: true, checked: 0, failed: 0, total: 5 }, 202)
        },
      }),
      '/ai-search/readiness',
    )
    const check = wrapper.findAll('button').find((b) => b.text() === 'Check AI readiness again')
    if (!check) throw new Error('no check button')
    await check.trigger('click')
    await settle()
    expect(wrapper.text()).toContain('Checking 2 of 5 sites again.')
    expect(check.attributes('disabled')).toBeDefined()
  })

  it('shows no check to an account that is no administrator', async () => {
    const { wrapper } = await mountApp(api(), '/ai-search/readiness')
    expect(wrapper.findAll('button').some((b) => b.text() === 'Check AI readiness again')).toBe(false)
  })

  it('says when the table cannot load', async () => {
    const { wrapper } = await mountApp(routes({ 'GET /api/v1/ai-readiness': () => problem(500, 'Internal Server Error', 'Down.') }), '/ai-search/readiness')
    expect(wrapper.get('[role="alert"]').text()).toContain('Down.')
  })
})
