import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { definition } from '../../../testing/definition'
import { buttonNamed, manageApi } from '../../../testing/manageApi'
import { type Handler, json, mountApp, settle } from '../../../testing/mountApp'

type Sent = { method: string; path: string; body: unknown }
const saved = definition({ current: { version: 4, text: 'Makers of metal belts compete.', savedAt: '2026-11-01T10:00:00Z', savedBy: 'Joris' } })

function api(sent: Sent[] = [], d = definition()) {
  const record = (answer: Handler): Handler => async (request) => {
    const text = await request.text()
    sent.push({ method: request.method, path: new URL(request.url).pathname, body: text === '' ? null : JSON.parse(text) })
    return answer(request)
  }
  return manageApi({
    'GET /api/v1/admin/competitor-definition': () => json(d),
    'PUT /api/v1/admin/competitor-definition': record(() => json(saved)),
    'POST /api/v1/admin/competitor-definition/versions/2/restore': record(() => json(saved)),
    'POST /api/v1/admin/competitor-candidates/review': record(() => json({ running: true }, 202)),
    'POST /api/v1/admin/competitor-checks': record(() => json({ running: true }, 202)),
  })
}
const widget = (wrapper: VueWrapper, title: string) => {
  const found = wrapper.findAll('section').find((s) => s.find('h2').exists() && s.get('h2').text() === title)
  if (!found) throw new Error(`no widget ${title}`)
  return found
}
const hasWidget = (wrapper: VueWrapper, title: string) => wrapper.findAll('h2').some((h) => h.text() === title)

describe('the competitor definition', () => {
  it('shows the text in use, what checking again would take, and every version', async () => {
    const { wrapper } = await mountApp(api(), '/manage/competitors/definition')
    expect(wrapper.get('h1').text()).toBe('Competitor definition')
    expect(wrapper.get('header a[href="/competitors"]').text()).toBe('See the competitors')
    const box = widget(wrapper, 'The definition')
    expect(box.get('textarea').element.value).toBe('Only makers of eyelink belts compete.')
    expect(box.text()).toContain('Version 3, saved 1 November 2026. Saving makes a new version and does not start a check.')
    const again = widget(wrapper, 'Check again')
    expect(again.attributes('data-span')).toBe('12')
    expect(again.text()).toContain('12 competitors and 40 candidates were checked against an older version of the definition.')
    expect(again.text()).toContain('52 checks, about $0.26 from the Anthropic key.')
    expect(widget(wrapper, 'About the definition').text()).toContain('Some checks used an older version')
    const rows = widget(wrapper, 'Versions').findAll('li').map((li) => li.text())
    expect(rows[0]).toContain('Version 3')
    expect(rows[0]).toContain('In use')
    expect(rows[1]).toContain('Version 2')
    expect(rows[1]).toContain('In the code until 27 October 2026.')
    expect(rows[1]).toContain('Restore version 2')
  })

  it('saves an edit as a new version, and starts no check', async () => {
    const sent: Sent[] = []
    const { wrapper } = await mountApp(api(sent), '/manage/competitors/definition')
    await widget(wrapper, 'The definition').get('textarea').setValue('Makers of metal belts compete.')
    expect(widget(wrapper, 'The definition').text()).toContain('You have unsaved edits.')
    await buttonNamed(wrapper, 'Save as a new version').trigger('click')
    await settle()
    expect(sent).toEqual([{ method: 'PUT', path: '/api/v1/admin/competitor-definition', body: { text: 'Makers of metal belts compete.' } }])
    expect(wrapper.text()).toContain('The definition is saved as version 4. Nothing was rechecked yet.')
  })

  it('copies a version into the box, and saving it restores that version', async () => {
    const sent: Sent[] = []
    const { wrapper } = await mountApp(api(sent), '/manage/competitors/definition')
    await buttonNamed(wrapper, 'Restore version 2').trigger('click')
    expect(widget(wrapper, 'The definition').get('textarea').element.value).toBe('A competitor makes metal belts.')
    expect(wrapper.text()).toContain('Version 2 is copied into the text box. Save it to make a new version.')
    await buttonNamed(wrapper, 'Save as a new version').trigger('click')
    await settle()
    expect(sent).toEqual([{ method: 'POST', path: '/api/v1/admin/competitor-definition/versions/2/restore', body: null }])
  })

  it('checks the candidates and the list again', async () => {
    const sent: Sent[] = []
    const { wrapper } = await mountApp(api(sent), '/manage/competitors/definition')
    await buttonNamed(widget(wrapper, 'Check again'), 'Check again').trigger('click')
    await settle()
    expect(sent.map((s) => s.path).sort()).toEqual(['/api/v1/admin/competitor-candidates/review', '/api/v1/admin/competitor-checks'])
    expect(wrapper.text()).toContain('Claude Haiku is checking the candidates and the list again.')
  })

  it('offers no check again when every check used this version', async () => {
    const { wrapper } = await mountApp(api([], definition({ outdatedCompetitors: 0, outdatedCandidates: 0 })), '/manage/competitors/definition')
    expect(hasWidget(wrapper, 'Check again')).toBe(false)
    expect(widget(wrapper, 'About the definition').text()).toContain('Every check used this version')
  })
})

describe('the Competitor list', () => {
  it('has a card for the definition, and says when it changed since the checks', async () => {
    const { wrapper } = await mountApp(api(), '/manage/competitors')
    expect(widget(wrapper, 'Competitor definition').get('a').attributes('href')).toBe('/manage/competitors/definition')
    const changed = widget(wrapper, 'The definition changed')
    expect(changed.text()).toContain('The list and the candidates were checked against an older version of the definition.')
    expect(changed.get('a').attributes('href')).toBe('/manage/competitors/definition')
  })

  it('says nothing changed when every check used the definition in use', async () => {
    const { wrapper } = await mountApp(api([], definition({ outdatedCompetitors: 0, outdatedCandidates: 0 })), '/manage/competitors')
    expect(hasWidget(wrapper, 'The definition changed')).toBe(false)
  })
})
