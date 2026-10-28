import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { Analysis } from '../../../infrastructure/api/types.gen'
import { ANALYSIS_ID, analysis, competitor, me } from '../../../testing/fixtures'
import { type Handler, json, mountApp, problem, routes, settle } from '../../../testing/mountApp'

function api(extra: Record<string, Handler> = {}) {
  return routes({
    'GET /api/v1/me': () => json(me({ isAdmin: true })),
    'GET /api/v1/competitors': () => json({ items: [competitor('Twentebelt')] }),
    'GET /api/v1/terms': () => json({ items: [], total: 0, facets: { languages: [], tiers: [], families: [] } }),
    [`GET /api/v1/analyses/${ANALYSIS_ID}`]: () => json(analysis()),
    ...extra,
  })
}

const pill = (wrapper: VueWrapper, group: string, label: string) => {
  const button = wrapper.get(`[role="group"][aria-label="${group}"]`).findAll('button').find((b) => b.text() === label)
  if (!button) throw new Error(`no pill ${label}`)
  return button
}

const scan = (overrides: Partial<Analysis>) => api({ [`GET /api/v1/analyses/${ANALYSIS_ID}`]: () => json(analysis(overrides)) })

describe('a scan', () => {
  it('has a page of its own, with a way back to Competitors', async () => {
    const { wrapper, router } = await mountApp(api(), `/scan/${ANALYSIS_ID}`)
    expect(document.title).toBe('Scan · Findability Explorer')
    expect(wrapper.get('h1').text()).toBe('What the scan found.')
    const back = wrapper.findAll('a').find((a) => a.text().includes('Back to Competitors'))
    await back?.trigger('click')
    await settle()
    expect(router.currentRoute.value.fullPath).toBe('/competitors')
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
      `/scan/${ANALYSIS_ID}`,
    )
    expect(wrapper.get('[role="progressbar"]').attributes('aria-valuetext')).toBe('Reading page 4 of 30')
    await wrapper.findAll('a').find((a) => a.text() === 'Show the recent scans')?.trigger('click')
    await settle()
    expect(router.currentRoute.value.fullPath).toBe('/competitors')
  })

  it('offers an administrator the form to scan another website, and a viewer only the scan', async () => {
    const admin = await mountApp(api(), `/scan/${ANALYSIS_ID}`)
    expect(admin.wrapper.get('#scan-title').text()).toBe('Scan a competitor')
    expect(admin.wrapper.find('form[aria-label="Scan a website"]').exists()).toBe(true)
    admin.wrapper.unmount()
    const viewer = await mountApp(api({ 'GET /api/v1/me': () => json(me()) }), `/scan/${ANALYSIS_ID}`)
    expect(viewer.wrapper.get('#scan-title').text()).toBe('Scans')
    expect(viewer.wrapper.find('form[aria-label="Scan a website"]').exists()).toBe(false)
  })
})
