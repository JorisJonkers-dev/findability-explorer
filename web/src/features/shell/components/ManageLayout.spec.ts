import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { me } from '../../../testing/fixtures'
import { type ManageCounts, manageCounts } from '../../../testing/manageFixtures'
import { mountAt } from '../../../testing/mountAt'
import { json, routes, settle } from '../../../testing/mountApp'
import ManageLayout from './ManageLayout.vue'

const PROPS = {
  title: 'Candidates',
  purpose: 'The companies the AI responses name that are not on the competitor list yet.',
  dataLine: 'From the Prompt runs so far.',
  results: { to: '/competitors', label: 'See the competitors' },
  helpTopic: 'competitors',
}

function layout(path: string, options: { isAdmin?: boolean; counts?: Partial<ManageCounts>; props?: Record<string, unknown> } = {}) {
  const { isAdmin = true, counts = {}, props = PROPS } = options
  return mountAt(ManageLayout, path, routes({ 'GET /api/v1/me': () => json(me({ isAdmin })), ...manageCounts(counts) }), {
    props,
    slots: { default: '<section aria-labelledby="w"><h2 id="w">Waiting</h2></section>' },
  })
}

/** The menu's links, as [label, address, current], in order. */
function menu(wrapper: VueWrapper) {
  return wrapper.findAll('nav[aria-label="Manage"] a').map((a): [string, string, string] => [a.text(), a.attributes('href') ?? '', a.attributes('aria-current') ?? ''])
}

describe('the Manage layout', () => {
  it('lists the sections down the side, with the open one folded out and the current page marked', async () => {
    const { wrapper } = await layout('/manage/competitors/candidates')
    expect(menu(wrapper)).toEqual([
      ['Activity', '/manage', ''],
      ['Prompt test', '/manage/prompt-test', ''],
      ['Website scans', '/manage/scans', ''],
      ['Competitor list', '/manage/competitors', ''],
      ['Candidates', '/manage/competitors/candidates', 'page'],
      ['Search term list', '/manage/terms', ''],
      ['Recommendations', '/manage/recommendations', ''],
      ['Keys and spending', '/manage/keys', ''],
    ])
    expect(wrapper.get('nav[aria-label="Manage"] [data-open]').text()).toContain('Competitor list')
  })

  it('marks a section index as the current page, with no other section folded out', async () => {
    const { wrapper } = await layout('/manage/keys', { props: { ...PROPS, title: 'Keys and spending' } })
    const links = menu(wrapper)
    expect(links.filter(([, , current]) => current === 'page')).toEqual([['Keys and spending', '/manage/keys', 'page']])
    expect(links.map(([label]) => label)).not.toContain('Candidates')
    expect(links.map(([label]) => label)).not.toContain('Waiting terms')
  })

  it('keeps a detail page’s section folded out, with no subpage marked', async () => {
    const { wrapper } = await layout('/manage/scans/5b0c')
    expect(wrapper.get('nav[aria-label="Manage"] [data-open]').text()).toContain('Website scans')
    expect(wrapper.findAll('nav[aria-label="Manage"] [aria-current]')).toHaveLength(0)
  })

  it('shows a count on a folded section, and on its subpage once the section is open', async () => {
    const { wrapper } = await layout('/manage/competitors/candidates', { counts: { candidates: 5, terms: 8, runs: 1, scans: 2 } })
    const labels = Object.fromEntries(menu(wrapper).map(([label, href]): [string, string] => [href, label]))
    expect(labels['/manage/competitors']).toBe('Competitor list')
    expect(labels['/manage/competitors/candidates']).toBe('Candidates 5 waiting')
    expect(labels['/manage/terms']).toBe('Search term list 8 waiting')
    expect(labels['/manage/prompt-test']).toBe('Prompt test 1 running')
    expect(labels['/manage/scans']).toBe('Website scans 2 running')
    expect(labels['/manage/keys']).toBe('Keys and spending')
    // The count shows as a figure; the word after it is for screen readers.
    expect(wrapper.get('nav[aria-label="Manage"] a[href="/manage/terms"] .sr-only').text()).toBe('waiting')
  })

  it('moves the count back to the section when it folds', async () => {
    const { wrapper, router } = await layout('/manage/terms/waiting', { counts: { terms: 8 } })
    expect(menu(wrapper).find(([, href]) => href === '/manage/terms/waiting')?.[0]).toBe('Waiting terms 8 waiting')
    await router.push('/manage/keys')
    await settle()
    expect(menu(wrapper).find(([, href]) => href === '/manage/terms')?.[0]).toBe('Search term list 8 waiting')
  })

  it('shows the page’s title, purpose, data line, Results link, sections, and Help link beside the menu', async () => {
    const { wrapper } = await layout('/manage/competitors/candidates', { props: { ...PROPS, back: { to: '/manage/competitors', label: 'Back to Competitor list' } } })
    expect(wrapper.get('header').text()).toContain('Manage, Competitor list')
    expect(wrapper.get('h1').text()).toBe('Candidates')
    expect(wrapper.get('[data-part="purpose"]').text()).toBe(PROPS.purpose)
    expect(wrapper.get('[data-part="data-line"]').text()).toBe('From the Prompt runs so far.')
    expect(wrapper.get('header a[href="/competitors"]').text()).toBe('See the competitors')
    expect(wrapper.get('h2').text()).toBe('Waiting')
    expect(wrapper.get('a[href="/help/competitors"]').text()).toBe('How this is measured, in Help')
    expect(wrapper.text()).toContain('Back to Competitor list')
  })

  it('names only Manage in the eyebrow of an address outside the menu', async () => {
    const { wrapper } = await layout('/elsewhere', { props: { title: 'Nowhere', purpose: 'Nothing.' } })
    expect(wrapper.get('header').text()).toContain('Manage')
    expect(wrapper.get('header').text()).not.toContain(',')
  })

  it('tells someone who is no administrator that Manage is not for them, and asks nothing of the server', async () => {
    const { wrapper, requests } = await layout('/manage/competitors/candidates', { isAdmin: false })
    expect(wrapper.find('nav[aria-label="Main"]').exists()).toBe(true)
    expect(wrapper.find('nav[aria-label="Manage"]').exists()).toBe(false)
    expect(wrapper.get('h1').text()).toBe('Manage')
    expect(wrapper.text()).toContain('Only administrators can start, stop, and change the work behind the results.')
    expect(wrapper.get('a[data-part="to-overview"]').text()).toBe('Go to the Overview')
    expect(wrapper.find('h2').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Candidates')
    expect(requests()).toEqual(['GET /api/v1/me'])
  })
})
