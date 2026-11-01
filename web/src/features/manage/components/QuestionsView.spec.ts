import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { PromptSetDraft } from '../../../infrastructure/api/types.gen'
import { me } from '../../../testing/fixtures'
import { manageCounts } from '../../../testing/manageFixtures'
import { type Handler, json, mountApp, problem, routes, settle } from '../../../testing/mountApp'
import { promptHistory, promptSets, promptSettings } from '../fixtures'

function server(extra: Record<string, Handler> = {}) {
  const bodies: PromptSetDraft[] = []
  const handle = routes({
    'GET /api/v1/me': () => json(me({ isAdmin: true })),
    ...manageCounts(),
    'GET /api/v1/admin/prompt-test': () => json(promptSettings()),
    'GET /api/v1/prompt-test/prompt-sets': () => json(promptSets()),
    'GET /api/v1/prompt-test/history': () => json(promptHistory()),
    'POST /api/v1/admin/prompt-sets': async (r) => {
      bodies.push((await r.json()) as PromptSetDraft)
      return json({ version: 3, publishedAt: '2026-11-01T12:00:00Z', note: 'Adds 1 question.', questions: [] }, 201)
    },
    ...extra,
  })
  return { handle, bodies }
}

type Found = ReturnType<VueWrapper['get']>

function widget(wrapper: VueWrapper, title: string): Found {
  const heading = wrapper.findAll('section > div > h2').find((h) => h.text() === title)
  if (!heading) throw new Error(`no widget ${title}`)
  return wrapper.get(`section[aria-labelledby="${heading.attributes('id') ?? ''}"]`)
}

function button(w: Found | VueWrapper, text: string) {
  const b = w.findAll('button').find((x) => x.text().startsWith(text))
  if (!b) throw new Error(`no button ${text}`)
  return b
}

const family = (wrapper: VueWrapper, id: string) => wrapper.get(`section[aria-labelledby="family-${id}"]`)
const summary = (wrapper: VueWrapper) => widget(wrapper, 'Publish as version 3').get('[data-part="publish-summary"]').text()

describe('Questions', () => {
  it('opens one editor grouped by belt family, each question with its scope and Remove, and its text below', async () => {
    const { wrapper } = await mountApp(server().handle, '/manage/prompt-test/questions')
    expect(wrapper.get('h1').text()).toBe('Questions')
    expect(wrapper.findAll('section[aria-labelledby^="family-"] > h3').map((h) => h.text())).toEqual([
      'Eyelink 1 question',
      'Open spiral 0 questions',
      'Closed spiral 0 questions',
      'Curve-going (Tri-Flex) 0 questions',
      'Other metal belts 1 question',
    ])
    const eyelink = family(wrapper, 'eyelink')
    expect(eyelink.get('[data-part="question"]').text()).toContain('Dutch, Twente')
    expect((eyelink.get('textarea').element as HTMLTextAreaElement).value).toBe('Wie levert draadogenbanden in Twente?')
    // No English line under the questions.
    expect(eyelink.text()).not.toContain('Who supplies eyelink belts in Twente?')
    expect(summary(wrapper)).toBe('No changes to publish yet. Edit, add, or remove a question in the draft, and this card says what would change.')
    expect(button(widget(wrapper, 'Publish as version 3'), 'Publish as version 3').attributes('disabled')).toBeDefined()
    expect(wrapper.get('details summary').text()).toBe('The published set, version 2, 2 questions')
  })

  it('changes only the draft when a question is edited or removed, and Discard the draft resets it', async () => {
    const { handle, bodies } = server()
    const { wrapper } = await mountApp(handle, '/manage/prompt-test/questions')
    await family(wrapper, 'eyelink').get('textarea').setValue('Wie levert oogbanden in Twente?')
    expect(summary(wrapper)).toContain('Adds no questions, changes 1, and removes none. A run will cost about the same as now.')
    await button(family(wrapper, 'other'), 'Remove').trigger('click')
    expect(family(wrapper, 'other').text()).toContain('No questions for this belt family yet.')
    expect(summary(wrapper)).toContain('changes 1, and removes 1. A run will cost about $0.13 less, or $1.50 less a year on a monthly schedule.')
    expect(bodies).toEqual([])
    await button(widget(wrapper, 'Publish as version 3'), 'Discard the draft').trigger('click')
    expect(wrapper.text()).toContain('The draft is reset to the published set.')
    expect((family(wrapper, 'eyelink').get('textarea').element as HTMLTextAreaElement).value).toBe('Wie levert draadogenbanden in Twente?')
    expect(summary(wrapper)).toContain('No changes to publish yet.')
  })

  it('adds a question in a new scope from the add row, offers one for every other family, and publishes the draft', async () => {
    const { handle, bodies } = server()
    const { wrapper } = await mountApp(handle, '/manage/prompt-test/questions')
    const eyelink = family(wrapper, 'eyelink')
    await button(eyelink, '+ Add a question').trigger('click')
    const form = eyelink.get('form')
    // It starts on the scope the family lacks.
    expect((form.get('select').element as HTMLSelectElement).value).toBe('english')
    expect(form.findAll('select').map((s) => s.findAll('option').at(-1)?.text())).toEqual(['A new scope…'])
    await form.get('select').setValue('__new')
    const [, language] = form.findAll('select')
    await language?.setValue('fr')
    await form.get('input').setValue('Lyon')
    await form.get('textarea').setValue('Quel fabricant de bandes à œillets recommandez-vous ?')
    await form.trigger('submit')
    expect(wrapper.get('[data-part="draft-notice"]').text()).toBe('French, Lyon is a new scope, and the question is added to the draft.')
    expect(family(wrapper, 'eyelink').text()).toContain('French, LyonNew')
    expect(summary(wrapper)).toContain('Adds 1 question (1 in French), changes none, and removes none. Adds 1 scope (French, Lyon).')
    expect(summary(wrapper)).toContain('A run will cost about $0.13 more, or $1.50 more a year on a monthly schedule.')

    await button(wrapper.get('[data-part="scope-offer"]'), 'Add one for every other belt family').trigger('click')
    expect(wrapper.text()).toContain('4 questions added to the draft for French, Lyon. Translate each one before you publish.')
    expect(family(wrapper, 'open_spiral').text()).toContain('French, Lyon')

    await button(widget(wrapper, 'Publish as version 3'), 'Publish as version 3').trigger('click')
    await settle()
    expect(bodies).toHaveLength(1)
    expect(bodies[0]?.scopes).toEqual([{ id: 'fr-lyon', name: 'French, Lyon', language: 'fr', region: 'Lyon' }])
    expect(bodies[0]?.questions.find((q) => q.scope === 'fr-lyon' && q.family === 'eyelink')).toEqual({
      family: 'eyelink',
      intent: 'product',
      scope: 'fr-lyon',
      text: 'Quel fabricant de bandes à œillets recommandez-vous ?',
    })
    expect(bodies[0]?.questions).toHaveLength(7)
    expect(wrapper.text()).toContain('Version 3 is published. The charts mark the date the questions changed.')
  })

  it('adds a question in a known scope, asks for its text first, and Cancel closes the row', async () => {
    const { wrapper } = await mountApp(server().handle, '/manage/prompt-test/questions')
    const other = family(wrapper, 'other')
    await button(other, '+ Add a question').trigger('click')
    expect((other.get('form select').element as HTMLSelectElement).value).toBe('twente')
    await other.get('form').trigger('submit')
    expect(other.get('[role="alert"]').text()).toBe('Write the question first.')
    await button(other, 'Cancel').trigger('click')
    expect(other.find('form').exists()).toBe(false)
    await button(other, '+ Add a question').trigger('click')
    await other.get('form textarea').setValue('Waarom scheurt onze band?')
    await other.get('form').trigger('submit')
    expect(wrapper.get('[data-part="draft-notice"]').text()).toBe('The question is added to the draft.')
    expect(wrapper.find('[data-part="scope-offer"]').exists()).toBe(false)
    expect(family(wrapper, 'other').findAll('[data-part="question"]')).toHaveLength(2)
  })

  it('restores a version into the draft, and says what the server refused', async () => {
    const { wrapper } = await mountApp(
      server({ 'POST /api/v1/admin/prompt-sets': () => problem(409, 'Conflict', 'Another publication took this version number first. Publish again.') }).handle,
      '/manage/prompt-test/questions',
    )
    const history = widget(wrapper, 'Version history')
    expect(history.findAll('details summary').map((s) => s.text())).toEqual([
      'Version 2 PublishedPublished on 27 October 2026, 2 questions',
      'Version 1 Published on 20 October 2026, 1 question',
    ])
    await button(history, 'Restore version 1 into the draft').trigger('click')
    expect(wrapper.text()).toContain('Version 1 is copied into the draft. Publish the draft to make a new version.')
    expect(summary(wrapper)).toContain('Restores version 1. Adds no questions, changes none, and removes 1.')
    await button(widget(wrapper, 'Publish as version 3'), 'Publish as version 3').trigger('click')
    await settle()
    expect(widget(wrapper, 'Publish as version 3').get('[role="alert"]').text()).toBe('Another publication took this version number first. Publish again.')
  })

  it('says what the server refused when the questions cannot be read', async () => {
    const { wrapper } = await mountApp(server({ 'GET /api/v1/prompt-test/prompt-sets': () => problem(500, 'Internal Server Error', 'Down.') }).handle, '/manage/prompt-test/questions')
    expect(wrapper.get('[role="alert"]').text()).toBe('Down.')
  })
})
