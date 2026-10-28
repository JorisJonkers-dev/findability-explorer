import { mount, RouterLinkStub } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { Row } from '../domain/compare'
import CompetitorList from './CompetitorList.vue'

const rows: Row[] = Array.from({ length: 10 }, (_, i) => ({ id: `c${String(i)}`, name: `Maker ${String(i)}`, share: i === 9 ? undefined : i / 20, ready: i % 6 }))
const tribelt = { name: 'Tribelt', share: 0.3, ready: 4 }

function render(props: Partial<InstanceType<typeof CompetitorList>['$props']> = {}) {
  return mount(CompetitorList, {
    props: { rows, tribelt, group: [], outOf: 5, ...props },
    global: { stubs: { RouterLink: RouterLinkStub } },
  })
}

describe('the competitor list', () => {
  it('ranks by AI answers, shows the first eight, and links each row to its page', () => {
    const wrapper = render()
    const links = wrapper.findAllComponents(RouterLinkStub)
    expect(links).toHaveLength(8)
    expect(links[0]?.props('to')).toBe('/competitors/c8')
    expect(links[0]?.attributes('aria-label')).toBe('Maker 8. AI answers that name it: 40%. AI readiness checks passed: 2 of 5. Open its page.')
    expect(wrapper.get('[aria-pressed="true"]').text()).toBe('AI answers')
  })

  it('puts Tribelt on top for comparison, and leaves it out when it is not known', () => {
    const wrapper = render()
    const first = wrapper.get('ol li')
    expect(first.text()).toContain('Tribelt')
    expect(first.text()).toContain('For comparison')
    expect(first.text()).toContain('30%')
    expect(first.text()).toContain('4 of 5')
    expect(render({ tribelt: undefined }).text()).not.toContain('For comparison')
  })

  it('says what a row has no value for', () => {
    const wrapper = render({ rows: [{ id: 'new', name: 'New Belts', share: undefined, ready: undefined }] })
    expect(wrapper.get('ol a').text()).toContain('No run yet')
    expect(wrapper.get('ol a').text()).toContain('Not scanned')
  })

  it('ranks by readiness or by name when asked', async () => {
    const wrapper = render({ rows: rows.slice(0, 3) })
    const pick = (label: string) => wrapper.findAll('[role="group"] button').find((b) => b.text() === label)
    await pick('AI readiness')?.trigger('click')
    expect(pick('AI readiness')?.attributes('aria-pressed')).toBe('true')
    expect(wrapper.findAllComponents(RouterLinkStub).map((l) => l.props('to'))).toEqual(['/competitors/c2', '/competitors/c1', '/competitors/c0'])
    await pick('Name')?.trigger('click')
    expect(wrapper.findAllComponents(RouterLinkStub).map((l) => l.props('to'))).toEqual(['/competitors/c0', '/competitors/c1', '/competitors/c2'])
  })

  it('shows the rest on request, and the first few again after', async () => {
    const wrapper = render()
    const more = wrapper.get('button[aria-expanded]')
    expect(more.text()).toBe('Show all 10 competitors')
    await more.trigger('click')
    expect(wrapper.findAllComponents(RouterLinkStub)).toHaveLength(10)
    expect(more.attributes('aria-expanded')).toBe('true')
    expect(more.text()).toBe('Show only the first 8')
    await more.trigger('click')
    expect(wrapper.findAllComponents(RouterLinkStub)).toHaveLength(8)
  })

  it('has no button for a short list', () => {
    expect(render({ rows: rows.slice(0, 8) }).find('button[aria-expanded]').exists()).toBe(false)
  })

  it('says which group companies it leaves out', () => {
    expect(render().text()).not.toContain('left out')
    expect(render({ group: ['Tribelt Group'] }).text()).toContain('Tribelt Group is left out: it is a group company of Tribelt, not a competitor.')
    expect(render({ group: ['A', 'B'] }).text()).toContain('A and B are left out: they are group companies of Tribelt, not competitors.')
  })
})
