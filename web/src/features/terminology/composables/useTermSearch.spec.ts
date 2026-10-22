import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { defineComponent, h, ref } from 'vue'
import { configureApi } from '../../../infrastructure/http'
import { term, termSearch } from '../../../testing/fixtures'
import { json, problem } from '../../../testing/mountApp'
import { useTermMap, useTermSearch } from './useTermSearch'

const NONE = { q: '', language: null, tier: null, family: null, user: '' }

/** Runs both composables against fetch, on one query client. */
function setup(fetch: (input: RequestInfo | URL) => Promise<Response>) {
  const sent: string[] = []
  configureApi({
    baseUrl: 'http://app.test',
    fetch: (input) => {
      const url = new URL((input as Request).url)
      sent.push(url.search)
      return fetch(input)
    },
  })
  let search: ReturnType<typeof useTermSearch> | undefined
  let map: ReturnType<typeof useTermMap> | undefined
  mount(
    defineComponent({
      setup: () => {
        search = useTermSearch(ref(NONE))
        map = useTermMap()
        return () => h('div')
      },
    }),
    { global: { plugins: [[VueQueryPlugin, { queryClient: new QueryClient({ defaultOptions: { queries: { retry: false } } }) }]] } },
  )
  if (!search || !map) throw new Error('the composables did not mount')
  return { search, map, sent }
}

describe('useTermSearch and useTermMap', () => {
  it('have nothing before the first response', () => {
    const { search, map } = setup(() => new Promise<Response>(() => undefined))
    expect(search.terms.value).toEqual([])
    expect(search.total.value).toBe(0)
    expect(search.facets.value).toEqual({ languages: [], tiers: [], families: [] })
    expect(search.isLoading.value).toBe(true)
    expect(search.error.value).toBeNull()
    expect(map.terms.value).toEqual([])
    expect(map.total.value).toBe(0)
  })

  it('share one request for the unfiltered map', async () => {
    const { search, map, sent } = setup(() => Promise.resolve(json(termSearch([term()], { total: 146 }))))
    await flushPromises()
    expect(sent).toEqual(['?limit=200'])
    expect(search.terms.value).toEqual([term()])
    expect(search.total.value).toBe(146)
    expect(search.facets.value.languages).toHaveLength(2)
    expect(map.terms.value).toEqual([term()])
    expect(map.total.value).toBe(146)
  })

  it('describes a failed search', async () => {
    const { search } = setup(() => Promise.resolve(problem(400, 'Bad Request', 'The query is too long.')))
    await flushPromises()
    expect(search.error.value).toBe('The query is too long.')
  })
})
