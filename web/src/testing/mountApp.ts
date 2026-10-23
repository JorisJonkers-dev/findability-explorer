import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { flushPromises, mount } from '@vue/test-utils'
import { onTestFinished, vi } from 'vitest'
import { createMemoryHistory } from 'vue-router'
import App from '../app/App.vue'
import { createAppRouter } from '../app/router'
import { loginUrl } from '../features/shell/domain/auth'
import { configureApi } from '../infrastructure/http'
import { me } from './fixtures'

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

/**
 * Answers each request by its method and path ("GET /api/v1/terms"); anything else is a 404.
 * Someone is signed in unless the table says otherwise: GET /api/v1/me answers me().
 */
export function routes(table: Record<string, Handler>): Handler {
  const all: Record<string, Handler> = { 'GET /api/v1/me': () => json(me()), ...table }
  return (request) => {
    const { pathname } = new URL(request.url)
    const handler = all[`${request.method} ${pathname}`]
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
  // jsdom has no scrolling; the router's scrollBehavior would log that on every page change.
  vi.spyOn(window, 'scrollTo').mockImplementation(() => undefined)
  const router = createAppRouter(createMemoryHistory())
  // A 401 sends the browser to sign in; here the "browser" is a spy, given the app's own address.
  const assign = vi.fn<(url: string) => void>()
  configureApi({
    baseUrl: 'http://app.test',
    fetch,
    onUnauthorized: () => {
      assign(loginUrl(router.currentRoute.value.fullPath))
    },
  })
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
  return { wrapper, fetch, router, queryClient, requests, assign }
}
