import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { Router } from 'vue-router'
import { mountApp, routes, settle } from '../../../testing/mountApp'

/** Where a navigation was headed; the guard stops it before the view it would load. */
function headedFor(router: Router): string[] {
  const seen: string[] = []
  router.beforeEach((to) => {
    seen.push(to.fullPath)
    return false
  })
  return seen
}

const pills = (wrapper: VueWrapper) => wrapper.findAll('[role="group"][aria-label="Measure"] button')

describe('the Buyers view', () => {
  it('opens on the contact rate, with its heading, figures and suggestions', async () => {
    const { wrapper } = await mountApp(routes({}), '/buyers')
    expect(document.title).toBe('Buyers · Findability Explorer')
    expect(wrapper.get('h1').text()).toBe('Paid search buys traffic, not buyers.')
    expect(wrapper.findAll('a').some((a) => a.text() === 'Back to Overview')).toBe(true)
    expect(pills(wrapper).map((b) => [b.text(), b.attributes('aria-pressed')])).toEqual([
      ['Users', 'false'],
      ['Contact rate', 'true'],
      ['Engagement time', 'false'],
      ['Engaged sessions', 'false'],
      ['Returning users', 'false'],
    ])
    expect(wrapper.get('[role="img"]').attributes('aria-label')).toBe('Contact rate: paid search 0.04%, organic search 1.73%, AI assistants 0.00%.')
    expect(wrapper.get('[aria-live="polite"]').text()).toBe('Organic visitors contact Tribelt 43 times more often than paid visitors.')
    expect(wrapper.findAll('tbody tr')).toHaveLength(7)
    expect(wrapper.get('aside').text()).toContain('Goal G2')
    expect(wrapper.find('img[alt="Close-up of a Tribelt metal mesh belt"]').exists()).toBe(false)
  })

  it('switches the chart to another measure', async () => {
    const { wrapper } = await mountApp(routes({}), '/buyers')
    await pills(wrapper)[0]?.trigger('click')
    expect(pills(wrapper)[0]?.attributes('aria-pressed')).toBe('true')
    expect(pills(wrapper)[1]?.attributes('aria-pressed')).toBe('false')
    expect(wrapper.get('[role="img"]').attributes('aria-label')).toBe('Users: paid search 18,772, organic search 640, AI assistants 13.')
    expect(wrapper.get('[aria-live="polite"]').text()).toBe('Paid search brings 29 times more users than organic search.')
    await pills(wrapper)[2]?.trigger('click')
    expect(wrapper.get('[role="img"]').text()).toContain('1:04')
  })

  it('goes back to the overview', async () => {
    const { wrapper, router } = await mountApp(routes({}), '/buyers')
    const seen = headedFor(router)
    await wrapper.findAll('nav[aria-label="Main"] a').find((a) => a.text() === 'Overview')?.trigger('click')
    await settle()
    expect(seen).toEqual(['/'])
  })
})
