import type {
  AiReadiness,
  AiReadinessRow,
  AiReadinessTable,
  Analysis,
  AnalysisSummary,
  BotRule,
  Competitor,
  Me,
  PromptAnswer,
  PromptExtraction,
  PromptRun,
  Product,
  PromptTestResults,
  Term,
  TermSearch,
} from '../infrastructure/api/types.gen'

export const ANALYSIS_ID = '5b0c1a9e-2d3f-4e5a-8b6c-7d8e9f0a1b2c'

export function term(overrides: Partial<Term> = {}): Term {
  return {
    term: 'TwenteFlex',
    language: 'en',
    family: 'tri_flex_turn_curve',
    tier: 'product',
    usedBy: ['Twentebelt'],
    source: 'https://www.twentebelt.com/conveyor-belts/',
    origin: 'research',
    ...overrides,
  }
}

export function termSearch(items: Term[], overrides: Partial<TermSearch> = {}): TermSearch {
  return {
    items,
    total: items.length,
    facets: {
      languages: [
        { value: 'de', count: 32 },
        { value: 'en', count: 14 },
      ],
      tiers: [{ value: 'product', count: 20 }],
      families: [{ value: 'eyelink', count: 7 }],
    },
    ...overrides,
  }
}

/** A company's id, made from its name as the server makes it. */
export function slug(name: string): string {
  return name
    .toLowerCase()
    .normalize('NFD')
    .replace(/\p{Mn}/gu, '')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-|-$/g, '')
}

export function competitor(name: string, overrides: Partial<Competitor> = {}): Competitor {
  return {
    id: slug(name),
    source: 'research',
    role: 'competitor',
    websiteUrl: 'https://www.twentebelt.com/',
    host: 'twentebelt.com',
    scans: 0,
    name,
    country: 'NL',
    website: 'https://www.twentebelt.com',
    languages: ['en', 'nl'],
    groupCompany: false,
    beltNames: [{ name: 'TwenteFlex spiral belts', family: 'tri_flex_turn_curve' }],
    homepageTitle: 'unverified',
    archived: false,
    ...overrides,
  }
}

export function summary(overrides: Partial<AnalysisSummary> = {}): AnalysisSummary {
  return {
    id: ANALYSIS_ID,
    competitorId: 'twentebelt',
    url: 'https://www.twentebelt.com',
    status: 'done',
    maxPages: 15,
    pagesCrawled: 15,
    createdAt: '2026-10-22T09:30:00Z',
    finishedAt: '2026-10-22T09:31:10Z',
    ...overrides,
  }
}

export function analysis(overrides: Partial<Analysis> = {}, summaryOverrides: Partial<AnalysisSummary> = {}): Analysis {
  return {
    summary: summary(summaryOverrides),
    pages: [
      {
        url: 'https://www.twentebelt.com/conveyor-belts/',
        title: 'Conveyor belts - Twentebelt',
        description: 'Metal conveyor belts for the food processing industry.',
        language: 'en',
        wordCount: 412,
      },
    ],
    keywords: [
      { phrase: 'spiral belts', language: 'en', score: 12.25, pages: 3, tier: 'product', family: 'tri_flex_turn_curve', closestTerm: 'spiral belt / spiral conveyor belt', inMap: false },
      { phrase: 'metal conveyor belts', language: 'en', score: 42.5, pages: 6, tier: 'broad', family: 'all', closestTerm: 'metal conveyor belt(s)', inMap: true },
      { phrase: 'förderbänder', language: 'de', score: 20, pages: 2, tier: 'broad', inMap: false },
    ],
    knownTerms: [
      { term: 'balanced weave belt', language: 'en', family: 'open_spiral', tier: 'product', occurrences: 9, usedByTribelt: false },
      { term: 'eyelink belt', language: 'en', family: 'eyelink', tier: 'product', occurrences: 2, usedByTribelt: true },
    ],
    matching: 'done',
    equivalents: [
      {
        product: 'TwenteFlex spiral belts',
        family: 'tri_flex_turn_curve',
        reason: 'A curve-going spiral-tower belt, like Tri-Flex.',
        searchTerms: [
          { term: 'spiral belt', language: 'en' },
          { term: 'Spiralband', language: 'de' },
        ],
      },
    ],
    ...overrides,
  }
}

/** The account signed in: not an administrator unless the test says so. */
export function me(overrides: Partial<Me> = {}): Me {
  return { subject: 'joris', name: 'Joris Jonkers', isAdmin: false, ...overrides }
}

const BOT_TOKENS = ['Googlebot', 'Google-Extended', 'Bingbot', 'OAI-SearchBot', 'ChatGPT-User', 'GPTBot', 'Claude-SearchBot', 'ClaudeBot', 'PerplexityBot', 'CCBot']

/** Every crawler's rule: allowed and named, unless access names it otherwise. */
export function bots(access: Record<string, BotRule['access']> = {}, named = true): BotRule[] {
  return BOT_TOKENS.map((bot) => {
    const a = access[bot] ?? 'allowed'
    return { bot, access: a, named, followsAdvice: a !== 'blocked' }
  })
}

/** A site that follows every recommendation, as the mirror does, unless overridden. */
export function readiness(overrides: Partial<AiReadiness> = {}): AiReadiness {
  const r: AiReadiness = {
    robots: 'found',
    bots: bots(),
    sitemap: { declared: true, files: 2, addresses: 124, languages: ['nl', 'de', 'en'] },
    llmsTxt: { present: true, bytes: 5120, links: 124 },
    llmsFullTxt: true,
    markdown: true,
    homeRead: true,
    structuredData: ['Organization', 'Product'],
    languages: ['nl', 'de', 'en'],
    stack: { cms: 'WordPress', seoPlugin: 'Yoast SEO' },
    advice: (['sitemapDeclared', 'sitemapLanguages', 'llmsTxt', 'markdown', 'organization', 'product', 'languages'] as const).map((check) => ({ check, ok: true })),
    gaps: 0,
    ...overrides,
  }
  return r
}

/** A row of the AI readiness table. */
export function readinessRow(name: string, overrides: Partial<AiReadinessRow> = {}): AiReadinessRow {
  return { competitorId: slug(name), name, role: 'competitor', scan: summary({ competitorId: slug(name) }), readiness: readiness(), ...overrides }
}

/** The AI readiness table of rows, with no readiness check run since the server started. */
export function readinessTable(items: AiReadinessRow[], check: Partial<AiReadinessTable['check']> = {}): AiReadinessTable {
  return { items, check: { running: false, checked: 0, failed: 0, total: 0, ...check } }
}

export const RUN_ID = '9d8e7f6a-5b4c-4d3e-8f2a-1b0c9d8e7f6a'

/** A Prompt run of two providers: Tribelt named in Dutch Twente supplier answers, cited with web search. */
export function promptRun(overrides: Partial<PromptRun> = {}): PromptRun {
  return {
    id: RUN_ID,
    promptSetVersion: 1,
    factSheetVersion: 1,
    trigger: 'manual',
    startedBy: 'joris',
    startedAt: '2026-11-01T06:00:00Z',
    finishedAt: '2026-11-01T06:20:00Z',
    status: 'done',
    models: [
      { provider: 'openai', model: 'openai/gpt-oss-20b', free: false },
      { provider: 'google', model: 'google/gemma:free', free: true, fallback: 'google/gemini-2.5-flash-lite' },
    ],
    tasks: 360,
    answered: 358,
    failed: 2,
    inputTokens: 40_000,
    outputTokens: 90_000,
    costUsd: 0.42,
    byProvider: [
      { provider: 'openai', answered: 180, failed: 0, costUsd: 0.3, models: ['openai/gpt-oss-20b'] },
      { provider: 'google', answered: 178, failed: 2, costUsd: 0.12, models: ['google/gemma:free', 'google/gemini-2.5-flash-lite'] },
    ],
    cells: [
      { provider: 'openai', scope: 'twente', family: 'eyelink', mode: 'knowledge', answers: 3, named: 1, cited: 0 },
      { provider: 'openai', scope: 'twente', family: 'eyelink', mode: 'web', answers: 3, named: 2, cited: 1 },
      { provider: 'google', scope: 'germany', family: 'open_spiral', mode: 'knowledge', answers: 3, named: 0, cited: 0 },
      { provider: 'google', scope: 'germany', family: 'open_spiral', mode: 'web', answers: 3, named: 0, cited: 0 },
    ],
    ...overrides,
  }
}

export function promptResults(overrides: Partial<PromptTestResults> = {}): PromptTestResults {
  return {
    promptSetVersion: 1,
    runs: [promptRun()],
    mentions: [
      { competitorId: 'twentebelt', name: 'Twentebelt', role: 'competitor', answers: 9, first: 4, averagePosition: 1.6 },
      { competitorId: 'tribelt', name: 'Tribelt B.V.', role: 'tribelt', answers: 3, first: 1, averagePosition: 2 },
    ],
    wrongClaims: [
      { answerId: '1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d', runId: RUN_ID, at: '2026-11-01T06:10:00Z', provider: 'openai', model: 'openai/gpt-oss-20b', promptId: 'eyelink-supplier-twente', mode: 'knowledge', correctness: 'wrong', claim: 'Tribelt is in Hengelo.' },
    ],
    ...overrides,
  }
}

let answers = 0

/** A read answer of the run; companies are the names in order, a known one as "Name:id". */
export function promptAnswer(
  overrides: Partial<PromptAnswer> = {},
  companies: string[] = ['Twentebelt:twentebelt', 'Tribelt:tribelt'],
  extraction: Partial<PromptExtraction> = {},
): PromptAnswer {
  return {
    id: `00000000-0000-4000-8000-${String((answers += 1)).padStart(12, '0')}`,
    promptId: 'eyelink-application-twente',
    family: 'eyelink',
    kind: 'application',
    scope: 'twente',
    language: 'nl',
    prompt: 'We bouwen een doorloopoven voor een industriële bakkerij. Welke fabrikanten in de regio Twente?',
    promptSetVersion: 2,
    provider: 'openai',
    model: 'openai/gpt-oss-20b',
    mode: 'knowledge',
    text: 'Twentebelt in Hengelo and Tribelt in Haaksbergen make eyelink belts.',
    citations: [],
    at: '2026-11-01T06:10:00Z',
    costUsd: 0.0012,
    inputTokens: 900,
    outputTokens: 300,
    extraction: {
      failed: false,
      companies: companies.map((c, i) => {
        const [name, id] = c.split(':') as [string, string | undefined]
        return { name, position: i + 1, ...(id ? { competitorId: id } : {}) }
      }),
      addresses: [],
      namesTribelt: companies.some((c) => c.endsWith(':tribelt')),
      citesTribelt: false,
      wrongClaims: [],
      costUsd: 0.0002,
      ...extraction,
    },
    ...overrides,
  }
}

/** A belt family against the field: eyelink, with a seller, four terms and 40 answers. */
export function product(overrides: Partial<Product> = {}): Product {
  return {
    family: 'eyelink',
    tribeltNames: ['Draadogenbanden'],
    sellers: [{ competitorId: 'twentebelt', name: 'Twentebelt', names: ['Oogbanden'] }],
    terms: [
      { term: 'draadogenband', language: 'nl', tier: 'product', usedBy: 9, usedByTribelt: false },
      { term: 'eyelink belt', language: 'en-US', tier: 'product', usedBy: 5, usedByTribelt: true },
      { term: 'Drahtösenband', language: 'de', tier: 'product', usedBy: 3, usedByTribelt: false },
      { term: 'bande à maillons', language: 'fr', tier: 'product', usedBy: 1, usedByTribelt: false },
    ],
    termsUsed: 1,
    answers: 40,
    named: 12,
    byScope: [],
    mostNamed: [],
    ...overrides,
  }
}

/** An improvement's details: why, how and when it is done, and no scan parts. */
export const details = {
  why: 'It helps.',
  steps: ['Do the first thing.', 'Do the second thing.'],
  doneWhen: 'It is in place.',
  parts: [] as { name: string; present: boolean }[],
}
