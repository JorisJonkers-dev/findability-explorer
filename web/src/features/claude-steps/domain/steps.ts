import type { ClaudeStep, ClaudeStepId, ClaudeStepModel, ClaudeStepVersion } from '../../../infrastructure/api/types.gen'
import { dollars } from '../../settings/domain/usage'

/** What one part of a step's data is, and where it comes from. */
export type Part = { name: string; what: string; from: string }
/** One field of a step's reply, and what it means. */
export type Field = { field: string; what: string }

/** What the pages say about a step beyond what the server stores: copied from the reviewed prototype. */
export type StepInfo = {
  does: string
  section: { label: string; to: string }
  results: { label: string; to: string }
  jobLabel: string
  gets: readonly Part[]
  replies: readonly Field[]
  change: string
}

/** The steps the server has. Each new step joins here with its words. */
export const STEP_INFO: Record<ClaudeStepId, StepInfo> = {
  matching: {
    does: 'Reads the pages of a scan and matches the products on them to Tribelt’s five belt families.',
    section: { label: 'Website scans', to: '/manage/scans' },
    results: { label: 'See the competitors', to: '/competitors' },
    jobLabel: 'a scan',
    gets: [
      { name: 'Tribelt’s belt families', what: 'The five families, each with its id, name, and a description of how it is built and used.', from: 'Fixed in the tool' },
      { name: 'The website text', what: 'The text of the pages the scan read, cut off at a length limit. The model is told when it was cut.', from: 'The scan' },
    ],
    replies: [
      { field: 'product', what: 'The website’s own name for a product.' },
      { field: 'family', what: 'The Tribelt belt family it corresponds to.' },
      { field: 'reason', what: 'One sentence on why.' },
      { field: 'searchTerms', what: 'The words the website uses for it, each with its language (nl, de, or en).' },
    ],
    change:
      'Scans from now on match the new way. Earlier scans keep their matches, so a scan before and after the change may differ for that reason alone.',
  },
  recommendations: {
    does: 'Reads the gaps the rules found and the tribelt.nl pages, then says where each change belongs and words it.',
    section: { label: 'Recommendations', to: '/manage/recommendations' },
    results: { label: 'See the recommendations', to: '/recommendations' },
    jobLabel: 'a Recommendation run',
    gets: [
      {
        name: 'The gaps',
        what: 'Each place where tribelt.nl falls behind its competitors: its kind, the facts the rules found, and the benefit and effort scores they gave it.',
        from: 'The newest Prompt run, the newest scan of each site, and the search term list',
      },
      { name: 'The tribelt.nl pages', what: 'The address, title, and text of every page.', from: 'The newest scan of tribelt.nl' },
    ],
    replies: [
      { field: 'title, where, why', what: 'What to change, on which page or none, and why.' },
      { field: 'steps', what: 'Two to four steps someone at Tribelt can follow.' },
      { field: 'doneWhen', what: 'What the next scan of tribelt.nl will find when it is done.' },
      { field: 'benefit, effort, scoreReason', what: 'The scores, each moved at most one point from the rules’ score, with a reason when moved.' },
    ],
    change: 'The next Recommendation run uses it. Recommendations already written stay until a run replaces them.',
  },
}

/** A step's words. */
export function stepInfo(id: ClaudeStepId): StepInfo {
  return STEP_INFO[id]
}

/** The default model, Claude Haiku 5.5, which the tool's own estimates assume. */
export const DEFAULT_MODEL = 'claude-haiku-5-5'

const DAY = new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'long', year: 'numeric', timeZone: 'UTC' })

/** The day a version was saved: "1 November 2026". */
export function savedOn(iso: string): string {
  return DAY.format(new Date(iso))
}

/** The page's data line: which version the step uses, and who saved it when. */
export function dataLine(step: Pick<ClaudeStep, 'fixedInstructions' | 'current'>): string {
  const v = step.current
  const paid = 'Paid by the Anthropic key.'
  if (step.fixedInstructions) return `${v.modelName} since ${savedOn(v.createdAt)}. ${paid}`
  if (v.number === 1) return `Version 1, the instructions the tool came with. ${paid}`
  return `Version ${String(v.number)}, saved ${savedOn(v.createdAt)} by ${v.createdBy}. ${paid}`
}

/** A version in the history: "Version 2, 1 November 2026, by Joris". */
export function versionLine(v: Pick<ClaudeStepVersion, 'number' | 'createdAt' | 'createdBy'>): string {
  return `Version ${String(v.number)}, ${savedOn(v.createdAt)}, by ${v.createdBy}`
}

/** A model in the picker, the default marked. */
export function modelLabel(m: ClaudeStepModel): string {
  return m.id === DEFAULT_MODEL ? `${m.name} (default)` : m.name
}

/** What one job costs with the chosen model, and how much more or less than now; empty for an unknown model. */
export function costLine(step: Pick<ClaudeStep, 'id' | 'models' | 'current' | 'jobCostUsd'>, model: string): string {
  const m = step.models.find((x) => x.id === model)
  if (!m) return ''
  const job = stepInfo(step.id).jobLabel
  if (model === step.current.model) return `About ${dollars(m.jobCostUsd)} for ${job}.`
  const diff = m.jobCostUsd - step.jobCostUsd
  return `About ${dollars(m.jobCostUsd)} for ${job}, ${dollars(Math.abs(diff))} ${diff >= 0 ? 'more' : 'less'} than now.`
}

/** What the editor holds: the instructions and the model. */
export type Draft = { text: string; model: string }

/** Whether the editor differs from the version the step uses. */
export function isDirty(step: Pick<ClaudeStep, 'current'>, draft: Draft): boolean {
  return draft.text !== step.current.instructions || draft.model !== step.current.model
}
