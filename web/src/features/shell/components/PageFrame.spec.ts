import { describe, expect, it } from 'vitest'
import { me } from '../../../testing/fixtures'
import { mountAt } from '../../../testing/mountAt'
import { json, routes } from '../../../testing/mountApp'
import PageFrame from './PageFrame.vue'

const FULL = {
  section: 'ai-responses',
  title: 'AI responses',
  purpose: 'How often AI assistants name Tribelt, and who they name instead.',
  dataLine: 'From the Prompt run of 27 October 2026.',
  manage: { to: '/manage/prompt-test', label: 'Manage the Prompt test' },
  helpTopic: 'prompt-test',
}

function frame(props: Record<string, unknown>, isAdmin = false) {
  return mountAt(PageFrame, '/ai-responses/all', routes({ 'GET /api/v1/me': () => json(me({ isAdmin })) }), {
    props,
    slots: { default: '<section aria-labelledby="t"><h2 id="t">Over time</h2></section>', aside: '<button type="button">Rank by value</button>' },
  })
}

describe('the page frame', () => {
  it('shows the top bar, the section, the title, the purpose, the data line, the sections, and the link to Help', async () => {
    const { wrapper } = await frame(FULL)
    expect(wrapper.find('nav[aria-label="Main"]').exists()).toBe(true)
    expect(wrapper.get('header').text()).toContain('02 AI responses')
    expect(wrapper.get('h1').text()).toBe('AI responses')
    expect(wrapper.get('[data-part="purpose"]').text()).toBe('How often AI assistants name Tribelt, and who they name instead.')
    expect(wrapper.get('[data-part="data-line"]').text()).toBe('From the Prompt run of 27 October 2026.')
    expect(wrapper.get('h2').text()).toBe('Over time')
    expect(wrapper.get('header button').text()).toBe('Rank by value')
    expect(wrapper.get('a[href="/help/prompt-test"]').text()).toBe('How this is measured, in Help')
  })

  it('keeps the parts in the order of the spec', async () => {
    const { wrapper } = await frame({ ...FULL, back: { to: '/ai-responses', label: 'Back to AI responses' } }, true)
    const text = wrapper.text()
    const order = ['Back to AI responses', 'AI responses', 'How often', 'From the Prompt run', 'Manage the Prompt test', 'Over time', 'How this is measured']
    const at = order.map((part) => text.indexOf(part, text.indexOf('Sign out')))
    expect(at.every((i) => i >= 0)).toBe(true)
    expect([...at].sort((a, b) => a - b)).toEqual(at)
  })

  it('links a detail page back to where it came from', async () => {
    const { wrapper } = await frame({ ...FULL, back: { to: '/ai-responses', label: 'Back to AI responses' } })
    expect(wrapper.findAll('a[href="/ai-responses"]').map((a) => a.text())).toContain('Back to AI responses')
  })

  it('links to its Manage page only for an administrator', async () => {
    const reader = await frame(FULL)
    expect(reader.wrapper.find('a[href="/manage/prompt-test"]').exists()).toBe(false)
    expect(reader.wrapper.text()).not.toContain('Manage the Prompt test')
    const admin = await frame(FULL, true)
    expect(admin.wrapper.get('a[href="/manage/prompt-test"]').text()).toBe('Manage the Prompt test')
  })

  it('leaves out the parts a page does not have', async () => {
    const { wrapper } = await frame({ title: 'Help', purpose: 'Short topics.' }, true)
    expect(wrapper.find('[data-part="data-line"]').exists()).toBe(false)
    expect(wrapper.find('header').text()).not.toContain('Back to')
    expect(wrapper.text()).not.toContain('How this is measured, in Help')
    expect(wrapper.text()).not.toContain('Manage the')
  })
})
