import { describe, expect, it } from 'vitest'
import { json, problem, routes } from '../../../testing/mountApp'
import { mountAt } from '../../../testing/mountAt'
import TermPages from './TermPages.vue'

const items = [
  { competitorId: 'tribelt', name: 'Tribelt B.V.', role: 'tribelt', pages: [{ url: 'https://www.tribelt.nl/en/eyelink', title: 'Eyelink belts' }] },
  { competitorId: 'esfo', name: 'ESFO', role: 'competitor', pages: [{ url: 'https://www.esfo.test/belts/', title: '' }] },
  { competitorId: 'twentebelt', name: 'Twentebelt', role: 'competitor', pages: [] },
]

describe('the pages that use a search term', () => {
  it('shows, side by side, where competitors use it and where tribelt.nl does', async () => {
    const { wrapper, requests } = await mountAt(TermPages, '/terms/x', routes({ 'GET /api/v1/term-pages': () => json({ items }) }), {
      props: { term: 'eyelink belt', language: 'en' },
    })
    expect(requests()).toContain('GET /api/v1/term-pages?term=eyelink%20belt&language=en')
    expect(wrapper.findAll('h2').map((h) => h.text())).toEqual(['Where competitors use it', 'On tribelt.nl'])
    const competitors = wrapper.get('section[aria-labelledby="term-pages-competitors"]')
    expect(competitors.get('a[href="/competitors/esfo"]').text()).toBe('ESFO')
    const page = competitors.get('a[href="https://www.esfo.test/belts/"]')
    expect([page.text(), page.attributes('target'), page.attributes('rel')]).toEqual(['www.esfo.test/belts', '_blank', 'noopener'])
    expect(competitors.text()).toContain('Its newest scan found the term before scans recorded the pages. The next scan lists them.')
    const tribelt = wrapper.get('section[aria-labelledby="term-pages-tribelt"]')
    expect(tribelt.get('a').text()).toBe('Eyelink belts')
    expect(wrapper.html()).toContain('min-[1100px]:grid-cols-2')
  })

  it('says when nobody uses the term, and when the pages cannot load', async () => {
    const { wrapper } = await mountAt(TermPages, '/terms/x', routes({ 'GET /api/v1/term-pages': () => json({ items: [] }) }), { props: { term: 'x', language: 'en' } })
    expect(wrapper.text()).toContain('No competitor’s newest scan found this term.')
    expect(wrapper.text()).toContain('tribelt.nl’s newest scan did not find this term.')
    const { wrapper: failing } = await mountAt(TermPages, '/terms/x', routes({ 'GET /api/v1/term-pages': () => problem(500, 'Internal Server Error', 'Down.') }), {
      props: { term: 'x', language: 'en' },
    })
    expect(failing.get('[role="alert"]').text()).toContain('Down.')
  })
})
