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
    const shown = TOPICS.flatMap((t) => [t.title, ...topicBody(t, { runUsd: 1, cadence: 'monthly' }), ...t.links.map((l) => l.label)])
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
    const estimates = { runUsd: 1.125, cadence: 'weekly' as const }
    expect(topicBody(topic('prompt-test'), estimates)).toContain('A run costs about $1.13 today. It runs by itself once a week.')
    expect(topicBody(topic('prompt-test'), { ...estimates, cadence: 'monthly' })).toContain('A run costs about $1.13 today. It runs by itself once a month.')
    expect(topicBody(topic('schedule'), estimates)).toContain('On the current schedule the Prompt test costs about $58.50 a year.')
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

describe('the topics the pages link to', () => {
  // Every page's "How this is measured, in Help" and every other link into Help, read from the source.
  const sources = import.meta.glob(['../../../**/*.vue', '../../../**/*.ts', '!../../../**/*.spec.ts', '!../../help/**'], {
    query: '?raw',
    import: 'default',
    eager: true,
  })
  const linked = new Set<string>()
  for (const text of Object.values(sources)) {
    for (const m of text.matchAll(/help-topic="([a-z0-9-]+)"|helpTopic: '([a-z0-9-]+)'|['"`]\/help\/([a-z0-9-]+)/g)) linked.add(m[1] ?? m[2] ?? m[3] ?? '')
  }
  // A goal page picks its topic by the goal's process.
  for (const id of ['prompt-test', 'readiness', 'terms']) linked.add(id)

  it('finds the links', () => {
    expect([...linked]).toEqual(
      expect.arrayContaining(['goals', 'terms', 'competitors', 'recommendations', 'websites', 'prompt-test', 'rescan', 'questions', 'models', 'claude-steps', 'add-term', 'rec-run', 'edit-recommendation', 'done']),
    )
  })

  it('opens a topic for each of them', () => {
    for (const id of linked) expect(openAt(id).topic, id).toBeDefined()
  })
})

describe('the topics of what Manage now does', () => {
  const to = (id: string) => topic(id).links.map((l) => l.to)
  const text = (id: string) => topicBody(topic(id), undefined).join(' ')

  it('send each Manage link to its own subpage', () => {
    expect(to('goals')).toContain('/manage/recommendations/goals')
    expect(to('schedule')).toEqual(['/manage/prompt-test/schedule'])
    expect(to('fact-sheet')).toEqual(['/manage/prompt-test/fact-sheet'])
    expect(to('stop')).toEqual(['/manage', '/manage/prompt-test/runs'])
    expect(to('rescan')).toEqual(['/manage/scans', '/manage/scans/queue'])
    expect(to('term-review')).toEqual(['/manage/terms/review', '/manage/terms/waiting'])
    expect(to('done')).toEqual(['/manage/recommendations/tick-off'])
    expect(to('first-setup')).toEqual(['/manage/keys', '/manage/keys/budgets'])
  })

  it('explain the two dollar budgets, what happens when one runs out, and where the spending shows', () => {
    expect(topic('budgets').title).toBe('Change the monthly budgets')
    expect(text('budgets')).toContain('There are two budgets in dollars, one for each key')
    expect(to('budgets')).toEqual(['/manage/prompt-test/schedule', '/manage/keys/budgets'])
    expect(text('over-budget')).toContain('asks for a second click before it starts')
    expect(text('costs')).toContain('Activity shows this month’s spending')
    expect(to('costs')).toEqual(['/manage'])
  })

  it('explain the competitor definition and its versions', () => {
    expect(text('definition')).toContain('Each save is a version that can be opened and restored.')
    expect(to('definition')).toEqual(['/manage/competitors/definition'])
    expect(to('competitors')).toContain('/manage/competitors/definition')
  })

  it('explain stopping work, the switch of the term review, and Stopped by hand', () => {
    expect(text('stop')).toContain('Stopped by hand')
    expect(topic('term-review').title).toBe('Review new terms, or switch the review off')
    expect(text('term-review')).toContain('Switch it off')
  })

  it('explain the two checks of a response against the fact sheet', () => {
    expect(text('read-response')).toContain('Wrong about Tribelt')
    expect(text('read-response')).toContain('Left out')
  })

  it('explain what a scan keeps: the sitemap’s text and the pages per term', () => {
    expect(topic('scan-keeps').group).toBe('websites')
    expect(text('scan-keeps')).toContain('sitemap')
    expect(text('scan-keeps')).toContain('the pages that use it')
  })
})

describe('the topics of the questions, the models, the Claude steps, and terms added by hand', () => {
  const to = (id: string) => topic(id).links.map((l) => l.to)
  const text = (id: string) => topicBody(topic(id), undefined).join(' ')

  it('explain editing and publishing the questions, a new scope, and the versions', () => {
    expect(topic('questions').title).toBe('Change the questions')
    expect(text('questions')).toContain('Nothing changes until you publish.')
    expect(text('questions')).toContain('Add a question')
    expect(text('questions')).toContain('restored')
    expect(topic('language').title).toBe('Ask in another language, such as French or Polish')
    expect(text('language')).toContain('A new scope')
    expect(to('questions')).toEqual(['/manage/prompt-test/questions'])
    expect(to('language')).toEqual(['/manage/prompt-test/questions'])
  })

  it('explain the model list: the catalogue, the recommended five, the cost of a run, and the break in the charts', () => {
    expect(topic('models').title).toBe('Choose which models are asked')
    expect(text('models')).toContain('OpenRouter’s catalogue')
    expect(text('models')).toContain('the five providers')
    expect(text('models')).toContain('The new cost of a run shows before you save.')
    expect(text('models')).toContain('trend charts')
    expect(text('breaks')).toContain('the model that reads the responses')
    expect(to('models')).toEqual(['/manage/prompt-test/models'])
  })

  it('explain the Claude steps: model, instructions, versions, and the reading step set in the code', () => {
    expect(topic('claude-steps').title).toBe('Change a Claude model or its instructions')
    expect(text('claude-steps')).toContain('every version can be restored')
    expect(text('claude-steps')).toContain('writing recommendations')
    expect(text('claude-steps')).toContain('For that step, only the model is chosen.')
    expect(text('claude-steps')).toContain('the AI responses charts mark the date')
    expect(to('claude-steps')).toEqual(['/manage/claude-steps'])
  })

  it('explain adding and removing a search term by hand', () => {
    expect(topic('add-term').title).toBe('Add or remove a search term by hand')
    expect(text('add-term')).toContain('marked as such')
    expect(to('add-term')).toEqual(['/manage/terms/edit'])
  })

  it('name the new words in the glossary', () => {
    const words = TOPICS.filter((t) => t.group === 'words').map((t) => t.title)
    expect(words).toEqual(expect.arrayContaining(['Claude step', 'Model list', 'Recommendation run', 'Prompt set version']))
    expect(text('word-model-list')).toBe('The models a Prompt run asks, chosen from OpenRouter’s catalogue.')
  })
})

describe('the topics of the Recommendation run and its recommendations', () => {
  const to = (id: string) => topic(id).links.map((l) => l.to)
  const text = (id: string) => topicBody(topic(id), undefined).join(' ')

  it('explain what a run reads, starting it, what it replaces and keeps, and the warning about old page text', () => {
    expect(topic('rec-run').title).toBe('Write new recommendations')
    expect(text('rec-run')).toContain('uses the newest Prompt run, scans, and search term list')
    expect(text('rec-run')).toContain('Start one in Manage, Recommendations.')
    expect(text('rec-run')).toContain('keeps the ones marked done or edited')
    expect(text('rec-run')).toContain('the page text is older than the responses')
    expect(to('rec-run')).toEqual(['/manage/recommendations', '/manage/claude-steps/recommendations'])
  })

  it('explain how a rule ticks off each kind, and which ones you tick off yourself', () => {
    const done = text('done')
    for (const kind of ['A search term', 'A new page', 'A listing', 'An AI readiness check', 'A fact sheet correction']) expect(done).toContain(kind)
    expect(done).toContain('Rules decide, never a model')
    expect(to('done')).toEqual(['/manage/recommendations/tick-off'])
  })

  it('explain editing a recommendation a run wrote', () => {
    expect(topic('edit-recommendation').title).toBe('Edit a recommendation a run wrote')
    expect(text('edit-recommendation')).toContain('kept when the next run replaces the others')
    expect(to('edit-recommendation')).toEqual(['/manage/recommendations/edit'])
  })

  it('say the ranked list holds the run’s recommendations too', () => {
    expect(text('recommendations')).toContain('The rest come from a Recommendation run')
  })
})
