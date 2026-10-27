import { flushPromises } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { competitor } from '../../../testing/fixtures'
import { json, problem } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { useCompany } from './useCompany'
import { useRemoveCompany } from './useCompanyAdmin'

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
