import { describe, expect, it } from 'vitest'
import type { Plan } from '../../../infrastructure/api/types.gen'
import { details, me } from '../../../testing/fixtures'
import { type Handler, json, mountApp, routes, settle } from '../../../testing/mountApp'

function plan(done = false): Plan {
  return {
    catalogueVersion: 1,
    improvements: [
      { id: 'submit-search-engines', title: 'Submit tribelt.nl to Google and Bing', moves: 'AI answers in Gemini and Copilot', measure: 'aiShareOfVoice', auto: false, done, ...details, why: 'Gemini answers from Google’s index.', tribelt: 0.24, ...(done ? { tickedBy: 'joris', tickedAt: '2026-11-02T09:00:00Z' } : {}), leader: { competitorId: 'ashworth', name: 'Ashworth', value: 0.57 } },
      { id: 'llms-files', title: 'Add llms.txt and llms-full.txt', moves: 'AI readiness', measure: 'readinessChecks', auto: true, done: true, ...details, parts: [{ name: 'llms.txt', present: true }, { name: 'llms-full.txt', present: false }], tribelt: 3, leader: { competitorId: 'martens', name: 'Märtens', value: 5 } },
    ],
    watched: [
      { target: 'aiShareOfVoice', now: 0.24, goal: 0.4, met: false },
      { target: 'citations', goal: 20 },
      { target: 'termsUsed', now: 45, goal: 35, met: true },
    ],
    tribeltTermsFrom: 'scan',
    eyeOn: [
      { why: 'mostNamed', leader: { competitorId: 'ashworth', name: 'Ashworth', value: 0.57 } },
      { why: 'twente', leader: { competitorId: 'twentebelt', name: 'Twentebelt', value: 0.6 } },
    ],
  }
}

function api(admin = false, extra: Record<string, Handler> = {}) {
  return routes({ 'GET /api/v1/me': () => json(me({ isAdmin: admin })), 'GET /api/v1/plan': () => json(plan()), ...extra })
}

describe('the plan', () => {
  it('walks through the weekly routine, a step at a time', async () => {
    const { wrapper } = await mountApp(api(), '/plan')
    expect(document.title).toBe('The plan · Findability Explorer')
    expect(wrapper.get('h1').text()).toBe('Check every Monday. Fix one thing a week. Watch the numbers move.')
    const steps = wrapper.findAll('section[aria-labelledby="week-title"] ol button')
    expect(steps.map((s) => s.attributes('aria-pressed'))).toEqual(['true', 'false', 'false', 'false'])
    expect(wrapper.get('[aria-live="polite"] a').attributes('href')).toBe('/ai-search')
    await steps[1]?.trigger('click')
    expect(wrapper.get('[aria-live="polite"]').text()).toContain('Catching up shows where Tribelt trails the leaders.')
    expect(wrapper.get('[aria-live="polite"] a').attributes('href')).toBe('/catching-up')
  })

  it('lists the improvements in order, with what is done and who to look at', async () => {
    const { wrapper } = await mountApp(api(), '/plan')
    expect(wrapper.get('#fix-title').text()).toBe('Improvement checklist')
    const items = wrapper.findAll('ol[aria-label="Improvement checklist"] > li')
    expect(items).toHaveLength(2)
    expect(items[0]?.text()).toContain('1. Submit tribelt.nl to Google and Bing')
    expect(items[0]?.text()).toContain('Gemini answers from Google’s index.')
    expect(items[0]?.text()).toContain('Moves AI answers in Gemini and Copilot. 24% of the AI answers in the newest Prompt run name Tribelt. Ashworth leads with 57%.')
    expect(items[0]?.get('details summary').text()).toBe('How to do it')
    expect(items[0]?.findAll('details ol li').map((li) => li.text())).toEqual(['Do the first thing.', 'Do the second thing.'])
    expect(items[0]?.get('details').text()).toContain('Done whenIt is in place.')
    expect(items[1]?.text()).toContain('The latest scan found llms.txt, but not llms-full.txt.')
    expect(items[1]?.text()).toContain('tribelt.nl passes 3 of five AI readiness checks. Märtens leads with 5.')
    expect(items[0]?.text()).toContain('An administrator ticks this off.')
    expect(items[0]?.get('a').attributes('href')).toBe('/competitors/ashworth')
    expect(items[1]?.text()).toContain('The latest scan of tribelt.nl checks this.')
    expect((items[1]?.get('input').element as HTMLInputElement).checked).toBe(true)
    // A viewer ticks nothing.
    expect(items.every((i) => i.get('input').attributes('disabled') !== undefined)).toBe(true)
    expect(wrapper.text()).toContain('1 of 2 done')
  })

  it('shows the numbers to watch against their targets, and who to keep an eye on', async () => {
    const { wrapper } = await mountApp(api(), '/plan')
    const watch = wrapper.get('aside[aria-labelledby="watch-title"]')
    expect(watch.text()).toContain('AI answers in the newest Prompt run that name Tribelt24% to 40%')
    expect(watch.text()).toContain('Web answers in the newest Prompt run that cite tribelt.nlNo data yet to 20')
    // A met target says so, draws no target behind its bar, and asks to be raised.
    expect(watch.text()).toContain("Search terms from the map found on tribelt.nl's newest scanMet: 45, target was 35")
    expect(watch.findAll('span.border-r-4')).toHaveLength(2)
    expect(watch.text()).toContain('A target you have met is done. An administrator can raise it.')
    expect(watch.text()).toContain('Ashworth is named most in AI answers.')
    expect(watch.text()).toContain('Twentebelt is named most when buyers ask for Twente.')
    expect(watch.find('button').exists()).toBe(false)
  })

  it('lets an administrator tick an improvement off and set the targets', async () => {
    const sent: unknown[] = []
    const { wrapper } = await mountApp(
      api(true, {
        'PUT /api/v1/admin/plan/improvements/submit-search-engines': async (r) => {
          sent.push(await r.json())
          return json(plan(true))
        },
        'PUT /api/v1/admin/plan/targets': async (r) => {
          sent.push(await r.json())
          return json(plan(true))
        },
        'GET /api/v1/standings': () => json({ measures: [], trailing: 0, open: [] }),
      }),
      '/plan',
    )
    const box = wrapper.get('#imp-submit-search-engines')
    await box.setValue(true)
    await settle()
    expect(wrapper.text()).toContain('Ticked off by joris.')
    await wrapper.findAll('button').find((b) => b.text() === 'Change the targets')?.trigger('click')
    const inputs = wrapper.findAll('form[aria-label="Targets"] input')
    expect((inputs[0]?.element as HTMLInputElement).value).toBe('0.4')
    await inputs[1]?.setValue('25')
    await wrapper.get('form[aria-label="Targets"]').trigger('submit')
    await settle()
    expect(sent).toEqual([
      { done: true },
      {
        targets: [
          { target: 'aiShareOfVoice', value: 0.4 },
          { target: 'citations', value: 25 },
          { target: 'termsUsed', value: 35 },
        ],
      },
    ])
    expect(wrapper.find('form[aria-label="Targets"]').exists()).toBe(false)
    await wrapper.findAll('button').find((b) => b.text() === 'Change the targets')?.trigger('click')
    await wrapper.findAll('button').find((b) => b.text() === 'Cancel')?.trigger('click')
    expect(wrapper.find('form[aria-label="Targets"]').exists()).toBe(false)
  })
})
