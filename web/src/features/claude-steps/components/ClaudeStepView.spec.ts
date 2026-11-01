import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { ClaudeStep } from '../../../infrastructure/api/types.gen'
import { claudeStep, stepVersion } from '../../../testing/claudeStepFixtures'
import { buttonNamed, manageApi } from '../../../testing/manageApi'
import { json, mountApp, problem, settle } from '../../../testing/mountApp'

const widget = (wrapper: VueWrapper, title: string) => {
  const found = wrapper.findAll('section').find((s) => s.find('h2').exists() && s.get('h2').text() === title)
  if (!found) throw new Error(`no widget ${title}`)
  return found
}

function server(initial: ClaudeStep = claudeStep()) {
  let current = initial
  const bodies: unknown[] = []
  const handle = manageApi({
    'GET /api/v1/admin/claude-steps/matching': () => json(current),
    'GET /api/v1/admin/claude-steps': () =>
      json({ items: [{ id: current.id, title: current.title, fixedInstructions: current.fixedInstructions, current: current.current, jobCostUsd: current.jobCostUsd }] }),
    'POST /api/v1/admin/claude-steps/matching/versions': async (r) => {
      const body = (await r.json()) as { model: string; instructions: string; restoredFrom?: number }
      bodies.push(body)
      const next = stepVersion({ number: current.current.number + 1, model: body.model, instructions: body.instructions, note: body.restoredFrom ? `Restored from version ${String(body.restoredFrom)}.` : '' })
      current = { ...current, current: next, versions: [...current.versions, next] }
      return json(current, 201)
    },
  })
  return { handle, bodies }
}

describe('a Claude step', () => {
  it('shows its instructions, model, what it gets and replies, and its versions, under the section it works for', async () => {
    const { wrapper } = await mountApp(server().handle, '/manage/claude-steps/matching')
    expect(wrapper.get('h1').text()).toBe('Matching products')
    expect(wrapper.get('[data-part="data-line"]').text()).toBe('Version 2, saved 1 November 2026 by Joris. Paid by the Anthropic key.')
    expect(wrapper.get('nav[aria-label="Manage"] a[aria-current="page"]').text()).toBe('Matching products')
    expect(wrapper.findAll('a').find((a) => a.text().includes('Back to Website scans'))?.attributes('href')).toBe('/manage/scans')
    expect(wrapper.findAll('section > div > h2').map((h) => h.text())).toEqual(['Instructions', 'Model', 'What the model gets', 'What the model replies', 'Version history'])
    expect(wrapper.get<HTMLTextAreaElement>('textarea').element.value).toBe('Match the metal belts.')
    expect(widget(wrapper, 'Model').findAll('option').map((o) => o.text())).toEqual(['Claude Haiku 5.5 (default)', 'Claude Sonnet 5.5', 'Claude Opus 5.5'])
    expect(wrapper.get('[data-part="cost"]').text()).toBe('About $0.088 for a scan.')
    expect(widget(wrapper, 'What the model gets').text()).toContain('From the scan')
    expect(widget(wrapper, 'What the model replies').text()).toContain('searchTerms')
    expect(wrapper.get('ul[aria-label="Versions"]').findAll('li').map((li) => li.find('span span').text())).toEqual([
      'Version 2, 1 November 2026, by JorisIn use',
      'Version 1, 20 October 2026, by the tool',
    ])
    expect(buttonNamed(wrapper, 'Save as a new version').attributes('disabled')).toBeDefined()
  })

  it('saves edited instructions and a cheaper model as a new version, and says what it costs against now', async () => {
    const { handle, bodies } = server()
    const { wrapper } = await mountApp(handle, '/manage/claude-steps/matching')
    await wrapper.get('textarea').setValue('Match only metal belts.')
    await widget(wrapper, 'Model').get('select').setValue('claude-haiku-5-5')
    expect(wrapper.get('[data-part="cost"]').text()).toBe('About $0.0044 for a scan, $0.0836 less than now.')
    await buttonNamed(wrapper, 'Save as a new version').trigger('click')
    await settle()
    expect(bodies).toEqual([{ model: 'claude-haiku-5-5', instructions: 'Match only metal belts.' }])
    expect(wrapper.text()).toContain('Version 3 is saved. Scans from now on match the new way.')
    expect(wrapper.get<HTMLTextAreaElement>('textarea').element.value).toBe('Match only metal belts.')
    expect(wrapper.get('[data-part="data-line"]').text()).toContain('Version 3')
  })

  it('undoes changes, goes back to the original, and restores a version as a new one', async () => {
    const { handle, bodies } = server()
    const { wrapper } = await mountApp(handle, '/manage/claude-steps/matching')
    const box = () => wrapper.get<HTMLTextAreaElement>('textarea').element.value
    await wrapper.get('textarea').setValue('Something else.')
    await buttonNamed(wrapper, 'Undo the changes').trigger('click')
    expect(box()).toBe('Match the metal belts.')
    await buttonNamed(wrapper, 'Back to the original').trigger('click')
    expect(box()).toBe('Compare the site with Tribelt.')
    expect(wrapper.text()).toContain('The original instructions and model are in the box. Save to use them.')
    await buttonNamed(wrapper, 'Undo the changes').trigger('click')
    await buttonNamed(wrapper, 'Restore version 1').trigger('click')
    expect(box()).toBe('Compare the site with Tribelt.')
    expect((widget(wrapper, 'Model').get('select').element as HTMLSelectElement).value).toBe('claude-haiku-5-5')
    await buttonNamed(wrapper, 'Save as a new version').trigger('click')
    await settle()
    expect(bodies).toEqual([{ model: 'claude-haiku-5-5', instructions: 'Compare the site with Tribelt.', restoredFrom: 1 }])
    expect(wrapper.text()).toContain('Restored from version 1.')
  })

  it('keeps a version’s fold open while the page redraws', async () => {
    const { wrapper, queryClient } = await mountApp(server().handle, '/manage/claude-steps/matching')
    const fold = () => wrapper.get('ul[aria-label="Versions"] details').element as HTMLDetailsElement
    fold().open = true
    await queryClient.invalidateQueries()
    await settle()
    expect(fold().open).toBe(true)
  })

  it('shows instructions set in the code read-only, with only the model to save', async () => {
    const { handle, bodies } = server(claudeStep({ fixedInstructions: true }))
    const { wrapper } = await mountApp(handle, '/manage/claude-steps/matching')
    expect(wrapper.find('textarea').exists()).toBe(false)
    expect(widget(wrapper, 'Instructions').text()).toContain('Set in the code')
    expect(wrapper.text()).toContain('Model history')
    await widget(wrapper, 'Model').get('select').setValue('claude-opus-5-5')
    await buttonNamed(wrapper, 'Save the model').trigger('click')
    await settle()
    expect(bodies).toEqual([{ model: 'claude-opus-5-5', instructions: 'Match the metal belts.' }])
  })

  it('says why a step could not be read or saved', async () => {
    const broken = await mountApp(manageApi({ 'GET /api/v1/admin/claude-steps/matching': () => problem(500, 'Internal Server Error') }), '/manage/claude-steps/matching')
    expect(broken.wrapper.get('[role="alert"]').text()).toBe('Internal Server Error')
    broken.wrapper.unmount()
    const { wrapper } = await mountApp(
      manageApi({
        'GET /api/v1/admin/claude-steps/matching': () => json(claudeStep()),
        'POST /api/v1/admin/claude-steps/matching/versions': () => problem(422, 'Unprocessable Entity', 'The instructions are empty.'),
      }),
      '/manage/claude-steps/matching',
    )
    await wrapper.get('textarea').setValue(' ')
    await buttonNamed(wrapper, 'Save as a new version').trigger('click')
    await settle()
    expect(wrapper.get('[role="alert"]').text()).toBe('The instructions are empty.')
  })
})

describe('the Claude models and instructions', () => {
  it('lists every step with its model, version, and the cost of one job, linked to its page', async () => {
    const { wrapper } = await mountApp(server().handle, '/manage/claude-steps')
    expect(wrapper.get('h1').text()).toBe('Claude models and instructions')
    expect(wrapper.get('nav[aria-label="Manage"] a[aria-current="page"]').text()).toBe('Models and instructions')
    const rows = wrapper.findAll('table[aria-label="Claude steps"] tbody tr').map((r) => r.findAll('td').map((td) => td.text()))
    expect(rows).toEqual([
      [
        'Matching productsReads the pages of a scan and matches the products on them to Tribelt’s five belt families.',
        'Website scans',
        'Claude Sonnet 5.5',
        'Version 21 November 2026',
        '$0.088a scan',
      ],
    ])
    expect(wrapper.get('table a').attributes('href')).toBe('/manage/claude-steps/matching')
  })

  it('says when the steps could not be read, and marks instructions set in the code', async () => {
    const broken = await mountApp(manageApi({ 'GET /api/v1/admin/claude-steps': () => problem(500, 'Internal Server Error') }), '/manage/claude-steps')
    expect(broken.wrapper.get('[role="alert"]').text()).toBe('Internal Server Error')
    broken.wrapper.unmount()
    const { wrapper } = await mountApp(server(claudeStep({ fixedInstructions: true })).handle, '/manage/claude-steps')
    expect(wrapper.get('table tbody').text()).toContain('Set in the code')
  })
})

describe('the step where it works', () => {
  it('shows a card on Website scans and a widget on Scan settings, each linked to the step', async () => {
    const scans = await mountApp(server().handle, '/manage/scans')
    expect(widget(scans.wrapper, 'Matching products').text()).toContain('Done by Claude Sonnet 5.5.')
    expect(widget(scans.wrapper, 'Matching products').get('a').attributes('href')).toBe('/manage/claude-steps/matching')
    // A scan's estimate is the step's job with its model now.
    expect(widget(scans.wrapper, 'Scan tribelt.nl').text()).toContain('About $0.088.')
    scans.wrapper.unmount()
    const settings = await mountApp(server().handle, '/manage/scans/settings')
    expect(widget(settings.wrapper, 'Model and instructions').text()).toContain('Claude Sonnet 5.5 does this step, with instructions version 2. About $0.088 for a scan.')
  })
})
