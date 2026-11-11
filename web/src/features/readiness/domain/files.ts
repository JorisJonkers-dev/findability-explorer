import type { AiReadiness, ReadinessFiles } from '../../../infrastructure/api/types.gen'
import { oxford } from '../../../shared/words'

/** The files a scan fetches from a site, in the order of their tabs. */
export const FILE_KEYS = ['robots', 'sitemap', 'llms', 'llmsFull'] as const

export type FileKey = (typeof FILE_KEYS)[number]

const FILE_NAMES: Record<FileKey, string> = { robots: 'robots.txt', sitemap: 'sitemap.xml', llms: 'llms.txt', llmsFull: 'llms-full.txt' }

/**
 * What the scan has of a file: its text, none found, robots.txt not reached, or a sitemap found by
 * a scan made before scans kept the sitemap's text.
 */
export type FileState = 'found' | 'missing' | 'unreachable' | 'notKept'

/**
 * One tab of the files viewer: the file's name, what the scan has of it, its text, whether that
 * was cut, the name it downloads under, and the address it came from (the sitemap's; empty else).
 */
export type FileTab = { key: FileKey; name: string; state: FileState; text: string; truncated: boolean; fileName: string; source: string }

/** The tab an address names, robots.txt when it names none. */
export function fileKeyOf(value: unknown): FileKey {
  return FILE_KEYS.find((k) => k === value) ?? 'robots'
}

/** A tab per file, from the files the scan kept and what it recorded of the site. */
export function fileTabs(files: ReadinessFiles | undefined, r: AiReadiness | undefined): FileTab[] {
  const text = (t: string | undefined, truncated: boolean | undefined) => ({
    state: t ? ('found' as const) : ('missing' as const),
    text: t ?? '',
    truncated: Boolean(t) && truncated === true,
  })
  const robots = text(files?.robotsTxt, files?.robotsTruncated)
  const sitemap = text(files?.sitemapTxt, files?.sitemapTruncated)
  const found = r !== undefined && (r.sitemap.declared || r.sitemap.files > 0)
  const byKey: Record<FileKey, Omit<FileTab, 'key' | 'name' | 'fileName' | 'source'>> = {
    robots: robots.state === 'missing' && r?.robots === 'unreachable' ? { ...robots, state: 'unreachable' } : robots,
    sitemap: sitemap.state === 'missing' && found ? { ...sitemap, state: 'notKept' } : sitemap,
    llms: text(files?.llmsTxt, files?.llmsTruncated),
    llmsFull: text(files?.llmsFullTxt, files?.llmsFullTruncated),
  }
  const source = (key: FileKey) => (key === 'sitemap' ? (files?.sitemapUrl ?? '') : '')
  return FILE_KEYS.map((key) => ({ key, name: FILE_NAMES[key], ...byKey[key], fileName: fileNameOf(source(key)) || FILE_NAMES[key], source: source(key) }))
}

/** The last part of an address's path, such as "sitemap_index.xml"; empty for an address without one. */
function fileNameOf(address: string): string {
  return address.replace(/[?#].*$/, '').split('/').slice(3).filter(Boolean).at(-1) ?? ''
}

/** "Found: robots.txt and llms.txt. Not found: llms-full.txt." */
export function filesFoundLine(tabs: readonly Pick<FileTab, 'name' | 'state'>[]): string {
  const found = tabs.filter((t) => t.state === 'found' || t.state === 'notKept').map((t) => t.name)
  const missing = tabs.filter((t) => t.state === 'missing' || t.state === 'unreachable').map((t) => t.name)
  if (found.length === 0) return 'No file was found.'
  return missing.length > 0 ? `Found: ${oxford(found)}. Not found: ${oxford(missing)}.` : `Found: ${oxford(found)}.`
}

/** What the viewer says in place of a file it has no text of; empty for a file it shows. */
export function missingSentence(tab: Pick<FileTab, 'key' | 'name' | 'state'>, host: string): string {
  switch (tab.state) {
    case 'found':
      return ''
    case 'notKept':
      return "This scan was made before scans kept the sitemap's text. Scan the site again to read it."
    case 'unreachable':
      return `The scan could not reach ${tab.name} on ${host}.`
    case 'missing':
      return `The scan found no ${tab.name} on ${host}.${tab.key === 'llms' || tab.key === 'llmsFull' ? ' Adding one is one of the eight AI readiness checks.' : ''}`
  }
}

/** A file's lines and its size in bytes, as UTF-8. */
export function fileSize(text: string): { lines: number; bytes: number } {
  return { lines: text.split('\n').length, bytes: new TextEncoder().encode(text).length }
}

/** Text made safe to set as HTML. */
export function escapeHtml(text: string): string {
  return text.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;').replaceAll('"', '&quot;').replaceAll("'", '&#39;')
}

/**
 * A file's lines, numbered from 1, as HTML: every match of the query, whatever its case, in a mark,
 * and the text around and inside each match escaped, so neither the file nor the query adds markup
 * and a query never matches inside an escape such as &amp;. hit is true for a line with a match;
 * hits counts the matches.
 */
export function searchLines(text: string, q: string): { lines: { n: number; html: string; hit: boolean }[]; hits: number } {
  const needle = q.trim()
  const pattern = needle === '' ? null : new RegExp(needle.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'gi')
  let hits = 0
  const lines = text.split('\n').map((line, i) => {
    let html = ''
    let last = 0
    let found = 0
    for (const m of pattern ? line.matchAll(pattern) : []) {
      html += `${escapeHtml(line.slice(last, m.index))}<mark>${escapeHtml(m[0])}</mark>`
      last = m.index + m[0].length
      found++
    }
    hits += found
    return { n: i + 1, html: html + escapeHtml(line.slice(last)), hit: found > 0 }
  })
  return { lines, hits }
}
