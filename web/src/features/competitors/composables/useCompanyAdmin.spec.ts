import { flushPromises } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { competitor, summary } from '../../../testing/fixtures'
import { json, problem } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { useCompany } from './useCompany'
import { useAddCompanies, useAddCompany, useRemoveCompany, useRescan, useUpdateCompany } from './useCompanyAdmin'

describe('useRemoveCompany', () => {
  it('asks for the other companies again, but not for the one it removed', async () => {
    const { state, sent } = mountComposable(
      () => ({ gone: useCompany(ref('gone')), kept: useCompany(ref('kept')), remove: useRemoveCompany() }),
      (request) =>
        request.method === 'DELETE' ? new Response(null, { status: 204 }) : json({ competitor: competitor('Belts'), scans: [] }),
    )
    await flushPromises()
    sent.length = 0
    const onRemoved = vi.fn()

    state.remove.remove('gone', onRemoved)
    await flushPromises()

    expect(onRemoved).toHaveBeenCalledOnce()
    expect(sent).toContain('DELETE /api/v1/competitors/gone')
    expect(sent).toContain('GET /api/v1/competitors/kept')
    expect(sent).not.toContain('GET /api/v1/competitors/gone')
  })

  it('says why the server refused to delete a company', async () => {
    const { state } = mountComposable(useRemoveCompany, () => problem(409, 'Conflict', 'A scan of this company is still queued or running.'))
    const onRemoved = vi.fn()
    state.remove('busy', onRemoved)
    await flushPromises()
    expect(onRemoved).not.toHaveBeenCalled()
    expect(state.error.value).toBe('A scan of this company is still queued or running.')
  })
})

describe('useAddCompanies', () => {
  it('adds the companies, says what came of it, and refreshes the lists', async () => {
    const batch = { items: [], scans: 1 }
    const { state, sent } = mountComposable(useAddCompanies, () => json(batch))
    const onDone = vi.fn()
    state.add([{ name: 'New Belts', url: 'https://new-belts.com' }], true, onDone)
    await flushPromises()
    expect(onDone).toHaveBeenCalledWith(batch)
    expect(state.batch.value).toEqual(batch)
    expect(state.error.value).toBeNull()
    expect(sent).toContain('POST /api/v1/admin/competitors/batch')
  })

  it('says why the server refused', async () => {
    const { state } = mountComposable(useAddCompanies, () => problem(403, 'Forbidden'))
    const onDone = vi.fn()
    state.add([], false, onDone)
    await flushPromises()
    expect(onDone).not.toHaveBeenCalled()
    expect(state.error.value).toBe('Only administrators can change the companies.')
  })
})

describe('useAddCompany', () => {
  it('adds a company and hands it over, or says why the server refused', async () => {
    const added = competitor('New Belts', { source: 'added' })
    let refuse = false
    const { state } = mountComposable(useAddCompany, () => (refuse ? problem(409, 'Conflict', 'Twentebelt has this website already.') : json(added, 201)))
    const onAdded = vi.fn()
    state.add('New Belts', 'https://new-belts.com', onAdded)
    await flushPromises()
    expect(onAdded).toHaveBeenCalledWith(added)
    expect(state.error.value).toBeNull()
    refuse = true
    state.add('New Belts', 'https://www.twentebelt.com', onAdded)
    await flushPromises()
    expect(state.error.value).toBe('Twentebelt has this website already.')
    state.reset()
    await flushPromises()
    expect(state.error.value).toBeNull()
  })
})

describe('useUpdateCompany', () => {
  it('saves a change, or says why the server refused', async () => {
    let refuse = false
    const { state, sent } = mountComposable(useUpdateCompany, () => (refuse ? problem(403, 'Forbidden') : json(competitor('Twentebelt'))))
    const onSaved = vi.fn()
    state.update('twentebelt', { name: 'Twentebelt B.V.' }, onSaved)
    await flushPromises()
    expect(onSaved).toHaveBeenCalled()
    expect(sent).toContain('PATCH /api/v1/competitors/twentebelt')
    refuse = true
    state.update('twentebelt', { name: 'x' }, onSaved)
    await flushPromises()
    expect(state.error.value).toBe('Only administrators can change the companies.')
  })
})

describe('useRescan', () => {
  it('follows the queued scans until each has finished, or says why the server refused', async () => {
    const queued = summary({ id: '11111111-1111-4111-8111-111111111111', status: 'queued' })
    let refuse = false
    const { state } = mountComposable(useRescan, (request) => {
      if (request.method === 'POST') return refuse ? problem(403, 'Forbidden') : json({ queued: [queued], skipped: [] }, 202)
      return json({ items: [{ ...queued, status: 'done' }] })
    })
    expect(state.progress.value).toBeNull()
    state.rescan({ competitorIds: ['twentebelt'] })
    await flushPromises()
    expect(state.batch.value?.queued).toHaveLength(1)
    expect(state.progress.value?.finished).toBe(true)
    expect(state.error.value).toBeNull()
    refuse = true
    state.rescan({ competitorIds: ['twentebelt'] })
    await flushPromises()
    expect(state.error.value).toBe('Only administrators can change the companies.')
  })
})
