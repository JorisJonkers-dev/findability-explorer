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
  reading: {
    does: 'Reads each response for the companies it names, the sources it cites, and what it says about Tribelt.',
    section: { label: 'Prompt test', to: '/manage/prompt-test' },
    results: { label: 'See the AI responses', to: '/ai-responses' },
    jobLabel: 'reading 100 responses',
    gets: [
      { name: 'The fact sheet', what: 'What is true about Tribelt, to check the response against.', from: 'Prompt test, Fact sheet: the version in use when the run started' },
      { name: 'The buyer’s question', what: 'The question as it was asked, in its own language.', from: 'The published Prompt set' },
      { name: 'The response', what: 'The full text the AI assistant gave.', from: 'The Prompt run' },
      { name: 'The cited sources', what: 'The web addresses the assistant listed. Only with web search.', from: 'The Prompt run' },
    ],
    replies: [
      { field: 'companies', what: 'Every company the response names as a maker or supplier, in order, with the web address it gives.' },
      { field: 'addresses', what: 'Every web address the response cites or links to.' },
      { field: 'namesTribelt, citesTribelt', what: 'Whether it names Tribelt, and whether it links to a Tribelt address.' },
      { field: 'correctness', what: 'Whether what it says about Tribelt agrees with the fact sheet: none, correct, partly, or wrong.' },
      { field: 'wrongClaims', what: 'Each claim about Tribelt the fact sheet contradicts, quoted.' },
      { field: 'leftOut', what: 'Each fact from the fact sheet a buyer would want that the response does not mention.' },
    ],
    change:
      'Responses from now on are read by the new model, so the AI responses charts mark the date and do not join the lines across it. Earlier runs keep how they were read. Read them again on Prompt test, Runs.',
  },
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
  candidateMerge: {
    does: 'Groups the names in the AI responses that are one company, such as a name with and without its legal form.',
    section: { label: 'Competitor list', to: '/manage/competitors' },
    results: { label: 'See the competitors', to: '/competitors' },
    jobLabel: 'tidying the waiting names',
    gets: [
      { name: 'The competitor list', what: 'The id, name, and website of every company on the list.', from: 'Competitor list, Companies' },
      { name: 'The candidate names', what: 'Each name with its id, its other spellings, its websites, and how many responses named it.', from: 'The AI responses of the Prompt runs' },
    ],
    replies: [{ field: 'clusters', what: 'Groups of names that are one company, each with the id of the company on the list it is, or none.' }],
    change: 'The next tidy uses it. Names already grouped stay grouped.',
  },
  candidateCheck: {
    does: 'Reads a company’s website and decides, against the competitor definition, whether it competes. It classifies candidates and checks the list.',
    section: { label: 'Competitor list', to: '/manage/competitors' },
    results: { label: 'See the competitors', to: '/competitors' },
    jobLabel: 'checking 100 companies',
    gets: [
      { name: 'The competitor definition', what: 'The rule for what counts as a competitor, sent with the companies.', from: 'Competitor list, Competitor definition: the version in use' },
      { name: 'The companies', what: 'Each company with its names, websites, what the responses say of it, and the text of its home page.', from: 'Candidates, or the competitor list, and a quick read of each website' },
    ],
    replies: [
      { field: 'competitor', what: 'Whether the company competes with Tribelt.' },
      { field: 'sure', what: 'Whether the model is sure. When it is not, a person decides.' },
      { field: 'reason', what: 'One sentence on why.' },
    ],
    change: 'Checks from now on use it. Decisions already made stay, and decisions an administrator made are never changed.',
  },
  termReview: {
    does: 'Decides which search terms found on competitors’ websites join the list, and what each one’s buyer intent is.',
    section: { label: 'Search term list', to: '/manage/terms' },
    results: { label: 'See the search terms', to: '/terms' },
    jobLabel: 'reviewing 100 search terms',
    gets: [
      { name: 'The belt families', what: 'The id of each family and what it covers.', from: 'Fixed in the tool' },
      { name: 'Examples from the list', what: 'Search terms already on the list, with their language, belt family, and buyer intent, to show the style.', from: 'Search term list' },
      { name: 'The search terms to review', what: 'Each with its id, how many competitors use it and which, and the closest search term already on the list.', from: 'Waiting terms, found by the scans' },
    ],
    replies: [
      { field: 'accept', what: 'Whether the search term joins the list.' },
      { field: 'term, language, family, tier', what: 'The search term as the list should hold it, and how to file it, its buyer intent in tier. Empty when rejected.' },
      { field: 'sure', what: 'Whether the model is sure. When it is not, a person decides.' },
      { field: 'reason', what: 'One sentence on why.' },
    ],
    change: 'Search terms reviewed from now on are decided the new way. Terms already on the list stay.',
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
