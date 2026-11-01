import { describe, expect, it } from 'vitest'
import { mountApp, routes } from '../../../testing/mountApp'

describe('Help', () => {
  it('says its topics are coming, in the page frame, under Help in the top bar', async () => {
    const { wrapper, router } = await mountApp(routes({}), '/help')
    expect(wrapper.get('h1').text()).toBe('Help')
    expect(wrapper.get('[data-part="purpose"]').text()).toBe('Short topics on how each figure is measured, and where the work behind it is done.')
    expect(wrapper.get('h2').text()).toBe('Help is coming')
    expect(wrapper.get('nav[aria-label="Main"] a[aria-current="page"]').text()).toBe('Help')
    await router.push('/help/prompt-test')
    expect(wrapper.get('h1').text()).toBe('Help')
  })
})
