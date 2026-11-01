import type { Competitor, PromptAnswer, Term } from '../../../infrastructure/api/types.gen'
import { PRODUCT_FAMILIES } from '../../../shared/vocabulary'
import { oxford } from '../../../shared/words'
import type { Measure } from './compare'
import { scanDate } from './table'

/** How a sentence names the belts of a family a company sells. */
const SELLS: Record<(typeof PRODUCT_FAMILIES)[number], string> = {
  eyelink: 'eyelink',
  open_spiral: 'open spiral',
  closed_spiral: 'closed spiral',
  tri_flex_turn_curve: 'Tri-Flex',
  other: 'other metal',
}

/** A competitor's purpose sentence: its site, and the belt families it sells. */
export function competitorPurpose(c: Pick<Competitor, 'host' | 'website' | 'beltNames'>): string {
  const families = PRODUCT_FAMILIES.filter((f) => c.beltNames.some((b) => b.family === f))
  const sells = families.length > 0 ? `It sells ${oxford(families.map((f) => SELLS[f]))} belts.` : 'No belt names are recorded for it yet.'
  return `${c.host ?? c.website}, on the competitor list. ${sells}`
}

/** Where a competitor page's figures come from: the newest Prompt run, and the company's newest scan. */
export function detailDataLine(name: string, runAt: string | undefined, scanAt: string | undefined): string {
  const run = runAt === undefined ? 'No Prompt run has finished yet' : `From the Prompt run of ${scanDate(runAt)}`
  const scan = scanAt === undefined ? `${name} has not been scanned yet` : `the scan of ${name} on ${scanDate(scanAt)}`
  return `${run}${runAt === undefined ? ', and ' : ' and '}${scan}.`
}

/** The measures on which the competitor is ahead of Tribelt, a sentence each. */
export function aheadLines(measures: readonly Measure[]): string[] {
  return measures.filter((m) => m.them !== undefined && m.them > (m.us ?? -1)).map((m) => `${m.label}: ${m.themText}, against Tribelt’s ${m.usText}.`)
}

/** The terms a competitor uses and tribelt.nl does not, the most used first: the first few, and how many in all. */
export function theirTerms<T extends Pick<Term, 'term' | 'usedBy' | 'tribeltUses' | 'makers'>>(terms: readonly T[], id: string, count = 10): { shown: T[]; total: number } {
  const theirs = terms
    .filter((t) => t.tribeltUses !== true && t.makers?.some((m) => m.competitorId === id) === true)
    .sort((a, b) => b.usedBy.length - a.usedBy.length || a.term.localeCompare(b.term))
  return { shown: theirs.slice(0, count), total: theirs.length }
}

/** A bar of how often one model names the competitor. */
export type ModelBar = { label: string; value: number; display: string; tone: 'blue-grey'; highlight: false }

/** For each model of the run, the share of its read responses that name the competitor. */
export function modelShares(answers: readonly Pick<PromptAnswer, 'model' | 'failure' | 'extraction'>[], id: string): ModelBar[] {
  const read = answers.flatMap((a) => (a.failure === undefined && a.extraction !== undefined && !a.extraction.failed ? [{ model: a.model, companies: a.extraction.companies }] : []))
  const models = [...new Set(read.map((a) => a.model))].sort()
  return models.map((model) => {
    const own = read.filter((a) => a.model === model)
    const value = own.filter((a) => a.companies.some((c) => c.competitorId === id)).length / own.length
    return { label: model, value, display: `${String(Math.round(value * 100))}%`, tone: 'blue-grey', highlight: false }
  })
}
