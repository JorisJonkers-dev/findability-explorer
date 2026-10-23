import { describe, expect, it } from 'vitest'
import { LOGOUT_ACTION, loginUrl } from './auth'

describe('signing in and out', () => {
  it('sends the user to auth-api and back to the page they were on', () => {
    expect(loginUrl('/terms?q=spiral belt&tier=spec')).toBe('/auth/login?return=%2Fterms%3Fq%3Dspiral%20belt%26tier%3Dspec')
    expect(loginUrl('/')).toBe('/auth/login?return=%2F')
    expect(LOGOUT_ACTION).toBe('/auth/logout')
  })
})
