import type {
  AiReadiness,
  AiReadinessRow,
  Analysis,
  AnalysisSummary,
  BotRule,
  Competitor,
  Me,
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
