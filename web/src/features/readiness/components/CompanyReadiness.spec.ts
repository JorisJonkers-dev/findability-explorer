import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { bots, me, readiness, readinessRow, readinessTable, summary } from '../../../testing/fixtures'
import { type Handler, json, mountApp, routes } from '../../../testing/mountApp'

const twentebelt = readinessRow('Twentebelt', {
  scan: summary({ number: 4 }),
  readiness: readiness({ bots: bots({ GPTBot: 'blocked' }), llmsFullTxt: false, stack: { cms: 'WordPress', botProtection: 'Cloudflare' } }),
})
const table = [twentebelt, readinessRow('Lesjöfors', { role: 'group' }), readinessRow('Unibelt', { readiness: undefined, scan: undefined })]

const aiSearch = {
  goal: 0.4,
  followed: [],
  trend: [],
  crawlers: [{ bot: 'GPTBot', lab: 'OpenAI', gets: 'What future ChatGPT models know', feeds: 'future', access: 'allowed' }],
  keepingOut: [],
  ranking: [],
  sites: 3,
  questions: 0,
  providers: 0,
  readinessChecks: 8,
  checks: [],
}

const api = (extra: Record<string, Handler> = {}) =>
  routes({
    'GET /api/v1/ai-readiness': () => json(readinessTable(table)),
    'GET /api/v1/ai-search': () => json(aiSearch),
    ...extra,
  })

const section = (wrapper: VueWrapper, title: string) => {
  const h2 = wrapper.findAll('h2').find((h) => h.text() === title)
  if (!h2) throw new Error(`no section ${title}`)
  return wrapper.get(`section[aria-labelledby="${h2.attributes('id') ?? ''}"]`)
}

describe('one site', () => {
  it('is framed by the site, with the way back to AI readiness', async () => {
    const { wrapper } = await mountApp(api(), '/ai-readiness/twentebelt')
    expect(wrapper.get('h1').text()).toBe('Twentebelt’s AI readiness')
    expect(wrapper.get('[data-part="purpose"]').text()).toBe('twentebelt.com passes 6½ of the 8 checks.')
    expect(wrapper.get('[data-part="data-line"]').text()).toBe('From scan 4 of 22 October 2026.')
    expect(wrapper.get('a[href="/ai-readiness"]').text()).toContain('AI readiness')
    expect(wrapper.findAll('h2').map((h) => h.text())).toEqual(['The eight checks', 'The structured data found', 'The files as found', 'Crawler policy'])
  })

  it('shows, in the left column, the checks with their evidence, the structured data, and the files', async () => {
    const { wrapper } = await mountApp(api(), '/ai-readiness/twentebelt')
    const left = wrapper.get('[data-column="left"]')
    const checks = section(wrapper, 'The eight checks')
    expect(left.element.contains(checks.element)).toBe(true)
    expect(checks.text()).toContain('Built with WordPress and Cloudflare.')
    const items = checks.findAll('li')
    expect(items).toHaveLength(8)
    expect(items[0]?.text()).toContain('AI crawlers let in')
    expect(items[0]?.text()).toContain('9 of the 10 crawlers are allowed in, and 1 is blocked.')
    expect(items[6]?.text()).toContain('The site has no llms-full.txt.')
    expect(section(wrapper, 'The structured data found').text()).toContain('Organization and Product.')
    const files = section(wrapper, 'The files as found')
    expect(left.element.contains(files.element)).toBe(true)
    expect(files.text()).toContain('llms.txtFound, 5,120 bytes, 124 links.')
    const read = files.get('a')
    expect(read.text()).toBe('Read the files')
    expect(read.attributes('href')).toBe('/ai-readiness/twentebelt/files')
  })

  it('shows the crawler policy in the right column', async () => {
    const { wrapper } = await mountApp(api(), '/ai-readiness/twentebelt')
    const policy = section(wrapper, 'Crawler policy')
    expect(wrapper.get('[data-column="right"]').element.contains(policy.element)).toBe(true)
    expect(policy.text()).toContain('Blocks GPTBot.')
    const rows = policy.findAll('tbody tr').map((r) => r.findAll('th, td').map((c) => c.text()))
    expect(rows).toHaveLength(10)
    expect(rows[5]).toEqual(['GPTBot', 'What future ChatGPT models knowOpenAI', 'Blocked'])
    expect(rows[0]?.[0]).toBe('Googlebot')
  })

  it('leads on to the competitor page and the scan history', async () => {
    const { wrapper } = await mountApp(api(), '/ai-readiness/twentebelt')
    expect(wrapper.get('a[href="/competitors/twentebelt"]').text()).toBe('Twentebelt’s competitor page')
    expect(wrapper.get('a[href="/competitors/twentebelt/history"]').text()).toBe('Scan history')
  })

  it('leads an administrator to the scans', async () => {
    const { wrapper } = await mountApp(api({ 'GET /api/v1/me': () => json(me({ isAdmin: true })) }), '/ai-readiness/twentebelt')
    expect(wrapper.get('a[href="/manage/scans"]').text()).toBe('Manage the scans')
  })

  it('says when no scan has recorded the site, or the site is unknown', async () => {
    const { wrapper } = await mountApp(api(), '/ai-readiness/unibelt')
    expect(wrapper.get('h1').text()).toBe('Unibelt’s AI readiness')
    expect(wrapper.text()).toContain('No scan has recorded the AI readiness of Unibelt yet.')
    expect(wrapper.find('a[href="/ai-readiness/unibelt/files"]').exists()).toBe(false)
    const { wrapper: unknown } = await mountApp(api(), '/ai-readiness/nobody')
    expect(unknown.get('h1').text()).toBe('Site not found')
    expect(unknown.text()).toContain('No scan has been read for this site. Go back to AI readiness and pick one.')
  })
})
