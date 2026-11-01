import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { overviewRoutes } from '../../../testing/homeFixtures'
import { mountApp, problem, routes, settle } from '../../../testing/mountApp'

/** The section under the heading title. */
function region(w: VueWrapper, title: string) {
  const found = w.findAll('section').find((s) => s.find('h2').exists() && s.get('h2').text() === title)
  if (!found) throw new Error(`no section ${title}`)
  return found
}

describe('the Overview', () => {
  it('says what it measures, where its figures come from, and sums them up in a sentence', async () => {
    const { wrapper } = await mountApp(routes(overviewRoutes()), '/')
    expect(document.title).toBe('Overview · Findability Explorer')
    expect(wrapper.get('h1').text()).toBe('Overview')
    expect(wrapper.get('[data-part="purpose"]').text()).toBe('How easily buyers find Tribelt in search and AI assistants, measured against five goals.')
    expect(wrapper.get('[data-part="data-line"]').text()).toBe('From the Prompt run of 1 November 2026 and the scans up to 28 October 2026.')
    expect(wrapper.get('[data-part="summary"]').text()).toBe(
      'AI responses name Tribelt more often than any competitor, tribelt.nl passes 3 of the 8 AI readiness checks, and tribelt.nl uses 30 of the 387 search terms buyers use.',
    )
  })

  it('lists the five goals with Tribelt, the goal, the best competitor, and a progress bar, each opening its page', async () => {
    const { wrapper, router } = await mountApp(routes(overviewRoutes()), '/')
    const goals = region(wrapper, 'Goals')
    expect(goals.get('h2').text()).toBe('Goals')
    expect(goals.text()).toContain('Click a goal to see who is ahead and what would improve it.')
    expect(goals.findAll('th[scope="col"]').map((t) => t.text())).toEqual(['Goal', 'Tribelt', 'The goal', 'The best competitor', 'Progress'])
    const rows = goals.findAll('tbody tr')
    expect(rows.map((r) => r.findAll('th, td').slice(0, 4).map((c) => c.text()))).toEqual([
      ['AI responses that name Tribelt', '24%', '40%', 'Ashworth, 57%'],
      ['AI responses about the Twente region name Tribelt', '20%', '50%', 'Twentebelt, 60%'],
      ['Web responses that cite tribelt.nl', '10%', '50%', 'Märtens, 30%'],
      ['AI readiness checks passed', '3 of 8', '8 of 8', 'Märtens, 8 of 8'],
      ['Search terms tribelt.nl uses', '45', '35', 'Ashworth, 40'],
    ])
    expect(goals.findAll('tbody a').map((a) => a.attributes('href'))).toEqual([
      '/goals/aiShareOfVoice',
      '/goals/twenteShareOfVoice',
      '/goals/citationShare',
      '/goals/readinessChecks',
      '/goals/termsUsed',
    ])
    const bar = rows[0]?.get('[role="img"]')
    expect(bar?.attributes('aria-label')).toBe('Tribelt now 24%, the best competitor 57%, the goal 40%.')
    expect(bar?.findAll('[data-mark]').map((m) => m.attributes('data-mark'))).toEqual(['now', 'best', 'goal'])
    expect(goals.get('[data-part="legend"]').text()).toBe('Tribelt nowThe best competitorThe goal')
    await goals.get('a[href="/goals/readinessChecks"]').trigger('click')
    await settle()
    expect(router.currentRoute.value.fullPath).toBe('/goals/readinessChecks')
  })

  it('has a numbered tile per part of the app', async () => {
    const { wrapper } = await mountApp(routes(overviewRoutes()), '/')
    const tiles = wrapper.findAll('nav[aria-label="Parts of the app"] a')
    expect(tiles.map((t) => [t.attributes('href'), t.text()])).toEqual([
      ['/ai-responses', '02AI responses24%of AI responses name Tribelt.'],
      ['/ai-readiness', '03AI readiness3 of 8AI readiness checks passed by tribelt.nl.'],
      ['/terms', '04Search terms30of the 387 search terms appear on tribelt.nl.'],
      ['/competitors', '05Competitors24competitors make metal belts buyers could choose instead.'],
      ['/recommendations', '06Recommendations3changes to make on tribelt.nl, ranked by value.'],
    ])
  })

  it('says when each process last ran and what comes next, with Manage for administrators', async () => {
    const reader = await mountApp(routes(overviewRoutes()), '/')
    const latest = region(reader.wrapper, 'Latest runs and scans')
    expect(latest.get('h2').text()).toBe('Latest runs and scans')
    expect(latest.findAll('tbody tr').map((r) => r.find('th').text())).toEqual(['Prompt test', 'Website scans', 'Search term list', 'Recommendations'])
    expect(latest.text()).toContain('tribelt.nl scanned on 26 October 2026')
    expect(latest.findAll('a').map((a) => a.text())).not.toContain('Manage')
    reader.wrapper.unmount()
    const admin = await mountApp(routes(overviewRoutes({}, true)), '/')
    const forAdmin = region(admin.wrapper, 'Latest runs and scans')
    expect(forAdmin.findAll('a').filter((a) => a.text() === 'Manage').map((a) => a.attributes('href'))).toEqual(['/manage/prompt-test', '/manage/scans', '/manage/terms', '/manage/recommendations'])
    expect(forAdmin.text()).toContain('The next run is scheduled for 8 November 2026.')
  })

  it('speaks of goals and the best competitor, never of targets or leaders', async () => {
    const { wrapper } = await mountApp(routes(overviewRoutes()), '/')
    expect(wrapper.text().toLowerCase()).not.toMatch(/target|leader/)
  })

  it('says when the goals fail to load', async () => {
    const { wrapper } = await mountApp(routes(overviewRoutes({ 'GET /api/v1/plan': () => problem(500, 'Internal Server Error', 'Down.') })), '/')
    expect(wrapper.get('[role="alert"]').text()).toContain('Down.')
    expect(wrapper.get('h1').text()).toBe('Overview')
  })
})
