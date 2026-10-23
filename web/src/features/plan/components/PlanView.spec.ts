import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { mountApp, routes } from '../../../testing/mountApp'

const phases = (wrapper: VueWrapper) => wrapper.findAll('[role="group"][aria-label="Phases"] button')
const moves = (wrapper: VueWrapper) => wrapper.findAll('section[aria-labelledby="mv-title"] button')
const detail = (wrapper: VueWrapper) => wrapper.get('section[aria-labelledby="rm-title"] [aria-live="polite"]')

describe('the plan view', () => {
  it('opens on the first phase and the first move', async () => {
    const { wrapper } = await mountApp(routes({}), '/plan')
    expect(document.title).toBe('The plan · Findability Explorer')
    expect(wrapper.get('h1').text()).toBe('Measure first. Then build. Then expand.')
    expect(phases(wrapper).map((p) => p.attributes('aria-pressed'))).toEqual(['true', 'false', 'false'])
    expect(detail(wrapper).text()).toContain('Measure and fix · Months 0–3')
    expect(detail(wrapper).text()).toContain('Run the 40-prompt baseline test.')
    expect(moves(wrapper).map((m) => m.attributes('aria-expanded'))).toEqual(['true', 'false', 'false', 'false', 'false'])
    expect(wrapper.findAll('section[aria-labelledby="gl-title"], aside li')).toHaveLength(5)
  })

  it('shows a phase when it is picked', async () => {
    const { wrapper } = await mountApp(routes({}), '/plan')
    await phases(wrapper)[2]?.trigger('click')
    expect(phases(wrapper).map((p) => p.attributes('aria-pressed'))).toEqual(['false', 'false', 'true'])
    expect(detail(wrapper).text()).toContain('Expand and learn · Months 6–12')
    expect(detail(wrapper).text()).toContain('go or no-go on the guidance tool')
  })

  it('opens one move at a time, and closes the open one', async () => {
    const { wrapper } = await mountApp(routes({}), '/plan')
    const panel = (n: string) => wrapper.get(`#move-${n}`)
    expect(moves(wrapper)[1]?.attributes('aria-controls')).toBe('move-2')
    await moves(wrapper)[1]?.trigger('click')
    expect(moves(wrapper).map((m) => m.attributes('aria-expanded'))).toEqual(['false', 'true', 'false', 'false', 'false'])
    expect(panel('2').isVisible()).toBe(true)
    expect(panel('1').isVisible()).toBe(false)
    expect(panel('2').text()).toContain('Bid on a qualified-lead goal')
    await moves(wrapper)[1]?.trigger('click')
    expect(moves(wrapper).map((m) => m.attributes('aria-expanded'))).toEqual(['false', 'false', 'false', 'false', 'false'])
  })
})
