import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { AiReadiness } from '../../../infrastructure/api/types.gen'
import { bots, readiness, readinessRow } from '../../../testing/fixtures'
import { json, mountApp, problem, routes } from '../../../testing/mountApp'

const gappy: AiReadiness = readiness({
  bots: bots({ GPTBot: 'blocked' }),
  llmsTxt: { present: false, bytes: 0, links: 0 },
  advice: readiness().advice.map((a) => (a.check === 'llmsTxt' ? { ...a, ok: false } : a)),
  gaps: 2,
})

const table = [
  readinessRow('Twentebelt', { readiness: gappy }),
  readinessRow('ESFO'),
  readinessRow('Unibelt', { readiness: undefined, scan: undefined }),
  readinessRow('Lesjöfors', { role: 'group', readiness: gappy }),
  readinessRow('Mirror of tribelt.nl', { competitorId: 'tribelt-mirror', role: 'mirror' }),
  readinessRow('Tribelt B.V.', { competitorId: 'tribelt-b-v', role: 'tribelt', readiness: gappy }),
]

const api = () => routes({ 'GET /api/v1/ai-readiness': () => json({ items: table }) })

/** Each band's company names, top to bottom. */
const bands = (wrapper: VueWrapper) =>
  wrapper.findAll('tbody').map((b) => `${b.attributes('aria-label') ?? ''}: ${b.findAll('tr th[scope="row"]').map((th) => th.text()).join(', ')}`)

describe('the AI readiness view', () => {
  it('pins Tribelt and the mirror above the group and the competitors', async () => {
    const { wrapper } = await mountApp(api(), '/ai-search/readiness')
    expect(document.title).toBe('AI readiness · Findability Explorer')
    expect(wrapper.get('h1').text()).toBe('Who lets search engines and AI assistants in.')
    expect(bands(wrapper)).toEqual([
      'Tribelt today, and the target: Tribelt B.V., Mirror of tribelt.nl',
      "Tribelt's group: Lesjöfors",
      'Competitors: Twentebelt, ESFO, Unibelt',
    ])
    expect(wrapper.text()).toContain('6 of 6 companies')
    const twente = wrapper.findAll('tbody')[2]?.findAll('tr')[1]
    expect(twente?.text()).toContain('Blocked, a gap')
    expect(wrapper.findAll('tbody')[2]?.findAll('tr')[3]?.text()).toContain('Not checked')
  })

  it('sorts a band by any column, and back, keeping Tribelt and the mirror on top', async () => {
    const { wrapper } = await mountApp(api(), '/ai-search/readiness')
    const gptbot = wrapper.get('button[aria-label^="GPTBot (OpenAI training)"]')
    await gptbot.trigger('click')
    expect(gptbot.attributes('aria-label')).toBe('GPTBot (OpenAI training), sorted descending')
    expect(bands(wrapper)[2]).toBe('Competitors: ESFO, Twentebelt, Unibelt')
    await gptbot.trigger('click')
    expect(bands(wrapper)[2]).toBe('Competitors: Unibelt, Twentebelt, ESFO')
    expect(bands(wrapper)[0]).toBe('Tribelt today, and the target: Tribelt B.V., Mirror of tribelt.nl')
    await gptbot.trigger('click')
    expect(gptbot.attributes('aria-label')).toBe('GPTBot (OpenAI training), not sorted')
  })

  it('filters the group and the competitors by a gap', async () => {
    const { wrapper } = await mountApp(api(), '/ai-search/readiness')
    await wrapper.get('select').setValue('bot:GPTBot')
    expect(bands(wrapper)).toEqual([
      'Tribelt today, and the target: Tribelt B.V., Mirror of tribelt.nl',
      "Tribelt's group: Lesjöfors",
      'Competitors: Twentebelt',
    ])
    expect(wrapper.text()).toContain('4 of 6 companies')
    await wrapper.get('select').setValue('')
    expect(wrapper.text()).toContain('6 of 6 companies')
  })

  it('says when the table cannot load', async () => {
    const { wrapper } = await mountApp(routes({ 'GET /api/v1/ai-readiness': () => problem(500, 'Internal Server Error', 'Down.') }), '/ai-search/readiness')
    expect(wrapper.get('[role="alert"]').text()).toContain('Down.')
  })
})
