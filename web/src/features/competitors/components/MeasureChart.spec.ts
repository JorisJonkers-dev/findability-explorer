import { mount, RouterLinkStub } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { measuresOf } from '../domain/compare'
import MeasureChart from './MeasureChart.vue'

const measures = measuresOf({ name: 'Twentebelt', share: 0.4, tribeltShare: 0.2, ready: 3, tribeltReady: 5, outOf: 5, sitemap: undefined, tribeltSitemap: 0.85, terms: 12, tribeltTerms: 1200, mapTerms: 2400 })

const render = (list = measures) => mount(MeasureChart, { props: { name: 'Twentebelt', measures: list }, global: { stubs: { RouterLink: RouterLinkStub } } })

describe('the measure chart', () => {
  it('sets the competitor against Tribelt on each measure, and says so for a screen reader', () => {
    const wrapper = render()
    expect(wrapper.get('h2').text()).toBe('Twentebelt against Tribelt')
    const buttons = wrapper.findAll('button')
    expect(buttons.map((b) => b.attributes('aria-label'))).toEqual([
      'AI answers that name it: Twentebelt 40%, Tribelt 20%.',
      'AI readiness checks passed: Twentebelt 3 of 5, Tribelt 5 of 5.',
      'Pages its sitemap lists: Twentebelt Not known, Tribelt 85%.',
      'Search terms from the list on its site: Twentebelt 12, Tribelt 1,200.',
    ])
  })

  it('draws the bars to scale, and none for what is not known', () => {
    const wrapper = render()
    const widths = (i: number) => wrapper.findAll('button')[i]?.findAll('[style]').map((e) => e.attributes('style'))
    expect(widths(0)).toEqual(['width: 100%;', 'width: 50%;'])
    expect(widths(2)).toEqual(['width: 0%;', 'width: 85%;'])
  })

  it('reads the first measure, and what the one picked means', async () => {
    const wrapper = render()
    const buttons = wrapper.findAll('button')
    const reading = wrapper.get('[aria-live="polite"]')
    expect(buttons.map((b) => b.attributes('aria-pressed'))).toEqual(['true', 'false', 'false', 'false'])
    expect(reading.text()).toContain('Twentebelt is named in 40% of the answers')
    expect(reading.getComponent(RouterLinkStub).props('to')).toBe('/ai-search/answers')
    await buttons[3]?.trigger('click')
    expect(buttons.map((b) => b.attributes('aria-pressed'))).toEqual(['false', 'false', 'false', 'true'])
    expect(reading.text()).toContain('tribelt.nl uses 1,200 of the 2,400 terms on the search term list.')
    expect(reading.getComponent(RouterLinkStub).text()).toBe('Open the search terms')
  })

  it('shows no reading without measures', () => {
    expect(render([]).find('[aria-live]').exists()).toBe(false)
  })
})
