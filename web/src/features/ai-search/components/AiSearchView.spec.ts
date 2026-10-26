import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { mountApp, routes } from '../../../testing/mountApp'

const rows = (wrapper: VueWrapper) => wrapper.findAll('ul[aria-label="Crawlers"] button')
const detail = (wrapper: VueWrapper) => wrapper.get('section[aria-labelledby="tl-title"] [aria-live="polite"]')

describe('the AI search view', () => {
  it('opens on Googlebot, which never came', async () => {
    const { wrapper } = await mountApp(routes({}), '/ai-search')
    expect(document.title).toBe('AI search · Findability Explorer')
    expect(wrapper.get('h1').text()).toBe('Google never came. AI crawlers read everything.')
    expect(wrapper.text()).toContain('Log of 30 September – 23 October 2026')
    expect(wrapper.text()).toContain('8,630 requests logged')
    expect(rows(wrapper).map((r) => r.attributes('aria-pressed'))).toEqual(['false', 'false', 'false', 'false', 'true', 'false'])
    expect(detail(wrapper).text()).toContain('Googlebot · Search index')
    expect(detail(wrapper).text()).toContain('No visits at all in more than three weeks')
    expect(wrapper.text()).toContain('No visits in more than three weeks')
  })

  it('tells what a crawler did when it is picked', async () => {
    const { wrapper } = await mountApp(routes({}), '/ai-search')
    await rows(wrapper)[0]?.trigger('click')
    expect(rows(wrapper)[0]?.attributes('aria-pressed')).toBe('true')
    expect(rows(wrapper)[0]?.attributes('aria-label')).toBe('ClaudeBot: 1 day with visits, on 10 October.')
    expect(detail(wrapper).text()).toContain('ClaudeBot · Model training')
    expect(detail(wrapper).text()).toContain('142 reads of robots.txt')
    await rows(wrapper)[3]?.trigger('click')
    expect(detail(wrapper).text()).toContain('GPTBot · Unverified')
    expect(detail(wrapper).text()).toContain('It may not be GPTBot at all.')
  })

  it('lists the prompt test and the ad surfaces', async () => {
    const { wrapper } = await mountApp(routes({}), '/ai-search')
    const [prompts, ads] = wrapper.findAll('table')
    expect(prompts?.get('caption').text()).toBe('Whether Tribelt appeared, per search type and assistant')
    expect(prompts?.findAll('tbody tr').map((r) => r.text())).toEqual([
      expect.stringMatching(/^Supplier search\s*No\s*No\s*Yes\s*In both searches$/),
      expect.stringMatching(/^Technical and quotation\s*No\s*No\s*FAQ cited once\s*No$/),
    ])
    expect(ads?.findAll('tbody th').map((t) => t.text())).toEqual(['Inside Gemini', 'Inside AI Overviews', 'Around AI Overviews', 'AI Mode', 'ChatGPT'])
    expect(wrapper.get('ul[aria-label="Named instead"]').findAll('li')).toHaveLength(6)
    expect(wrapper.text()).toContain('Goal G4')
  })

  it('leads on to the AI readiness of the competitors', async () => {
    const { wrapper } = await mountApp(routes({}), '/ai-search')
    expect(wrapper.get('a[href="/ai-search/readiness"]').text()).toBe('Who follows this: AI readiness of the competitors')
  })
})
