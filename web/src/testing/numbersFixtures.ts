import type { Overview, Plan, Standings } from '../infrastructure/api/types.gen'
import { details } from './fixtures'

const ashworth = { competitorId: 'ashworth', name: 'Ashworth', value: 0.57 }

/** A plan that watches every number: one met, one without data, and improvements for three measures. */
export function planFixture(): Plan {
  return {
    catalogueVersion: 1,
    improvements: [
      { id: 'submit-search-engines', title: 'Submit tribelt.nl to Google and Bing', moves: 'Helps Tribelt appear in Gemini and Copilot answers.', measure: 'aiShareOfVoice', auto: false, done: false, ...details },
      { id: 'llms-files', title: 'Add llms.txt and llms-full.txt', moves: 'Improves AI readiness.', measure: 'readinessChecks', auto: true, done: true, ...details },
      { id: 'cite-pages', title: 'Publish pages worth citing', moves: 'Gets more answers to cite tribelt.nl.', measure: 'citationShare', auto: false, done: false, ...details },
      { id: 'answer-questions', title: 'Answer buyers questions', moves: 'Helps Tribelt appear in more answers.', measure: 'aiShareOfVoice', auto: false, done: false, ...details },
    ],
    watched: [
      { target: 'aiShareOfVoice', now: 0.24, goal: 0.4, max: 1, met: false },
      { target: 'twenteShareOfVoice', now: 0.2, goal: 0.5, max: 1, met: false },
      { target: 'citationShare', now: 0.1, goal: 0.5 },
      { target: 'readinessChecks', now: 3, goal: 8, max: 8, met: false },
      { target: 'termsUsed', now: 45, goal: 35, met: true },
    ],
    tribeltTermsFrom: 'scan',
    readinessChecks: 8,
    eyeOn: [
      { why: 'mostNamed', leader: ashworth },
      { why: 'twente', leader: { competitorId: 'twentebelt', name: 'Twentebelt', value: 0.6 } },
    ],
  }
}

/** Standings that agree with planFixture. */
export function standingsFixture(): Standings {
  return {
    measures: [
      { measure: 'aiShareOfVoice', tribelt: 0.24, median: 0.1, leader: ashworth, competitors: 4, trails: true, leads: false },
      { measure: 'citationShare', tribelt: 0.1, median: 0.05, leader: { competitorId: 'martens', name: 'Märtens', value: 0.3 }, competitors: 3, trails: true, leads: false },
      { measure: 'readinessChecks', tribelt: 3, median: 2, leader: { competitorId: 'martens', name: 'Märtens', value: 8 }, competitors: 4, trails: true, leads: false },
      { measure: 'termsUsed', tribelt: 45, median: 20, leader: { competitorId: 'ashworth', name: 'Ashworth', value: 40 }, competitors: 4, trails: false, leads: true },
    ],
    trailing: 3,
    open: [],
    readinessChecks: 8,
    tribeltTermsFrom: 'scan',
  }
}

/** The overview figures: 30 of 387 terms, 24 competitors, and Tribelt named in 24% of the AI answers. */
export function overviewFixture(): Overview {
  return {
    terms: { total: 387, tribelt: 30, languages: [{ language: 'en', terms: 200, tribelt: 20 }] },
    competitors: { makers: 24, families: [{ family: 'eyelink', makers: 9 }] },
    aiSearch: { answers: 50, companies: [{ competitorId: 'tribelt', role: 'tribelt', name: 'Tribelt', answers: 12, share: 0.24 }] },
  }
}
