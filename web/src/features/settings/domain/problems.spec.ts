import { describe, expect, it } from 'vitest'
import { FORBIDDEN_MESSAGE, isForbidden, settingsProblem } from './problems'

const problem = (status: number, detail?: string) => ({ type: 'about:blank', title: 'T', status, ...(detail ? { detail } : {}) })

describe('settings problems', () => {
  it('recognises a refusal for not being an administrator', () => {
    expect(isForbidden(problem(403))).toBe(true)
    expect(isForbidden(problem(401))).toBe(false)
    expect(isForbidden(new Error('x'))).toBe(false)
    expect(settingsProblem(problem(403), 'load')).toBe(FORBIDDEN_MESSAGE)
  })

  it("puts the provider's reason after what was refused", () => {
    expect(settingsProblem(problem(422, 'invalid x-api-key'), 'add')).toBe('The provider refused this key. invalid x-api-key')
    expect(settingsProblem(problem(422), 'save')).toBe('This key can no longer be read. Choose another key.')
    expect(settingsProblem(problem(422), 'remove')).toBe('This key cannot be removed here.')
  })

  it('says when the provider did not answer, or the key is gone', () => {
    expect(settingsProblem(problem(502), 'models')).toBe('The provider did not answer. Try again in a minute.')
    expect(settingsProblem(problem(502, 'timeout'), 'models')).toBe('The provider did not answer: timeout Try again in a minute.')
    expect(settingsProblem(problem(404), 'test')).toBe('That key no longer exists. Reload the page.')
  })

  it('explains a conflict, preferring the server', () => {
    expect(settingsProblem(problem(409, 'A key with this label exists.'), 'add')).toBe('A key with this label exists.')
    expect(settingsProblem(problem(409), 'add')).toBe('Someone changed these settings at the same time. Reload the page and try again.')
  })

  it('falls back to the general description', () => {
    expect(settingsProblem(problem(500, 'Boom.'), 'load')).toBe('Boom.')
    expect(settingsProblem(new TypeError('Failed to fetch'), 'turnOff')).toBe('Something went wrong. Try again.')
  })
})
