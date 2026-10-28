import { describe, expect, it } from 'vitest'
import type { Plan, PlanImprovement } from '../../../infrastructure/api/types.gen'
import { details, me } from '../../../testing/fixtures'
import { type Handler, json, mountApp, routes, settle } from '../../../testing/mountApp'

const leader = { competitorId: 'ashworth', name: 'Ashworth', value: 0.57 }

function imp(id: string, benefit: number, effort: number, extra: Partial<PlanImprovement> = {}): PlanImprovement {
  return { id, title: `Title ${id}`, moves: `Moves ${id}.`, measure: 'aiShareOfVoice', auto: false, done: false, ...details, why: `Why ${id}.`, benefit, effort, tribelt: 0.24, leader, ...extra }
}

function plan(improvements: PlanImprovement[]): Plan {
  return {
    catalogueVersion: 2,
    improvements,
    watched: [{ target: 'aiShareOfVoice', now: 0.24, goal: 0.4, max: 1, met: false }],
    tribeltTermsFrom: 'scan',
    readinessChecks: 8,
    eyeOn: [],
  }
}

const base = (): PlanImprovement[] => [
  imp('a', 4, 1),
  imp('b', 3, 2, { auto: true, measure: 'readinessChecks', tribelt: 3, parts: [{ name: 'llms.txt', present: true }, { name: 'llms-full.txt', present: false }] }),
  imp('c', 4, 3, { leader: undefined }),
]

function api(admin: boolean, current: Plan, extra: Record<string, Handler> = {}) {
  return routes({ 'GET /api/v1/me': () => json(me({ isAdmin: admin })), 'GET /api/v1/plan': () => json(current), ...extra })
}

describe('a recommendation', () => {
  it('shows the figures, the steps, the measure and the leader', async () => {
    const { wrapper } = await mountApp(api(false, plan(base())), '/recommendations/a')
    expect(wrapper.get('h1').text()).toBe('Title a')
    expect(wrapper.text()).toContain('Why a.')
    const figures = wrapper.get('section[aria-label="Key figures"]').text()
    expect(figures).toContain('Large benefit')
    expect(figures).toContain('Four out of five.')
    expect(figures).toContain('A few hours')
    expect(figures).toContain('The least work of all three recommendations.')
    expect(figures).toContain('1st')
    expect(figures).toContain('It comes before the other two.')
    expect(wrapper.text()).toContain('Done when It is in place.')
    expect(wrapper.text()).toContain('AI answers that name Tribelt')
    expect(wrapper.text()).toContain('Moves a.')
    const lead = wrapper.get('a[href="/competitors/ashworth"]')
    expect(lead.text()).toContain('Leads on this measure.')
    expect(wrapper.text()).toContain('An administrator marks this one as done')
    expect(wrapper.get('header').text()).toContain('Back to Recommendations')
    expect(wrapper.find('[role="status"]').exists()).toBe(false)
  })

  it('lists the others that improve the same measure', async () => {
    const { wrapper } = await mountApp(api(false, plan(base())), '/recommendations/a')
    const also = wrapper.get('a[href="/recommendations/c"]')
    expect(also.text()).toContain('Title c')
    expect(also.text()).toContain('A few days of work. Large benefit.')
    expect(wrapper.text()).toContain('Also improves this measure')
    expect(wrapper.find('a[href="/recommendations/b"]').exists()).toBe(false)
    // Without a leader or a measure shared, neither is shown.
    const alone = await mountApp(api(false, plan([imp('c', 4, 3, { leader: undefined })])), '/recommendations/c')
    expect(alone.wrapper.text()).not.toContain('The leader to look at')
    expect(alone.wrapper.text()).not.toContain('Also improves this measure')
  })

  it('ticks a step off and back for the reader alone', async () => {
    const { wrapper } = await mountApp(api(false, plan(base())), '/recommendations/a')
    const steps = () => wrapper.findAll('ol button')
    expect(steps().map((s) => s.attributes('aria-pressed'))).toEqual(['false', 'false'])
    await steps()[0]?.trigger('click')
    expect(steps()[0]?.attributes('aria-pressed')).toBe('true')
    expect(steps()[0]?.text()).toContain('✓')
    await steps()[0]?.trigger('click')
    expect(steps()[0]?.attributes('aria-pressed')).toBe('false')
  })

  it('says a scan checks it, and what the scan found', async () => {
    const { wrapper } = await mountApp(api(true, plan(base())), '/recommendations/b')
    expect(wrapper.text()).toContain('The latest scan of tribelt.nl checks this one.')
    expect(wrapper.text()).toContain('The latest scan found llms.txt, but not llms-full.txt.')
    expect(wrapper.text()).toContain('Ranked by best value')
    expect(wrapper.find('button[aria-pressed][class*="bg-black px-5"]').exists()).toBe(false)
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
          current = plan(base().map((i) => (i.id === 'a' ? { ...i, done: body.done, tickedBy: 'joris' } : i)))
          return json(current)
        },
        'GET /api/v1/standings': () => json({ measures: [], trailing: 0, open: [], readinessChecks: 8 }),
      }),
      '/recommendations/a',
    )
    const toggle = () => wrapper.findAll('button').find((b) => b.text() === 'Mark as done' || b.text() === 'Undo, it is not done')
    await toggle()?.trigger('click')
    await settle()
    expect(wrapper.text()).toContain('Marked as done by joris.')
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

  it('shows a viewer no button', async () => {
    const { wrapper } = await mountApp(api(false, plan(base())), '/recommendations/a')
    expect(wrapper.text()).not.toContain('Mark as done')
  })

  it('says who marked it as done, and places no rank on it', async () => {
    const done = base().map((i) => (i.id === 'a' ? { ...i, done: true, tickedBy: 'joris' } : i))
    const { wrapper } = await mountApp(api(false, plan(done)), '/recommendations/a')
    expect(wrapper.get('[role="status"]').text()).toContain('Marked as done. The next recommendation moves up.')
    expect(wrapper.text()).toContain('Marked as done by joris.')
    expect(wrapper.text()).not.toContain('Ranked by best value')
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
