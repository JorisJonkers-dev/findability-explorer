import { describe, expect, it } from 'vitest'
import type { Plan, PlanImprovement } from '../../../infrastructure/api/types.gen'
import { usesOldWord } from '../../../shared/words'
import { details, me } from '../../../testing/fixtures'
import { type Handler, json, mountApp, routes, settle } from '../../../testing/mountApp'

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

// a trails the best competitor, b is checked by a scan, c has no best competitor, l leads.
const base = (): PlanImprovement[] => [
  imp('a', 4, 1),
  imp('b', 3, 2, { auto: true, measure: 'readinessChecks', tribelt: 3, leader: { ...leader, value: 7 }, parts: [{ name: 'llms.txt', present: true }, { name: 'llms-full.txt', present: false }] }),
  imp('c', 4, 3, { leader: undefined, measure: 'sitemapCoverage', tribelt: 0.5 }),
  imp('l', 2, 2, { tribelt: 0.25, leader: { ...leader, value: 0.15 } }),
]

function api(admin: boolean, current: Plan, extra: Record<string, Handler> = {}) {
  return routes({ 'GET /api/v1/me': () => json(me({ isAdmin: admin })), 'GET /api/v1/plan': () => json(current), ...extra })
}

const section = (w: Awaited<ReturnType<typeof mountApp>>['wrapper'], id: string) => w.get(`section[aria-labelledby="${id}"]`)

describe('a recommendation', () => {
  it('shows the frame and its five sections in order', async () => {
    const { wrapper } = await mountApp(api(false, plan(base())), '/recommendations/a')
    expect(wrapper.get('h1').text()).toBe('Title a')
    expect(wrapper.text()).toContain('Back to Recommendations')
    expect(wrapper.get('[data-part="purpose"]').text()).toBe('The steps to make this change, why it ranks where it does, and how to tell it is done.')
    expect(wrapper.get('[data-part="data-line"]').text()).toBe('Written by hand, and applies in general.')
    expect(wrapper.findAll('h2').map((h) => h.text())).toEqual(['Where, and how', 'Why it ranks here', 'The goal it improves', 'Steps', 'When it counts as done'])
    expect(section(wrapper, 'where-title').text()).toContain('This applies to the whole site.')
    // Each reason is a paragraph of its own.
    expect(section(wrapper, 'why-title').findAll('p').map((p) => p.text())).toEqual([
      'Why a.',
      'Moves a.',
      'Benefit 4 of 5. Effort 1 of 5. It is work for the web agency.',
      'It ranks 1st of the 4 open recommendations by best value.',
    ])
    expect(section(wrapper, 'steps-title').findAll('ol li').map((li) => li.text())).toEqual(['Do the first thing.', 'Do the second thing.'])
    expect(section(wrapper, 'done-title').text()).toContain('Open')
    expect(section(wrapper, 'done-title').text()).toContain('It is in place.')
    expect(wrapper.find('a[href="/help/recommendations"]').exists()).toBe(true)
  })

  it('says the best competitor is ahead when Tribelt trails, with the bar and the competitor to look at', async () => {
    const { wrapper } = await mountApp(api(false, plan(base())), '/recommendations/a')
    const goal = section(wrapper, 'goal-title')
    expect(goal.text()).toContain('AI responses that name Tribelt.')
    expect(goal.text()).toContain('Tribelt now 20%, the goal 40%. Example Belts is ahead of Tribelt, with 30% against 20%.')
    const bar = goal.get('[role="img"]')
    expect(bar.attributes('aria-label')).toBe('Tribelt now 20%, the best competitor 30%, the goal 40%')
    expect(bar.get('[data-part="now"]').attributes('style')).toContain('width: 20%')
    expect(bar.get('[data-part="leader"]').attributes('style')).toContain('left: 30%')
    expect(bar.get('[data-part="goal"]').attributes('style')).toContain('left: 40%')
    expect(goal.text()).toContain('Tribelt now')
    expect(goal.text()).toContain('The best competitor')
    expect(goal.text()).toContain('The goal')
    expect(goal.get('a[href="/competitors/example-belts"]').text()).toBe('Example Belts')
    // "AI" keeps its capitals inside the sentence.
    expect(goal.text()).toContain('Example Belts is the best competitor on AI responses that name Tribelt. Look at how it does this.')
    // The page keeps to the tool's words: goal, best competitor, response.
    expect(usesOldWord(wrapper.text())).toBe(false)
  })

  it('says Tribelt is ahead when it leads', async () => {
    const { wrapper } = await mountApp(api(false, plan(base())), '/recommendations/l')
    const goal = section(wrapper, 'goal-title')
    expect(goal.text()).toContain('Tribelt now 25%, the goal 40%. Tribelt is ahead of the best competitor, Example Belts, at 15%.')
    expect(goal.text()).not.toContain('is ahead of Tribelt')
  })

  it('has no goal or competitor to show where none watches the measure', async () => {
    const { wrapper } = await mountApp(api(false, plan(base())), '/recommendations/c')
    const goal = section(wrapper, 'goal-title')
    expect(goal.text()).toContain('Pages the sitemaps list.')
    expect(goal.text()).toContain('Tribelt now 50%. No goal watches this measure.')
    expect(goal.find('[data-part="leader"]').exists()).toBe(false)
    expect(goal.find('[data-part="goal"]').exists()).toBe(false)
    expect(goal.text()).not.toContain('Look at how it does this.')
    expect(goal.text()).not.toContain('The best competitor')
  })

  it('says a scan checks it, and what the scan found, with no button', async () => {
    const { wrapper } = await mountApp(api(true, plan(base())), '/recommendations/b')
    const done = section(wrapper, 'done-title')
    expect(done.text()).toContain('The latest scan of tribelt.nl checks this one.')
    expect(done.text()).toContain('The latest scan found llms.txt, but not llms-full.txt.')
    expect(section(wrapper, 'why-title').text()).toContain('It is a change to the site or its files.')
    expect(section(wrapper, 'goal-title').text()).toContain('Example Belts is ahead of Tribelt, with 7 of 8 against 3 of 8.')
    expect(wrapper.text()).not.toContain('Mark as done')
  })

  it('lets an administrator mark it as done and undo that', async () => {
    const sent: unknown[] = []
    let current = plan(base())
    const { wrapper } = await mountApp(
      api(true, current, {
        'GET /api/v1/plan': () => json(current),
        'PUT /api/v1/admin/plan/improvements/a': async (r) => {
          const body = (await r.json()) as { done: boolean }
          sent.push(body)
          current = plan(base().map((i) => (i.id === 'a' ? { ...i, done: body.done, tickedBy: 'joris', tickedAt: '2026-11-02T09:00:00Z' } : i)))
          return json(current)
        },
        'GET /api/v1/standings': () => json({ measures: [], trailing: 0, open: [], readinessChecks: 8 }),
      }),
      '/recommendations/a',
    )
    expect(section(wrapper, 'done-title').text()).toContain('A scan cannot check this one, so an administrator marks it.')
    const toggle = () => wrapper.findAll('button').find((b) => b.text() === 'Mark as done' || b.text() === 'Undo, it is not done')
    await toggle()?.trigger('click')
    await settle()
    expect(wrapper.get('[role="status"]').text()).toBe('Marked done on 2 November 2026, by joris.')
    expect(section(wrapper, 'done-title').text()).toContain('Done')
    expect(section(wrapper, 'why-title').text()).not.toContain('It ranks')
    expect(toggle()?.text()).toBe('Undo, it is not done')
    await toggle()?.trigger('click')
    await settle()
    expect(sent).toEqual([{ done: true }, { done: false }])
    expect(toggle()?.text()).toBe('Mark as done')
  })

  it('says when a change could not be saved', async () => {
    const { wrapper } = await mountApp(api(true, plan(base()), { 'PUT /api/v1/admin/plan/improvements/a': () => json({ title: 'Down', status: 500 }, 500) }), '/recommendations/a')
    await wrapper.findAll('button').find((b) => b.text() === 'Mark as done')?.trigger('click')
    await settle()
    expect(wrapper.find('p[role="alert"]').exists()).toBe(true)
  })

  it('shows a viewer no button, and no link to Manage', async () => {
    const { wrapper } = await mountApp(api(false, plan(base())), '/recommendations/a')
    expect(wrapper.text()).not.toContain('Mark as done')
    expect(wrapper.find('a[href="/manage/recommendations"]').exists()).toBe(false)
  })

  it('says when no recommendation has the address', async () => {
    const { wrapper } = await mountApp(api(false, plan(base())), '/recommendations/nope')
    expect(wrapper.get('h1').text()).toBe('Recommendation')
    expect(wrapper.text()).toContain('No recommendation has this address.')
    expect(wrapper.get('p a[href="/recommendations"]').text()).toBe('See all the recommendations.')
  })

  it('says when it cannot load', async () => {
    const { wrapper } = await mountApp(routes({ 'GET /api/v1/me': () => json(me()), 'GET /api/v1/plan': () => json({ title: 'Down', status: 500 }, 500) }), '/recommendations/a')
    expect(wrapper.find('[role="alert"]').exists()).toBe(true)
  })
})
