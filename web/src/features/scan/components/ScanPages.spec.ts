import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { competitor, readinessTable, summary } from '../../../testing/fixtures'
import { buttonNamed, manageApi, noContent } from '../../../testing/manageApi'
import { type Handler, json, mountApp, problem, settle } from '../../../testing/mountApp'

const RUN = '00000000-0000-4000-8000-0000000000a1'
const WAIT = '00000000-0000-4000-8000-0000000000b2'
const scans = [
  summary({ id: RUN, status: 'running', competitorId: 'twentebelt', pagesCrawled: 3, maxPages: 30, createdAt: '2026-10-28T10:00:00Z' }),
  summary({ id: WAIT, status: 'queued', competitorId: 'ashworth', pagesCrawled: 0, maxPages: 60, createdAt: '2026-10-28T10:01:00Z' }),
]
const companies = [
  competitor('Twentebelt', { lastScan: summary({ id: RUN, createdAt: '2026-10-27T08:00:00Z', number: 2, pagesCrawled: 22, maxPages: 60 }), scans: 2 }),
  competitor('Ashworth', { lastScan: summary({ id: WAIT, createdAt: '2026-10-28T05:48:00Z', pagesCrawled: 60, maxPages: 60 }), scans: 1 }),
  competitor('Never', { scans: 0 }),
]
const api = (extra: Record<string, Handler> = {}) => manageApi({ 'GET /api/v1/analyses': () => json({ items: scans }), ...extra }, companies)
const widget = (wrapper: VueWrapper, title: string) => {
  const found = wrapper.findAll('section').find((s) => s.find('h2').exists() && s.get('h2').text() === title)
  if (!found) throw new Error(`no widget ${title}`)
  return found
}

describe('the scan queue', () => {
  it('lists the running scans with Stop and the waiting ones with Remove, under Website scans', async () => {
    const { wrapper } = await mountApp(api(), '/manage/scans/queue')
    expect(document.title).toBe('Scan queue · Findability Explorer')
    expect(wrapper.get('h1').text()).toBe('Scan queue')
    expect(wrapper.get('nav[aria-label="Manage"] a[aria-current="page"]').text()).toMatch(/^Scan queue/)
    expect(wrapper.findAll('a').find((a) => a.text().includes('Back to Website scans'))?.attributes('href')).toBe('/manage/scans')
    expect(widget(wrapper, 'Running now').get('ul[aria-label="Scans running"]').text()).toContain('Scan of Twentebelt')
    expect(widget(wrapper, 'Waiting').get('ul[aria-label="Scans waiting"]').text()).toContain('Scan of Ashworth')
    expect(widget(wrapper, 'The queue').findAll('[data-part="value"]').map((v) => v.text())).toEqual(['1', '1'])
  })

  it('stops a scan, removes a waiting one, removes all waiting, and stops the rescan', async () => {
    const sent: string[] = []
    const record = (what: string) => () => (sent.push(what), noContent())
    const { wrapper } = await mountApp(
      api({
        [`POST /api/v1/admin/analyses/${RUN}/stop`]: record('stop'),
        [`POST /api/v1/admin/analyses/${WAIT}/stop`]: record('remove'),
        'DELETE /api/v1/admin/analyses/queue': record('remove all'),
        'POST /api/v1/admin/rescans/stop': record('stop rescan'),
      }),
      '/manage/scans/queue',
    )
    await widget(wrapper, 'Running now').get('ul button').trigger('click')
    await settle()
    expect(widget(wrapper, 'Running now').text()).toContain('The scan of Twentebelt is stopped. It keeps the pages it read.')
    await widget(wrapper, 'Waiting').get('ul button').trigger('click')
    await settle()
    await buttonNamed(wrapper, 'Remove all waiting scans').trigger('click')
    await settle()
    await buttonNamed(wrapper, 'Stop the rescan').trigger('click')
    await settle()
    expect(sent).toEqual(['stop', 'remove', 'remove all', 'stop rescan'])
  })

  it('says when nothing runs or waits, and shows the readiness check with its Stop while it runs', async () => {
    let stopped = false
    const { wrapper } = await mountApp(
      api({
        'GET /api/v1/analyses': () => json({ items: [summary()] }),
        'GET /api/v1/ai-readiness': () => json(readinessTable([], { running: true, checked: 3, total: 9 })),
        'POST /api/v1/admin/readiness-checks/stop': () => ((stopped = true), noContent()),
      }),
      '/manage/scans/queue',
    )
    expect(widget(wrapper, 'Waiting').text()).toContain('No scan is waiting.')
    expect(wrapper.text()).not.toContain('Remove all waiting scans')
    expect(wrapper.get('[data-part="readiness-check"]').text()).toContain('3 of 9 sites')
    await buttonNamed(wrapper, 'Stop the check').trigger('click')
    await settle()
    expect(stopped).toBe(true)
  })

  it('says why a stop was refused', async () => {
    const { wrapper } = await mountApp(api({ 'POST /api/v1/admin/rescans/stop': () => problem(404, 'Not Found', 'No scan of a rescan waits or runs.') }), '/manage/scans/queue')
    await buttonNamed(wrapper, 'Stop the rescan').trigger('click')
    await settle()
    expect(wrapper.get('[role="alert"]').text()).toBe('No scan of a rescan waits or runs.')
  })
})

describe('recent scans', () => {
  it('lists each company’s newest scan, newest first, opening its scan', async () => {
    const { wrapper } = await mountApp(api(), '/manage/scans/recent')
    expect(wrapper.get('h1').text()).toBe('Recent scans')
    const rows = wrapper.findAll('table[aria-label="Scans"] tbody tr').map((r) => r.findAll('td').map((td) => td.text()))
    expect(rows).toEqual([
      ['Ashworth', '1', '28 October 2026, 05:48', '60 of 60'],
      ['Twentebelt', '2', '27 October 2026, 08:00', '22 of 60'],
    ])
    expect(wrapper.get('table[aria-label="Scans"] tbody a').attributes('href')).toBe(`/manage/scans/${WAIT}`)
    expect(widget(wrapper, 'At a glance').text()).toContain('The newest scan is of Ashworth, on 28 October 2026.')
  })

  it('sorts by a column', async () => {
    const { wrapper } = await mountApp(api(), '/manage/scans/recent')
    await wrapper.get('th button[aria-label="Company, not sorted"]').trigger('click')
    expect(wrapper.findAll('table tbody tr').map((r) => r.get('td').text())).toEqual(['Ashworth', 'Twentebelt'])
    await wrapper.get('th button[aria-label="Company, sorted ascending"]').trigger('click')
    expect(wrapper.findAll('table tbody tr').map((r) => r.get('td').text())).toEqual(['Twentebelt', 'Ashworth'])
  })

  it('says when no company was scanned', async () => {
    const { wrapper } = await mountApp(manageApi({}, [competitor('Never')]), '/manage/scans/recent')
    expect(wrapper.text()).toContain('No company has been scanned yet.')
  })
})

describe('scan settings', () => {
  it('keeps the default number of pages and offers the model that matches products', async () => {
    const { wrapper } = await mountApp(api(), '/manage/scans/settings')
    expect(wrapper.get('h1').text()).toBe('Scan settings')
    const pages = wrapper.get<HTMLInputElement>('#set-pages')
    expect(pages.element.value).toBe('60')
    pages.element.value = '250'
    await pages.trigger('change')
    expect(pages.element.value).toBe('100')
    pages.element.value = '60'
    await pages.trigger('change')
    expect(widget(wrapper, 'Product matching').text()).toContain('In use now')
  })

  it('says why the settings could not be read', async () => {
    const { wrapper } = await mountApp(api({ 'GET /api/v1/admin/llm': () => problem(500, 'Internal Server Error') }), '/manage/scans/settings')
    expect(widget(wrapper, 'Product matching').find('[role="alert"]').exists()).toBe(true)
  })
})

