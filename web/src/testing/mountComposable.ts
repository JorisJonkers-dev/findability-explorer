import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { mount } from '@vue/test-utils'
import { onTestFinished, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { configureApi } from '../infrastructure/http'
import type { Handler } from './mountApp'

/**
 * Runs a composable inside a component, on a fresh query client, with the network answered by
 * handle. Returns what the composable returned, and the requests sent as "METHOD /path?query".
 */
export function mountComposable<T>(use: () => T, handle: Handler) {
  const sent: string[] = []
  configureApi({
    baseUrl: 'http://app.test',
    fetch: vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init)
      const url = new URL(request.url)
      sent.push(`${request.method} ${url.pathname}${url.search}`)
      return Promise.resolve(handle(request))
    }),
  })
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  let state: T | undefined
  const wrapper = mount(defineComponent({ setup: () => ((state = use()), () => h('div')) }), {
    global: { plugins: [[VueQueryPlugin, { queryClient }]] },
  })
  onTestFinished(() => {
    wrapper.unmount()
    queryClient.clear()
  })
  if (state === undefined) throw new Error('the composable did not mount')
  return { state, sent, queryClient }
}
