import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { analysis, summary } from '../../../testing/fixtures'
import { json, problem, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { useRecentScans } from './useRecentScans'
import { useStartScan } from './useStartScan'

const NEW_ID = '6c1d2b0f-3e4a-4f6b-9c7d-8e9f0a1b2c3d'

describe('useStartScan', () => {
  it('hands over the new id and lists the recent scans again', async () => {
    const { state, sent } = mountComposable(
      () => ({ start: useStartScan(), recent: useRecentScans() }),
      routes({
        'POST /api/v1/analyses': async (request) => {
          expect(await request.json()).toEqual({ url: 'https://www.twentebelt.com', maxPages: 10 })
          return json(analysis({}, { id: NEW_ID }), 202)
        },
        'GET /api/v1/analyses': () => json({ items: [summary()] }),
      }),
    )
    await flushPromises()
    const started: string[] = []
    state.start.start('https://www.twentebelt.com', 10, (id) => started.push(id))
    expect(state.start.isStarting.value).toBe(true)
    await flushPromises()
    await flushPromises()
    expect(started).toEqual([NEW_ID])
    expect(sent.filter((r) => r.startsWith('GET'))).toHaveLength(2)
    expect(state.start.error.value).toBeNull()
  })

  it('places a refusal, and forgets it on reset', async () => {
    const { state } = mountComposable(useStartScan, () => problem(422, 'Unprocessable', 'The host is not public.'))
    state.start('https://10.0.0.1.example', 15, () => undefined)
    await flushPromises()
    await flushPromises()
    expect(state.error.value).toEqual({ field: 'url', message: 'The host is not public.' })
    state.reset()
    await flushPromises()
    expect(state.error.value).toBeNull()
  })
})
