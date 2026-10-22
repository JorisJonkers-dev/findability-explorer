import type {
  Analysis,
  AnalysisSummary,
  Competitor,
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
