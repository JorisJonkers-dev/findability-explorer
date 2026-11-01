import { describe, expect, it, vi } from 'vitest'
import { ANALYSIS_ID, readiness, readinessRow, readinessTable, summary } from '../../../testing/fixtures'
import { type Handler, json, mountApp, problem, routes, settle } from '../../../testing/mountApp'

const tribelt = readinessRow('Tribelt B.V.', {
  competitorId: 'tribelt',
  role: 'tribelt',
  scan: summary({ competitorId: 'tribelt', url: 'https://www.tribelt.nl', number: 2 }),
  readiness: readiness({ llmsFullTxt: false }),
})
const robots = 'User-agent: *\nDisallow: /admin/\n\nUser-agent: GPTBot\nDisallow: /admin/'
const SITEMAP = '<?xml version="1.0"?>\n<urlset>\n  <url><loc>https://www.tribelt.nl/de/</loc></url>\n</urlset>'
const files = {
  robotsTxt: robots,
  robotsTruncated: false,
  llmsTxt: '# Tribelt\n> Metal belts',
  llmsTruncated: false,
  llmsFullTxt: '',
  llmsFullTruncated: false,
  sitemapUrl: 'https://www.tribelt.nl/sitemap_index.xml',
  sitemapTxt: SITEMAP,
  sitemapTruncated: false,
}

const api = (extra: Record<string, Handler> = {}) =>
  routes({
    'GET /api/v1/ai-readiness': () => json(readinessTable([tribelt, readinessRow('Unibelt', { readiness: undefined, scan: undefined })])),
    [`GET /api/v1/analyses/${ANALYSIS_ID}/readiness-files`]: () => json(files),
    ...extra,
  })

describe('the files found on a site', () => {
  it('is framed by the site, with the way back to its page, from its newest scan', async () => {
    const { wrapper, requests } = await mountApp(api(), '/ai-readiness/tribelt/files')
    expect(wrapper.get('h1').text()).toBe('Files found on tribelt.nl')
    expect(wrapper.get('[data-part="purpose"]').text()).toBe("The files the site's newest scan fetched. Pick a file to read, search, copy, or download it.")
    expect(wrapper.get('[data-part="data-line"]').text()).toBe('From scan 2 of 22 October 2026.')
    expect(wrapper.get('a[href="/ai-readiness/tribelt"]').text()).toContain('Tribelt’s AI readiness')
    expect(requests()).toContain(`GET /api/v1/analyses/${ANALYSIS_ID}/readiness-files`)
    expect(wrapper.text()).toContain('Found: robots.txt, sitemap.xml, and llms.txt. Not found: llms-full.txt.')
  })

  it('has a tab per file, robots.txt open, with its lines numbered, and its lines and size in the header', async () => {
    const { wrapper } = await mountApp(api(), '/ai-readiness/tribelt/files')
    const tabs = wrapper.findAll('[role="tab"]')
    expect(tabs.map((t) => t.text())).toEqual(['robots.txt', 'sitemap.xml', 'llms.txt', 'llms-full.txt'])
    expect(tabs[0]?.attributes('aria-selected')).toBe('true')
    const panel = wrapper.get('[role="tabpanel"]')
    expect(panel.get('h3').text()).toBe('robots.txt')
    expect(panel.text()).toContain('5 lines')
    expect(panel.text()).toContain('69 bytes')
    expect(panel.findAll('[data-line]').map((l) => l.get('[data-n]').text())).toEqual(['1', '2', '3', '4', '5'])
  })

  it('highlights and counts what the search finds', async () => {
    const { wrapper } = await mountApp(api(), '/ai-readiness/tribelt/files')
    await wrapper.get('input[type="search"]').setValue('admin')
    const panel = wrapper.get('[role="tabpanel"]')
    expect(panel.findAll('mark').map((m) => m.text())).toEqual(['admin', 'admin'])
    expect(panel.findAll('[data-line].bg-orange-pale')).toHaveLength(2)
    expect(panel.text()).toContain('2 matches')
    await wrapper.get('input[type="search"]').setValue('<script>')
    expect(panel.findAll('mark')).toHaveLength(0)
    expect(panel.text()).toContain('0 matches')
  })

  it('copies the file to the clipboard, and downloads it as a file named after it', async () => {
    const writeText = vi.fn(() => Promise.resolve())
    Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })
    const createObjectURL = vi.fn(() => 'blob:robots')
    const revokeObjectURL = vi.fn()
    Object.assign(URL, { createObjectURL, revokeObjectURL })
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => undefined)
    const { wrapper } = await mountApp(api(), '/ai-readiness/tribelt/files')
    await wrapper.findAll('button').find((b) => b.text() === 'Copy')?.trigger('click')
    await settle()
    expect(writeText).toHaveBeenCalledWith(robots)
    expect(wrapper.text()).toContain('The file is copied.')
    await wrapper.findAll('button').find((b) => b.text() === 'Download')?.trigger('click')
    expect(createObjectURL).toHaveBeenCalled()
    expect((click.mock.contexts[0] as HTMLAnchorElement).download).toBe('robots.txt')
    click.mockRestore()
  })

  it('says when copying is not allowed', async () => {
    Object.defineProperty(navigator, 'clipboard', { value: { writeText: () => Promise.reject(new Error('denied')) }, configurable: true })
    const { wrapper } = await mountApp(api(), '/ai-readiness/tribelt/files')
    await wrapper.findAll('button').find((b) => b.text() === 'Copy')?.trigger('click')
    await settle()
    expect(wrapper.text()).toContain('Copying is not allowed here.')
  })

  it('opens another tab, keeping it in the address, and lists a file the scan did not find as not found', async () => {
    const { wrapper, router } = await mountApp(api(), '/ai-readiness/tribelt/files')
    await wrapper.findAll('[role="tab"]')[3]?.trigger('click')
    await settle()
    expect(router.currentRoute.value.query.file).toBe('llmsFull')
    expect(wrapper.get('[role="tabpanel"]').text()).toContain('The scan found no llms-full.txt on tribelt.nl. Adding one is one of the eight AI readiness checks.')
    expect(wrapper.findAll('button').some((b) => b.text() === 'Copy')).toBe(false)
  })

  it('shows the sitemap’s text, from its address, and downloads it under its own name', async () => {
    const createObjectURL = vi.fn(() => 'blob:sitemap')
    Object.assign(URL, { createObjectURL, revokeObjectURL: vi.fn() })
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => undefined)
    const { wrapper } = await mountApp(api(), '/ai-readiness/tribelt/files?file=sitemap')
    const panel = wrapper.get('[role="tabpanel"]')
    expect(panel.get('h3').text()).toBe('sitemap.xml')
    expect(panel.text()).toContain('From https://www.tribelt.nl/sitemap_index.xml')
    expect(panel.text()).toContain('4 lines')
    expect(panel.findAll('[data-line]')[2]?.text()).toContain('<url><loc>https://www.tribelt.nl/de/</loc></url>')
    await wrapper.get('input[type="search"]').setValue('/de/')
    expect(panel.findAll('mark').map((m) => m.text())).toEqual(['/de/'])
    await wrapper.findAll('button').find((b) => b.text() === 'Download')?.trigger('click')
    expect((click.mock.contexts[0] as HTMLAnchorElement).download).toBe('sitemap_index.xml')
    click.mockRestore()
  })

  it('says when the scan was made before scans kept the sitemap', async () => {
    const { wrapper } = await mountApp(
      api({ [`GET /api/v1/analyses/${ANALYSIS_ID}/readiness-files`]: () => json({ ...files, sitemapUrl: '', sitemapTxt: '' }) }),
      '/ai-readiness/tribelt/files?file=sitemap',
    )
    expect(wrapper.get('[role="tabpanel"]').text()).toContain("This scan was made before scans kept the sitemap's text. Scan the site again to read it.")
  })

  it('says when no scan recorded the site, and when the files cannot load', async () => {
    const { wrapper } = await mountApp(api(), '/ai-readiness/unibelt/files')
    expect(wrapper.text()).toContain('No scan of this site was read.')
    const { wrapper: failing } = await mountApp(api({ [`GET /api/v1/analyses/${ANALYSIS_ID}/readiness-files`]: () => problem(500, 'Internal Server Error', 'Down.') }), '/ai-readiness/tribelt/files')
    expect(failing.get('[role="alert"]').text()).toContain('Down.')
  })
})
