import type { LlmEvent } from '../../../infrastructure/api/types.gen'
import { atLabel } from './time'

export type EventLine = {
  key: string
  when: string
  text: string
  actor: string
  /** Whether the provider accepted the key, for a test or an addition; null for any other change. */
  ok: boolean | null
}

/** The recent changes and tests, newest first, as the list shows them. */
export function eventLines(events: readonly LlmEvent[]): EventLine[] {
  return [...events]
    .sort((a, b) => Date.parse(b.at) - Date.parse(a.at))
    .map((e, index) => ({
      key: `${e.at}-${String(index)}`,
      when: atLabel(e.at),
      text: e.message,
      actor: e.actor,
      ok: e.ok ?? null,
    }))
}
