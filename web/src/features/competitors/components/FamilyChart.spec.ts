import { mount, RouterLinkStub } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { FamilyBar } from '../domain/compare'
import FamilyChart from './FamilyChart.vue'

const bars: FamilyBar[] = [
  { family: 'eyelink', label: 'Eyelink', makers: 4, tribelt: 'Draadogenbanden', path: '/terms/families/eyelink' },
  { family: 'open_spiral', label: 'Open spiral', makers: 2, tribelt: null, path: '/terms/families/open-spiral' },
]

const render = (props: { bars: FamilyBar[] } = { bars }) => mount(FamilyChart, { props, global: { stubs: { RouterLink: RouterLinkStub } } })

describe('the family chart', () => {
  it('has a button and a bar for each family, and describes itself', () => {
    const wrapper = render()
    expect(wrapper.get('[role="group"]').attributes('aria-label')).toBe('Competitors that sell each belt family. Eyelink: 4. Open spiral: 2.')
    const buttons = wrapper.findAll('[role="group"] button')
    expect(buttons.map((b) => b.text())).toEqual(['Eyelink4', 'Open spiral2'])
    expect(buttons[0]?.get('[style]').attributes('style')).toContain('width: 100%')
    expect(buttons[1]?.get('[style]').attributes('style')).toContain('width: 50%')
  })

  it('opens on the first family, and says how Tribelt sells the one picked', async () => {
    const wrapper = render()
    const buttons = wrapper.findAll('[role="group"] button')
    expect(buttons.map((b) => b.attributes('aria-pressed'))).toEqual(['true', 'false'])
    const reading = wrapper.get('[aria-live="polite"]')
    expect(reading.text()).toContain('4 of the competitors sell it. Tribelt sells it as Draadogenbanden.')
    expect(reading.getComponent(RouterLinkStub).props('to')).toBe('/terms/families/eyelink')
    expect(reading.getComponent(RouterLinkStub).text()).toBe('Open the eyelink page')
    await buttons[1]?.trigger('click')
    expect(buttons.map((b) => b.attributes('aria-pressed'))).toEqual(['false', 'true'])
    expect(reading.text()).toContain('Tribelt does not sell this family.')
    expect(reading.getComponent(RouterLinkStub).props('to')).toBe('/terms/families/open-spiral')
  })

  it('shows nothing picked without families', () => {
    const wrapper = render({ bars: [] })
    expect(wrapper.find('[aria-live]').exists()).toBe(false)
    expect(wrapper.findAll('[role="group"] button')).toHaveLength(0)
  })
})
