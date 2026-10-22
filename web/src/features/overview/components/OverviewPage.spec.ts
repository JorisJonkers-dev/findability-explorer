import { describe, expect, it } from 'vitest'
import { analysis, summary } from '../../../testing/fixtures'
import { json, mountApp, routes } from '../../../testing/mountApp'

const NEW_ID = '6c1d2b0f-3e4a-4f6b-9c7d-8e9f0a1b2c3d'
const api = routes({
  'GET /api/v1/analyses': () => json({ items: [summary()] }),
  'POST /api/v1/analyses': () => json(analysis({}, { id: NEW_ID, status: 'queued' }), 202),
  [`GET /api/v1/analyses/${NEW_ID}`]: () => json(analysis({}, { id: NEW_ID, status: 'queued' })),
  'GET /api/v1/terms': () => json({ items: [], total: 0, facets: { languages: [], tiers: [], families: [] } }),
  'GET /api/v1/competitors': () => json({ items: [] }),
})

describe('the overview', () => {
  it("tells the research's story as the report has it", async () => {
    const { wrapper } = await mountApp(api, '/')
    expect(wrapper.get('figure').text()).toContain('Tri-Flex')
    expect(wrapper.findAll('figure li').map((li) => li.text())).toEqual([
      'TwenteFlexTwentebelt, NL',
      'Omni-GridAshworth, US',
      'Cam-GridCambridge, US',
      'RA rondloopbandESFO, NL',
    ])
    expect(wrapper.findAll('article h3').map((h) => h.text())).toEqual([
      'Paid search brings visitors, not buyers',
      'Gemini never named Tribelt',
      'AI crawlers read everything, Google nothing',
    ])
    expect(wrapper.text()).toContain('0.04%')
    expect(wrapper.text()).toContain('1.73%')
    expect(wrapper.text()).toContain('146 terms in 10 languages, taken from the websites of 25 competitors')
    expect(wrapper.findAll('section[aria-labelledby="map-title"] a').map((a) => a.attributes('href'))).toEqual([
      '/terminology',
      '/competitors',
    ])
  })

  it('checks the quick scan address here', async () => {
    const { wrapper, requests } = await mountApp(api, '/')
    await wrapper.get('form[aria-label="Quick scan"]').trigger('submit')
    expect(wrapper.get('#quick-error').text()).toBe('Enter the address of a website.')
    expect(wrapper.get('#quick').attributes('aria-invalid')).toBe('true')
    await wrapper.get('#quick').setValue('x')
    expect(wrapper.find('#quick-error').exists()).toBe(false)
    expect(requests().filter((r) => r.startsWith('POST'))).toEqual([])
  })

  it('hands a good address to the scan page, which starts the scan', async () => {
    const { wrapper, router, requests } = await mountApp(api, '/')
    await wrapper.get('#quick').setValue('https://www.twentebelt.com')
    await wrapper.get('form[aria-label="Quick scan"]').trigger('submit')
    const { settle } = await import('../../../testing/mountApp')
    await settle()
    expect(requests()).toContain('POST /api/v1/analyses')
    expect(router.currentRoute.value.path).toBe(`/scan/${NEW_ID}`)
  })
})
