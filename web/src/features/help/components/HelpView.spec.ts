import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { me } from '../../../testing/fixtures'
import { json, mountApp, routes, settle } from '../../../testing/mountApp'
import { GROUPS, TOPICS } from '../domain/topics'
import { promptSettings } from '../fixtures'

type Wrapper = Awaited<ReturnType<typeof mountApp>>['wrapper']

const reader = () => routes({})
const admin = () =>
  routes({
    'GET /api/v1/me': () => json(me({ isAdmin: true })),
    'GET /api/v1/admin/prompt-test': () => json(promptSettings({ runUsd: 1.5, monthUsd: 6 }, 'weekly')),
  })

let scrolled: string[] = []
beforeEach(() => {
  scrolled = []
  // jsdom does not scroll; record which element was asked to come into view.
  Element.prototype.scrollIntoView = vi.fn(function (this: Element) {
    scrolled.push(this.id)
  })
})
afterEach(() => {
  Reflect.deleteProperty(Element.prototype, 'scrollIntoView')
})

const groupButtons = (w: Wrapper) => w.findAll('[role="group"][aria-label="Topics"] button')
const button = (w: Wrapper, label: string) => groupButtons(w).find((b) => b.text().startsWith(label))
const pressed = (w: Wrapper) => w.get('[role="group"][aria-label="Topics"] [aria-pressed="true"]').text()
const titles = (w: Wrapper) => w.findAll('article h3').map((h) => h.text())

describe('Help', () => {
  it('opens on Getting started in the page frame, with every group and its count', async () => {
    const { wrapper } = await mountApp(reader(), '/help')
    expect(wrapper.get('h1').text()).toBe('Help')
    expect(wrapper.get('[data-part="purpose"]').text()).toBe('Short explanations of how the Findability Explorer works, each with a link to the place where you do it.')
    expect(wrapper.get('[data-part="data-line"]').text()).toBe(`${String(TOPICS.length)} topics in 9 groups. Type to search them.`)
    expect(wrapper.get('nav[aria-label="Main"] a[aria-current="page"]').text()).toBe('Help')
    expect(groupButtons(wrapper).map((b) => b.text())).toEqual([
      `Everything ${String(TOPICS.length)}`,
      ...GROUPS.map((g) => `${g.label} ${String(TOPICS.filter((t) => t.group === g.id).length)}`),
    ])
    expect(pressed(wrapper)).toMatch(/^Getting started/)
    expect(titles(wrapper)).toEqual(TOPICS.filter((t) => t.group === 'start').map((t) => t.title))
    expect(wrapper.get('[data-part="topics"]').classes()).toContain('min-[1100px]:grid-cols-2')
  })

  it('shows one group at a time, and the glossary as a two-column list', async () => {
    const { wrapper } = await mountApp(reader(), '/help')
    await button(wrapper, 'Costs and budgets')?.trigger('click')
    expect(pressed(wrapper)).toMatch(/^Costs and budgets/)
    expect(titles(wrapper)).toEqual(TOPICS.filter((t) => t.group === 'costs').map((t) => t.title))
    expect(wrapper.findAll('h2').map((h) => h.text())).toContain('Costs and budgets')

    await button(wrapper, 'Words used in this tool')?.trigger('click')
    const glossary = wrapper.get('dl')
    expect(glossary.classes()).toContain('sm:grid-cols-2')
    expect(glossary.findAll('dt').map((d) => d.text())).toContain('Typical competitor')
    expect(wrapper.text()).toContain('The middle value across all competitors on the list.')
    expect(wrapper.get('nav[aria-label="Help topics"]').text()).toContain('words explained')
  })

  it('searches every group as you type, with the matches marked and counted', async () => {
    const { wrapper } = await mountApp(reader(), '/help')
    await wrapper.get('input[type="search"]').setValue('budget')
    expect(pressed(wrapper)).toMatch(/^Everything/)
    expect(titles(wrapper)).toContain('What happens when a budget runs out')
    expect(wrapper.findAll('mark').map((m) => m.text().toLowerCase())).toContain('budget')
    expect(button(wrapper, 'Costs and budgets')?.text()).toMatch(/Costs and budgets [1-9]/)
    expect(wrapper.get('nav[aria-label="Help topics"]').text()).toContain('Costs and budgets')

    await wrapper.get('input[type="search"]').setValue('zzzz nothing')
    expect(wrapper.get('[data-part="empty"]').text()).toBe('Nothing in Help matches “zzzz nothing”. Try a shorter word, or pick Everything.')
  })

  it('hides Manage links from a reader, and says an administrator does it', async () => {
    const { wrapper } = await mountApp(reader(), '/help/results-manage')
    const article = wrapper.get('#help-results-manage')
    expect(article.findAll('a').map((a) => a.attributes('href'))).not.toContain('/manage')
    expect(article.text()).toContain('An administrator does this in Manage.')
    expect(wrapper.find('header a[href="/manage"]').exists()).toBe(false)
    // A reader's topics leave the figures out; only administrators may read the estimates.
    await button(wrapper, 'The Prompt test')?.trigger('click')
    expect(wrapper.get('#help-prompt-test').text()).not.toContain('$')
  })

  it('gives an administrator the Manage links, and the figures from the live estimates', async () => {
    const { wrapper } = await mountApp(admin(), '/help/prompt-test')
    await settle()
    const article = wrapper.get('#help-prompt-test')
    expect(article.text()).toContain('A run costs about $1.50 today. It runs by itself once a week.')
    expect(article.findAll('a').map((a) => a.attributes('href'))).toEqual(['/ai-responses', '/manage/prompt-test'])
    expect(article.text()).not.toContain('An administrator does this in Manage.')
  })

  it("opens a topic's address at its group, with the topic in view", async () => {
    const { wrapper, router } = await mountApp(reader(), '/help/buyer-intent')
    expect(pressed(wrapper)).toMatch(/^Reading the results/)
    expect(wrapper.get('#help-buyer-intent').attributes('data-hit')).toBe('true')
    expect(wrapper.get('#help-buyer-intent').text()).toContain('Application. The buyer describes what the belt is for, such as a bakery oven.')
    expect(scrolled).toEqual(['help-buyer-intent'])

    await router.push('/help/over-budget')
    await settle()
    expect(pressed(wrapper)).toMatch(/^Costs and budgets/)
    expect(scrolled).toEqual(['help-buyer-intent', 'help-over-budget'])

    await router.push('/help/targets')
    await settle()
    expect(wrapper.get('#help-goals').attributes('data-hit')).toBe('true')

    await router.push('/help/word-gap')
    await settle()
    expect(wrapper.get('#help-word-gap').attributes('data-hit')).toBe('true')
    expect(scrolled.at(-1)).toBe('help-word-gap')

    await router.push('/help')
    await settle()
    expect(pressed(wrapper)).toMatch(/^Getting started/)
    expect(wrapper.findAll('[data-hit="true"]')).toHaveLength(0)
  })
})
