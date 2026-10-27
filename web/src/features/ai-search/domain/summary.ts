import type { AiSearchSummary, CrawlerAccess, KeepingOut, ReadinessRank, ShareTrendPoint } from '../../../infrastructure/api/types.gen'

/** A share as the page writes it: "24%". */
export function pct(share: number): string {
  return `${String(Math.round(share * 100))}%`
}

/** A date as the chart writes it: "2 Nov". */
export function shortDate(iso: string): string {
  return new Date(iso).toLocaleDateString('en-GB', { day: 'numeric', month: 'short', timeZone: 'UTC' })
}

/** The chart's drawing box, in its own units. */
export const CHART = { left: 70, right: 1040, top: 100, bottom: 600 } as const

/** The top of the chart's scale: the next 20% above the highest share or the goal, at least 20%. */
export function scaleTop(trend: readonly ShareTrendPoint[], goal: number): number {
  const highest = Math.max(goal, ...trend.flatMap((p) => p.shares))
  return Math.max(0.2, Math.ceil(highest * 5 - 1e-9) / 5)
}

/** Where a share sits on the chart's vertical axis. */
export function yOf(share: number, top: number): number {
  return CHART.bottom - (share / top) * (CHART.bottom - CHART.top)
}

/** Where the run at index sits on the horizontal axis; a single run sits in the middle. */
export function xOf(index: number, runs: number): number {
  if (runs <= 1) return (CHART.left + CHART.right) / 2
  return CHART.left + (index * (CHART.right - CHART.left)) / (runs - 1)
}

/** One followed company's line: its points, where it ends, and its label. */
export type Line = { name: string; tribelt: boolean; points: string; endX: number; endY: number; labelY: number; label: string }

/** The chart's lines, Tribelt's last so it draws on top; labels pushed apart so they never overlap. */
export function linesOf(summary: Pick<AiSearchSummary, 'followed' | 'trend'>, top: number): Line[] {
  const { followed, trend } = summary
  const lines = followed.map((c, i): Line => {
    const shares = trend.map((p) => p.shares[i] ?? 0)
    const last = shares.at(-1) ?? 0
    const endY = yOf(last, top)
    return {
      name: c.name,
      tribelt: c.role === 'tribelt',
      points: shares.map((s, j) => `${xOf(j, shares.length).toFixed(1)},${yOf(s, top).toFixed(1)}`).join(' '),
      endX: xOf(shares.length - 1, shares.length),
      endY,
      labelY: endY + 10,
      label: `${c.name} ${pct(last)}`,
    }
  })
  // Labels at least 36 units apart, from the top down.
  const byY = [...lines].sort((a, b) => a.labelY - b.labelY)
  for (let i = 1; i < byY.length; i++) {
    const above = byY[i - 1] as Line
    const line = byY[i] as Line
    if (line.labelY - above.labelY < 36) line.labelY = above.labelY + 36
  }
  return [...lines.filter((l) => !l.tribelt), ...lines.filter((l) => l.tribelt)]
}

/** Tribelt's share in the newest run and in the first one the trend shows, or undefined before any run. */
export function tribeltShares(summary: Pick<AiSearchSummary, 'followed' | 'trend'>): { now: number; first: number } | undefined {
  const i = summary.followed.findIndex((c) => c.role === 'tribelt')
  const first = summary.trend.at(0)
  const last = summary.trend.at(-1)
  if (i < 0 || !first || !last) return undefined
  return { now: last.shares[i] ?? 0, first: first.shares[i] ?? 0 }
}

/** The line beside the big figure: how it moved since the first run shown. */
export function trendLine(summary: Pick<AiSearchSummary, 'followed' | 'trend'>): string {
  const t = tribeltShares(summary)
  if (!t) return 'No Prompt run has finished yet.'
  if (summary.trend.length === 1) return 'of the answers in the first Prompt run'
  const first = shortDate(summary.trend[0]?.runStartedAt ?? '')
  if (t.now === t.first) return `of the answers in the newest run, the same as on ${first}`
  return `of the answers in the newest run, ${t.now > t.first ? 'up' : 'down'} from ${pct(t.first)} on ${first}`
}

/** A chart description for a screen reader, from the data. */
export function chartLabel(summary: Pick<AiSearchSummary, 'followed' | 'trend' | 'goal'>): string {
  const parts = summary.followed.map((c, i) => {
    const first = summary.trend.at(0)?.shares[i] ?? 0
    const last = summary.trend.at(-1)?.shares[i] ?? 0
    return `${c.name} went from ${pct(first)} to ${pct(last)}.`
  })
  return [`Share of answers naming each company over ${String(summary.trend.length)} Prompt runs.`, ...parts, `The goal is ${pct(summary.goal)}.`].join(' ')
}

/** How a crawler's access reads, with a mark that differs in shape as well as colour. */
export const ACCESS: Record<CrawlerAccess | 'unknown', { text: string; mark: string; ok: boolean | null }> = {
  allowed: { text: 'Let in', mark: '●', ok: true },
  partly: { text: 'Partly', mark: '◐', ok: true },
  blocked: { text: 'Kept out', mark: '✕', ok: false },
  unknown: { text: 'Not checked', mark: '–', ok: null },
}

/** Names as a sentence: "A", "A and B", "A, B and 3 more". */
export function nameList(names: readonly string[], shown = 3): string {
  if (names.length <= 1) return names.join('')
  if (names.length <= shown) return `${names.slice(0, -1).join(', ')} and ${names.at(-1) ?? ''}`
  return `${names.slice(0, shown).join(', ')} and ${String(names.length - shown)} more`
}

/** What tribelt.nl does with the crawlers, from its newest scan. */
export function tribeltNote(crawlers: AiSearchSummary['crawlers']): string {
  if (crawlers.every((c) => c.access === undefined)) return 'tribelt.nl has not been scanned yet.'
  const out = crawlers.filter((c) => c.access === 'blocked').map((c) => c.bot)
  if (out.length === 0) return 'tribelt.nl lets every one of them in.'
  return `tribelt.nl keeps ${nameList(out, 4)} out.`
}

/** Who among the competitors keeps crawlers out. */
export function keepingOutNote(keepingOut: readonly KeepingOut[]): string {
  if (keepingOut.length === 0) return 'No competitor keeps a crawler out.'
  const verb = keepingOut.length === 1 ? 'shuts' : 'shut'
  return `${nameList(keepingOut.map((k) => k.name))} ${verb} some of them out.`
}

/** Whether a ranked row is Tribelt's. */
export function isTribelt(r: Pick<ReadinessRank, 'role'>): boolean {
  return r.role === 'tribelt'
}
