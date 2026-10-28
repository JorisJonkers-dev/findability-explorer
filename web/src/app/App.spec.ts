import { describe, expect, it } from 'vitest'
import type { RouteLocationNormalized } from 'vue-router'
import { ANALYSIS_ID, analysis, competitor, me, summary, termSearch } from '../testing/fixtures'
import { json, mountApp, problem, routes, settle } from '../testing/mountApp'

const api = routes({
  'GET /api/v1/terms': () => json(termSearch([])),
  'GET /api/v1/competitors': () => json({ items: [competitor('Twentebelt')] }),
  'GET /api/v1/competitors/twentebelt': () => json({ competitor: competitor('Twentebelt'), scans: [] }),
  'GET /api/v1/analyses': () => json({ items: [summary()] }),
  [`GET /api/v1/analyses/${ANALYSIS_ID}`]: () => json(analysis()),
})

describe('signing in', () => {
  it('waits for the account before it shows anything', async () => {
    let answer: (response: Response) => void = () => undefined
    const pending = new Promise<Response>((resolve) => {
      answer = resolve
    })
    const { wrapper } = await mountApp(routes({ 'GET /api/v1/me': () => pending }), '/recommendations')
    expect(wrapper.text()).toContain('Signing in…')
    expect(wrapper.find('h1').exists()).toBe(false)
    answer(json(me()))
    await settle()
    expect(wrapper.get('h1').text()).toBe('Check every Monday. Fix one thing a week. Watch the numbers move.')
  })

  it('sends the browser to sign in when the session has ended, and back to the same page after', async () => {
    const { assign, wrapper } = await mountApp(routes({ 'GET /api/v1/me': () => problem(401, 'Unauthorized') }), '/terms?q=spiral')
    expect(assign).toHaveBeenCalledWith('/auth/login?return=%2Fterms%3Fq%3Dspiral')
    expect(wrapper.find('h1').exists()).toBe(false)
  })

  it('does the same for any other request that answers 401', async () => {
    const { assign } = await mountApp(
      routes({ 'GET /api/v1/competitors': () => problem(401, 'Unauthorized'), 'GET /api/v1/terms': () => json(termSearch([])) }),
      '/competitors',
    )
    expect(assign).toHaveBeenCalledWith('/auth/login?return=%2Fcompetitors')
  })

  it('says when the account cannot be loaded', async () => {
    const { wrapper } = await mountApp(routes({ 'GET /api/v1/me': () => problem(500, 'Internal Server Error') }), '/')
    expect(wrapper.get('[role="alert"]').text()).toBe('Internal Server Error')
  })

  it('offers Settings to an administrator only, and signs out with a form', async () => {
    const reader = await mountApp(api, '/recommendations')
    const bar = reader.wrapper.get('nav[aria-label="Main"]')
    expect(bar.find('a[href="/settings"]').exists()).toBe(false)
    const form = bar.get('form')
    expect(form.attributes('method')).toBe('post')
    expect(form.attributes('action')).toBe('/auth/logout')
    expect(form.get('button[type="submit"]').text()).toBe('Sign out')
    reader.wrapper.unmount()

    const admin = await mountApp(routes({ 'GET /api/v1/me': () => json(me({ isAdmin: true })) }), '/recommendations')
    expect(admin.wrapper.get('nav[aria-label="Main"] a[href="/settings"]').text()).toBe('Settings')
  })
})

describe('the top bar', () => {
  it('links to every section in order and marks the one shown', async () => {
    const { wrapper } = await mountApp(api, '/ai-search/answers')
    const links = wrapper.findAll('nav[aria-label="Main"] a').map((a) => [a.text(), a.attributes('href'), a.attributes('aria-current') ?? ''])
    expect(links).toEqual([
      ['Findability Explorer', '/', ''],
      ['Overview', '/', ''],
      ['AI search', '/ai-search', 'page'],
      ['Competitors', '/competitors', ''],
      ['Search terms', '/terms', ''],
      ['Recommendations', '/recommendations', ''],
    ])
  })

  it("marks Settings on a settings page", async () => {
    const { wrapper } = await mountApp(routes({ 'GET /api/v1/me': () => json(me({ isAdmin: true })) }), '/settings/terms')
    expect(wrapper.get('nav[aria-label="Main"] a[href="/settings"]').attributes('aria-current')).toBe('page')
    expect(wrapper.get('nav[aria-label="Settings"] a[aria-current="page"]').text()).toBe('Term review')
  })

  it('has no section tabs, and a detail page links back to its section', async () => {
    const { wrapper } = await mountApp(api, '/ai-search/answers')
    expect(wrapper.find('nav[aria-label="In this section"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('Back to AI search')
  })

  it('redirects the old routes to the new ones', async () => {
    const { router } = await mountApp(api, '/catching-up')
    expect(router.currentRoute.value.fullPath).toBe('/')
    const moves: [string, string][] = [
      ['/plan', '/recommendations'],
      ['/products', '/terms'],
      ['/products/open-spiral', '/terms/families/open-spiral'],
      ['/ai-search/prompt-test', '/ai-search/answers'],
      ['/competitors/all', '/competitors'],
    ]
    for (const [from, to] of moves) {
      await router.push(from)
      await settle()
      expect(router.currentRoute.value.fullPath).toBe(to)
    }
  })

  it('opens a page per belt family, with a link back to Search terms', async () => {
    const { wrapper, router } = await mountApp(api, '/terms/families/open-spiral')
    expect(wrapper.get('h1').text()).toBe('Open spiral')
    expect(wrapper.text()).toContain('Back to Search terms')
    await router.push('/terms/families/nothing')
    await settle()
    expect(wrapper.get('h1').text()).toBe('No such belt family.')
  })
})

describe('the routes', () => {
  it('titles each view', async () => {
    const { router } = await mountApp(api, '/')
    expect(document.title).toBe('Overview · Findability Explorer')
    const titles: [string, string][] = [
      ['/buyers', 'Buyers'],
      ['/ai-search', 'AI search'],
      ['/competitors', 'Competitors'],
      [`/scan/${ANALYSIS_ID}`, 'Scan'],
      ['/terms', 'Search terms'],
      ['/recommendations', 'Recommendations'],
    ]
    for (const [path, title] of titles) {
      await router.push(path)
      await settle()
      expect(document.title).toBe(`${title} · Findability Explorer`)
    }
  })

  it('titles a view without its own title after the app', async () => {
    const { router } = await mountApp(api, '/')
    router.addRoute({ path: '/plain', component: { render: () => null } })
    await router.push('/plain')
    await settle()
    expect(document.title).toBe('Findability Explorer')
  })

  it('sends unknown paths to the poster', async () => {
    const { router } = await mountApp(api, '/no/such/page')
    expect(router.currentRoute.value.fullPath).toBe('/')
  })

  it("sends earlier versions' addresses to their new views, keeping the query", async () => {
    const { router } = await mountApp(api, '/terminology?q=spiral&tier=product')
    expect(router.currentRoute.value.fullPath).toBe('/terms?q=spiral&tier=product')
    await router.push('/keywords?q=mesh')
    expect(router.currentRoute.value.fullPath).toBe('/terms?q=mesh')
    await router.push('/scan?x=1')
    expect(router.currentRoute.value.fullPath).toBe('/competitors?x=1')
    await router.push('/analyses')
    expect(router.currentRoute.value.fullPath).toBe('/competitors')
    await router.push(`/analyses/${ANALYSIS_ID}`)
    expect(router.currentRoute.value.fullPath).toBe(`/scan/${ANALYSIS_ID}`)
    await router.push('/settings/candidates')
    expect(router.currentRoute.value.fullPath).toBe('/settings/competitors')
  })

  it('keeps the scroll position within a view, and starts a new view at the top', async () => {
    const { router } = await mountApp(api, '/terms')
    const scroll = router.options.scrollBehavior
    // A resolved location stands in for the normalized one the router passes after a navigation.
    const at = (path: string) => router.resolve(path) as RouteLocationNormalized
    expect(await scroll?.(at('/terms?q=a'), at('/terms'), null)).toBe(false)
    expect(await scroll?.(at(`/scan/${ANALYSIS_ID}`), at('/competitors'), null)).toBe(false)
    expect(await scroll?.(at('/competitors'), at('/terms'), null)).toEqual({ top: 0 })
    expect(await scroll?.(at('/competitors'), at('/terms'), { left: 0, top: 120 })).toEqual({ left: 0, top: 120 })
  })
})

describe('the settings menu', () => {
  it('shows how many terms wait in Term review on every settings page', async () => {
    const { wrapper } = await mountApp(
      routes({
        'GET /api/v1/me': () => json(me({ isAdmin: true })),
        'GET /api/v1/admin/term-candidates': () => json({ items: [], counts: { pending: 8, accepted: 0, rejected: 0 }, review: { running: false, waiting: 0 } }),
      }),
      '/settings/prompt-test',
    )
    expect(wrapper.get('nav[aria-label="Settings"] a[href="/settings/terms"]').text()).toBe('Term review 8')
    expect(wrapper.find('nav[aria-label="Settings"] [aria-label="8 waiting"]').exists()).toBe(true)
  })
})
