import { describe, expect, it } from 'vitest'
import { createMemoryHistory } from 'vue-router'
import { createAppRouter } from '../../../app/router'
import { BUYER_INTENT } from '../../../shared/tiers'
import { TIERS } from '../../../shared/vocabulary'
import { usesOldWord } from '../../../shared/words'
import { GROUPS, groupCounts, highlight, openAt, searchTopics, topicBody, topicsByGroup, TOPICS, type Topic } from './topics'

const topic = (id: string): Topic => {
  const t = TOPICS.find((x) => x.id === id)
  if (!t) throw new Error(`no topic ${id}`)
  return t
}

describe('the Help topics', () => {
  it("has the spec's nine groups, in its order, and every topic belongs to one", () => {
    expect(GROUPS.map((g) => g.label)).toEqual([
      'Getting started',
      'Reading the results',
      'The Prompt test',
      'Websites and AI readiness',
      'Search terms',
      'Competitors',
      'Recommendations',
      'Costs and budgets',
      'Words used in this tool',
    ])
    const ids = GROUPS.map((g) => g.id)
    for (const t of TOPICS) expect(ids, t.id).toContain(t.group)
    for (const g of ids) expect(TOPICS.some((t) => t.group === g), g).toBe(true)
  })

  it('gives every topic its own id, as an address takes it', () => {
    const ids = TOPICS.map((t) => t.id)
    expect(new Set(ids).size).toBe(ids.length)
    for (const id of ids) expect(id).toMatch(/^[a-z0-9]+(-[a-z0-9]+)*$/)
  })

  it('links only to addresses the app has', () => {
    const router = createAppRouter(createMemoryHistory())
    const links = TOPICS.flatMap((t) => t.links.map((l) => ({ id: t.id, ...l })))
    expect(links.length).toBeGreaterThan(20)
    for (const l of links) {
      const matched = router.resolve(l.to).matched
      expect(matched.length, `${l.id}: ${l.to}`).toBeGreaterThan(0)
      expect(matched.at(-1)?.path, `${l.id}: ${l.to}`).not.toBe('/:rest(.*)*')
    }
  })

  it('marks every Manage link as for administrators only', () => {
    for (const t of TOPICS) {
      for (const l of t.links) expect(l.adminOnly === true, `${t.id}: ${l.to}`).toBe(l.to.startsWith('/manage'))
    }
  })

  it("says the tool's words, never the old ones, in what it shows", () => {
    const shown = TOPICS.flatMap((t) => [t.title, ...topicBody(t, { runUsd: 1, monthUsd: 1, cadence: 'monthly' }), ...t.links.map((l) => l.label)])
    expect(shown.filter(usesOldWord)).toEqual([])
    expect(GROUPS.map((g) => g.label).filter(usesOldWord)).toEqual([])
  })

  it('explains every kind of buyer intent in one sentence', () => {
    const body = topicBody(topic('buyer-intent'), undefined)
    for (const t of TIERS) expect(body).toContain(`${BUYER_INTENT[t].label}. ${BUYER_INTENT[t].help}`)
  })

  it('keeps the glossary to the words group, one sentence each, with no links', () => {
    const words = TOPICS.filter((t) => t.group === 'words')
    expect(words.length).toBeGreaterThan(20)
    for (const w of words) {
      expect(w.id).toMatch(/^word-/)
      expect(w.body).toHaveLength(1)
      expect(w.links).toEqual([])
    }
    expect(words.map((w) => w.title)).toEqual(expect.arrayContaining(['Response', 'Search term', 'Buyer intent', 'Best competitor', 'Typical competitor', 'Goal', 'Partial run']))
  })
})

describe('the figures in a topic', () => {
  it('come from the live estimates, and are left out without them', () => {
    const estimates = { runUsd: 1.125, monthUsd: 4.5, cadence: 'weekly' as const }
    expect(topicBody(topic('prompt-test'), estimates)).toContain('A run costs about $1.13 today. It runs by itself once a week.')
    expect(topicBody(topic('prompt-test'), { ...estimates, cadence: 'monthly' })).toContain('A run costs about $1.13 today. It runs by itself once a month.')
    expect(topicBody(topic('schedule'), estimates)).toContain('On the current schedule the Prompt test costs about $54.00 a year.')
    expect(topicBody(topic('costs'), estimates)[0]).toBe('A Prompt run costs about $1.13.')
    for (const id of ['prompt-test', 'schedule', 'costs']) {
      const without = topicBody(topic(id), undefined)
      expect(without.join(' '), id).not.toMatch(/\$/)
      expect(without.length, id).toBeGreaterThan(0)
    }
  })
})

describe('searchTopics', () => {
  it('finds a topic by any word of its title, sentences, keywords, or links, in every group', () => {
    expect(searchTopics('budget', 'all').map((t) => t.title)).toContain('What happens when a budget runs out')
    expect(searchTopics('STOP a run', 'all').map((t) => t.id)).toContain('stop')
    expect(searchTopics('leader', 'all').map((t) => t.id)).toContain('best-competitor')
    expect(searchTopics('intent tier', 'all').map((t) => t.id)).toEqual(expect.arrayContaining(['buyer-intent', 'word-buyer-intent']))
    expect(searchTopics('Waiting terms', 'all').map((t) => t.id)).toContain('term-review')
    expect(searchTopics('nothing like this at all', 'all')).toEqual([])
  })

  it('keeps to one group when asked, and lists every topic of it for no words', () => {
    const costs = searchTopics('', 'costs')
    expect(costs.length).toBeGreaterThan(0)
    expect(costs).toEqual(TOPICS.filter((t) => t.group === 'costs'))
    expect(searchTopics('   ', 'all')).toEqual([...TOPICS])
    expect(searchTopics('budget', 'start').every((t) => t.group === 'start')).toBe(true)
  })

  it('counts the matches in each group, and in all', () => {
    const counts = groupCounts('budget')
    expect(counts.all).toBe(searchTopics('budget', 'all').length)
    for (const g of GROUPS) expect(counts[g.id]).toBe(searchTopics('budget', g.id).length)
    expect(groupCounts('').all).toBe(TOPICS.length)
  })

  it('groups topics in the order of the groups, leaving out empty ones', () => {
    const grouped = topicsByGroup(searchTopics('budget', 'all'))
    const order = GROUPS.map((g) => g.id)
    const at = grouped.map((g) => order.indexOf(g.group.id))
    expect(at).toEqual([...at].sort((a, b) => a - b))
    expect(grouped.length).toBeGreaterThan(1)
    expect(grouped.every((g) => g.topics.length > 0)).toBe(true)
  })
})

describe('openAt', () => {
  it("opens a topic's address at its group with the topic, and a group's at the group; a topic wins over a group of its name", () => {
    expect(openAt('buyer-intent')).toEqual({ group: 'results', topic: 'buyer-intent' })
    expect(openAt('costs')).toEqual({ group: 'costs', topic: 'costs' })
    expect(openAt('words')).toEqual({ group: 'words' })
  })

  it("takes the prototype's old topic names to the new ones", () => {
    expect(openAt('targets')).toEqual({ group: 'results', topic: 'goals' })
    expect(openAt('leader')).toEqual({ group: 'results', topic: 'best-competitor' })
    expect(openAt('intent')).toEqual({ group: 'results', topic: 'buyer-intent' })
    expect(openAt('read-answer')).toEqual({ group: 'results', topic: 'read-response' })
    expect(openAt('websites')).toEqual({ group: 'websites', topic: 'readiness' })
  })

  it('opens Getting started for no topic, or one Help does not have', () => {
    expect(openAt(undefined)).toEqual({ group: 'start' })
    expect(openAt('no-such-topic')).toEqual({ group: 'start' })
  })
})

describe('highlight', () => {
  it('splits text where the search matches, in any case', () => {
    expect(highlight('A budget, and Budgets', 'budget')).toEqual([
      { text: 'A ', hit: false },
      { text: 'budget', hit: true },
      { text: ', and ', hit: false },
      { text: 'Budget', hit: true },
      { text: 's', hit: false },
    ])
    expect(highlight('Costs (in $)', '(in $')).toEqual([
      { text: 'Costs ', hit: false },
      { text: '(in $', hit: true },
      { text: ')', hit: false },
    ])
  })

  it('leaves text whole for no search, or none found', () => {
    expect(highlight('Help', '  ')).toEqual([{ text: 'Help', hit: false }])
    expect(highlight('Help', 'zzz')).toEqual([{ text: 'Help', hit: false }])
  })
})
