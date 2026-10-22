import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { flushPromises, mount } from '@vue/test-utils'
import { onTestFinished, vi } from 'vitest'
import { createMemoryHistory } from 'vue-router'
import App from '../app/App.vue'
import { createAppRouter } from '../app/router'
import { configureApi } from '../infrastructure/http'

export type Handler = (request: Request) => Response | Promise<Response>

/** A JSON response, as the server sends it. */
export function json(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

/** An RFC 9457 problem response. */
export function problem(status: number, title: string, detail?: string): Response {
  return new Response(JSON.stringify({ type: 'about:blank', title, status, ...(detail ? { detail } : {}) }), {
    status,
    headers: { 'Content-Type': 'application/problem+json' },
  })
}

/** Answers each request by its method and path ("GET /api/v1/terms"); anything else is a 404. */
export function routes(table: Record<string, Handler>): Handler {
  return (request) => {
    const { pathname } = new URL(request.url)
    const handler = table[`${request.method} ${pathname}`]
    return handler ? handler(request) : problem(404, 'Not Found')
  }
}

/** Lets pending responses, renders and lazily loaded pages settle. */
export async function settle(): Promise<void> {
  await flushPromises()
  await vi.dynamicImportSettled()
  await flushPromises()
}

/** Mounts the whole app at path, with the network answered by handle. */
export async function mountApp(handle: Handler, path = '/') {
  const sent: string[] = []
  const fetch = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
    const request = input instanceof Request ? input : new Request(input, init)
    const url = new URL(request.url)
    sent.push(`${request.method} ${url.pathname}${url.search}`)
    return Promise.resolve(handle(request))
  })
  configureApi({ baseUrl: 'http://app.test', fetch })
  const router = createAppRouter(createMemoryHistory())
  await router.push(path)
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const wrapper = mount(App, {
    attachTo: document.body,
    global: { plugins: [router, [VueQueryPlugin, { queryClient }]] },
  })
  // Unmounting stops any polling, so one test's timers never reach the next.
  onTestFinished(() => {
    wrapper.unmount()
    queryClient.clear()
  })
  await router.isReady()
  await settle()
  /** The requests sent so far, as "METHOD /path?query". */
  const requests = () => [...sent]
  return { wrapper, fetch, router, queryClient, requests }
}
