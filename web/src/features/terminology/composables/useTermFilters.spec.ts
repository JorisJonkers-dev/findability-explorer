import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h, nextTick } from 'vue'
import { createMemoryHistory, createRouter, RouterView } from 'vue-router'
import { SEARCH_DELAY_MS, useTermFilters } from './useTermFilters'

/** Mounts the composable on a router at path, returning it and the router. */
async function setup(path = '/terminology') {
  let filters: ReturnType<typeof useTermFilters> | undefined
  const Page = defineComponent({ setup: () => ((filters = useTermFilters()), () => h('div')) })
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/terminology', component: Page },
      { path: '/elsewhere', component: { render: () => null } },
    ],
  })
  await router.push(path)
  const wrapper = mount(RouterView, { global: { plugins: [router] } })
  await flushPromises()
  if (!filters) throw new Error('the composable did not mount')
  return { filters, router, wrapper }
}

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
})

afterEach(() => {
  vi.useRealTimers()
})

describe('useTermFilters', () => {
  it('starts from the filters in the URL', async () => {
    const { filters } = await setup('/terminology?q=spiral&language=de&tier=bogus')
    expect(filters.filters.value).toEqual({ q: 'spiral', language: 'de', tier: null, family: null, user: '' })
    expect(filters.draft.value).toBe('spiral')
  })

  it('puts the search in the URL only once typing pauses', async () => {
    const { filters, router } = await setup()
    filters.draft.value = 'spi'
    await nextTick()
    vi.advanceTimersByTime(SEARCH_DELAY_MS - 1)
    filters.draft.value = 'spiral'
    await nextTick()
    vi.advanceTimersByTime(SEARCH_DELAY_MS - 1)
    await flushPromises()
    expect(router.currentRoute.value.query).toEqual({})
    vi.advanceTimersByTime(1)
    await flushPromises()
    expect(router.currentRoute.value.query).toEqual({ q: 'spiral' })
  })

  it('drops a pending search when the text is put back as it was', async () => {
    const { filters, router } = await setup('/terminology?q=belt')
    filters.draft.value = 'belts'
    await nextTick()
    filters.draft.value = 'belt'
    await nextTick()
    vi.advanceTimersByTime(SEARCH_DELAY_MS)
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe('/terminology?q=belt')
  })

  it('follows a query the URL changes, dropping what was being typed', async () => {
    const { filters, router } = await setup()
    filters.draft.value = 'typed'
    await nextTick()
    await router.replace('/terminology?q=linked')
    await flushPromises()
    expect(filters.draft.value).toBe('linked')
    vi.advanceTimersByTime(SEARCH_DELAY_MS)
    await flushPromises()
    expect(router.currentRoute.value.query).toEqual({ q: 'linked' })
  })

  it('changes one filter and keeps the others', async () => {
    const { filters, router } = await setup('/terminology?q=belt&family=eyelink')
    filters.update({ user: 'Twentebelt', language: 'xx' })
    await flushPromises()
    expect(router.currentRoute.value.query).toEqual({ q: 'belt', family: 'eyelink', user: 'Twentebelt' })
  })

  it('toggles a filter: on with a value, off with the same value again', async () => {
    const { filters, router } = await setup('/terminology?q=belt')
    filters.toggle('tier', 'product')
    await flushPromises()
    expect(router.currentRoute.value.query).toEqual({ q: 'belt', tier: 'product' })
    filters.toggle('tier', 'product')
    await flushPromises()
    expect(router.currentRoute.value.query).toEqual({ q: 'belt' })
  })

  it('clears every filter and the search box, and drops a pending search', async () => {
    const { filters, router } = await setup('/terminology?q=belt&family=eyelink')
    filters.draft.value = 'belts'
    await nextTick()
    filters.clear()
    await flushPromises()
    vi.advanceTimersByTime(SEARCH_DELAY_MS)
    await flushPromises()
    expect(router.currentRoute.value.query).toEqual({})
    expect(filters.draft.value).toBe('')
  })

  it('sends nothing once the page is gone', async () => {
    const { filters, router } = await setup()
    filters.draft.value = 'late'
    await nextTick()
    await router.push('/elsewhere')
    await flushPromises()
    vi.advanceTimersByTime(SEARCH_DELAY_MS)
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe('/elsewhere')
  })
})
