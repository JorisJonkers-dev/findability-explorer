import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { competitor, me, overview, promptAnswer, promptResults, RUN_ID, readinessRow, readinessTable, summary, term, termSearch } from '../../../testing/fixtures'
import { type Handler, json, mountApp, problem, routes } from '../../../testing/mountApp'

const SCAN_ID = '1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d'
const tribelt = competitor('Tribelt', { role: 'tribelt' })
const twentebelt = competitor('Twentebelt', { scans: 2, lastScan: summary() })
const martens = competitor('Martens', { scans: 0, websiteUrl: undefined, host: undefined, website: '', beltNames: [], lastScan: undefined })

function api(extra: Record<string, Handler> = {}) {
  return routes({
    'GET /api/v1/competitors': () => json({ items: [tribelt, twentebelt, martens] }),
    'GET /api/v1/competitors/twentebelt': () => json({ competitor: twentebelt, scans: [summary({ id: SCAN_ID, status: 'running' }), summary()] }),
    'GET /api/v1/competitors/martens': () => json({ competitor: martens, scans: [] }),
    'GET /api/v1/terms': () => json(termSearch([term({ term: 'TwenteFlex', usedBy: ['Twentebelt'] })])),
    'GET /api/v1/overview': () => json(overview()),
    'GET /api/v1/prompt-test': () => json(promptResults()),
    'GET /api/v1/ai-readiness': () => json(readinessTable([readinessRow('Tribelt', { role: 'tribelt' }), readinessRow('Twentebelt')])),
    [`GET /api/v1/prompt-runs/${RUN_ID}/answers`]: () =>
      json({ run: promptResults().runs[0], items: [promptAnswer({ model: 'openai/gpt' }), promptAnswer({ model: 'openai/gpt' }, ['Tribelt:tribelt'])] }),
    ...extra,
  })
}

const link = (wrapper: VueWrapper, text: string) => wrapper.findAll('a').find((a) => a.text() === text)

describe('a competitor’s page', () => {
  it('is in the frame, with its sections side by side, and the ways to look closer', async () => {
    const { wrapper } = await mountApp(api(), '/competitors/twentebelt')
    expect(document.title).toBe('Competitor · Findability Explorer')
    expect(wrapper.get('h1').text()).toBe('Twentebelt')
    expect(wrapper.get('[data-part="purpose"]').text()).toBe('twentebelt.com, on the competitor list. It sells Tri-Flex belts.')
    expect(wrapper.get('[data-part="data-line"]').text()).toBe('From the Prompt run of 1 November 2026 and the scan of Twentebelt on 22 October 2026.')
    expect(wrapper.findAll('h2').map((h) => h.text())).toEqual([
      'Against Tribelt',
      'Where it is ahead of Tribelt',
      'What it sells and what it calls it',
      'Search terms it uses that tribelt.nl does not',
      'In AI responses',
    ])
    expect(wrapper.get('section[aria-labelledby="ahead-title"]').text()).toContain('75%, against Tribelt’s 25%.')
    const belts = wrapper.get('section[aria-labelledby="belts-title"]')
    expect(belts.findAll('li').map((li) => li.text())).toContain('Tri-FlexTwenteFlex spiral belts')
    expect(belts.get('a[href="/terms/families/eyelink"]').text()).toBe('Eyelink')
    expect(wrapper.get('section[aria-labelledby="their-terms-title"] a').attributes('href')).toBe('/terms/en--twenteflex')
    expect(wrapper.get('section[aria-labelledby="models-title"] [role="img"]').attributes('aria-label')).toBe('How often each model names Twentebelt. openai/gpt: 50%.')
    const visit = wrapper.get('a[target="_blank"]')
    expect(visit.attributes('href')).toBe('https://www.twentebelt.com/')
    expect(visit.attributes('rel')).toBe('noopener noreferrer')
    expect(wrapper.get('a[href="/competitors/twentebelt/history"]').text()).toBe('See its scan history')
    expect(wrapper.text()).toContain('Back to Competitors')
  })

  it('gives an administrator one way into Manage, and no tools on the page itself', async () => {
    const reader = await mountApp(api(), '/competitors/twentebelt')
    expect(reader.wrapper.find('a[href="/manage/competitors/companies/twentebelt"]').exists()).toBe(false)
    reader.wrapper.unmount()
    const { wrapper } = await mountApp(api({ 'GET /api/v1/me': () => json(me({ isAdmin: true })) }), '/competitors/twentebelt')
    expect(wrapper.get('a[href="/manage/competitors/companies/twentebelt"]').text()).toBe('Manage this company')
    expect(wrapper.findAll('button').some((b) => /^(Scan again|Scan this website)$/.test(b.text()))).toBe(false)
    expect(wrapper.find('[aria-label="Manage this company"]').exists()).toBe(false)
    expect(link(wrapper, 'Open its latest scan')).toBeUndefined()
  })

  it('says when no scan has read the website', async () => {
    const { wrapper } = await mountApp(api(), '/competitors/martens')
    expect(wrapper.get('[data-part="data-line"]').text()).toBe('From the Prompt run of 1 November 2026 and Martens has not been scanned yet.')
    expect(wrapper.text()).toContain('Martens is not ahead of Tribelt on any measure.')
    expect(wrapper.text()).toContain('Every term this competitor uses is also on tribelt.nl.')
    expect(wrapper.find('a[target="_blank"]').exists()).toBe(false)
  })

  it('says when no company has the id, or the company cannot be loaded', async () => {
    const missing = await mountApp(api({ 'GET /api/v1/competitors/nobody': () => problem(404, 'Not Found') }), '/competitors/nobody')
    expect(missing.wrapper.get('[role="alert"]').text()).toBe('No company has this id.')
    missing.wrapper.unmount()
    const failing = await mountApp(api({ 'GET /api/v1/competitors/twentebelt': () => problem(500, 'Internal Server Error') }), '/competitors/twentebelt')
    expect(failing.wrapper.get('[role="alert"]').text()).toBe('Internal Server Error')
  })

  it('shows what a measure means when it is picked', async () => {
    const { wrapper } = await mountApp(api(), '/competitors/twentebelt')
    const chart = wrapper.get('section[aria-labelledby="compare-title"]')
    await chart.findAll('button').find((b) => b.text().startsWith('AI readiness checks passed'))?.trigger('click')
    expect(chart.get('[aria-live="polite"]').text()).toContain('Both sites pass 8 of 8 of the AI readiness checks.')
  })
})
