import { describe, expect, it } from 'vitest'
import { me } from '../../../testing/fixtures'
import { manageCounts } from '../../../testing/manageFixtures'
import { json, mountApp, routes } from '../../../testing/mountApp'

const admin = routes({ 'GET /api/v1/me': () => json(me({ isAdmin: true })), ...manageCounts({ terms: 2 }) })

describe('Recommendations in Manage', () => {
  it('says marking and the goals are on the Recommendations page for now, and links there', async () => {
    const { wrapper } = await mountApp(admin, '/manage/recommendations')
    expect(wrapper.get('h1').text()).toBe('Recommendations')
    expect(wrapper.get('nav[aria-label="Manage"] a[aria-current="page"]').text()).toBe('Recommendations')
    expect(wrapper.get('header a[href="/recommendations"]').text()).toBe('See the recommendations')
    expect(wrapper.get('section[aria-labelledby="tick-off-title"] a').attributes('href')).toBe('/recommendations')
  })
})
