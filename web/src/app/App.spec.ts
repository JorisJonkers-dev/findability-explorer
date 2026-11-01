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
    expect(wrapper.get('h1').text()).toBe('Recommendations')
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

  it('offers Manage to an administrator only, and signs out with a form', async () => {
    const reader = await mountApp(api, '/recommendations')
    const bar = reader.wrapper.get('nav[aria-label="Main"]')
    expect(bar.find('a[href="/manage"]').exists()).toBe(false)
    expect(reader.wrapper.find('a[href="/settings"]').exists()).toBe(false)
    const form = bar.get('form')
    expect(form.attributes('method')).toBe('post')
    expect(form.attributes('action')).toBe('/auth/logout')
    expect(form.get('button[type="submit"]').text()).toBe('Sign out')
    reader.wrapper.unmount()

    const admin = await mountApp(routes({ 'GET /api/v1/me': () => json(me({ isAdmin: true })) }), '/recommendations')
    expect(admin.wrapper.get('nav[aria-label="Main"] a[href="/manage"]').text()).toBe('Manage')
    expect(admin.wrapper.get('nav[aria-label="Main"] a[href="/manage"]').attributes('aria-current')).toBeUndefined()
  })
})

describe('the top bar', () => {
  it('links to every Results section in order and marks the one shown', async () => {
    const { wrapper } = await mountApp(api, '/ai-responses/all')
    const links = wrapper.findAll('nav[aria-label="Main"] a').map((a) => [a.text(), a.attributes('href'), a.attributes('aria-current') ?? ''])
    expect(links).toEqual([
      ['Findability Explorer', '/', ''],
      ['Overview', '/', ''],
      ['AI responses', '/ai-responses', 'page'],
      ['AI readiness', '/ai-readiness', ''],
      ['Search terms', '/terms', ''],
      ['Competitors', '/competitors', ''],
      ['Recommendations', '/recommendations', ''],
      ['Help', '/help', ''],
    ])
  })

  it('marks Manage, and no Results section, on every Manage page', async () => {
    const { wrapper, router } = await mountApp(routes({ 'GET /api/v1/me': () => json(me({ isAdmin: true })) }), '/manage/keys')
    for (const path of ['/manage', '/manage/keys', '/manage/terms/waiting']) {
      await router.push(path)
      await settle()
      const bar = wrapper.get('nav[aria-label="Main"]')
      expect(bar.get('a[href="/manage"]').attributes('aria-current')).toBe('page')
      expect(bar.findAll('a[aria-current="page"]')).toHaveLength(1)
    }
  })

  it('has no section tabs, and a detail page links back to its section', async () => {
    const { wrapper } = await mountApp(api, '/ai-responses/all')
    expect(wrapper.find('nav[aria-label="In this section"]').exists()).toBe(false)
    expect(wrapper.find('a[href="/ai-responses"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Back to AI responses')
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
    const { router } = await mountApp(routes({ 'GET /api/v1/me': () => json(me({ isAdmin: true })) }), '/')
    expect(document.title).toBe('Overview · Findability Explorer')
    const titles: [string, string][] = [
      ['/goals/aiShareOfVoice', 'Goal'],
      ['/ai-responses', 'AI responses'],
      ['/ai-responses/all', 'Every response'],
      ['/ai-readiness', 'AI readiness'],
      ['/terms', 'Search terms'],
      ['/terms/all', 'Every term'],
      ['/competitors', 'Competitors'],
      ['/recommendations', 'Recommendations'],
      ['/help', 'Help'],
      ['/help/prompt-test', 'Help'],
      ['/manage', 'Activity'],
      ['/manage/prompt-test', 'Prompt test'],
      ['/manage/scans', 'Website scans'],
      [`/manage/scans/${ANALYSIS_ID}`, 'Scan'],
      ['/manage/competitors/candidates', 'Candidates'],
      ['/manage/terms/waiting', 'Waiting terms'],
      ['/manage/recommendations', 'Recommendations'],
      ['/manage/keys', 'Keys and spending'],
    ]
    for (const [path, title] of titles) {
      await router.push(path)
      await settle()
      expect(document.title, path).toBe(`${title} · Findability Explorer`)
    }
  })

  it('says of every page whether it is in Results or in Manage', async () => {
    const { router } = await mountApp(api, '/')
    const pages = router.getRoutes().filter((r) => r.components)
    expect(pages.length).toBeGreaterThan(20)
    for (const page of pages) {
      expect(typeof page.meta.title, page.path).toBe('string')
      expect(page.meta.view, page.path).toBe(page.path.startsWith('/manage') ? 'manage' : 'results')
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
    expect(router.currentRoute.value.fullPath).toBe('/terms/all?q=spiral&tier=product')
    const moves: [string, string][] = [
      ['/keywords?q=mesh', '/terms/all?q=mesh'],
      ['/scan?x=1', '/competitors?x=1'],
      ['/analyses', '/competitors'],
      [`/analyses/${ANALYSIS_ID}?company=twentebelt`, `/manage/scans/${ANALYSIS_ID}?company=twentebelt`],
      ['/settings/candidates?status=added', '/manage/competitors/candidates?status=added'],
      ['/catching-up', '/'],
      ['/plan', '/recommendations'],
      ['/products', '/terms'],
      ['/products/open-spiral', '/terms/families/open-spiral'],
      ['/ai-search/prompt-test', '/ai-responses/all'],
      ['/competitors/all', '/competitors'],
    ]
    for (const [from, to] of moves) {
      await router.push(from)
      expect(router.currentRoute.value.fullPath, from).toBe(to)
    }
  })

  it("sends every 1.12.0 address to its new place, keeping the query", async () => {
    const { router } = await mountApp(api, '/')
    const moves: [string, string][] = [
      ['/numbers/aiShareOfVoice', '/goals/aiShareOfVoice'],
      ['/ai-search', '/ai-responses'],
      ['/ai-search/answers?model=openai', '/ai-responses/all?model=openai'],
      ['/ai-search/readiness', '/ai-readiness'],
      ['/buyers', '/'],
      [`/scan/${ANALYSIS_ID}?company=twentebelt`, `/manage/scans/${ANALYSIS_ID}?company=twentebelt`],
      ['/settings', '/manage/keys'],
      ['/settings/prompt-test', '/manage/prompt-test'],
      ['/settings/terms?status=accepted', '/manage/terms/waiting?status=accepted'],
      ['/settings/competitors', '/manage/competitors/candidates'],
      ['/how-it-works', '/help'],
    ]
    for (const [from, to] of moves) {
      await router.push(from)
      expect(router.currentRoute.value.fullPath, from).toBe(to)
    }
  })

  it('opens a Manage section on its one subpage until the section has a page of its own', async () => {
    const { router } = await mountApp(api, '/manage/competitors')
    expect(router.currentRoute.value.fullPath).toBe('/manage/competitors/candidates')
    await router.push('/manage/terms')
    expect(router.currentRoute.value.fullPath).toBe('/manage/terms/waiting')
  })

  it('keeps the scroll position within a view, and starts a new view at the top', async () => {
    const { router } = await mountApp(api, '/terms')
    const scroll = router.options.scrollBehavior
    // A resolved location stands in for the normalized one the router passes after a navigation.
    const at = (path: string) => router.resolve(path) as RouteLocationNormalized
    expect(await scroll?.(at('/terms?q=a'), at('/terms'), null)).toBe(false)
    expect(await scroll?.(at(`/manage/scans/${ANALYSIS_ID}`), at(`/manage/scans/other`), null)).toBe(false)
    expect(await scroll?.(at(`/manage/scans/${ANALYSIS_ID}`), at(`/competitors`), null)).toEqual({ top: 0 })
    expect(await scroll?.(at('/competitors'), at('/terms'), null)).toEqual({ top: 0 })
    expect(await scroll?.(at('/competitors'), at('/terms'), { left: 0, top: 120 })).toEqual({ left: 0, top: 120 })
    // A Help topic's address brings the topic into view itself.
    expect(await scroll?.(at('/help/stop'), at('/terms'), null)).toBe(false)
    expect(await scroll?.(at('/help'), at('/terms'), null)).toEqual({ top: 0 })
  })
})
