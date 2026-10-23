import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h, nextTick } from 'vue'
import { createMemoryHistory, createRouter, RouterView } from 'vue-router'
import { SEARCH_DELAY_MS, useTermView } from './useTermView'

/** Mounts the composable on a router at path, returning it and the router. */
async function setup(path = '/terms') {
  let state: ReturnType<typeof useTermView> | undefined
  const Page = defineComponent({ setup: () => ((state = useTermView()), () => h('div')) })
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/terms', component: Page },
      { path: '/elsewhere', component: { render: () => null } },
    ],
  })
  await router.push(path)
  const wrapper = mount(RouterView, { global: { plugins: [router] } })
  await flushPromises()
  if (!state) throw new Error('the composable did not mount')
  return { state, router, wrapper }
}

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
})

afterEach(() => {
  vi.useRealTimers()
})

describe('useTermView', () => {
  it('starts from the filters in the URL', async () => {
    const { state } = await setup('/terms?q=spiral&language=en-US&tier=bogus&gap=1')
    expect(state.view.value).toEqual({ q: 'spiral', language: 'en', tier: null, user: '', group: null, gap: true })
    expect(state.draft.value).toBe('spiral')
  })

  it('puts the search in the URL only once typing pauses', async () => {
    const { state, router } = await setup('/terms?tier=spec')
    state.draft.value = 'spi'
    await nextTick()
    vi.advanceTimersByTime(SEARCH_DELAY_MS - 1)
    state.draft.value = 'spiral'
    await nextTick()
    vi.advanceTimersByTime(SEARCH_DELAY_MS - 1)
    await flushPromises()
    expect(router.currentRoute.value.query).toEqual({ tier: 'spec' })
    vi.advanceTimersByTime(1)
    await flushPromises()
    expect(router.currentRoute.value.query).toEqual({ tier: 'spec', q: 'spiral' })
  })

  it('drops a pending search when the text is put back as it was', async () => {
    const { state, router } = await setup('/terms?q=belt')
    state.draft.value = 'belts'
    await nextTick()
    state.draft.value = 'belt'
    await nextTick()
    vi.advanceTimersByTime(SEARCH_DELAY_MS)
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe('/terms?q=belt')
  })

  it('follows a query the URL changes, dropping what was being typed', async () => {
    const { state, router } = await setup()
    state.draft.value = 'typed'
    await nextTick()
    await router.replace('/terms?q=linked')
    await flushPromises()
    expect(state.draft.value).toBe('linked')
    vi.advanceTimersByTime(SEARCH_DELAY_MS)
    await flushPromises()
    expect(router.currentRoute.value.query).toEqual({ q: 'linked' })
  })

  it('toggles a filter on and off, and clears one', async () => {
    const { state, router } = await setup('/terms?q=belt')
    state.toggle('tier', 'product')
    await flushPromises()
    expect(router.currentRoute.value.query).toEqual({ q: 'belt', tier: 'product' })
    state.toggle('user', 'Märtens')
    await flushPromises()
    expect(router.currentRoute.value.query).toEqual({ q: 'belt', tier: 'product', user: 'Märtens' })
    state.toggle('tier', 'product')
    await flushPromises()
    expect(router.currentRoute.value.query).toEqual({ q: 'belt', user: 'Märtens' })
    state.clear('user')
    await flushPromises()
    expect(router.currentRoute.value.query).toEqual({ q: 'belt' })
    state.toggle('group', 'technical')
    await flushPromises()
    state.clear('group')
    await flushPromises()
    expect(router.currentRoute.value.query).toEqual({ q: 'belt' })
  })

  it('shows only the terms Tribelt lacks, and every term again', async () => {
    const { state, router } = await setup()
    state.toggleGap()
    await flushPromises()
    expect(router.currentRoute.value.query).toEqual({ gap: '1' })
    state.toggleGap()
    await flushPromises()
    expect(router.currentRoute.value.query).toEqual({})
  })

  it('sends nothing once the view is gone', async () => {
    const { state, router } = await setup()
    state.draft.value = 'late'
    await nextTick()
    await router.push('/elsewhere')
    await flushPromises()
    vi.advanceTimersByTime(SEARCH_DELAY_MS)
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe('/elsewhere')
  })
})
