import type { IntentTier, PromptFamily, PromptSetDraft, PromptSetScope, PublishedPromptSet } from '../../../infrastructure/api/types.gen'
import { languageLabel } from '../../../shared/vocabulary'
import { costLabel, FAMILIES } from '../../prompt-test/domain/results'

/**
 * The draft of the Prompt set an administrator edits on Questions. It changes nothing until it is
 * published; publishing makes it the next version.
 */

/** One question of the draft: key tells the rows apart, id is the published question it keeps. */
export type DraftQuestion = { key: string; id?: string; family: PromptFamily; intent: IntentTier; scope: string; text: string; english?: string }

/** The draft: its questions in order, the new scopes it asks in, and the version it was restored from. */
export type Draft = { questions: DraftQuestion[]; scopes: PromptSetScope[]; restoredFrom?: number }

/** The languages a new scope can ask in, by their two-letter code. */
export const SCOPE_LANGUAGES: readonly string[] = ['nl', 'de', 'en', 'fr', 'pl', 'es', 'it', 'da', 'sv']

/** A published version as a draft, unchanged. */
export function draftOf(set: PublishedPromptSet): Draft {
  return {
    questions: set.questions.map((q) => ({
      key: q.id,
      id: q.id,
      family: q.family,
      intent: q.intent,
      scope: q.scope,
      text: q.text,
      ...(q.english ? { english: q.english } : {}),
    })),
    scopes: [],
  }
}

/** A version copied into the draft, to publish again as a new version. */
export function restoreOf(set: PublishedPromptSet): Draft {
  return { ...draftOf(set), restoredFrom: set.version }
}

/** A key no question of the draft has: "new-" and one more than the highest number used. */
function freshKey(draft: Draft): string {
  const used = draft.questions.map((q) => /^new-(\d+)$/.exec(q.key)?.[1]).filter((n) => n !== undefined).map(Number)
  return `new-${String(Math.max(0, ...used) + 1)}`
}

/** The intent a new question of a family takes: the one its questions in the draft have most, else application. */
export function intentFor(draft: Draft, family: PromptFamily): IntentTier {
  const counts = new Map<IntentTier, number>()
  for (const q of draft.questions) if (q.family === family) counts.set(q.intent, (counts.get(q.intent) ?? 0) + 1)
  let best: IntentTier = 'application'
  let most = 0
  for (const [intent, n] of counts) {
    if (n > most) {
      best = intent
      most = n
    }
  }
  return best
}

/** The draft with a new question at the end of its family. */
export function addQuestion(draft: Draft, family: PromptFamily, scope: string, text: string, english?: string): Draft {
  const q: DraftQuestion = { key: freshKey(draft), family, intent: intentFor(draft, family), scope, text: text.trim(), ...(english ? { english } : {}) }
  return { ...draft, questions: [...draft.questions, q] }
}

/** The draft without a question; a new scope no question asks in any more leaves with it. */
export function removeQuestion(draft: Draft, key: string): Draft {
  const questions = draft.questions.filter((q) => q.key !== key)
  return { ...draft, questions, scopes: draft.scopes.filter((s) => questions.some((q) => q.scope === s.id)) }
}

/** The draft with a question's text changed; its English no longer says the same, so it goes. */
export function editQuestion(draft: Draft, key: string, text: string): Draft {
  return {
    ...draft,
    questions: draft.questions.map((q) => {
      if (q.key !== key) return q
      const edited: DraftQuestion = { ...q, text }
      delete edited.english
      return edited
    }),
  }
}

/** Every scope the draft can ask in: the known ones, then its new ones. */
export function scopesOf(known: readonly PromptSetScope[], draft: Draft): PromptSetScope[] {
  return [...known, ...draft.scopes]
}

/** A scope's name, or its id when no scope has it. */
export function scopeNameOf(scopes: readonly PromptSetScope[], id: string): string {
  return scopes.find((s) => s.id === id)?.name ?? id
}

/** The first scope a family has no question in yet, so the add row starts where a question is missing. */
export function firstGap(draft: Draft, scopes: readonly PromptSetScope[], family: PromptFamily): string {
  const gap = scopes.find((s) => !draft.questions.some((q) => q.family === family && q.scope === s.id))
  return (gap ?? scopes[0])?.id ?? ''
}

/** A scope's id from its language and region: "fr-lyon", or "fr" with no region. */
export function scopeID(language: string, region: string): string {
  const slug = region
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
  return (slug ? `${language}-${slug}` : language).slice(0, 40).replace(/-+$/, '')
}

/**
 * The scope a language and region make: an existing one with that id or name, or a new one, which
 * the draft then asks in. made says whether it is new.
 */
export function makeScope(draft: Draft, known: readonly PromptSetScope[], language: string, region: string): { draft: Draft; scope: PromptSetScope; made: boolean } {
  const where = region.trim()
  const name = where ? `${languageLabel(language)}, ${where}` : `${languageLabel(language)}, no region`
  const id = scopeID(language, where)
  const found = scopesOf(known, draft).find((s) => s.id === id || s.name === name)
  if (found) return { draft, scope: found, made: false }
  const scope: PromptSetScope = { id, name, language, ...(where ? { region: where } : {}) }
  return { draft: { ...draft, scopes: [...draft.scopes, scope] }, scope, made: true }
}

/** The families with no question in a scope yet. */
export function familiesWithout(draft: Draft, scope: string): PromptFamily[] {
  return FAMILIES.map((f) => f.value).filter((f) => !draft.questions.some((q) => q.family === f && q.scope === scope))
}

/**
 * A question in a scope for every family without one, each starting from that family's question in
 * English, so the administrator translates it before publishing.
 */
export function fillScope(draft: Draft, scope: string): { draft: Draft; added: number } {
  const missing = familiesWithout(draft, scope)
  let next = draft
  for (const family of missing) {
    const source = draft.questions.find((q) => q.family === family && q.english) ?? draft.questions.find((q) => q.family === family)
    next = addQuestion(next, family, scope, source?.english ?? source?.text ?? 'Write the question as a buyer would ask it.')
  }
  return { draft: next, added: missing.length }
}

/** The questions of one family in the draft, in the scopes' order. */
export type FamilyRows = { family: PromptFamily; label: string; questions: DraftQuestion[] }

/** The draft by belt family, every family, each family's questions in the scopes' order. */
export function familyRows(draft: Draft, scopes: readonly PromptSetScope[]): FamilyRows[] {
  const order = (id: string) => {
    const i = scopes.findIndex((s) => s.id === id)
    return i < 0 ? scopes.length : i
  }
  return FAMILIES.map((f) => ({
    family: f.value,
    label: f.label,
    questions: draft.questions.filter((q) => q.family === f.value).sort((a, b) => order(a.scope) - order(b.scope)),
  }))
}

/** What the draft changes against the published set. */
export type DraftDiff = { added: DraftQuestion[]; changed: DraftQuestion[]; removed: number; scopes: PromptSetScope[]; reordered: boolean }

/** What the draft adds, changes, and removes against the published set. */
export function diffOf(published: PublishedPromptSet, draft: Draft): DraftDiff {
  const before = new Map(published.questions.map((q) => [q.id, q]))
  const kept = new Set(draft.questions.flatMap((q) => (q.id ? [q.id] : [])))
  const added = draft.questions.filter((q) => !q.id || !before.has(q.id))
  const changed = draft.questions.filter((q) => {
    const p = q.id ? before.get(q.id) : undefined
    return p !== undefined && (p.text !== q.text.trim() || p.scope !== q.scope || p.family !== q.family || p.intent !== q.intent)
  })
  const removed = published.questions.filter((q) => !kept.has(q.id)).length
  const order = draft.questions.map((q) => q.id ?? '').join('\n')
  const reordered = added.length === 0 && removed === 0 && order !== published.questions.map((q) => q.id).join('\n')
  return { added, changed, removed, scopes: draft.scopes, reordered }
}

/** Whether publishing the draft would change nothing. */
export function isUnchanged(d: DraftDiff): boolean {
  return d.added.length === 0 && d.changed.length === 0 && d.removed === 0 && d.scopes.length === 0 && !d.reordered
}

const NUMBER = new Intl.NumberFormat('en-GB')

function questions(n: number): string {
  return n === 1 ? '1 question' : `${NUMBER.format(n)} questions`
}

/**
 * What publishing changes, as Publish says it: "Adds 2 questions (2 in French), changes 1, and
 * removes none. Adds 1 scope (French, Lyon)."
 */
export function publishLine(d: DraftDiff, draft: Draft, scopes: readonly PromptSetScope[]): string {
  const languages = new Map<string, number>()
  for (const q of d.added) {
    const language = languageLabel(scopes.find((s) => s.id === q.scope)?.language ?? '')
    languages.set(language, (languages.get(language) ?? 0) + 1)
  }
  const inLanguages = [...languages].map(([l, n]) => `${NUMBER.format(n)} in ${l}`).join(', ')
  const adds = d.added.length === 0 ? 'Adds no questions' : `Adds ${questions(d.added.length)} (${inLanguages})`
  const changes = d.changed.length === 0 ? 'none' : NUMBER.format(d.changed.length)
  const removes = d.removed === 0 ? 'none' : NUMBER.format(d.removed)
  const parts = [`${adds}, changes ${changes}, and removes ${removes}.`]
  if (draft.restoredFrom) parts.unshift(`Restores version ${String(draft.restoredFrom)}.`)
  if (d.scopes.length > 0) parts.push(`Adds ${d.scopes.length === 1 ? '1 scope' : `${String(d.scopes.length)} scopes`} (${d.scopes.map((s) => s.name).join('; ')}).`)
  return parts.join(' ')
}

/**
 * What a run will cost more or less: a run costs about the same per question as now, so the change
 * is the difference in questions times that, per run and per year on the schedule.
 */
export function costSentence(runUsd: number, publishedQuestions: number, draftQuestions: number, cadence: 'monthly' | 'weekly'): string {
  const perQuestion = publishedQuestions > 0 ? runUsd / publishedQuestions : 0
  const delta = perQuestion * (draftQuestions - publishedQuestions)
  if (Math.abs(delta) < 0.005) return 'A run will cost about the same as now.'
  const runsPerYear = cadence === 'weekly' ? 52 : 12
  const dir = (x: number) => `${costLabel(Math.abs(x))} ${x > 0 ? 'more' : 'less'}`
  return `A run will cost about ${dir(delta)}, or ${dir(delta * runsPerYear)} a year on a ${cadence} schedule.`
}

/** The draft as publishPromptSet takes it: only the new scopes a question asks in. */
export function requestOf(draft: Draft): PromptSetDraft {
  const used = new Set(draft.questions.map((q) => q.scope))
  return {
    questions: draft.questions.map((q) => ({
      ...(q.id ? { id: q.id } : {}),
      family: q.family,
      intent: q.intent,
      scope: q.scope,
      text: q.text.trim(),
      ...(q.english ? { english: q.english } : {}),
    })),
    scopes: draft.scopes.filter((s) => used.has(s.id)),
    ...(draft.restoredFrom ? { restoredFrom: draft.restoredFrom } : {}),
  }
}
