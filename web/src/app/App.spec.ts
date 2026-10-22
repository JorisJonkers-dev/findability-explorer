import { describe, expect, it } from 'vitest'
import type { RouteLocationNormalized } from 'vue-router'
import { ANALYSIS_ID, analysis, summary, termSearch } from '../testing/fixtures'
import { json, mountApp, problem, routes, settle } from '../testing/mountApp'

const api = routes({
  'GET /api/v1/terms': () => json(termSearch([])),
  'GET /api/v1/competitors': () => json({ items: [] }),
  'GET /api/v1/analyses': () => json({ items: [summary({ createdAt: new Date().toISOString() }), summary({ id: 'f'.repeat(8) + '-0000-4000-8000-000000000000', url: 'https://www.ashworth.com', status: 'running' })] }),
  [`GET /api/v1/analyses/${ANALYSIS_ID}`]: () => json(analysis()),
})

describe('the shell', () => {
  it('names the app and links to every section', async () => {
    const { wrapper } = await mountApp(api, '/')
    expect(wrapper.text()).toContain('Findability Explorer')
    expect(wrapper.text()).toContain('How metal belts are named, and found')
    const nav = wrapper.get('nav[aria-label="Main"]')
    expect(nav.findAll('a').map((a) => [a.text(), a.attributes('href')])).toEqual([
      ['Overview', '/'],
      ['Scan a competitor', '/scan'],
      ['Terminology', '/terminology'],
      ['Competitors', '/competitors'],
    ])
    expect(wrapper.text()).toContain('Built for Tribelt B.V. in the Digital Strategy course, University of Twente.')
  })

  it('opens on the overview', async () => {
    const { wrapper } = await mountApp(api, '/')
    expect(wrapper.get('h1').text()).toBe('One belt, five names. Buyers search with all of them.')
    expect(document.title).toBe('Overview · Findability Explorer')
  })

  it("marks the current section's link, also on the pages under it", async () => {
    const { router, wrapper } = await mountApp(api, '/competitors')
    const current = () => wrapper.findAll('nav[aria-label="Main"] [aria-current="page"]').map((a) => a.text())
    expect(current()).toEqual(['Competitors'])
    await router.push(`/scan/${ANALYSIS_ID}`)
    await settle()
    expect(current()).toEqual(['Scan a competitor'])
    expect(document.title).toBe('Scan a competitor · Findability Explorer')
    await router.push('/')
    await settle()
    expect(current()).toEqual(['Overview'])
  })

  it('lists the recent scans, marking the one on screen', async () => {
    const { router, wrapper } = await mountApp(api, '/')
    const recent = wrapper.get('section[aria-labelledby="recent-title"]')
    expect(recent.findAll('a').map((a) => a.findAll('span').map((s) => s.text()))).toEqual([
      ['twentebelt.com', 'today'],
      ['ashworth.com', 'scanning'],
    ])
    expect(recent.get('a').attributes('href')).toBe(`/scan/${ANALYSIS_ID}`)
    await router.push(`/scan/${ANALYSIS_ID}`)
    await settle()
    expect(recent.findAll('[aria-current="page"]').map((a) => a.attributes('href'))).toEqual([`/scan/${ANALYSIS_ID}`])
    // On a phone the same list folds into a disclosure.
    expect(wrapper.get('details summary').text()).toBe('Recent scans')
  })

  it('says when there are no scans yet, or they cannot be loaded', async () => {
    const empty = await mountApp(routes({ 'GET /api/v1/analyses': () => json({ items: [] }) }), '/')
    expect(empty.wrapper.get('section[aria-labelledby="recent-title"]').text()).toContain('No scans yet.')
    empty.wrapper.unmount()
    const failing = await mountApp(routes({ 'GET /api/v1/analyses': () => problem(500, 'Internal Server Error') }), '/')
    expect(failing.wrapper.get('section[aria-labelledby="recent-title"]').text()).toContain('Internal Server Error')
  })

  it('titles a page without its own title after the app', async () => {
    const { router } = await mountApp(api, '/')
    router.addRoute({ path: '/plain', component: { render: () => null } })
    await router.push('/plain')
    await settle()
    expect(document.title).toBe('Findability Explorer')
  })
})

describe('the routes', () => {
  it('sends unknown paths to the overview', async () => {
    const { router } = await mountApp(api, '/no/such/page')
    expect(router.currentRoute.value.fullPath).toBe('/')
  })

  it("sends the first version's addresses to their new pages, keeping the query", async () => {
    const { router } = await mountApp(api, '/keywords?q=spiral&tier=product')
    expect(router.currentRoute.value.fullPath).toBe('/terminology?q=spiral&tier=product')
    await router.push('/analyses?x=1')
    expect(router.currentRoute.value.fullPath).toBe('/scan?x=1')
    await router.push(`/analyses/${ANALYSIS_ID}`)
    expect(router.currentRoute.value.fullPath).toBe(`/scan/${ANALYSIS_ID}`)
  })

  it('keeps the scroll position for a query change, and starts a new page at the top', async () => {
    const { router } = await mountApp(api, '/terminology')
    const scroll = router.options.scrollBehavior
    // A resolved location stands in for the normalized one the router passes after a navigation.
    const at = (path: string) => router.resolve(path) as RouteLocationNormalized
    expect(await scroll?.(at('/terminology?q=a'), at('/terminology'), null)).toBe(false)
    expect(await scroll?.(at('/competitors'), at('/terminology'), null)).toEqual({ top: 0 })
    expect(await scroll?.(at('/competitors'), at('/terminology'), { left: 0, top: 120 })).toEqual({ left: 0, top: 120 })
  })
})
