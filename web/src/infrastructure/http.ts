import * as z from 'zod'
import { client } from './api/client.gen'

export type ApiOptions = {
  baseUrl?: string
  fetch?: typeof fetch
  /** Called when the server answers 401: the session ended, or never began. */
  onUnauthorized?: () => void
}

let unauthorized: number | undefined

/** Points the generated client at the page's own origin: the Go binary serves the API beside the app. */
export function configureApi(options: ApiOptions = {}): void {
  // The server's Content-Security-Policy allows no eval, so zod must not probe for it.
  z.config({ jitless: true })
  client.setConfig({ baseUrl: options.baseUrl ?? '', ...(options.fetch ? { fetch: options.fetch } : {}) })
  if (unauthorized !== undefined) client.interceptors.response.eject(unauthorized)
  const onUnauthorized = options.onUnauthorized
  unauthorized = onUnauthorized
    ? client.interceptors.response.use((response) => {
        if (response.status === 401) onUnauthorized()
        return response
      })
    : undefined
}
