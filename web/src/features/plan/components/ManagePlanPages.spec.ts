import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { Plan, PlanImprovement } from '../../../infrastructure/api/types.gen'
import { details } from '../../../testing/fixtures'
import { buttonNamed, manageApi } from '../../../testing/manageApi'
import { recommendationRuns } from '../../../testing/recommendationRuns'
import { type Handler, json, mountApp, problem, settle } from '../../../testing/mountApp'

const item = (id: string, title: string, auto: boolean, done: boolean): PlanImprovement => ({ ...details, id, title, moves: 'citations', measure: 'citationShare', auto, done })

function planOf(items: PlanImprovement[]): Plan {
  return {
    catalogueVersion: 1,
    improvements: items,
    watched: [
      { target: 'aiShareOfVoice', now: 0.2, goal: 0.25, max: 1 },
      { target: 'twenteShareOfVoice', goal: 0.4, max: 1 },
      { target: 'citationShare', goal: 0.1, max: 1 },
      { target: 'readinessChecks', goal: 6 },
      { target: 'termsUsed', goal: 40, max: 663 },
    ],
    eyeOn: [],
    readinessChecks: 8,
  }
}

function server(extra: Record<string, Handler> = {}) {
  let items = [item('listings', 'Get listed', false, false), item('sitemap', 'Submit the sitemap', true, false), item('about', 'Fix the About page', false, true)]
  const sent: unknown[] = []
  const handle = manageApi({
    'GET /api/v1/plan': () => json(planOf(items)),
    'GET /api/v1/admin/recommendation-runs': () => json(recommendationRuns()),
    'PUT /api/v1/admin/plan/improvements/listings': async (r) => {
      const body = (await r.json()) as { done: boolean }
      sent.push(body)
      items = items.map((i) => (i.id === 'listings' ? { ...i, done: body.done } : i))
      return json(planOf(items))
    },
    'PUT /api/v1/admin/plan/targets': async (r) => {
      sent.push(await r.json())
      return json(planOf(items))
    },
    ...extra,
  })
  return { handle, sent }
}
const widget = (wrapper: VueWrapper, title: string) => {
  const found = wrapper.findAll('section').find((s) => s.find('h2').exists() && s.get('h2').text() === title)
  if (!found) throw new Error(`no widget ${title}`)
  return found
}

describe('Recommendations in Manage', () => {
  it('opens on the open recommendations and a card each for ticking off and the goals', async () => {
    const { wrapper } = await mountApp(server().handle, '/manage/recommendations')
    expect(wrapper.get('h1').text()).toBe('Recommendations')
    expect(wrapper.get('header a[href="/recommendations"]').text()).toBe('See the recommendations')
    expect(widget(wrapper, 'Open recommendations').text()).toContain('21 done')
    expect(widget(wrapper, 'Tick off yourself').text()).toContain('1waiting')
    expect(widget(wrapper, 'Goals').get('a').attributes('href')).toBe('/manage/recommendations/goals')
  })

  it('offers a Recommendation run with what it reads, replaces, and costs, and lists the past runs', async () => {
    const { wrapper } = await mountApp(server().handle, '/manage/recommendations')
    const stale = widget(wrapper, 'The page text is older than the responses')
    expect(stale.text()).toContain('tribelt.nl was last scanned before the newest Prompt run.')
    expect(stale.get('a').attributes('href')).toBe('/manage/scans')
    const start = widget(wrapper, 'Start a Recommendation run')
    expect(start.text()).toContain('Reads the newest Prompt run (1 November 2026)')
    expect(start.text()).toContain('This replaces 12 open recommendations. Those you marked as done or edited stay.')
    expect(start.text()).toContain('40 gaps to write, about $0.07 from the Anthropic key.')
    expect(start.find('a[href="/manage/claude-steps/recommendations"]').exists()).toBe(true)
    const rows = widget(wrapper, 'Past runs').findAll('tbody tr').map((r) => r.findAll('td').map((td) => td.text()))
    expect(rows[0]).toEqual(['1 November 2026, 08:00', 'Done', 'claude-haiku-5-5', '38', '10', '2', '$0.06'])
    expect(rows[1]?.[1]).toBe('Stopped by hand')
  })

  it('starts a run, and stops the one that runs', async () => {
    const sent: string[] = []
    let running = false
    const { wrapper } = await mountApp(
      server({
        'GET /api/v1/admin/recommendation-runs': () => json(recommendationRuns({ running, stale: false, runs: [] })),
        'POST /api/v1/admin/recommendation-runs': () => {
          sent.push('start')
          running = true
          return json(recommendationRuns({ running: true, stale: false, runs: [] }), 202)
        },
        'POST /api/v1/admin/recommendation-runs/stop': () => {
          sent.push('stop')
          return new Response(null, { status: 204 })
        },
      }).handle,
      '/manage/recommendations',
    )
    expect(wrapper.findAll('h2').some((h) => h.text() === 'The page text is older than the responses')).toBe(false)
    expect(widget(wrapper, 'Past runs').text()).toContain('No Recommendation run has run yet.')
    await buttonNamed(wrapper, 'Start a Recommendation run').trigger('click')
    await settle()
    expect(wrapper.text()).toContain('The Recommendation run started.')
    await buttonNamed(wrapper, 'Stop the Recommendation run').trigger('click')
    await settle()
    expect(sent).toEqual(['start', 'stop'])
    expect(wrapper.text()).toContain('The run stops after the batch it writes.')
  })

  it('asks for an Anthropic key when there is none', async () => {
    const { wrapper } = await mountApp(server({ 'GET /api/v1/admin/recommendation-runs': () => json(recommendationRuns({ noKey: true, model: undefined })) }).handle, '/manage/recommendations')
    const start = widget(wrapper, 'Start a Recommendation run')
    expect(start.get('a[href="/manage/keys/anthropic"]').text()).toBe('Add an Anthropic key')
    expect(start.findAll('button').length).toBe(0)
  })
})

describe('tick off yourself', () => {
  it('marks one as done, and undoes it from the Done list', async () => {
    const { handle, sent } = server()
    const { wrapper } = await mountApp(handle, '/manage/recommendations/tick-off')
    expect(wrapper.get('h1').text()).toBe('Tick off yourself')
    expect(widget(wrapper, 'Waiting').findAll('li').map((li) => li.get('a').text())).toEqual(['Get listed'])
    expect(widget(wrapper, 'Done').findAll('li').map((li) => li.get('a').text())).toEqual(['Fix the About page'])
    await buttonNamed(wrapper, 'Mark as done: Get listed').trigger('click')
    await settle()
    expect(widget(wrapper, 'Waiting').text()).toContain('Nothing waits to be ticked off.')
    expect(wrapper.text()).toContain('“Get listed” is marked as done. Undo opens it again.')
    await buttonNamed(wrapper, 'Undo: Get listed').trigger('click')
    await settle()
    expect(wrapper.text()).toContain('“Get listed” is open again.')
    expect(sent).toEqual([{ done: true }, { done: false }])
  })

  it('says when nothing is done yet, and when the plan could not be read', async () => {
    const empty = await mountApp(manageApi({ 'GET /api/v1/plan': () => json(planOf([])) }), '/manage/recommendations/tick-off')
    expect(widget(empty.wrapper, 'Done').text()).toContain('Nothing is marked as done yet.')
    empty.wrapper.unmount()
    const broken = await mountApp(manageApi({ 'GET /api/v1/plan': () => problem(500, 'Internal Server Error') }), '/manage/recommendations/tick-off')
    expect(broken.wrapper.get('[role="alert"]').text()).toBe('Internal Server Error')
  })
})

describe('the goals', () => {
  it('shows the five, shares in percent, and saves them as fractions', async () => {
    const { handle, sent } = server()
    const { wrapper } = await mountApp(handle, '/manage/recommendations/goals')
    expect(wrapper.get('h1').text()).toBe('Goals')
    const inputs = wrapper.findAll<HTMLInputElement>('form[aria-label="The five goals"] input')
    expect(wrapper.findAll('form[aria-label="The five goals"] label').map((l) => l.text())).toEqual([
      'AI responses that name Tribelt (percent)',
      'AI responses about the Twente region that name Tribelt (percent)',
      'Web responses that cite tribelt.nl (percent)',
      'AI readiness checks passed (0 to 8)',
      'Search terms tribelt.nl uses',
    ])
    expect(inputs.map((i) => i.element.value)).toEqual(['25', '40', '10', '6', '40'])
    await inputs[0]?.setValue('30')
    await wrapper.get('form[aria-label="The five goals"]').trigger('submit')
    await settle()
    expect(sent).toEqual([
      {
        targets: [
          { target: 'aiShareOfVoice', value: 0.3 },
          { target: 'twenteShareOfVoice', value: 0.4 },
          { target: 'citationShare', value: 0.1 },
          { target: 'readinessChecks', value: 6 },
          { target: 'termsUsed', value: 40 },
        ],
      },
    ])
    expect(wrapper.text()).toContain('The goals are saved.')
  })

  it('refuses a value out of range, and says why a save failed', async () => {
    const { wrapper } = await mountApp(server({ 'PUT /api/v1/admin/plan/targets': () => problem(422, 'Unprocessable', 'A goal is out of range.') }).handle, '/manage/recommendations/goals')
    const inputs = wrapper.findAll<HTMLInputElement>('form[aria-label="The five goals"] input')
    await inputs[3]?.setValue('9')
    await wrapper.get('form[aria-label="The five goals"]').trigger('submit')
    expect(wrapper.get('[role="alert"]').text()).toBe('Type a number from 0 to 8 for AI readiness checks passed.')
    await inputs[3]?.setValue('7')
    await wrapper.get('form[aria-label="The five goals"]').trigger('submit')
    await settle()
    expect(wrapper.get('[role="alert"]').text()).toBe('A goal is out of range.')
    expect(wrapper.text()).not.toContain('The goals are saved.')
  })
})
