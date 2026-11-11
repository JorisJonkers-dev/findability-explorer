import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { mount, type VueWrapper } from '@vue/test-utils'
import { onTestFinished, vi } from 'vitest'
import type { Component } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { configureApi } from '../infrastructure/http'
import { type Handler, settle } from './mountApp'

/**
 * Mounts one component at an address, with a router that knows every address and the network
 * answered by handle: for a frame or a layout, whose look depends on the route and the account.
 */
export async function mountAt(component: Component, path: string, handle: Handler, options: { props?: Record<string, unknown>; slots?: Record<string, string> } = {}) {
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
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:rest(.*)*', component: { render: () => null } }] })
  await router.push(path)
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const wrapper = mount(component, { props: options.props, slots: options.slots, global: { plugins: [router, [VueQueryPlugin, { queryClient }]] } }) as unknown as VueWrapper
  onTestFinished(() => {
    wrapper.unmount()
    queryClient.clear()
  })
  await settle()
  return { wrapper, router, requests: () => [...sent] }
}
