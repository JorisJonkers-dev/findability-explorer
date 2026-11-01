import type { PromptFamily, PromptSetQuestion, PromptSetVersionInfo } from '../../../infrastructure/api/types.gen'
import { FAMILIES } from '../../prompt-test/domain/results'

/** A question as Questions lists it: its scope, the text as asked, and its English when known. */
export type ListedQuestion = { id: string; scope: string; text: string; english?: string }

/** The questions of one belt family. */
export type QuestionGroup = { family: PromptFamily; label: string; questions: ListedQuestion[] }

/** The questions grouped by belt family, in the families' order; a family without questions is left out. */
export function questionGroups(questions: readonly PromptSetQuestion[]): QuestionGroup[] {
  return FAMILIES.map((f) => ({
    family: f.value,
    label: f.label,
    questions: questions
      .filter((q) => q.family === f.value)
      .map((q) => ({ id: q.id, scope: q.scopeName, text: q.text, ...(q.english ? { english: q.english } : {}) })),
  })).filter((g) => g.questions.length > 0)
}

const DAY = new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'long', year: 'numeric', timeZone: 'UTC' })

/** "Published on 27 October 2026, 200 questions". */
export function versionLine(version: Pick<PromptSetVersionInfo, 'publishedAt' | 'questions'>): string {
  const n = version.questions === 1 ? '1 question' : `${String(version.questions)} questions`
  return `Published on ${DAY.format(new Date(version.publishedAt))}, ${n}`
}
