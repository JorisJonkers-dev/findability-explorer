import { type Crawler, CRAWLERS, type CrawlerId, LOG_DAYS } from './content'

/** Where a day falls on the timeline, from 0 (30 September) to 1 (23 October). */
export function dayPosition(day: number): number {
  return Math.min(Math.max(day / LOG_DAYS, 0), 1)
}

/** The crawler with this id. */
export function crawler(id: CrawlerId): Crawler {
  return CRAWLERS.find((c) => c.id === id) as Crawler
}

/** The line under a crawler's name: its owner, and whether its visits could be verified. */
export function ownerLine(c: Pick<Crawler, 'owner' | 'verified'>): string {
  return c.verified === false ? `${c.owner} · unverified` : c.owner
}

/** A crawler's row in words, for a screen reader: "Googlebot: no visits." or "ClaudeBot: 1 visit, on 10 October." */
export function rowLabel(c: Pick<Crawler, 'name' | 'events' | 'verified'>): string {
  if (c.events.length === 0) return `${c.name}: no visits.`
  const days = c.events.map((e) => `${String(e.day <= 0 ? 30 : e.day)} ${e.day <= 0 ? 'September' : 'October'}`)
  const visits = `${String(c.events.length)} ${c.events.length === 1 ? 'day' : 'days'} with visits`
  const unverified = c.verified === false ? ', unverified' : ''
  return `${c.name}: ${visits}${unverified}, ${days.length === 1 ? 'on' : 'from'} ${days.length === 1 ? String(days[0]) : `${String(days[0])} to ${String(days.at(-1))}`}.`
}
