import { describe, expect, it } from 'vitest'
import type { PromptRunFigures, PromptSetVersionInfo } from '../../../infrastructure/api/types.gen'
import { breaksOf, chartNotes, chartSeries, forVersion, LANGUAGE_MEASURES, parseVersion, partialNote, seriesOf, tableRows, versionNote, versionOptions } from './history'

const versions = [
  { version: 1, publishedAt: '2026-10-26T15:04:51Z', questions: 90, note: 'A supplier, an application, and a problem question for each belt family, in six scopes.' },
  { version: 2, publishedAt: '2026-10-27T06:35:33Z', questions: 20, note: 'Kept the application question a buyer most often asks, in four scopes.' },
]
const runA: PromptRunFigures = { runId: 'a', startedAt: '2026-10-27T06:03:12Z', promptSetVersion: 1, tasks: 1260, answered: 62, read: 47, partial: true, failure: 'OpenRouter answered 401: User not found.', all: 0.4, dutch: 0.48 }
const runB: PromptRunFigures = { runId: 'b', startedAt: '2026-10-27T08:36:32Z', promptSetVersion: 2, tasks: 200, answered: 200, read: 183, partial: false, all: 0.25, dutch: 0.41, english: 0.1 }
const runs = [runA, runB]

describe('the Prompt test history', () => {
  it('gives a series per measure, leaving out runs without a value', () => {
    expect(seriesOf(runs, (r) => r.english)).toEqual([{ x: '2026-10-27T08:36:32Z', y: 0.1, partial: false }])
  })
  it('keeps only one version, or all', () => {
    expect(forVersion(runs, 2).map((r) => r.runId)).toEqual(['b'])
    expect(forVersion(runs, 'all')).toHaveLength(2)
  })
  it('breaks where a version starts with runs on both sides', () => {
    expect(breaksOf(versions, runs)).toEqual([{ x: '2026-10-27T06:35:33Z', label: 'Prompt set version 2', note: 'The questions changed here (prompt set version 2), so the figures before and after are not the same measure.' }])
    expect(breaksOf(versions, forVersion(runs, 2))).toEqual([])
  })
  it('breaks where the model list changed between two runs, beside the versions', () => {
    const before = { ...runB, runId: 'c', startedAt: '2026-10-28T06:00:00Z', models: ['openai/gpt-oss-20b', 'google/gemma:free'] }
    const same = { ...runB, runId: 'd', startedAt: '2026-10-29T06:00:00Z', models: ['google/gemma:free', 'openai/gpt-oss-20b'] }
    const after = { ...runB, runId: 'e', startedAt: '2026-10-30T06:00:00Z', models: ['openai/gpt-oss-20b', 'openai/gpt-5-nano', 'mistralai/small'] }
    const dropped = { ...runB, runId: 'f', startedAt: '2026-10-31T06:00:00Z', models: ['openai/gpt-oss-20b', 'openai/gpt-5-nano'] }
    expect(breaksOf(versions, [runA, runB, before, same, after, dropped])).toEqual([
      { x: '2026-10-27T06:35:33Z', label: 'Prompt set version 2', note: 'The questions changed here (prompt set version 2), so the figures before and after are not the same measure.' },
      {
        x: '2026-10-30T06:00:00Z',
        label: 'Model list',
        note: 'The model list changed here (added openai/gpt-5-nano and mistralai/small, and removed google/gemma:free), so the figures before and after are not the same measure.',
      },
      { x: '2026-10-31T06:00:00Z', label: 'Model list', note: 'The model list changed here (removed mistralai/small), so the figures before and after are not the same measure.' },
    ])
  })
  it('explains a partial run, and names a refused key plainly', () => {
    expect(partialNote(runA)).toBe('27 October 2026, 06:03: a partial run of version 1. Only 62 of its 1,260 requests got a response (the rest failed because the OpenRouter key was refused), and 47 of those could be read, so its figure rests on few responses.')
  })
  it('says what a version asks, and when it has one run', () => {
    expect(versionNote(versions, 1, forVersion(runs, 1))).toBe('Version 1, published 26 October 2026: 90 questions. A supplier, an application, and a problem question for each belt family, in six scopes. It has one run so far, so there is no line yet.')
    expect(versionNote(versions, 'all', runs)).toBe('The questions changed with each version, so the line breaks between versions. Version 1 asked 90 questions, and version 2 asks 20 questions.')
  })
})

describe('the partial-run note', () => {
  const partial: PromptRunFigures = { ...runA, failure: 'The model is rate limited.' }
  it('quotes any other failure without its full stop', () => {
    expect(partialNote(partial)).toContain('(the rest failed because The model is rate limited)')
  })
  it('leaves the reason out when the run gave none', () => {
    expect(partialNote({ ...partial, failure: undefined })).toBe('27 October 2026, 06:03: a partial run of version 1. Only 62 of its 1,260 requests got a response, and 47 of those could be read, so its figure rests on few responses.')
  })
})

describe('the version note', () => {
  it('says nothing more of a version with several runs, or none', () => {
    const v2 = 'Version 2, published 27 October 2026: 20 questions. Kept the application question a buyer most often asks, in four scopes.'
    expect(versionNote(versions, 2, [runB, { ...runB, runId: 'c' }])).toBe(v2)
    expect(versionNote(versions, 2, [])).toBe(v2)
  })
  it('says nothing of a version that is not published', () => {
    expect(versionNote(versions, 3, [])).toBe('')
  })
  it('counts one question as one, and does not speak of breaks with one version', () => {
    const one: PromptSetVersionInfo[] = [{ version: 1, publishedAt: '2026-10-26T15:04:51Z', questions: 1, note: 'One question.' }]
    expect(versionNote(one, 'all', [])).toBe('Version 1 asks 1 question.')
    expect(versionNote(one, 1, [])).toBe('Version 1, published 26 October 2026: 1 question. One question.')
    expect(versionNote([], 'all', [])).toBe('')
  })
  it('lists three versions with the Oxford comma', () => {
    const three = [...versions, { version: 3, publishedAt: '2026-10-28T06:00:00Z', questions: 24, note: 'More.' }]
    expect(versionNote(three, 'all', runs)).toBe('The questions changed with each version, so the line breaks between versions. Version 1 asked 90 questions, version 2 asked 20 questions, and version 3 asks 24 questions.')
  })
})

describe('the version filter', () => {
  it('offers every version, the newest first and called current', () => {
    expect(versionOptions(versions)).toEqual([
      { value: 'all', label: 'All versions' },
      { value: 2, label: 'Version 2, current' },
      { value: 1, label: 'Version 1' },
    ])
  })
  it('reads the choice from the address, falling back to all versions', () => {
    expect(parseVersion('1', versions)).toBe(1)
    expect(parseVersion(['2'], versions)).toBe(2)
    expect(parseVersion('3', versions)).toBe('all')
    expect(parseVersion('x', versions)).toBe('all')
    expect(parseVersion(undefined, versions)).toBe('all')
    expect(parseVersion(null, versions)).toBe('all')
    expect(parseVersion([null], versions)).toBe('all')
  })
})

describe('the notes under a chart', () => {
  it('notes each break, then each partial run that has a point', () => {
    expect(chartNotes(breaksOf(versions, runs), runs, LANGUAGE_MEASURES)).toEqual([
      '27 October 2026: The questions changed here (prompt set version 2), so the figures before and after are not the same measure.',
      partialNote(runA),
    ])
  })
  it('leaves out a partial run without a point on the chart', () => {
    expect(chartNotes([], runs, [{ label: 'English', tone: 'violet', pick: (r) => r.english }])).toEqual([])
  })
})

describe('the figures as a table', () => {
  it('gives a row per run, oldest first, with a value per measure', () => {
    expect(tableRows(runs, LANGUAGE_MEASURES)).toEqual([
      { x: '2026-10-27T06:03:12Z', when: '27 October 2026, 06:03', partial: true, values: [0.4, 0.48, undefined, undefined] },
      { x: '2026-10-27T08:36:32Z', when: '27 October 2026, 08:36', partial: false, values: [0.25, 0.41, undefined, 0.1] },
    ])
  })
  it('names the languages the questions were asked in', () => {
    expect(LANGUAGE_MEASURES.map((m) => [m.label, m.tone, m.strong ?? false])).toEqual([
      ['All', 'orange', true],
      ['Dutch', 'blue', false],
      ['German', 'aqua', false],
      ['English', 'violet', false],
    ])
    expect(LANGUAGE_MEASURES.map((m) => m.pick(runB))).toEqual([0.25, 0.41, undefined, 0.1])
  })
})

describe('the lines a chart draws', () => {
  it('gives a line per measure, in its tone, the strong one marked', () => {
    expect(chartSeries(runs, LANGUAGE_MEASURES.slice(0, 2))).toEqual([
      { label: 'All', tone: 'orange', strong: true, points: [{ x: '2026-10-27T06:03:12Z', y: 0.4, partial: true }, { x: '2026-10-27T08:36:32Z', y: 0.25, partial: false }] },
      { label: 'Dutch', tone: 'blue', points: [{ x: '2026-10-27T06:03:12Z', y: 0.48, partial: true }, { x: '2026-10-27T08:36:32Z', y: 0.41, partial: false }] },
    ])
  })
})
