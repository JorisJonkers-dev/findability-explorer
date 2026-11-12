import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SortButton from './SortButton.vue'

describe('the sort button', () => {
  it('names the column by its label and says how it is sorted', () => {
    const wrapper = mount(SortButton, { props: { label: 'Company', direction: null } })
    expect(wrapper.get('button').attributes('aria-label')).toBe('Company, not sorted')
    expect(wrapper.text()).toBe('Company')
  })

  it('opens the name with a shorter header that is shown instead of the label', () => {
    const wrapper = mount(SortButton, { props: { label: 'Product', short: 'Prod.', direction: null } })
    expect(wrapper.get('button').attributes('aria-label')).toBe('Prod. Product, not sorted')
    expect(wrapper.text()).toBe('Prod.')
  })

  it('does not repeat a short header that is the label', () => {
    const wrapper = mount(SortButton, { props: { label: 'Broad', short: 'Broad', direction: null } })
    expect(wrapper.get('button').attributes('aria-label')).toBe('Broad, not sorted')
  })
})
