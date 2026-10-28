import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { PromptAnswer, PromptMode } from '../../../infrastructure/api/types.gen'
import { promptAnswer, promptRun } from '../../../testing/fixtures'
import PromptGrid from './PromptGrid.vue'

/** An answer from OpenAI to its own question that puts Tribelt at a place in its list, from 1. */
function at(position: number, mode: PromptMode = 'knowledge'): PromptAnswer {
  return promptAnswer({ promptId: `q${String(position)}`, mode }, [...Array.from({ length: position - 1 }, (_, i) => `C${String(i)}`), 'Tribelt:tribelt'])
}

/** Answers that show every mark a square can: places 1 to 5 and 9, named without a place, not named, failed and unread; and a missing one. */
function everyMark(): PromptAnswer[] {
  return [
    ...[1, 2, 3, 4, 5, 9].map((p) => at(p)),
    promptAnswer({ promptId: 'unplaced' }, ['A:a'], { namesTribelt: true }),
    promptAnswer({ promptId: 'none' }, ['A:a']),
    promptAnswer({ promptId: 'failed', failure: 'Rate limited.', extraction: undefined }),
    promptAnswer({ promptId: 'unread' }, [], { failed: true }),
  ]
}

function grid(answers: PromptAnswer[], mode: PromptMode | 'all' = 'all') {
  return mount(PromptGrid, { props: { run: promptRun(), answers, mode, scope: 'all', selected: undefined } })
}

describe('the answer grid and its legend', () => {
  it('shows every mark a square can show in the legend, with the same colours and border', () => {
    const wrapper = grid(everyMark())
    const legend = wrapper.findAll('ul[aria-label="What the squares mean"] li > span').map((s) => ({ text: s.text(), classes: s.attributes('class') }))
    const cells = wrapper.findAll('table button')
    expect(new Set(cells.map((b) => b.text()))).toEqual(new Set(['1', '2', '3', '4', '5', '9', '•', '', '–']))
    for (const cell of cells) {
      const entry = legend.find((l) => l.classes === cell.attributes('class'))
      expect(entry, cell.attributes('aria-label')).toBeDefined()
      if (/^\d+$/.test(cell.text())) expect(entry?.text).toMatch(/^\d+$/)
      else expect(entry?.text).toBe(cell.text())
    }
  })

  it('says what each mark means', () => {
    const wrapper = grid(everyMark())
    expect(wrapper.findAll('ul[aria-label="What the squares mean"] li').map((l) => l.text())).toEqual([
      '1 Tribelt is named first',
      '2 Tribelt is named second or third',
      '4 Tribelt is named fourth or later',
      '• Tribelt is named, but not in a ranked list',
      'Tribelt is not named',
      '– No answer came back',
    ])
    const heads = wrapper.findAll('thead tr:nth-child(2) th')
    expect(heads.map((th) => th.attributes('title'))).toEqual(['from its own knowledge', 'with web search', 'from its own knowledge', 'with web search'])
    expect(heads[0]?.get('.sr-only').text()).toBe('from its own knowledge')
  })

  it('shows only the column of the mode it is narrowed to', () => {
    const wrapper = grid([at(1, 'web')], 'web')
    expect(wrapper.findAll('thead tr:nth-child(2) th').map((th) => th.attributes('title'))).toEqual(['with web search', 'with web search'])
  })

  it('describes every square in full, for hover and screen readers', () => {
    const wrapper = grid(everyMark(), 'knowledge')
    const labels = wrapper.findAll('table button').filter((b) => b.attributes('aria-label')?.startsWith('OpenAI'))
    const head = 'OpenAI, from its own knowledge, Eyelink, Twente, in Dutch:'
    expect(labels.map((b) => b.attributes('aria-label'))).toEqual([
      `${head} Tribelt is named first.`,
      `${head} Tribelt is named 2nd.`,
      `${head} Tribelt is named 3rd.`,
      `${head} Tribelt is named 4th.`,
      `${head} Tribelt is named 5th.`,
      `${head} Tribelt is named 9th.`,
      `${head} Tribelt is named, but not in a ranked list.`,
      `${head} Tribelt is not named.`,
      `${head} No answer came back.`,
      `${head} No answer came back.`,
    ])
    for (const b of labels) expect(b.attributes('title')).toBe(b.attributes('aria-label'))
  })
})
