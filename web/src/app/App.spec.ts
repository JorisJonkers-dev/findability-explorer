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
    const { wrapper } = await mountApp(routes({ 'GET /api/v1/me': () => pending }), '/plan')
    expect(wrapper.text()).toContain('Signing in…')
    expect(wrapper.find('h1').exists()).toBe(false)
    answer(json(me()))
    await settle()
    expect(wrapper.get('h1').text()).toBe('Measure first. Then build. Then expand.')
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

  it('names who is signed in, offers Settings to an administrator only, and signs out with a form', async () => {
    const reader = await mountApp(api, '/plan')
    const account = reader.wrapper.get('nav[aria-label="Account"]')
    expect(account.text()).toContain('Signed in as Joris Jonkers')
    expect(account.find('a[href="/settings"]').exists()).toBe(false)
    const form = account.get('form')
    expect(form.attributes('method')).toBe('post')
    expect(form.attributes('action')).toBe('/auth/logout')
    expect(form.get('button[type="submit"]').text()).toBe('Sign out')
    reader.wrapper.unmount()

    const admin = await mountApp(routes({ 'GET /api/v1/me': () => json(me({ isAdmin: true })) }), '/plan')
    expect(admin.wrapper.get('nav[aria-label="Account"] a[href="/settings"]').text()).toBe('Settings')
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
      ['/plan', 'The plan'],
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
    await router.push('/competitors/twentebelt')
    expect(router.currentRoute.value.fullPath).toBe('/competitors?company=twentebelt')
    await router.push('/scan?x=1')
    expect(router.currentRoute.value.fullPath).toBe('/competitors?x=1')
    await router.push('/analyses')
    expect(router.currentRoute.value.fullPath).toBe('/competitors')
    await router.push(`/analyses/${ANALYSIS_ID}`)
    expect(router.currentRoute.value.fullPath).toBe(`/scan/${ANALYSIS_ID}`)
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
