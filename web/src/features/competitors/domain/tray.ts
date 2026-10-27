import type { CompetitorBatch, CompetitorCandidate } from '../../../infrastructure/api/types.gen'
import { candidateWebsite } from '../../prompt-test/domain/settings'
import { fold } from '../../terminology/domain/fold'
import { hostOf, normalizeHost, readableName } from './company'

/** A company picked to add: its name, its website (empty until known) and the candidate it comes from. */
export type TrayItem = { name: string; website: string; candidateId?: string }

/** A known company, as the tray matches against it. */
type Known = { name: string; host?: string | undefined }

const HOST = /^[a-z0-9-]+(\.[a-z0-9-]+)*\.[a-z]{2,}$/

/** The host a typed or pasted text names, "berndorfband-group.com" from "https://www.berndorfband-group.com/en", or null. */
export function hostIn(text: string): string | null {
  const trimmed = text.trim().toLowerCase()
  if (trimmed === '') return null
  const host = hostOf(/^https?:\/\//.test(trimmed) ? trimmed : `https://${trimmed}`)
  return host !== null && HOST.test(host) ? host : null
}

/** The website to add for a typed or pasted text: as typed when it is an address, otherwise its host over https. */
export function websiteOf(text: string): string {
  const trimmed = text.trim()
  if (/^https?:\/\//i.test(trimmed)) return trimmed
  const host = hostIn(trimmed)
  return host === null ? '' : `https://${host}/`
}

/** A company's name as compared: folded, without a legal form or punctuation. */
export function nameKey(name: string): string {
  const LEGAL = new Set(['bv', 'b.v.', 'gmbh', 'ltd', 'inc', 'nv', 'n.v.', 'ag', 'llc', 'sa', 's.a.', 'spa', 'sl', 'co', 'kg'])
  return fold(name.replace(/\(.*$/s, ''))
    .split(/[^a-z0-9.]+/)
    .filter((w) => w !== '' && !LEGAL.has(w))
    .map((w) => w.replace(/\./g, ''))
    .filter((w) => w !== '')
    .join(' ')
}

/** The known company an item repeats, by its website's host or its name. */
export function duplicateOf<T extends Known>(item: TrayItem, known: readonly T[]): T | undefined {
  const host = hostIn(item.website)
  const key = nameKey(item.name)
  return known.find((k) => (host !== null && k.host !== undefined && normalizeHost(k.host) === host) || (key !== '' && nameKey(k.name) === key))
}

/** The tray with item added, unless the tray holds it already. */
export function addToTray(tray: readonly TrayItem[], item: TrayItem): TrayItem[] {
  const asKnown = tray.map((t) => ({ name: t.name, host: hostIn(t.website) ?? undefined }))
  return duplicateOf(item, asKnown) ? [...tray] : [...tray, item]
}

/** One suggestion while typing: a known company (already added), a named candidate, or a new company. */
export type Suggestion = {
  kind: 'known' | 'candidate' | 'new'
  name: string
  detail: string
  item?: TrayItem
}

/** The most suggestions shown at once. */
export const MAX_TRAY_SUGGESTIONS = 6

/**
 * What to suggest for a typed text: the known companies it matches, marked as added, then the
 * companies the AI answers named, most named first, then a new company of that name or website.
 */
export function traySuggestions(text: string, known: readonly Known[], candidates: readonly CompetitorCandidate[]): Suggestion[] {
  const q = fold(text.trim())
  if (q === '') return []
  const host = hostIn(text)
  const hit = (name: string, h: string | undefined) => fold(name).includes(q) || (h !== undefined && h.includes(q)) || (host !== null && h === host)
  const out: Suggestion[] = known
    .filter((k) => hit(k.name, k.host))
    .slice(0, 3)
    .map((k) => ({ kind: 'known', name: k.name, detail: 'Already a competitor' }))
  for (const c of [...candidates].sort((a, b) => b.answers - a.answers)) {
    const website = candidateWebsite(c)
    if (hit(c.name, hostIn(website) ?? undefined)) {
      out.push({
        kind: 'candidate',
        name: c.name,
        detail: `Named in ${String(c.answers)} AI ${c.answers === 1 ? 'answer' : 'answers'}`,
        item: { name: c.name, website, candidateId: c.id },
      })
    }
  }
  const fresh: Suggestion =
    host !== null
      ? { kind: 'new', name: `Add ${host}`, detail: `A new company called ${readableName(host)}`, item: { name: readableName(host), website: websiteOf(text) } }
      : { kind: 'new', name: `Add “${text.trim()}” as a new company`, detail: 'Add its website before adding it', item: { name: text.trim(), website: '' } }
  return [...out.slice(0, MAX_TRAY_SUGGESTIONS - 1), fresh]
}

/** A pasted list as tray items, one company per line: a name and a website in any order, split by a comma, a semicolon or a tab. */
export function parseList(text: string): TrayItem[] {
  return text
    .split('\n')
    .map((line) => line.trim())
    .filter((line) => line !== '')
    .map((line) => {
      const parts = line
        .split(/[,;\t]/)
        .map((p) => p.trim())
        .filter((p) => p !== '')
      const site = parts.find((p) => hostIn(p) !== null && !p.includes(' '))
      // A line of only a website is named after its host.
      const name = parts.find((p) => p !== site) ?? readableName(String(hostIn(String(site))))
      return { name, website: site === undefined ? '' : websiteOf(site) }
    })
}

/** The tray with a pasted list added: lines repeating a known company are skipped and named. */
export function pasteIntoTray(tray: readonly TrayItem[], text: string, known: readonly Known[]): { tray: TrayItem[]; skipped: string[] } {
  const skipped: string[] = []
  let next = [...tray]
  for (const item of parseList(text)) {
    const dup = duplicateOf(item, known)
    if (dup) skipped.push(`${item.name} (${dup.name})`)
    else next = addToTray(next, item)
  }
  return { tray: next, skipped }
}

/** What a batch did, in a sentence. */
export function batchSummary(batch: CompetitorBatch): string {
  const count = (outcome: string) => batch.items.filter((i) => i.outcome === outcome).length
  const added = count('added')
  const parts = [`Added ${String(added)} ${added === 1 ? 'competitor' : 'competitors'}.`]
  if (count('duplicate') > 0) parts.push(`Skipped ${String(count('duplicate'))} that ${count('duplicate') === 1 ? 'is a competitor' : 'are competitors'} already.`)
  if (count('rejected') > 0) parts.push(`Could not add ${String(count('rejected'))}.`)
  if (batch.scans > 0) parts.push(`Started ${String(batch.scans)} ${batch.scans === 1 ? 'scan' : 'scans'}.`)
  return parts.join(' ')
}
