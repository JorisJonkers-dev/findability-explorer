import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { Plan, PlanImprovement } from '../../../infrastructure/api/types.gen'
import { usesOldWord } from '../../../shared/words'
import { details, me } from '../../../testing/fixtures'
import { json, mountApp, routes } from '../../../testing/mountApp'

const leader = { competitorId: 'example-belts', name: 'Example Belts', value: 0.3 }

function imp(id: string, benefit: number, effort: number, extra: Partial<PlanImprovement> = {}): PlanImprovement {
  return { id, title: `Title ${id}`, moves: `Moves ${id}.`, measure: 'aiShareOfVoice', auto: false, done: false, ...details, why: `Why ${id}.`, benefit, effort, tribelt: 0.2, leader, ...extra }
}

function plan(improvements: PlanImprovement[]): Plan {
  return {
    catalogueVersion: 2,
    improvements,
    watched: [{ target: 'aiShareOfVoice', now: 0.2, goal: 0.4, max: 1, met: false }],
    tribeltTermsFrom: 'scan',
    readinessChecks: 8,
    eyeOn: [],
  }
}

// Best value first, as the server sends them: a (4, 1), b (3, 1) scanned, c (4, 3), e (4, 1), and d done.
const four = (): PlanImprovement[] => [
  imp('a', 4, 1),
  imp('b', 3, 1, { auto: true, measure: 'readinessChecks', tribelt: 3 }),
  imp('c', 4, 3, { measure: 'termsUsed', tribelt: 20 }),
  imp('e', 4, 1),
  imp('d', 5, 5, { done: true, tickedBy: 'joris', tickedAt: '2026-11-02T09:00:00Z' }),
]

function api(admin: boolean, current: Plan) {
  return routes({ 'GET /api/v1/me': () => json(me({ isAdmin: admin })), 'GET /api/v1/plan': () => json(current) })
}

const rows = (w: VueWrapper) => w.findAll('ul[aria-label="Recommendations"] > li')
const titles = (w: VueWrapper) => rows(w).map((r) => r.get('a').text())
const bubbles = (w: VueWrapper) => w.findAll('[role="group"][aria-label="Benefit against effort"] button')
const button = (w: VueWrapper, text: string) => w.findAll('button').find((b) => b.text() === text)

describe('the recommendations', () => {
  it('opens with the frame, the chart, the ranked list, and Done', async () => {
    const { wrapper } = await mountApp(api(false, plan(four())), '/recommendations')
    expect(document.title).toBe('Recommendations · Findability Explorer')
    expect(wrapper.get('h1').text()).toBe('Recommendations')
    expect(wrapper.get('[data-part="purpose"]').text()).toBe(
      'Changes to make on tribelt.nl, ranked by benefit against effort. Some apply in general. The rest come from the latest Recommendation run and name a page, search term, or site.',
    )
    expect(wrapper.get('[data-part="data-line"]').text()).toBe('Written by hand, version 2 of the catalogue. No Recommendation run has written any yet.')
    expect(wrapper.findAll('h2').map((h) => h.text())).toEqual(['Benefit against effort', 'Ranked list', 'Done'])
    expect(wrapper.find('a[href="/manage/recommendations"]').exists()).toBe(false)
    expect(wrapper.get('a[href="/help/recommendations"]').text()).toBe('How this is measured, in Help')
  })

  it('ranks the open ones best value first, each with its kind, where, scores, and goal', async () => {
    const { wrapper } = await mountApp(api(false, plan(four())), '/recommendations')
    const list = wrapper.get('section[aria-labelledby="list-title"]')
    expect(list.text()).toContain('4 recommendations. Best value is benefit divided by effort. Ties go to the larger benefit.')
    expect(titles(wrapper)).toEqual(['1. Title a', '2. Title b', '3. Title c', '4. Title e'])
    expect(rows(wrapper)[0]?.get('a').attributes('href')).toBe('/recommendations/a')
    const second = rows(wrapper)[1]?.text() ?? ''
    expect(second).toContain('AI readiness')
    expect(second).toContain('The whole site')
    expect(second).toContain('Benefit 3 of 5, effort 1 of 5')
    // "AI" keeps its capitals inside the sentence; other measures do not.
    expect(second).toContain('Improves AI readiness checks passed')
    expect(rows(wrapper)[2]?.text()).toContain('Improves search terms tribelt.nl uses')
    // Five rows show at a time, so four need no cue.
    expect(list.text()).not.toContain('Scroll inside the list')
    expect(list.get('ul[aria-label="Recommendations"]').classes()).toContain('overflow-y-auto')
  })

  it('says to scroll when the list holds more than five', async () => {
    const many = Array.from({ length: 7 }, (_, i) => imp(`r${String(i)}`, 3, 2))
    const { wrapper } = await mountApp(api(false, plan(many)), '/recommendations')
    expect(wrapper.text()).toContain('Scroll inside the list to see all 7.')
  })

  it('sorts by the column picked, and the other way round when picked again', async () => {
    const { wrapper } = await mountApp(api(false, plan(four())), '/recommendations')
    const sort = (name: string) => wrapper.findAll('[role="group"][aria-label="Sort by"] button').find((b) => b.text().startsWith(name))
    expect(sort('Best value')?.attributes('aria-pressed')).toBe('true')
    await sort('Effort')?.trigger('click')
    expect(sort('Effort')?.attributes('aria-pressed')).toBe('true')
    expect(titles(wrapper)).toEqual(['1. Title a', '2. Title b', '3. Title e', '4. Title c'])
    await sort('Effort')?.trigger('click')
    expect(titles(wrapper)).toEqual(['1. Title c', '2. Title e', '3. Title b', '4. Title a'])
    await sort('Name')?.trigger('click')
    expect(titles(wrapper)).toEqual(['1. Title a', '2. Title b', '3. Title c', '4. Title e'])
  })

  it('narrows the list to a bubble, and back to every bubble', async () => {
    const { wrapper } = await mountApp(api(false, plan(four())), '/recommendations')
    expect(bubbles(wrapper).map((b) => b.text())).toEqual(['2', '1', '1'])
    expect(bubbles(wrapper)[0]?.attributes('aria-label')).toBe('2 recommendations with benefit 4 of 5 and effort 1 of 5')
    await bubbles(wrapper)[0]?.trigger('click')
    expect(bubbles(wrapper)[0]?.attributes('aria-pressed')).toBe('true')
    expect(titles(wrapper)).toEqual(['1. Title a', '2. Title e'])
    expect(wrapper.get('section[aria-labelledby="list-title"]').text()).toContain('2 recommendations.')
    await button(wrapper, 'Show every bubble')?.trigger('click')
    expect(titles(wrapper)).toHaveLength(4)
    // Picking the same bubble twice lets it go.
    await bubbles(wrapper)[1]?.trigger('click')
    expect(titles(wrapper)).toEqual(['1. Title c'])
    await bubbles(wrapper)[1]?.trigger('click')
    expect(titles(wrapper)).toHaveLength(4)
    expect(button(wrapper, 'Show every bubble')).toBeUndefined()
  })

  it('filters by kind, and counts what it shows', async () => {
    const { wrapper } = await mountApp(api(false, plan(four())), '/recommendations')
    const show = wrapper.get('[role="group"][aria-label="Show"]')
    expect(show.findAll('button').map((b) => b.text())).toEqual(['All', 'AI readiness', 'General'])
    expect(wrapper.text()).toContain('4 open recommendations')
    await bubbles(wrapper)[0]?.trigger('click')
    await show.findAll('button')[1]?.trigger('click')
    expect(show.findAll('button')[1]?.attributes('aria-pressed')).toBe('true')
    expect(titles(wrapper)).toEqual(['1. Title b'])
    expect(wrapper.text()).toContain('1 open recommendation of this kind')
    expect(bubbles(wrapper).map((b) => b.text())).toEqual(['1'])
  })

  it('has no kind filter while every recommendation is of one kind', async () => {
    const { wrapper } = await mountApp(api(false, plan([imp('a', 4, 1)])), '/recommendations')
    expect(wrapper.find('[role="group"][aria-label="Show"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('1 open recommendation')
  })

  it('folds Done away, with when and by whom', async () => {
    const { wrapper } = await mountApp(api(false, plan(four())), '/recommendations')
    const done = wrapper.get('section[aria-labelledby="done-title"] details')
    expect(done.attributes('open')).toBeUndefined()
    expect(done.get('summary').text()).toBe('Done (1)')
    expect(done.get('a[href="/recommendations/d"]').text()).toBe('Title d')
    expect(done.text()).toContain('Marked done on 2 November 2026, by joris.')
  })

  it('says when nothing is done yet, and when everything is', async () => {
    const none = await mountApp(api(false, plan([imp('a', 4, 1)])), '/recommendations')
    expect(none.wrapper.get('section[aria-labelledby="done-title"]').text()).toContain('Nothing is marked done yet.')
    const all = await mountApp(api(false, plan([imp('d', 5, 5, { done: true })])), '/recommendations')
    expect(all.wrapper.text()).toContain('Every recommendation is done.')
    expect(all.wrapper.find('ul[aria-label="Recommendations"]').exists()).toBe(false)
  })

  it('gives an administrator the link to Manage, and nothing to change here', async () => {
    const { wrapper } = await mountApp(api(true, plan(four())), '/recommendations')
    expect(wrapper.get('a[href="/manage/recommendations"]').text()).toBe('Manage the recommendations')
    expect(button(wrapper, 'Mark as done')).toBeUndefined()
    expect(button(wrapper, 'Undo')).toBeUndefined()
    expect(usesOldWord(wrapper.text())).toBe(false)
  })

  it('says when it cannot load', async () => {
    const failed = await mountApp(routes({ 'GET /api/v1/me': () => json(me()), 'GET /api/v1/plan': () => json({ title: 'Down', status: 500 }, 500) }), '/recommendations')
    expect(failed.wrapper.find('[role="alert"]').exists()).toBe(true)
  })
})
