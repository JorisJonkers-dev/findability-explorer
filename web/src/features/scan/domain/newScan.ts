import { zProblem } from '../../../infrastructure/api/zod.gen'
import { describeError } from '../../../shared/describeError'

/** The page budgets the form offers; the contract allows any whole number from 1 to 30. */
export const PAGE_OPTIONS = [5, 10, 15, 30] as const
export const DEFAULT_PAGES = 15
export const MAX_URL_LENGTH = 500

export type ScanUrlCheck =
  | { kind: 'ok'; url: string }
  | { kind: 'empty' }
  | { kind: 'malformed' }
  | { kind: 'scheme' }
  | { kind: 'noHost' }
  | { kind: 'tooLong' }

/**
 * Checks a website address before it is sent: an absolute http or https URL with a host name.
 * A bare host ("twentebelt.com") is read as https. Whether the host is public is the server's
 * call; it answers 422 when it is not.
 */
export function checkScanUrl(raw: string): ScanUrlCheck {
  const typed = raw.trim()
  if (typed === '') return { kind: 'empty' }
  const text = /^[a-z][a-z\d+.-]*:/i.test(typed) ? typed : `https://${typed}`
  if (text.length > MAX_URL_LENGTH) return { kind: 'tooLong' }
  let url: URL
  try {
    url = new URL(text)
  } catch {
    return { kind: 'malformed' }
  }
  if (url.protocol !== 'http:' && url.protocol !== 'https:') return { kind: 'scheme' }
  // The parser accepts "https://intranet"; a host on the internet has a dot, or is an IPv6 literal.
  if (!url.hostname.includes('.') && !url.hostname.startsWith('[')) return { kind: 'noHost' }
  return { kind: 'ok', url: text }
}

/** The message to show for a check, or null when the address can be sent. */
export function describeScanUrlCheck(check: ScanUrlCheck): string | null {
  switch (check.kind) {
    case 'ok':
      return null
    case 'empty':
      return 'Enter the address of a website.'
    case 'malformed':
      return 'Enter a full address, like https://www.example.com.'
    case 'scheme':
      return 'Only http and https addresses can be scanned.'
    case 'noHost':
      return 'Enter an address with a full host name, like https://www.example.com.'
    case 'tooLong':
      return `Addresses can be at most ${String(MAX_URL_LENGTH)} characters.`
  }
}

/** Where a refused scan's message belongs: under the address, or under the whole form. */
export type StartError = { field: 'url' | 'form'; message: string }

/**
 * Places the server's answer to a refused scan. A 422 is about the address (not public, not
 * reachable), so it goes under the address; a 429 (too many scans) and anything else under the form.
 */
export function placeStartError(error: unknown): StartError {
  const problem = zProblem.safeParse(error)
  const field = problem.success && problem.data.status === 422 ? 'url' : 'form'
  return { field, message: describeError(error) }
}
