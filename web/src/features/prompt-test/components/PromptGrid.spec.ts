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
  it('shows every mark a square can show in the legend, with the same text, colours and border', () => {
    const wrapper = grid(everyMark())
    const legend = wrapper.findAll('ul[aria-label="Legend"] li > span').map((s) => ({ text: s.text(), classes: s.attributes('class') }))
    const cells = wrapper.findAll('table button')
    expect(new Set(cells.map((b) => b.text()))).toEqual(new Set(['1', '2', '3', '4', '5', '9', '?', '–', '✕']))
    for (const cell of cells) {
      const entry = legend.find((l) => l.classes === cell.attributes('class'))
      expect(entry, cell.attributes('aria-label')).toBeDefined()
      if (/^\d+$/.test(cell.text())) expect(entry?.text).toMatch(/^\d+$/)
      else expect(entry?.text).toBe(cell.text())
    }
  })

  it('says what each mark and each column means', () => {
    const wrapper = grid(everyMark())
    expect(wrapper.findAll('ul[aria-label="Legend"] li').map((l) => l.text())).toEqual([
      '1 Tribelt named first',
      '2 Named 2nd or 3rd',
      '4 Named 4th or later',
      '? Named, place unknown',
      '– Not named',
      '✕ No answer. The model gave none, or it could not be read.',
      'Own Own knowledge',
      'Web With web search',
    ])
    const heads = wrapper.findAll('thead tr:nth-child(2) th')
    expect(heads.map((th) => th.attributes('title'))).toEqual(['Own knowledge', 'With web search', 'Own knowledge', 'With web search'])
    expect(heads[0]?.get('.sr-only').text()).toBe('Own knowledge')
  })

  it('explains only the column it shows when narrowed to one mode', () => {
    const wrapper = grid([at(1, 'web')], 'web')
    expect(wrapper.findAll('ul[aria-label="Legend"] li').at(-1)?.text()).toBe('Web With web search')
    expect(wrapper.findAll('ul[aria-label="Legend"] li').map((l) => l.text())).not.toContain('Own Own knowledge')
  })

  it('describes every square in full, for hover and screen readers', () => {
    const wrapper = grid(everyMark(), 'knowledge')
    const labels = wrapper.findAll('table button').filter((b) => b.attributes('aria-label')?.startsWith('OpenAI'))
    expect(labels.map((b) => b.attributes('aria-label'))).toEqual([
      'OpenAI, own knowledge, Eyelink, Twente (Dutch): Tribelt named first',
      'OpenAI, own knowledge, Eyelink, Twente (Dutch): Tribelt named 2nd',
      'OpenAI, own knowledge, Eyelink, Twente (Dutch): Tribelt named 3rd',
      'OpenAI, own knowledge, Eyelink, Twente (Dutch): Tribelt named 4th',
      'OpenAI, own knowledge, Eyelink, Twente (Dutch): Tribelt named 5th',
      'OpenAI, own knowledge, Eyelink, Twente (Dutch): Tribelt named 9th',
      'OpenAI, own knowledge, Eyelink, Twente (Dutch): Tribelt named, place unknown',
      'OpenAI, own knowledge, Eyelink, Twente (Dutch): Tribelt not named',
      'OpenAI, own knowledge, Eyelink, Twente (Dutch): No answer, the model gave none or it could not be read',
      'OpenAI, own knowledge, Eyelink, Twente (Dutch): No answer, the model gave none or it could not be read',
    ])
    for (const b of labels) expect(b.attributes('title')).toBe(b.attributes('aria-label'))
  })
})
