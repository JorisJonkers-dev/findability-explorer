import { describe, expect, it } from 'vitest'
import { ANALYSIS_ID, analysis, competitor, me, readinessRow, summary, term, termSearch, readinessTable } from '../../../testing/fixtures'
import { type Handler, json, mountApp, problem, routes, settle } from '../../../testing/mountApp'

const martens = competitor('Märtens', {
  scans: 2,
  lastScan: summary({ competitorId: 'martens' }),
  websiteUrl: 'https://www.martens-belts.com/',
  beltNames: [
    { name: 'Drahtösengliederband', family: 'eyelink' },
    { name: 'Spiralgeflechtband', family: 'open_spiral' },
  ],
})

function api(admin = false, extra: Record<string, Handler> = {}) {
  return routes({
    'GET /api/v1/me': () => json(me({ isAdmin: admin })),
    'GET /api/v1/competitors': () => json({ items: [martens] }),
    'GET /api/v1/competitors/martens': () => json({ competitor: martens, scans: [summary({ status: 'running', id: '1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d' }), summary({ competitorId: 'martens' })] }),
    [`GET /api/v1/analyses/${ANALYSIS_ID}`]: () => json(analysis()),
    'GET /api/v1/terms': () => json(termSearch([])),
    'GET /api/v1/ai-readiness': () => json(readinessTable([readinessRow('Märtens', { competitorId: 'martens' })])),
    [`GET /api/v1/analyses/${ANALYSIS_ID}/readiness-files`]: () =>
      json({ robotsTxt: 'User-agent: *', robotsTruncated: false, llmsTxt: '', llmsTruncated: false, llmsFullTxt: '', llmsFullTruncated: false }),
    ...extra,
  })
}

describe("a company's latest scan", () => {
  it('shows the crawl of its newest finished scan, the belts it sells and its AI readiness', async () => {
    const { wrapper } = await mountApp(api(), '/competitors/martens')
    expect(document.title).toBe('Latest scan · Findability Explorer')
    expect(wrapper.get('h1').text()).toBe("Märtens sells 2 of Tribelt's five belt families.")
    expect(wrapper.text()).toContain('Back to Competitors')
    expect(wrapper.find('nav[aria-label="In this section"]').exists()).toBe(false)
    const belts = wrapper.get('section[aria-labelledby="belts-title"]')
    expect(belts.findAll('tbody tr').map((r) => r.text())).toContain('EyelinkDrahtösengliederband')
    expect(wrapper.get('section[aria-labelledby="crawl-title"] a[target="_blank"]').attributes('href')).toBe('https://www.martens-belts.com/')
    expect(wrapper.get('section[aria-labelledby="ready-title"]').text()).toContain('User-agent: *')
    expect(wrapper.findAll('button').some((b) => b.text() === 'Scan again')).toBe(false)
  })

  it('lets an administrator scan it again', async () => {
    let sent: unknown = null
    const { wrapper } = await mountApp(
      api(true, {
        'POST /api/v1/admin/rescans': async (r) => {
          sent = await r.json()
          return json({ queued: [], skipped: [] }, 202)
        },
      }),
      '/competitors/martens',
    )
    await wrapper.findAll('button').find((b) => b.text() === 'Scan again')?.trigger('click')
    await settle()
    expect(sent).toEqual({ competitorIds: ['martens'] })
  })

  it('filters its phrases, and the map terms it uses, by one language', async () => {
    const terms = [term({ term: 'Förderband', language: 'de', usedBy: ['Märtens'] }), term({ term: 'conveyor belt', language: 'en-US', usedBy: ['Märtens'] })]
    const { wrapper } = await mountApp(api(false, { 'GET /api/v1/terms': () => json(termSearch(terms)) }), '/competitors/martens')
    const used = () => wrapper.findAll('ul[aria-label="Its terms in the terminology map"] li').map((li) => li.text())
    const ranked = () => wrapper.findAll('ol[aria-label="Ranked phrases"] li').length
    expect(ranked()).toBe(3)
    expect(used()).toHaveLength(2)
    const group = wrapper.get('[role="group"][aria-label="Language"]')
    expect(group.findAll('button').map((b) => b.text())).toEqual(['All languages', 'German', 'English'])
    await group.findAll('button').find((b) => b.text() === 'English')?.trigger('click')
    await settle()
    expect(ranked()).toBe(2)
    expect(used()).toEqual([expect.stringContaining('conveyor belt')])
  })

  it('says when no company has the id', async () => {
    const { wrapper } = await mountApp(api(false, { 'GET /api/v1/competitors/nobody': () => problem(404, 'Not Found', 'No company has this id.') }), '/competitors/nobody')
    expect(wrapper.get('[role="alert"]').text()).toBe('No company has this id.')
  })
})
