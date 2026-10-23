import { describe, expect, it } from 'vitest'
import { llmSettings } from '../fixtures'
import { eventLines } from './events'

describe('the event list', () => {
  it('lists the newest first, with who did it and whether the key was accepted', () => {
    const events = [...llmSettings().events, { at: '2026-10-23T10:00:00Z', actor: 'ada', action: 'active_cleared' as const, message: 'Turned matching off.' }]
    expect(eventLines(events).map((e) => [e.when, e.actor, e.ok, e.text])).toEqual([
      ['23 Oct 2026, 10:00 UTC', 'ada', null, 'Turned matching off.'],
      ['23 Oct 2026, 09:30 UTC', 'joris', true, 'Tested anthropic key "Vault key": accepted.'],
      ['22 Oct 2026, 14:05 UTC', 'joris', true, 'Added openrouter key "Course budget key" (9f86d081884c).'],
    ])
    expect(new Set(eventLines(events).map((e) => e.key)).size).toBe(3)
  })
})
