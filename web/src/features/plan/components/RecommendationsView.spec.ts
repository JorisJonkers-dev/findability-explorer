import type { VueWrapper } from '@vue/test-utils'
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
    watched: [
      { target: 'aiShareOfVoice', now: 0.24, goal: 0.4, max: 1, met: false },
      { target: 'citationShare', goal: 20 },
      { target: 'termsUsed', now: 45, goal: 35, met: true },
    ],
    tribeltTermsFrom: 'scan',
    readinessChecks: 8,
    eyeOn: [],
  }
}

// Best value first, as the server sends them: a (4, 1), b (3, 1) scanned, c (4, 3), and d done.
const three = (): PlanImprovement[] => [
  imp('a', 4, 1),
  imp('b', 3, 1, { auto: true, measure: 'readinessChecks', tribelt: 3, leader: { ...leader, value: 8 } }),
  imp('c', 4, 3, { measure: 'termsUsed', tribelt: 20 }),
  imp('d', 5, 5, { done: true, tickedBy: 'joris', tickedAt: '2026-11-02T09:00:00Z' }),
]

function api(admin: boolean, current: Plan, extra: Record<string, Handler> = {}) {
  return routes({ 'GET /api/v1/me': () => json(me({ isAdmin: admin })), 'GET /api/v1/plan': () => json(current), ...extra })
}

const rows = (w: VueWrapper) => w.findAll('ul[aria-label="Recommendations"] > li')
const button = (w: VueWrapper, text: string) => w.findAll('button').find((b) => b.text() === text)

describe('the recommendations', () => {
  it('starts with the best value, and lists the rest in order', async () => {
    const { wrapper } = await mountApp(api(false, plan(three())), '/recommendations')
    expect(document.title).toBe('Recommendations · Findability Explorer')
    expect(wrapper.get('h1').text()).toBe('Start with the changes that help Tribelt most.')
    const first = wrapper.get('section[aria-labelledby="first-title"] article')
    expect(first.get('h2').text()).toBe('Title a')
    expect(first.text()).toContain('Do this first')
    expect(first.text()).toContain('Why a.')
    expect(first.text()).toContain('A few hours')
    expect(first.text()).toContain('Large benefit')
    expect(first.text()).toContain('Ashworth')
    expect(first.text()).toContain('Target')
    expect(first.get('a[href="/recommendations/a"]').text()).toBe('Open the recommendation')
    expect(rows(wrapper).map((r) => r.get('a').text())).toEqual(['Title b', 'Title c'])
    expect(rows(wrapper)[0]?.text()).toContain('AI readiness checks passed: 3 of eight, Ashworth 8 of eight')
    // The chart has one dot per open recommendation, ranked.
    expect(wrapper.findAll('[aria-label="Benefit against effort"] button').map((b) => b.text())).toEqual(['1', '2', '3'])
    expect(wrapper.text()).toContain('1 recommendation done')
  })

  it('ranks by the sort that is picked', async () => {
    const { wrapper } = await mountApp(api(false, plan(three())), '/recommendations')
    const pill = (name: string) => wrapper.findAll('[aria-label="Rank by"] button').find((b) => b.text() === name)
    expect(pill('Best value')?.attributes('aria-pressed')).toBe('true')
    await pill('Biggest benefit')?.trigger('click')
    expect(pill('Biggest benefit')?.attributes('aria-pressed')).toBe('true')
    expect(rows(wrapper).map((r) => r.get('a').text())).toEqual(['Title c', 'Title b'])
    await pill('Least work')?.trigger('click')
    expect(wrapper.get('section[aria-labelledby="first-title"] h2').text()).toBe('Title a')
    expect(rows(wrapper).map((r) => r.get('a').text())).toEqual(['Title b', 'Title c'])
  })

  it('shows another recommendation at the top when its dot is picked', async () => {
    const { wrapper } = await mountApp(api(false, plan(three())), '/recommendations')
    await wrapper.findAll('[aria-label="Benefit against effort"] button')[2]?.trigger('click')
    expect(wrapper.get('section[aria-labelledby="first-title"] h2').text()).toBe('Title c')
    expect(wrapper.get('section[aria-labelledby="first-title"]').text()).toContain('You picked this one')
    expect(rows(wrapper).map((r) => r.get('a').text())).toEqual(['Title a', 'Title b'])
    // Picking the best one again is just the best one.
    await wrapper.findAll('[aria-label="Benefit against effort"] button')[0]?.trigger('click')
    expect(wrapper.get('section[aria-labelledby="first-title"]').text()).toContain('Do this first')
    // So is picking from the list.
    await rows(wrapper)[1]?.findAll('button')[0]?.trigger('click')
    await rows(wrapper)[1]?.findAll('button').find((b) => b.text() === 'Show at the top')?.trigger('click')
    expect(wrapper.get('section[aria-labelledby="first-title"] h2').text()).toBe('Title c')
  })

  it('shows and hides the steps of the top one and of each listed one', async () => {
    const { wrapper } = await mountApp(api(false, plan(three())), '/recommendations')
    const top = wrapper.get('section[aria-labelledby="first-title"]')
    expect(top.text()).not.toContain('Do the first thing.')
    await top.findAll('button').find((b) => b.text() === 'Show the steps')?.trigger('click')
    expect(top.text()).toContain('Do the first thing.')
    expect(top.text()).toContain('Done when It is in place.')
    await top.findAll('button').find((b) => b.text() === 'Hide the steps')?.trigger('click')
    expect(top.text()).not.toContain('Do the first thing.')
    const second = rows(wrapper)[0]
    await second?.get('button').trigger('click')
    expect(second?.text()).toContain('Why b.')
    expect(second?.text()).toContain('Do the second thing.')
    expect(second?.find('a[href="/recommendations/b"]').exists()).toBe(true)
    await second?.get('button').trigger('click')
    expect(second?.text()).not.toContain('Do the second thing.')
  })

  it('marks as done and undoes, as an administrator, and not for a scanned one', async () => {
    const sent: { url: string; body: unknown }[] = []
    let current = plan(three())
    const put: Handler = async (r) => {
      sent.push({ url: new URL(r.url).pathname, body: await r.json() })
      const id = new URL(r.url).pathname.split('/').pop()
      current = plan(current.improvements.map((i) => (i.id === id ? { ...i, done: (sent.at(-1)?.body as { done: boolean }).done } : i)))
      return json(current)
    }
    const { wrapper } = await mountApp(
      routes({
        'GET /api/v1/me': () => json(me({ isAdmin: true })),
        'GET /api/v1/plan': () => json(current),
        'PUT /api/v1/admin/plan/improvements/a': put,
        'PUT /api/v1/admin/plan/improvements/d': put,
        'GET /api/v1/standings': () => json({ measures: [], trailing: 0, open: [], readinessChecks: 8 }),
      }),
      '/recommendations',
    )
    // The scanned one offers no button, even when opened.
    await rows(wrapper)[0]?.get('button').trigger('click')
    expect(rows(wrapper)[0]?.findAll('button').map((b) => b.text())).toEqual(['Hide the steps', 'Show at the top'])
    await button(wrapper, 'Mark as done')?.trigger('click')
    await settle()
    expect(sent).toEqual([{ url: '/api/v1/admin/plan/improvements/a', body: { done: true } }])
    expect(wrapper.get('section[aria-labelledby="first-title"] h2').text()).toBe('Title b')
    expect(wrapper.text()).toContain('2 recommendations done')
    // Undo is there for the hand-ticked one only.
    const undo = wrapper.findAll('section[aria-labelledby="done-title"] button')
    expect(undo.map((b) => b.text())).toEqual(['Undo', 'Undo'])
    await undo[1]?.trigger('click')
    await settle()
    expect(sent[1]).toEqual({ url: '/api/v1/admin/plan/improvements/d', body: { done: false } })
    expect(wrapper.text()).toContain('1 recommendation done')
  })

  it('marks one done from the list, and forgets a pick it was showing', async () => {
    let current = plan(three())
    const { wrapper } = await mountApp(
      routes({
        'GET /api/v1/me': () => json(me({ isAdmin: true })),
        'GET /api/v1/plan': () => json(current),
        'PUT /api/v1/admin/plan/improvements/c': () => {
          current = plan(current.improvements.map((i) => (i.id === 'c' ? { ...i, done: true } : i)))
          return json(current)
        },
        'GET /api/v1/standings': () => json({ measures: [], trailing: 0, open: [], readinessChecks: 8 }),
      }),
      '/recommendations',
    )
    await wrapper.findAll('[aria-label="Benefit against effort"] button')[2]?.trigger('click')
    await button(wrapper, 'Mark as done')?.trigger('click')
    await settle()
    expect(wrapper.get('section[aria-labelledby="first-title"] h2').text()).toBe('Title a')
    // From the list, with another one picked.
    await rows(wrapper)[0]?.get('button').trigger('click')
    expect(button(wrapper, 'Mark as done')).toBeDefined()
    await button(wrapper, 'Mark as done')?.trigger('click')
  })

  it('shows a viewer no way to change anything', async () => {
    const { wrapper } = await mountApp(api(false, plan(three())), '/recommendations')
    expect(button(wrapper, 'Mark as done')).toBeUndefined()
    expect(button(wrapper, 'Undo')).toBeUndefined()
    expect(wrapper.find('section[aria-labelledby="targets-title"]').exists()).toBe(false)
  })

  it('says when every recommendation is done', async () => {
    const { wrapper } = await mountApp(api(false, plan([imp('d', 5, 5, { done: true })])), '/recommendations')
    expect(wrapper.get('#all-done').text()).toBe('Every recommendation is done.')
    expect(wrapper.find('ul[aria-label="Recommendations"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('1 recommendation done')
  })

  it('has no done list while nothing is done', async () => {
    const { wrapper } = await mountApp(api(false, plan([imp('a', 4, 1)])), '/recommendations')
    expect(wrapper.find('section[aria-labelledby="done-title"]').exists()).toBe(false)
    expect(wrapper.find('section[aria-labelledby="next-title"]').exists()).toBe(false)
  })

  it('lets an administrator change the targets', async () => {
    const sent: unknown[] = []
    const { wrapper } = await mountApp(
      api(true, plan(three()), {
        'PUT /api/v1/admin/plan/targets': async (r) => {
          sent.push(await r.json())
          return json(plan(three()))
        },
        'GET /api/v1/standings': () => json({ measures: [], trailing: 0, open: [], readinessChecks: 8 }),
      }),
      '/recommendations',
    )
    expect(wrapper.get('section[aria-labelledby="targets-title"]').text()).toContain('The targets are shown on the overview.')
    await button(wrapper, 'Change the targets')?.trigger('click')
    const inputs = wrapper.findAll('form[aria-label="Targets"] input')
    expect((inputs[0]?.element as HTMLInputElement).value).toBe('0.4')
    await inputs[1]?.setValue('25')
    await inputs[2]?.setValue('')
    await wrapper.get('form[aria-label="Targets"]').trigger('submit')
    await settle()
    expect(sent).toEqual([
      {
        targets: [
          { target: 'aiShareOfVoice', value: 0.4 },
          { target: 'citationShare', value: 25 },
        ],
      },
    ])
    expect(wrapper.find('form[aria-label="Targets"]').exists()).toBe(false)
    await button(wrapper, 'Change the targets')?.trigger('click')
    await button(wrapper, 'Cancel')?.trigger('click')
    expect(wrapper.find('form[aria-label="Targets"]').exists()).toBe(false)
  })

  it('says when it cannot load, and while it loads', async () => {
    const failed = await mountApp(routes({ 'GET /api/v1/me': () => json(me()), 'GET /api/v1/plan': () => json({ title: 'Down', status: 500 }, 500) }), '/recommendations')
    expect(failed.wrapper.find('[role="alert"]').exists()).toBe(true)
  })

  it('says when a change could not be saved', async () => {
    const { wrapper } = await mountApp(
      api(true, plan(three()), { 'PUT /api/v1/admin/plan/improvements/a': () => json({ title: 'Down', status: 500 }, 500) }),
      '/recommendations',
    )
    await button(wrapper, 'Mark as done')?.trigger('click')
    await settle()
    expect(wrapper.find('p[role="alert"]').exists()).toBe(true)
  })
})
