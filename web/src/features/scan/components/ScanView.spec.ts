import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { Analysis } from '../../../infrastructure/api/types.gen'
import { ANALYSIS_ID, analysis, competitor, me } from '../../../testing/fixtures'
import { buttonNamed, manageApi, noContent } from '../../../testing/manageApi'
import { type Handler, json, mountApp, problem, settle } from '../../../testing/mountApp'

const NEW_ID = '6c1d2b0f-3e4a-4f6b-9c7d-8e9f0a1b2c3d'
const api = (extra: Record<string, Handler> = {}) =>
  manageApi({ [`GET /api/v1/analyses/${ANALYSIS_ID}`]: () => json(analysis({}, { number: 2, matchingModel: 'anthropic: claude-haiku-5-5' })), ...extra }, [competitor('Twentebelt')])

const pill = (wrapper: VueWrapper, group: string, label: string) => {
  const button = wrapper.get(`[role="group"][aria-label="${group}"]`).findAll('button').find((b) => b.text() === label)
  if (!button) throw new Error(`no pill ${label}`)
  return button
}
const scan = (overrides: Partial<Analysis>) => api({ [`GET /api/v1/analyses/${ANALYSIS_ID}`]: () => json(analysis(overrides)) })
const widget = (wrapper: VueWrapper, title: string) => {
  const found = wrapper.findAll('section').find((s) => s.find('h2').exists() && s.get('h2').text() === title)
  if (!found) throw new Error(`no widget ${title}`)
  return found
}

describe('a scan', () => {
  it('names its company, number, date, pages read and model, with the way back to Recent scans', async () => {
    const { wrapper, router } = await mountApp(api(), `/manage/scans/${ANALYSIS_ID}`)
    expect(document.title).toBe('Scan · Findability Explorer')
    expect(wrapper.get('h1').text()).toBe('Scan of Twentebelt')
    expect(wrapper.get('nav[aria-label="Manage"] [data-open]').text()).toContain('Website scans')
    expect(widget(wrapper, 'This scan').get('dl').text()).toBe(
      'CompanyTwentebeltScan number2Date22 October 2026, 09:30Pages read15 of 15Matching modelanthropic: claude-haiku-5-5',
    )
    expect(widget(wrapper, 'This scan').get('a').attributes('href')).toBe('/competitors/twentebelt')
    expect(wrapper.get('header a[href="/ai-readiness"]').text()).toBe('See AI readiness')
    await wrapper.findAll('a').find((a) => a.text().includes('Back to Recent scans'))?.trigger('click')
    await settle()
    expect(router.currentRoute.value.fullPath).toBe('/manage/scans/recent')
  })

  it('shows what a finished scan found: what Tribelt is missing, the ranked phrases, and the matched products', async () => {
    const { wrapper } = await mountApp(api(), `/manage/scans/${ANALYSIS_ID}`)
    const found = widget(wrapper, 'What the scan found')
    expect(found.text()).toContain('Tribelt is missing:')
    const phrases = () => found.findAll('ol[aria-label="Ranked phrases"] li').map((li) => li.text())
    expect(phrases()).toHaveLength(3)
    await pill(wrapper, 'Which phrases', 'Already on our list').trigger('click')
    expect(phrases()).toHaveLength(1)
    expect(found.get('details summary').text()).toBe("1 product matched to Tribelt's belts")
  })

  it('says what became of matching when it was off', async () => {
    const { wrapper } = await mountApp(scan({ matching: 'disabled', equivalents: [], keywords: [], knownTerms: [] }), `/manage/scans/${ANALYSIS_ID}`)
    expect(wrapper.text()).toContain('Product matching is off')
    expect(widget(wrapper, 'This scan').get('dl').text()).toContain('Matching modelNone')
  })

  it('says why a scan failed, that one was stopped by hand, or that it could not be loaded', async () => {
    const failed = await mountApp(api({ [`GET /api/v1/analyses/${ANALYSIS_ID}`]: () => json(analysis({}, { status: 'failed', failure: 'robots.txt forbids it.' })) }), `/manage/scans/${ANALYSIS_ID}`)
    expect(failed.wrapper.text()).toContain('Scan failed')
    expect(failed.wrapper.get('[role="alert"]').text()).toBe('robots.txt forbids it.')
    expect(failed.wrapper.findAll('h2').map((h) => h.text())).not.toContain('What the scan found')
    failed.wrapper.unmount()
    const stopped = await mountApp(
      api({ [`GET /api/v1/analyses/${ANALYSIS_ID}`]: () => json(analysis({}, { status: 'failed', stoppedByHand: true, pagesCrawled: 4 })) }),
      `/manage/scans/${ANALYSIS_ID}`,
    )
    expect(stopped.wrapper.text()).toContain('Stopped by hand. It keeps the 4 pages it read.')
    expect(stopped.wrapper.text()).not.toContain('Scan failed')
    stopped.wrapper.unmount()
    const missing = await mountApp(api({ [`GET /api/v1/analyses/${ANALYSIS_ID}`]: () => problem(404, 'Not Found', 'No such scan.') }), `/manage/scans/${ANALYSIS_ID}`)
    expect(missing.wrapper.get('[role="alert"]').text()).toBe('No such scan.')
  })

  it('stops a scan that reads pages', async () => {
    const sent: string[] = []
    const { wrapper } = await mountApp(
      api({
        [`GET /api/v1/analyses/${ANALYSIS_ID}`]: () => json(analysis({}, { status: 'running', pagesCrawled: 3, maxPages: 30 })),
        [`POST /api/v1/admin/analyses/${ANALYSIS_ID}/stop`]: () => (sent.push('stop'), noContent()),
      }),
      `/manage/scans/${ANALYSIS_ID}`,
    )
    expect(wrapper.get('[role="progressbar"]').attributes('aria-valuetext')).toBe('Reading page 4 of 30')
    await buttonNamed(wrapper, 'Stop the scan').trigger('click')
    await settle()
    expect(sent).toEqual(['stop'])
    expect(wrapper.text()).toContain('The scan of Twentebelt is stopped. It keeps the pages it read.')
  })

  it('takes a waiting scan off the queue', async () => {
    const { wrapper } = await mountApp(
      api({
        [`GET /api/v1/analyses/${ANALYSIS_ID}`]: () => json(analysis({}, { status: 'queued', pagesCrawled: 0 })),
        [`POST /api/v1/admin/analyses/${ANALYSIS_ID}/stop`]: noContent,
      }),
      `/manage/scans/${ANALYSIS_ID}`,
    )
    await buttonNamed(wrapper, 'Remove it from the queue').trigger('click')
    await settle()
    expect(wrapper.text()).toContain('The scan of Twentebelt is removed from the queue.')
  })

  it('scans the site again and opens the new scan', async () => {
    let body: unknown = null
    const { wrapper, router } = await mountApp(
      api({
        'POST /api/v1/analyses': async (request) => {
          body = await request.json()
          return json(analysis({}, { id: NEW_ID, status: 'queued' }), 202)
        },
        [`GET /api/v1/analyses/${NEW_ID}`]: () => json(analysis({}, { id: NEW_ID })),
      }),
      `/manage/scans/${ANALYSIS_ID}`,
    )
    await buttonNamed(wrapper, 'Scan again').trigger('click')
    await settle()
    expect(body).toEqual({ url: 'https://www.twentebelt.com', maxPages: 60 })
    expect(router.currentRoute.value.fullPath).toBe(`/manage/scans/${NEW_ID}`)
  })

  it('names a scan without a company after its site, and shows a viewer nothing', async () => {
    const lone = await mountApp(manageApi({ [`GET /api/v1/analyses/${ANALYSIS_ID}`]: () => json(analysis({}, { competitorId: undefined })) }), `/manage/scans/${ANALYSIS_ID}`)
    expect(lone.wrapper.get('h1').text()).toBe('Scan of twentebelt.com')
    lone.wrapper.unmount()
    const viewer = await mountApp(api({ 'GET /api/v1/me': () => json(me()) }), `/manage/scans/${ANALYSIS_ID}`)
    expect(viewer.wrapper.text()).toContain('Only administrators can start, stop, and change the work behind the results.')
    expect(viewer.wrapper.text()).not.toContain('Scan again')
  })
})

