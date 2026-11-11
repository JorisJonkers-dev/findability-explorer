import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { definition } from '../../../testing/definition'
import { json, problem, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { useCompetitorDefinition } from './useCompetitorDefinition'

const saved = definition({
  current: { version: 4, text: 'Makers of metal belts compete.', savedAt: '2026-11-01T10:00:00Z', savedBy: 'Joris' },
  outdatedCompetitors: 14,
})

function server(bodies: unknown[] = [], over: Record<string, () => Response> = {}) {
  const record = (answer: () => Response) => async (request: Request) => {
    const text = await request.text()
    bodies.push(text === '' ? null : JSON.parse(text))
    return answer()
  }
  return routes({
    'GET /api/v1/admin/competitor-definition': () => json(definition()),
    'PUT /api/v1/admin/competitor-definition': record(() => json(saved)),
    'POST /api/v1/admin/competitor-definition/versions/2/restore': record(() => json(saved)),
    'POST /api/v1/admin/competitor-candidates/review': () => json({ running: true }, 202),
    'POST /api/v1/admin/competitor-checks': () => json({ running: true }, 202),
    ...over,
  })
}

describe('useCompetitorDefinition', () => {
  it('asks nothing for a viewer', async () => {
    const { state, sent } = mountComposable(() => useCompetitorDefinition(ref(false)), server())
    await flushPromises()
    expect(sent).toEqual([])
    expect(state.definition.value).toBeUndefined()
    expect(state.draft.value).toBe('')
  })

  it('reads the definition into the text box, and saves an edit as a new version', async () => {
    const bodies: unknown[] = []
    const { state, sent } = mountComposable(() => useCompetitorDefinition(ref(true)), server(bodies))
    await flushPromises()
    expect(state.draft.value).toBe('Only makers of eyelink belts compete.')
    expect(state.dirty.value).toBe(false)
    expect(state.error.value).toBeNull()
    state.draft.value = 'Makers of metal belts compete.'
    expect(state.dirty.value).toBe(true)
    state.save()
    await flushPromises()
    expect(sent).toContain('PUT /api/v1/admin/competitor-definition')
    expect(bodies).toEqual([{ text: 'Makers of metal belts compete.' }])
    expect(state.notice.value).toBe('The definition is saved as version 4. Nothing was rechecked yet.')
    expect(state.definition.value?.current.version).toBe(4)
    expect(state.dirty.value).toBe(false)
  })

  it('refuses an empty text, and says when nothing changed', async () => {
    const { state, sent } = mountComposable(
      () => useCompetitorDefinition(ref(true)),
      server([], { 'PUT /api/v1/admin/competitor-definition': () => problem(409, 'Conflict', 'The competitor definition in use has this text already.') }),
    )
    await flushPromises()
    state.draft.value = '  '
    state.save()
    expect(state.error.value).toBe('The definition cannot be empty.')
    state.draft.value = 'Only makers of eyelink belts compete. '
    state.save()
    await flushPromises()
    expect(sent.filter((s) => s.startsWith('PUT'))).toHaveLength(1)
    expect(state.error.value).toBe('Nothing changed, so no new version was saved.')
  })

  it('copies a version into the text box, and saving it restores that version', async () => {
    const bodies: unknown[] = []
    const { state, sent } = mountComposable(() => useCompetitorDefinition(ref(true)), server(bodies))
    await flushPromises()
    state.restore(2)
    expect(state.draft.value).toBe('A competitor makes metal belts.')
    expect(state.notice.value).toBe('Version 2 is copied into the text box. Save it to make a new version.')
    state.save()
    await flushPromises()
    expect(sent).toContain('POST /api/v1/admin/competitor-definition/versions/2/restore')
    expect(bodies).toEqual([null])
    expect(state.notice.value).toBe('The definition is saved as version 4. Nothing was rechecked yet.')
    // A version no longer copied in, once edited, is saved as a new text.
    state.restore(2)
    state.draft.value = 'A competitor makes metal belts, also as spares.'
    state.save()
    await flushPromises()
    expect(bodies.at(-1)).toEqual({ text: 'A competitor makes metal belts, also as spares.' })
    state.restore(9) // no such version: nothing changes
    expect(state.draft.value).toBe('Makers of metal belts compete.')
  })

  it('checks the candidates and the list again', async () => {
    const { state, sent } = mountComposable(() => useCompetitorDefinition(ref(true)), server())
    await flushPromises()
    state.checkAgain()
    await flushPromises()
    expect(sent).toContain('POST /api/v1/admin/competitor-candidates/review')
    expect(sent).toContain('POST /api/v1/admin/competitor-checks')
    expect(state.notice.value).toBe('Claude Haiku is checking the candidates and the list again.')
  })

  it('says why a check could not start', async () => {
    const { state } = mountComposable(
      () => useCompetitorDefinition(ref(true)),
      server([], { 'POST /api/v1/admin/competitor-checks': () => problem(403, 'Forbidden') }),
    )
    await flushPromises()
    state.checkAgain()
    await flushPromises()
    expect(state.error.value).toBe('Only administrators can change the competitor definition.')
  })

  it('says while it saves or starts a check, and why the definition could not be read', async () => {
    const { state } = mountComposable(() => useCompetitorDefinition(ref(true)), server())
    await flushPromises()
    state.draft.value = 'Makers of metal belts compete.'
    state.save()
    expect(state.isSaving.value).toBe(true)
    state.checkAgain()
    expect(state.isChecking.value).toBe(true)
    await flushPromises()
    expect(state.isSaving.value).toBe(false)
    expect(state.isChecking.value).toBe(false)
    const failing = mountComposable(
      () => useCompetitorDefinition(ref(true)),
      server([], { 'GET /api/v1/admin/competitor-definition': () => problem(403, 'Forbidden') }),
    )
    await flushPromises()
    expect(failing.state.error.value).toBe('Only administrators can change the competitor definition.')
  })
})
