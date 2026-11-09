import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { CompetitorProfile, ScanComparison } from '../../../infrastructure/api/types.gen'
import { competitor, me, readinessRow, summary, readinessTable } from '../../../testing/fixtures'
import { type Handler, json, mountApp, problem, routes, settle } from '../../../testing/mountApp'
import { PROFILE_GROUPS } from '../domain/history'

const SCAN_1 = '11111111-1111-4111-8111-111111111111'
const SCAN_2 = '22222222-2222-4222-8222-222222222222'
const SCAN_3 = '33333333-3333-4333-8333-333333333333'

const twentebelt = competitor('Twentebelt', { scans: 3 })
const scans = [
  summary({ id: SCAN_3, number: 3, createdAt: '2026-10-23T09:30:00Z', matchingModel: 'anthropic: claude-haiku-5-5', maxPages: 60, pagesCrawled: 48 }),
  summary({ id: SCAN_2, number: 2, status: 'failed', createdAt: '2026-10-22T12:00:00Z', failure: 'Timed out', pagesCrawled: 0 }),
  summary({ id: SCAN_1, number: 1, createdAt: '2026-10-22T09:30:00Z' }),
]

const profile: CompetitorProfile = {
  competitorId: 'twentebelt',
  scan: scans[0],
  finishedScans: 2,
  languages: ['en', 'de'],
  groups: PROFILE_GROUPS.map((group) => ({
    group,
    terms:
      group === 'eyelink'
        ? [
            { phrase: 'eyelink belts', language: 'en', score: 42.5, pages: 6, tier: 'product', family: 'eyelink', inMap: true, firstScan: 1 },
            { phrase: 'draadogenband', language: 'nl', score: 12, pages: 2, tier: 'product', family: 'eyelink', closestTerm: 'draadogenbanden', inMap: false, firstScan: 3 },
            { phrase: 'eyelink conveyor', language: 'en', score: 8, pages: 1, tier: 'product', inMap: false, firstScan: 1 },
          ]
        : [],
  })),
}

const comparison: ScanComparison = {
  from: scans[2] ?? summary(),
  to: scans[0] ?? summary(),
  groups: PROFILE_GROUPS.map((group) => ({
    group,
    added: group === 'eyelink' ? [{ phrase: 'draadogenband', language: 'nl', tier: 'product', inMap: false, toScore: 12, toPages: 2 }] : [],
    removed: group === 'broad' ? [{ phrase: 'conveyor belts', language: 'en', tier: 'broad', inMap: true, fromScore: 20, fromPages: 4 }] : [],
    moved:
      group === 'eyelink'
        ? [{ phrase: 'eyelink belts', language: 'en', tier: 'product', inMap: true, fromScore: 12.04, toScore: 42.5, fromPages: 2, toPages: 6 }]
        : [],
  })),
}

function api(extra: Record<string, Handler> = {}) {
  return routes({
    'GET /api/v1/competitors/twentebelt': () => json({ competitor: twentebelt, scans }),
    'GET /api/v1/competitors/twentebelt/profile': () => json(profile),
    'GET /api/v1/competitors/twentebelt/scans/compare': () => json(comparison),
    ...extra,
  })
}

const group = (wrapper: VueWrapper, label: string) => {
  const button = wrapper.get('[role="group"][aria-label="Belt family"]').findAll('button').find((b) => b.text() === label)
  if (!button) throw new Error(`no group ${label}`)
  return button
}

describe('a company’s search terms and history', () => {
  it('shows the latest scan’s terms by family and language, the scans, and the latest two compared', async () => {
    const { wrapper, requests } = await mountApp(api(), '/competitors/twentebelt/history')
    expect(document.title).toBe('Over time · Findability Explorer')
    expect(wrapper.get('h1').text()).toBe('How Twentebelt changed, scan by scan.')
    expect(wrapper.text()).toContain('From Scan 3 of 23 Oct 2026, the latest of 2 finished scans.')
    expect(group(wrapper, 'Eyelink (3)').attributes('aria-pressed')).toBe('true')
    const tables = wrapper.findAll('section[aria-labelledby="profile-title"] table')
    expect(tables.map((t) => t.get('caption').text())).toEqual(['English (2)', 'Dutch (1)'])
    expect(tables[0]?.findAll('tbody tr')[0]?.text()).toContain('eyelink belts42.56\u00a0p.Productin the mapsince scan 1')
    expect(tables[1]?.text()).toContain('not in the map, closest: draadogenbanden')
    expect(tables[1]?.text()).toContain('new in this scan')
    expect(tables[0]?.text()).toContain('not in the map')

    await group(wrapper, 'Tri-Flex (0)').trigger('click')
    expect(wrapper.text()).toContain('The scan found no tri-flex terms.')

    const timeline = wrapper.get('ol[aria-label="Scans, newest first"]')
    expect(timeline.findAll('li').map((li) => li.text())).toEqual([
      'Scan 3, Finished23 Oct 2026. Finished. 48 of 60 pages.anthropic: claude-haiku-5-5',
      'Scan 2, Failed22 Oct 2026. Failed. 0 of 15 pages.no model',
      'Scan 1, Finished22 Oct 2026. Finished. 15 of 15 pages.no model',
    ])
    expect(timeline.get('a').attributes('href')).toBe(`/scan/${SCAN_3}?company=twentebelt`)

    expect(requests()).toContain(`GET /api/v1/competitors/twentebelt/scans/compare?from=${SCAN_1}&to=${SCAN_3}`)
    const compare = wrapper.get('section[aria-labelledby="compare-title"]')
    expect(compare.text()).toContain('1 added')
    expect(compare.text()).toContain('1 removed')
    expect(compare.text()).toContain('1 moved')
    expect(compare.findAll('h3').map((h) => h.text())).toEqual(['AI readiness', 'Eyelink', 'Broad'])
    const items = compare.findAll('li').map((li) => li.text())
    expect(items).toEqual([
      'Addeddraadogenband(nl) score 12, 2 pages',
      'Movedeyelink belts(en) score 12 → 42.5, pages 2 → 6',
      'Removedconveyor belts(en) was 20, 4 pages',
    ])
  })

  it('compares two other scans, and asks for two different ones', async () => {
    const { wrapper, requests } = await mountApp(api(), '/competitors/twentebelt/history')
    const [from, to] = wrapper.findAll('section[aria-labelledby="compare-title"] select')
    await to?.setValue(SCAN_1)
    expect(wrapper.text()).toContain('Pick two different scans.')
    await from?.setValue(SCAN_3)
    await settle()
    expect(requests()).toContain(`GET /api/v1/competitors/twentebelt/scans/compare?from=${SCAN_3}&to=${SCAN_1}`)
  })

  it('says when there is nothing to compare, no finished scan, or the comparison failed', async () => {
    const one = await mountApp(
      api({
        'GET /api/v1/competitors/twentebelt': () => json({ competitor: twentebelt, scans: [] }),
        'GET /api/v1/competitors/twentebelt/profile': () => json({ competitorId: 'twentebelt', finishedScans: 0, languages: [], groups: [] }),
      }),
      '/competitors/twentebelt/history',
    )
    expect(one.wrapper.text()).toContain('No scan of this company has finished yet.')
    expect(one.wrapper.text()).toContain('Not scanned yet.')
    expect(one.wrapper.text()).toContain('Two finished scans are needed to compare.')
    one.wrapper.unmount()

    const same = await mountApp(
      api({
        'GET /api/v1/competitors/twentebelt/scans/compare': () =>
          json({ ...comparison, groups: PROFILE_GROUPS.map((g) => ({ group: g, added: [], removed: [], moved: [] })) }),
      }),
      '/competitors/twentebelt/history',
    )
    expect(same.wrapper.text()).toContain('The two scans found the same terms.')
    same.wrapper.unmount()

    const failing = await mountApp(
      api({
        'GET /api/v1/competitors/twentebelt/scans/compare': () => problem(422, 'Unprocessable', 'The company has fewer than two finished scans.'),
        'GET /api/v1/competitors/twentebelt/profile': () => problem(500, 'Internal Server Error'),
      }),
      '/competitors/twentebelt/history',
    )
    expect(failing.wrapper.get('section[aria-labelledby="compare-title"] [role="alert"]').text()).toBe('The company has fewer than two finished scans.')
    expect(failing.wrapper.text()).toContain('Internal Server Error')
  })

  it('says when the company does not exist or cannot be loaded', async () => {
    const missing = await mountApp(api({ 'GET /api/v1/competitors/twentebelt': () => problem(404, 'Not Found') }), '/competitors/twentebelt/history')
    expect(missing.wrapper.get('[role="alert"]').text()).toBe('No company has this id.')
    expect(missing.wrapper.get('h1').text()).toBe('Search terms, scan by scan.')
    missing.wrapper.unmount()
    const failing = await mountApp(api({ 'GET /api/v1/competitors/twentebelt': () => problem(500, 'Internal Server Error') }), '/competitors/twentebelt/history')
    expect(failing.wrapper.get('[role="alert"]').text()).toBe('Internal Server Error')
  })
})

describe('a company’s AI readiness over time', () => {
  it('lists what changed in AI readiness between the two scans', async () => {
    const { wrapper } = await mountApp(
      api({
        'GET /api/v1/competitors/twentebelt/scans/compare': () =>
          json({ ...comparison, readiness: [{ field: 'bot:GPTBot', from: 'allowed', to: 'blocked' }, { field: 'llmsTxt', from: 'no', to: 'yes' }] }),
      }),
      '/competitors/twentebelt/history',
    )
    const changes = wrapper.get('[data-testid="readiness-changes"]')
    expect(changes.findAll('li').map((li) => li.text())).toEqual(['GPTBot: allowed → blocked', 'llms.txt: no → yes'])
  })

  it('says when nothing changed', async () => {
    const { wrapper } = await mountApp(
      api({ 'GET /api/v1/competitors/twentebelt/scans/compare': () => json({ ...comparison, readiness: [] }) }),
      '/competitors/twentebelt/history',
    )
    expect(wrapper.get('[data-testid="readiness-changes"]').text()).toContain('Nothing changed.')
  })

  it('shows administrators the raw files, with the lines about AI crawlers marked', async () => {
    const files = { robotsTxt: 'User-agent: GPTBot\nDisallow: /\n\nUser-agent: *\nAllow: /', robotsTruncated: false, llmsTxt: '# Twentebelt', llmsTruncated: true, llmsFullTxt: '', llmsFullTruncated: false }
    const { wrapper, requests } = await mountApp(
      api({
        'GET /api/v1/me': () => json(me({ isAdmin: true })),
        'GET /api/v1/ai-readiness': () => json(readinessTable([readinessRow('Twentebelt', { scan: summary({ id: SCAN_3 }) })])),
        [`GET /api/v1/analyses/${SCAN_3}/readiness-files`]: () => json(files),
      }),
      '/competitors/twentebelt/history',
    )
    expect(requests()).toContain(`GET /api/v1/analyses/${SCAN_3}/readiness-files`)
    const section = wrapper.get('section[aria-labelledby="files-title"]')
    expect(section.findAll('summary').map((s) => s.text())).toEqual(['robots.txt', 'llms.txt: the first 100 KB', 'llms-full.txt: not served'])
    const marked = section.findAll('pre')[0]?.findAll('span.font-semibold').map((l) => l.text())
    expect(marked).toEqual(['User-agent: GPTBot'])
  })

  it('says when no scan recorded the files, and shows them to every account', async () => {
    const admin = await mountApp(
      api({ 'GET /api/v1/me': () => json(me({ isAdmin: true })), 'GET /api/v1/ai-readiness': () => json(readinessTable([])) }),
      '/competitors/twentebelt/history',
    )
    expect(admin.wrapper.get('section[aria-labelledby="files-title"]').text()).toContain('No scan of it recorded its AI readiness yet.')
    admin.wrapper.unmount()
    const failing = await mountApp(
      api({
        'GET /api/v1/me': () => json(me({ isAdmin: true })),
        'GET /api/v1/ai-readiness': () => json(readinessTable([readinessRow('Twentebelt', { scan: summary({ id: SCAN_3 }) })])),
        [`GET /api/v1/analyses/${SCAN_3}/readiness-files`]: () => problem(500, 'Internal Server Error', 'Down.'),
      }),
      '/competitors/twentebelt/history',
    )
    expect(failing.wrapper.get('section[aria-labelledby="files-title"] [role="alert"]').text()).toContain('Down.')
    failing.wrapper.unmount()
    // The files are public on the site, so an account without the admin role reads them too.
    const viewer = await mountApp(api(), '/competitors/twentebelt/history')
    expect(viewer.wrapper.find('section[aria-labelledby="files-title"]').exists()).toBe(true)
  })
})
