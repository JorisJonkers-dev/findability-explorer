import { describe, expect, it } from 'vitest'
import { me } from '../../../testing/fixtures'
import { manageCounts } from '../../../testing/manageFixtures'
import { json, mountApp, routes } from '../../../testing/mountApp'

const admin = routes({ 'GET /api/v1/me': () => json(me({ isAdmin: true })), ...manageCounts({ terms: 2 }) })

describe('Activity', () => {
  it('opens Manage with a card per section, saying what each holds', async () => {
    const { wrapper } = await mountApp(admin, '/manage')
    expect(wrapper.get('h1').text()).toBe('Activity')
    expect(wrapper.get('nav[aria-label="Manage"] a[aria-current="page"]').text()).toBe('Activity')
    const cards = wrapper.findAll('ul[aria-label="The sections of Manage"] a')
    expect(cards.map((a) => a.attributes('href'))).toEqual([
      '/manage/prompt-test',
      '/manage/scans',
      '/manage/competitors',
      '/manage/terms',
      '/manage/recommendations',
      '/manage/keys',
    ])
    expect(cards[3]?.text()).toBe('Search term listThe search terms the scans found that wait for a decision.')
  })

  it('tells someone who is no administrator that Manage is for administrators, with the way to the Overview', async () => {
    const { wrapper } = await mountApp(routes({}), '/manage')
    expect(wrapper.text()).toContain('Only administrators can start, stop, and change the work behind the results.')
    expect(wrapper.get('a[data-part="to-overview"]').attributes('href')).toBe('/')
    expect(wrapper.find('ul[aria-label="The sections of Manage"]').exists()).toBe(false)
  })
})
