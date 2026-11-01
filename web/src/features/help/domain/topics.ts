import type { PromptCadence } from '../../../infrastructure/api/types.gen'
import { BUYER_INTENT } from '../../../shared/tiers'
import { TIERS } from '../../../shared/vocabulary'
import { yearUsd } from '../../manage/domain/estimate'
import { costLabel } from '../../prompt-test/domain/results'

/** The nine groups of Help, in the spec's order. */
export type GroupId = 'start' | 'results' | 'prompt' | 'websites' | 'terms' | 'competitors' | 'recommendations' | 'costs' | 'words'

export const GROUPS: readonly { id: GroupId; label: string }[] = [
  { id: 'start', label: 'Getting started' },
  { id: 'results', label: 'Reading the results' },
  { id: 'prompt', label: 'The Prompt test' },
  { id: 'websites', label: 'Websites and AI readiness' },
  { id: 'terms', label: 'Search terms' },
  { id: 'competitors', label: 'Competitors' },
  { id: 'recommendations', label: 'Recommendations' },
  { id: 'costs', label: 'Costs and budgets' },
  { id: 'words', label: 'Words used in this tool' },
]

/** Where a topic sends you to do the thing it explains. A link into Manage is for administrators only. */
export type TopicLink = { label: string; to: string; adminOnly?: boolean }

/** A figure a topic says from the app's own estimates: what a Prompt run costs, or a year of them. */
export type EstimateKind = 'run' | 'year' | 'cost'

/** One sentence of a topic, or the place of a figure from the live estimates. */
export type Sentence = string | { estimate: EstimateKind }

export type Topic = {
  /** Its address: `/help/:id`. */
  id: string
  group: GroupId
  title: string
  body: readonly Sentence[]
  links: readonly TopicLink[]
  /** More words a search finds it by, which the topic itself does not say. */
  keywords: string
}

/** The live estimates the figures in topics come from: the Prompt test's prices, for administrators. */
export type Estimates = { runUsd: number; cadence: PromptCadence }

const topic = (id: string, group: GroupId, title: string, body: readonly Sentence[], links: readonly TopicLink[], keywords: string): Topic => ({
  id,
  group,
  title,
  body,
  links,
  keywords,
})

const manage = (label: string, to: string): TopicLink => ({ label, to, adminOnly: true })

/** The glossary: each word, one sentence, and more words a search finds it by. */
const WORDS: readonly [string, string, string?][] = [
  ['Prompt test', 'The part of the tool that asks AI assistants the buyer questions, then checks what they say about Tribelt and its competitors.'],
  ['Goal', 'The number Tribelt wants to reach on one measure, such as the share of AI responses that name Tribelt.'],
  ['Search term', 'Words that buyers type when they look for metal conveyor belts.'],
  ['Fact sheet', 'What is true about Tribelt. Every response about Tribelt is checked against it.'],
  ['Competitor definition', 'The written rule that decides which companies count as competitors.'],
  ['Competitor candidate', 'A company that an AI response names but that is not yet on the competitor list.'],
  ['Activity', 'The list in Manage of everything that is running, each with its own Stop button.'],
  ['Results', 'What readers see, with the figures each process found.'],
  ['Manage', 'Where administrators start, stop, and change the work behind the results.'],
  ['Prompt run', 'One execution of the Prompt test: every question, to every model, from the model’s own knowledge and with web search.'],
  ['Partial run', 'A Prompt run that got fewer than half its responses.'],
  ['Prompt set version', 'One published Prompt set. Publishing makes a new version, and the charts mark a break where the version changes.'],
  ['Prompt set', 'The buyer questions a Prompt run asks. Each publication is a new version.'],
  ['Model list', 'The models a Prompt run asks, chosen from OpenRouter’s catalogue.'],
  ['Claude step', 'One piece of work a Claude model does, such as reading responses or matching products.'],
  ['Scope', 'Where and in which language a question asks: a region in its language, or a language with no region.'],
  ['Response', 'One model’s reply to one question, with what it names and cites.'],
  ['AI share of voice', 'The share of a Prompt run’s responses that name a company. If 30 of 100 responses name Tribelt, its share is 30 percent.'],
  ['Citation share', 'The share of the responses with web search that cite a company’s web address. If 10 of 50 such responses cite tribelt.nl, its share is 20 percent.'],
  ['Scan', 'The reading of one website: its pages, search terms, belts, and AI readiness.'],
  ['AI readiness', 'What a website tells search engines and AI assistants about itself, scored out of eight checks.'],
  ['Competitor', 'A company that makes metal conveyor belts Tribelt’s buyers could choose instead.'],
  ['Best competitor', 'The competitor with the best value on a measure in the latest data.'],
  ['Typical competitor', 'The middle value across all competitors on the list.', 'median'],
  ['Gap', 'A measure where the best competitor is ahead of Tribelt.'],
  ['Recommendation', 'One change to make on tribelt.nl, ranked by its benefit against its effort.'],
  ['Recommendation run', 'The final step that writes specific recommendations from the latest data.'],
  [
    'Haiku, Sonnet, and Opus',
    'AI models from Anthropic, called Claude. The tool uses them to read pages and responses and to write recommendations. Haiku is the cheapest and fastest, Sonnet is more careful, and Opus is the most careful and costs the most. Each step can use a different one.',
    'anthropic claude model haiku sonnet opus',
  ],
  ['Buyer intent', 'How far along a buyer is when they ask, from knowing the exact belt to a general search. See Buyer intent under Reading the results.', 'intent tier'],
]

const CHECKS: readonly [string, string][] = [
  ['AI crawlers let in', 'robots.txt lets search and AI crawlers read the site, so assistants can find and learn its pages.'],
  ['A sitemap', 'A sitemap lists every page, so a crawler finds pages that no link leads to.'],
  ['A sitemap in every language', 'The sitemap lists the pages in all three languages, so crawlers find each language version.'],
  ['Pages in Dutch, German, and English', 'The site has pages in Dutch, German, and English, the languages of the buyers in scope.'],
  [
    'Structured data about the company',
    'The home page says in structured data who the company is, where it is, and what it makes, so an assistant reads it as facts.',
  ],
  ['llms.txt', 'llms.txt is a short guide to the site for language models, with links to the pages that matter. Having it helps.'],
  ['llms-full.txt', 'llms-full.txt is the whole site as one plain text file, so an assistant can read it in one go. Having it as well as llms.txt helps more.'],
  ['Markdown pages', 'Each page is also offered as Markdown. A language model reads Markdown more reliably than a designed web page.'],
]

const GOALS: readonly [string, string][] = [
  ['AI responses that name Tribelt', "The share of a Prompt run's responses that name Tribelt."],
  ['AI responses about the Twente region name Tribelt', "The share of a Prompt run's responses about the Twente region that name Tribelt."],
  ['Web responses that cite tribelt.nl', "The share of a Prompt run's responses with web search that link to a tribelt.nl page."],
  ['AI readiness checks passed', 'How many of the eight AI readiness checks tribelt.nl passes.'],
  ['Search terms tribelt.nl uses', 'How many search terms on the list appear on tribelt.nl.'],
]

const slug = (s: string) => s.toLowerCase().replace(/[^a-z]+/g, '-')

/**
 * Every topic of Help, the reviewed prototype's copy, as far as the tool does it today.
 */
export const TOPICS: readonly Topic[] = [
  // Getting started
  topic(
    'what-is',
    'start',
    'What the Findability Explorer does',
    [
      'It measures how easily buyers find Tribelt when they search or ask an AI assistant which metal conveyor belt to use and who makes it.',
      'It does this with four processes: asking AI assistants the buyer questions, checking websites, comparing the search terms buyers use, and writing recommendations for tribelt.nl.',
    ],
    [{ label: 'See the Overview', to: '/' }],
    'about purpose overview',
  ),
  topic(
    'results-manage',
    'start',
    'Results and Manage',
    [
      'Results is what readers see: what the processes found. Manage is only for administrators. There you start, stop, tune, and pay for the work.',
      'Every Results page links to its Manage page for administrators, and every Manage page links back.',
    ],
    [manage('Open Manage', '/manage')],
    'admin administrator view reader',
  ),
  topic(
    'first-setup',
    'start',
    'Set the tool up for the first time',
    [
      'Add an OpenRouter key to pay for the Prompt test, and an Anthropic key to pay for reading responses, product matching, the term review, and the Recommendation runs.',
      'Then set a monthly budget for each key. Work stops by itself when its budget is spent.',
    ],
    [manage('Keys and spending', '/manage/keys'), manage('Budgets', '/manage/keys/budgets')],
    'key api openrouter anthropic install handover budget',
  ),
  topic(
    'fresh',
    'start',
    'How old the figures are',
    [
      'Every page says under its title where its figures come from and when. The Overview lists the latest runs and scans, and when the next Prompt run is due.',
    ],
    [{ label: 'Latest runs and scans', to: '/' }],
    'date age refresh updated',
  ),

  // Reading the results
  topic(
    'goals',
    'results',
    'The five goals',
    GOALS.map(([name, purpose]) => `${name}. ${purpose}`),
    [{ label: 'See the goals', to: '/' }, manage('Set the goals', '/manage/recommendations/goals')],
    'goal target numbers to watch progress',
  ),
  topic(
    'best-competitor',
    'results',
    'The best competitor and the typical competitor',
    [
      'The best competitor has the best value on a measure in the latest data. A tie goes to the name first alphabetically. Tribelt itself is never counted.',
      'The typical competitor is the middle value across all competitors. Half score higher, and half score lower. If competitors name Tribelt in 10, 20, and 60 percent of responses, the typical competitor is at 20 percent.',
    ],
    [{ label: 'See the competitors', to: '/competitors' }],
    'median best gap ahead leader',
  ),
  topic(
    'buyer-intent',
    'results',
    'Buyer intent',
    [
      'Buyer intent is how far along a buyer is when they ask. Every question and search term falls into one of five kinds.',
      ...TIERS.map((t) => `${BUYER_INTENT[t].label}. ${BUYER_INTENT[t].help}`),
    ],
    [
      { label: 'See the search terms', to: '/terms' },
      { label: 'See the AI responses', to: '/ai-responses' },
    ],
    'intent buyer intent kind tier intent tier specification product application problem broad',
  ),
  topic(
    'breaks',
    'results',
    'Why a trend line has a gap',
    [
      'A line breaks where the questions, the model list, or the model that reads the responses changed. The figures before and after are not the same measure, so the line is never joined. A note under the chart says what changed.',
    ],
    [{ label: 'A goal over time', to: '/goals/aiShareOfVoice' }],
    'trend chart version history',
  ),
  topic(
    'read-response',
    'results',
    'Read a response from an AI assistant',
    [
      'Every response opens on its own page, with the full text, the companies it named, the sources it cited, and how it compares with the fact sheet.',
      'The comparison has two parts. Wrong about Tribelt lists the claims that contradict the fact sheet. Left out lists the facts about Tribelt the response could have said and did not.',
      'In Every response, filter by search mode, scope, belt family, or model, and pick a square.',
    ],
    [{ label: 'Every response', to: '/ai-responses/all' }],
    'grid square response text csv download answer wrong claims left out fact sheet',
  ),

  // The Prompt test
  topic(
    'prompt-test',
    'prompt',
    'What the Prompt test does',
    [
      'A Prompt run asks every question in the Prompt set to every model on the Model list, once from the model’s own knowledge and once with web search. Haiku reads each response for the companies it names and the sources it cites.',
      { estimate: 'run' },
    ],
    [{ label: 'See the AI responses', to: '/ai-responses' }, manage('Manage the Prompt test', '/manage/prompt-test')],
    'prompt run ai share of voice chatgpt gemini',
  ),
  topic(
    'start-run',
    'prompt',
    'Start a Prompt run',
    [
      'Start one in Manage, Prompt test. The button shows what the run will cost and how much of the OpenRouter budget is left. If the run would cost more than is left, it asks for a second click.',
    ],
    [manage('Start a Prompt run', '/manage/prompt-test')],
    'run now begin',
  ),
  topic(
    'stop',
    'prompt',
    'Stop anything that is running',
    [
      'Activity lists everything that is running, each with its own Stop button. Stop first says what it keeps, for example the responses a Prompt run has so far. You can finish a stopped Prompt run later with a retry.',
      'Work you stop is marked Stopped by hand. Work its budget stops is marked Stopped by the budget.',
    ],
    [manage('Activity', '/manage'), manage('Prompt runs', '/manage/prompt-test/runs')],
    'cancel halt abort stop',
  ),
  topic(
    'questions',
    'prompt',
    'Change the questions',
    [
      'Edit the draft in Manage, Prompt test, Questions. Press Add a question under a belt family to write a new one in place. Nothing changes until you publish. Publishing makes a new version and says how much more or less a run will cost.',
      'Every earlier version can be restored into the draft, and publishing it makes a new version, so the history is never lost. The charts mark a break where the questions changed.',
    ],
    [manage('Questions', '/manage/prompt-test/questions')],
    'prompt set edit publish version restore add question draft',
  ),
  topic(
    'language',
    'prompt',
    'Ask in another language, such as French or Polish',
    [
      'In Manage, Prompt test, Questions, press Add a question under a belt family and pick A new scope: a language, with or without a region. The tool then offers to add a question in that scope for every other belt family. Edit the wording, and publish.',
    ],
    [manage('Questions', '/manage/prompt-test/questions')],
    'scope french polish region translate',
  ),
  topic(
    'models',
    'prompt',
    'Choose which models are asked',
    [
      'Manage, Prompt test, Models shows OpenRouter’s catalogue. The recommended models come first: the cheapest of the five providers whose assistants buyers use most. A provider can have several models, and each one gets its own column in the results.',
      'The new cost of a run shows before you save. Saving marks the date on the trend charts, so results from before and after are not compared.',
    ],
    [manage('Models', '/manage/prompt-test/models')],
    'model provider gemini gpt claude catalogue free recommended model list',
  ),
  topic(
    'schedule',
    'prompt',
    'Change how often the test runs',
    ['Choose monthly or weekly in Manage, Prompt test, Schedule.', { estimate: 'year' }],
    [manage('Schedule', '/manage/prompt-test/schedule')],
    'schedule cadence monthly weekly automatic',
  ),
  topic(
    'fact-sheet',
    'prompt',
    'Correct the fact sheet',
    [
      'The fact sheet says what is true about Tribelt. Every response about Tribelt is checked against it, and claims that contradict it are marked wrong. Each saved fact sheet is a version.',
    ],
    [manage('Fact sheet', '/manage/prompt-test/fact-sheet')],
    'facts wrong claims check',
  ),

  // Websites and AI readiness
  topic(
    'readiness',
    'websites',
    'The eight AI readiness checks',
    CHECKS.map(([name, why]) => `${name}. ${why}`),
    [{ label: 'See AI readiness', to: '/ai-readiness' }],
    'robots llms sitemap markdown structured data crawler',
  ),
  topic(
    'scan',
    'websites',
    'Scan tribelt.nl after the website changes',
    [
      'A scan reads up to 60 pages of a site, finds its search terms and belts, and checks its AI readiness. Reading pages is free. The goals and recommendations use the newest scan of tribelt.nl.',
    ],
    [manage('Website scans', '/manage/scans')],
    'scan crawl website pages',
  ),
  topic(
    'rescan',
    'websites',
    'Scan a competitor, or all of them',
    [
      'Scan one site, or rescan every competitor, those not scanned in 30 days, or those never scanned. Each choice shows how many scans it starts, what they cost, and how long they take.',
      'Waiting scans can be removed from the queue, one at a time or all at once.',
    ],
    [manage('Website scans', '/manage/scans'), manage('Scan queue', '/manage/scans/queue')],
    'rescan queue remove competitor',
  ),

  topic(
    'scan-keeps',
    'websites',
    'What a scan keeps',
    [
      'A scan keeps the text of the site’s sitemap and, for each search term on the list, the pages that use it.',
      'A site’s files page shows the sitemap, and a search term’s page lists where each company uses it.',
    ],
    [
      { label: 'See AI readiness', to: '/ai-readiness' },
      { label: 'Every term', to: '/terms/all' },
    ],
    'sitemap pages per term files evidence where used',
  ),

  // Search terms
  topic(
    'terms',
    'terms',
    'What the search term list is',
    [
      'The search terms buyers use for metal conveyor belts, in Dutch, German, and English. A page that uses a buyer’s words is more likely to be found for them.',
      'Scans find new terms on websites. The term review decides which of them join the list.',
    ],
    [{ label: 'See the search terms', to: '/terms' }],
    'keywords terms list language phrase',
  ),
  topic(
    'who-uses',
    'terms',
    'See who uses a search term',
    ['Every term has a page that shows which competitors use it, on which of their pages, and whether tribelt.nl does.'],
    [{ label: 'Every term', to: '/terms/all' }],
    'keyword competitor pages usage',
  ),
  topic(
    'add-term',
    'terms',
    'Add or remove a search term by hand',
    [
      'Sales staff often know words buyers use that no scan has found. Add them in Manage, Search term list, Add or remove terms. A term added by hand is marked as such, and scans look for it from the next scan on.',
      'A term can be taken off the list on the same page.',
    ],
    [manage('Add or remove terms', '/manage/terms/edit')],
    'add remove keyword manual by hand',
  ),
  topic(
    'term-review',
    'terms',
    'Review new terms, or switch the review off',
    [
      'The term review runs by itself after each scan and costs a little Anthropic budget. Switch it off, start it by hand, or accept and reject terms yourself.',
    ],
    [manage('Term review', '/manage/terms/review'), manage('Waiting terms', '/manage/terms/waiting')],
    'review terms keywords accept reject automatic',
  ),

  // Competitors
  topic(
    'competitors',
    'competitors',
    'Who counts as a competitor',
    [
      "A competitor makes metal conveyor belts Tribelt's buyers could choose instead. Makers of plastic or fabric belts and Tribelt's group companies are not competitors.",
      'The rule is written down as the competitor definition, which Haiku checks companies against.',
    ],
    [{ label: 'See the competitors', to: '/competitors' }, manage('Competitor definition', '/manage/competitors/definition')],
    'rival plastic fabric group',
  ),
  topic(
    'candidates',
    'competitors',
    'Add or dismiss companies that AI responses name',
    [
      'AI responses keep naming companies that are not on the list. Add the ones that compete and dismiss the rest, or let Haiku check them first. You can undo every decision.',
    ],
    [manage('Candidates', '/manage/competitors/candidates')],
    'candidate competitor add remove dismiss review',
  ),

  topic(
    'definition',
    'competitors',
    'Change the competitor definition',
    [
      'Edit it in Manage, Competitor list. Each save is a version that can be opened and restored. Saving does not recheck the companies; it offers a recheck with its cost.',
    ],
    [manage('Competitor definition', '/manage/competitors/definition')],
    'definition rule version restore',
  ),

  // Recommendations
  topic(
    'recommendations',
    'recommendations',
    'How the recommendations are ranked',
    [
      'Every recommendation has a benefit and an effort from 1 to 5. The list is ranked by benefit divided by effort. Some are written by hand and apply in general. The rest come from a Recommendation run and name a page, a search term, or a site.',
    ],
    [{ label: 'See the recommendations', to: '/recommendations' }],
    'rank priority benefit effort',
  ),
  topic(
    'rec-run',
    'recommendations',
    'Write new recommendations',
    [
      'A Recommendation run uses the newest Prompt run, scans, and search term list. Rules find where tribelt.nl falls behind and score each gap; a language model reads tribelt.nl to say where each change belongs and words it. It may move a score by one point, and must say why.',
      'Start one in Manage, Recommendations. Before it starts, the page says what it reads, how many open recommendations it replaces, and what it costs. When tribelt.nl was last scanned before the newest Prompt run, it warns that the page text is older than the responses: scan tribelt.nl first, or start the run anyway.',
      'A new run replaces the open generated recommendations, and keeps the ones marked done or edited. A run that is stopped, runs out of budget, or fails changes nothing.',
    ],
    [manage('Recommendation runs', '/manage/recommendations'), manage('Writing recommendations', '/manage/claude-steps/recommendations')],
    'generate run model write replace keep stale page text',
  ),
  topic(
    'done',
    'recommendations',
    'When a recommendation counts as done',
    [
      'Rules decide, never a model, from the newest scan of tribelt.nl and the newest Prompt run.',
      'A search term is done when the page it names uses it, or any page when it names none. A new page is done when tribelt.nl has a page about the belt family in that language. A listing is done when a web response that cites the site names Tribelt or links to tribelt.nl. An AI readiness check is done when it passes.',
      'A fact sheet correction no scan can see, so you tick it off yourself, as you do a hand-written recommendation no scan can check. A wrong click can be undone.',
    ],
    [manage('Recommendations to tick off yourself', '/manage/recommendations/tick-off')],
    'done tick complete check rule undo',
  ),
  topic(
    'edit-recommendation',
    'recommendations',
    'Edit a recommendation a run wrote',
    [
      'In Manage, Recommendations, Edit a recommendation, change the title, the page on tribelt.nl, why, the steps, when it counts as done, and the benefit and effort. An edited recommendation is kept when the next run replaces the others.',
      'The hand-written recommendations are part of the tool and cannot be edited.',
    ],
    [manage('Edit a recommendation', '/manage/recommendations/edit')],
    'edit change wording scores generated written',
  ),

  // Costs and budgets
  topic(
    'costs',
    'costs',
    'What things cost',
    [
      { estimate: 'cost' },
      'Readiness checks are free. Every Start button shows its own estimate first.',
      'Activity shows this month’s spending by kind of work and by key.',
    ],
    [manage('This month’s spending', '/manage')],
    'price money dollars estimate',
  ),
  topic(
    'budgets',
    'costs',
    'Change the monthly budgets',
    [
      'There are two budgets in dollars, one for each key: OpenRouter for the Prompt test, and Anthropic for everything Haiku or Sonnet does.',
      'The Prompt test’s budget is set in Manage, Prompt test, Schedule, and the Anthropic budget in Keys and spending, Budgets.',
    ],
    [manage('Prompt test budget', '/manage/prompt-test/schedule'), manage('Budgets', '/manage/keys/budgets')],
    'budget limit monthly dollars',
  ),
  topic(
    'over-budget',
    'costs',
    'What happens when a budget runs out',
    ['Work that would cost more than is left asks for a second click before it starts. Running work stops when its budget is spent, and keeps what it has done.'],
    [manage('Activity', '/manage')],
    'over budget limit stop',
  ),

  topic(
    'claude-steps',
    'costs',
    'Change a Claude model or its instructions',
    [
      'Six steps ask a Claude model: reading responses, matching products, tidying candidate names, checking companies, reviewing search terms, and writing recommendations. Each has a page in its own part of Manage, and Keys and spending lists all six.',
      'On a step’s page, pick the model and edit the instructions. The page shows what the step costs with each model, what the model gets with the instructions and where that comes from, and the parts it must reply with. Saving makes a new version, and every version can be restored.',
      'The data and the reply format are set by the program, so editing the instructions cannot break a step. The instructions for reading responses are set in the code, so every Prompt run is read the same way. For that step, only the model is chosen.',
      'Changing how responses are read changes the measure, so the AI responses charts mark the date.',
    ],
    [manage('Claude models and instructions', '/manage/claude-steps')],
    'prompt instructions model claude haiku sonnet opus change cost anthropic step version restore',
  ),

  // Words used in this tool
  ...WORDS.map(([word, meaning, keywords = '']) => topic(`word-${slug(word)}`, 'words', word, [meaning], [], `glossary word meaning definition ${keywords}`.trim())),
]

/** A topic's sentences as they read, each figure from the live estimates; a figure is left out without them. */
export function topicBody(topic: Topic, estimates: Estimates | undefined): string[] {
  return topic.body.flatMap((s) => {
    if (typeof s === 'string') return [s]
    return estimates ? [estimateSentence(s.estimate, estimates)] : []
  })
}

function estimateSentence(kind: EstimateKind, e: Estimates): string {
  switch (kind) {
    case 'run':
      return `A run costs about ${costLabel(e.runUsd)} today. It runs by itself once a ${e.cadence === 'weekly' ? 'week' : 'month'}.`
    case 'year':
      return `On the current schedule the Prompt test costs about ${costLabel(yearUsd(e.runUsd, e.cadence))} a year.`
    case 'cost':
      return `A Prompt run costs about ${costLabel(e.runUsd)}.`
  }
}

const haystack = new Map(
  TOPICS.map((t) => [t, [t.title, ...t.body.filter((s) => typeof s === 'string'), t.keywords, ...t.links.map((l) => l.label)].join(' ').toLowerCase()]),
)

/**
 * The topics that have every word of q in their title, sentences, keywords, or links, in one group
 * or in all; every topic of the group for no words.
 */
export function searchTopics(q: string, group: GroupId | 'all'): Topic[] {
  const words = q.toLowerCase().split(/\s+/).filter(Boolean)
  return TOPICS.filter((t) => (group === 'all' || t.group === group) && words.every((w) => haystack.get(t)?.includes(w)))
}

/** How many topics match q in each group, and in all. */
export function groupCounts(q: string): Record<GroupId | 'all', number> {
  const found = searchTopics(q, 'all')
  const counts = { all: found.length } as Record<GroupId | 'all', number>
  for (const g of GROUPS) counts[g.id] = found.filter((t) => t.group === g.id).length
  return counts
}

/** Topics under their groups, in the order of the groups, leaving out a group with none. */
export function topicsByGroup(topics: readonly Topic[]): { group: (typeof GROUPS)[number]; topics: Topic[] }[] {
  return GROUPS.map((group) => ({ group, topics: topics.filter((t) => t.group === group.id) })).filter((g) => g.topics.length > 0)
}

/** The prototype's topic names, which earlier links may carry, to the ones Help has now. */
const RENAMED: Record<string, string> = {
  targets: 'goals',
  leader: 'best-competitor',
  intent: 'buyer-intent',
  'read-answer': 'read-response',
  websites: 'readiness',
}

const isGroup = (id: string): id is GroupId => GROUPS.some((g) => g.id === id)

/**
 * What `/help/:topic` opens: a topic's group with the topic in view, a group's address at that
 * group, and Getting started for no topic or one Help does not have.
 */
export function openAt(param: string | undefined): { group: GroupId; topic?: string } {
  const id = param === undefined ? undefined : (RENAMED[param] ?? param)
  const found = TOPICS.find((t) => t.id === id)
  if (found) return { group: found.group, topic: found.id }
  if (id !== undefined && isGroup(id)) return { group: id }
  return { group: 'start' }
}

/** Text in parts, each marked where it matches the search as typed, in any case. */
export function highlight(text: string, q: string): { text: string; hit: boolean }[] {
  const needle = q.trim()
  if (!needle) return [{ text, hit: false }]
  const pattern = new RegExp(`(${needle.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi')
  return text
    .split(pattern)
    .filter((part) => part !== '')
    .map((part) => ({ text: part, hit: part.toLowerCase() === needle.toLowerCase() }))
}
