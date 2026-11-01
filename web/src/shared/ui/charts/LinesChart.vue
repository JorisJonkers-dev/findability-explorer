<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { dayMonth, longDate } from '../../dates'
import type { LinesBreak, LinesPoint, LinesSeries } from './lines'
import { TONE_BG, TONE_VAR } from './tones'

/**
 * Lines over time, one per series, ported from the prototype's drawLines. Runs are spaced evenly;
 * where the measure changed (a break), the lines stop and a shaded band, labelled, marks the new
 * measure. Each line ends in a label in the right margin (identity is never colour alone), joined to
 * its last point by a short line in its colour; a partial point has a dashed ring; a lone run sits at
 * the right edge, beside its labels. The chart is drawn at the width its box has, so text keeps its size.
 */
const props = withDefaults(
  defineProps<{
    series: readonly LinesSeries[]
    breaks?: readonly LinesBreak[]
    goal?: number
    goalLabel?: string
    format?: (v: number) => string
    label?: string
    height?: number
  }>(),
  { breaks: () => [], goal: undefined, goalLabel: 'Goal', format: (v: number) => `${String(Math.round(v * 100))}%`, label: 'Over time', height: 300 },
)

const box = ref<HTMLElement | null>(null)
const width = ref(860)
let observer: ResizeObserver | undefined
onMounted(() => {
  if (box.value?.clientWidth) width.value = box.value.clientWidth
  if (typeof ResizeObserver === 'undefined' || !box.value) return
  observer = new ResizeObserver((entries) => {
    const w = entries[0]?.contentRect.width
    if (w) width.value = w
  })
  observer.observe(box.value)
})
onBeforeUnmount(() => observer?.disconnect())

const STEPS = [0.05, 0.1, 0.2, 0.25]
const L = 46
const T = 30
const B = 44
const PAD = 18
const GAP = 17
const at = (iso: string): number => Date.parse(iso)
const byTime = (a: LinesPoint, b: LinesPoint): number => at(a.x) - at(b.x)

type End = { key: string; v: number; x?: string; text: string; colour: string; strong: boolean; goal: boolean; y: number; ly: number }

const chart = computed(() => {
  const { series, breaks, goal, format } = props
  const all = series.flatMap((s) => s.points)
  if (!all.length) return undefined
  const times = [...new Set(all.map((p) => p.x))].sort((a, b) => at(a) - at(b))
  const ends: End[] = series.flatMap((s) => {
    const last = [...s.points].sort(byTime).at(-1)
    return last ? [{ key: s.label, v: last.y, x: last.x, text: `${s.label} ${format(last.y)}`, colour: TONE_VAR[s.tone], strong: Boolean(s.strong), goal: false, y: 0, ly: 0 }] : []
  })
  if (goal !== undefined) ends.push({ key: 'goal', v: goal, text: `${props.goalLabel} ${format(goal)}`, colour: '#000', strong: false, goal: true, y: 0, ly: 0 })

  const W = Math.max(320, Math.round(width.value))
  const H = props.height
  const R = Math.max(...ends.map((e) => e.text.length)) * 7.4 + 34
  const x0 = L + PAD
  const x1 = W - R - PAD
  const right = x1 + PAD
  // A lone run sits at the right, beside its labels, so the joining lines cannot read as a trend.
  const X = (x: string): number => (times.length === 1 ? x1 : x0 + (times.indexOf(x) / (times.length - 1)) * (x1 - x0))
  const top = Math.max(goal ?? 0, ...all.map((p) => p.y)) * 1.12
  const step = STEPS.find((s) => top / s <= 6) ?? 0.25
  const ym = Math.max(step, Math.ceil(top / step - 1e-9) * step)
  const Y = (v: number): number => T + (1 - v / ym) * (H - T - B)

  // The shaded band from each break to the right edge: a new measure starts there.
  const bands = breaks.flatMap((b) => {
    const after = times.findIndex((t) => at(t) > at(b.x))
    if (after <= 0) return []
    const bx = (X(times[after - 1] as string) + X(times[after] as string)) / 2
    return [{ key: b.x, x: bx, y: T - 22, width: right - bx, height: H - B - T + 22, label: b.label }]
  })
  const grid = Array.from({ length: Math.round(ym / step) + 1 }, (_, i) => ({ y: Y(i * step), text: format(i * step) }))
  const days = times.map(dayMonth)
  const ticks = times.map((x, i) => ({ key: x, x: X(x), day: days[i] as string, time: days.filter((d) => d === days[i]).length > 1 ? x.slice(11, 16) : undefined }))

  const lines: { key: string; colour: string; width: number; points: string }[] = []
  const dots: { key: string; cx: number; cy: number; colour: string; strong: boolean; partial: boolean; title: string }[] = []
  series.forEach((s, si) => {
    const pts = [...s.points].sort(byTime)
    const segs: LinesPoint[][] = [[]]
    pts.forEach((p, i) => {
      const prev = pts[i - 1]
      if (prev && breaks.some((b) => at(b.x) > at(prev.x) && at(b.x) <= at(p.x))) segs.push([])
      segs[segs.length - 1]?.push(p)
    })
    segs.forEach((seg, i) => {
      if (seg.length > 1) lines.push({ key: `${String(si)}-${String(i)}`, colour: TONE_VAR[s.tone], width: s.strong ? 3 : 2, points: seg.map((p) => `${X(p.x).toFixed(1)},${Y(p.y).toFixed(1)}`).join(' ') })
    })
    pts.forEach((p) => {
      const title = `${s.label}, ${longDate(p.x)}: ${format(p.y)}${p.partial ? ' (partial run)' : ''}`
      dots.push({ key: `${String(si)}-${p.x}`, cx: X(p.x), cy: Y(p.y), colour: TONE_VAR[s.tone], strong: Boolean(s.strong), partial: Boolean(p.partial), title })
    })
  })

  // End labels in the right margin, at their value, pushed apart by at least 17px and kept inside the plot.
  for (const e of ends) e.y = e.ly = Y(e.v)
  const order = [...ends].sort((a, b) => a.y - b.y)
  for (let i = 1; i < order.length; i++) (order[i] as End).ly = Math.max((order[i] as End).ly, (order[i - 1] as End).ly + GAP)
  const floor = H - B
  const lowest = order.at(-1) as End
  if (lowest.ly > floor) {
    lowest.ly = floor
    for (let i = order.length - 2; i >= 0; i--) (order[i] as End).ly = Math.min((order[i] as End).ly, (order[i + 1] as End).ly - GAP)
  }
  const lx = right + 22
  const labels = ends.map((e) => {
    const from = e.goal ? right : X(e.x as string) + (e.strong ? 7 : 6)
    const path = `M${String(from)},${String(e.y)} C${String(lx - 10)},${String(e.y)} ${String(lx - 14)},${String(e.ly)} ${String(lx - 4)},${String(e.ly)}`
    return { key: e.key, path, colour: e.colour, goal: e.goal, bold: e.strong || e.goal, x: lx, y: e.ly + 4, text: e.text }
  })

  return { W, H, right, bands, grid, ticks, goalY: goal === undefined ? undefined : Y(goal), lines, dots, labels }
})

const partial = computed(() => props.series.some((s) => s.points.some((p) => p.partial)))
</script>

<template>
  <div class="flex min-w-0 flex-col gap-1.5">
    <ul data-part="legend" class="m-0 flex list-none flex-wrap items-center gap-x-4 gap-y-1 p-0 text-sm text-ink-2">
      <li v-for="s in series" :key="s.label" class="inline-flex items-center gap-1.5">
        <span aria-hidden="true" class="h-1 w-4 rounded-sm" :class="TONE_BG[s.tone]"></span>{{ s.label }}
      </li>
      <li v-if="goal !== undefined" class="inline-flex items-center gap-1.5">
        <span aria-hidden="true" class="w-4 border-t-2 border-dashed border-black"></span>{{ goalLabel }}
      </li>
      <li v-if="partial">A dashed ring marks a partial run.</li>
    </ul>
    <div ref="box" class="w-full min-w-0">
      <svg v-if="chart" role="img" :aria-label="label" :width="chart.W" :height="chart.H" :viewBox="`0 0 ${String(chart.W)} ${String(chart.H)}`" class="block h-auto max-w-full font-sans">
        <g v-for="b in chart.bands" :key="b.key" data-part="band">
          <rect :x="b.x" :y="b.y" :width="b.width" :height="b.height" fill="#fff" opacity="0.7" rx="8" />
          <text :x="b.x + 8" :y="T - 8" font-size="11.5" font-weight="600" fill="var(--color-muted)">{{ b.label }}</text>
        </g>
        <g v-for="g in chart.grid" :key="g.text" data-part="grid">
          <line :x1="L" :x2="chart.right" :y1="g.y" :y2="g.y" stroke="var(--color-grey)" />
          <text :x="L - 8" :y="g.y + 4" text-anchor="end" font-size="11.5" fill="var(--color-muted)">{{ g.text }}</text>
        </g>
        <g v-for="t in chart.ticks" :key="t.key" data-part="tick">
          <text :x="t.x" :y="chart.H - 24" text-anchor="middle" font-size="11.5" fill="var(--color-muted)">{{ t.day }}</text>
          <text v-if="t.time" :x="t.x" :y="chart.H - 10" text-anchor="middle" font-size="11" fill="var(--color-muted)">{{ t.time }}</text>
        </g>
        <line v-if="chart.goalY !== undefined" data-part="goal" :x1="L" :x2="chart.right" :y1="chart.goalY" :y2="chart.goalY" stroke="#000" stroke-width="1.5" stroke-dasharray="6 4" />
        <polyline v-for="l in chart.lines" :key="l.key" fill="none" :stroke="l.colour" :stroke-width="l.width" stroke-linejoin="round" stroke-linecap="round" :points="l.points" />
        <g v-for="d in chart.dots" :key="d.key">
          <title>{{ d.title }}</title>
          <circle :cx="d.cx" :cy="d.cy" r="10" fill="transparent" />
          <circle v-if="d.partial" data-part="ring" :cx="d.cx" :cy="d.cy" :r="d.strong ? 9 : 8" fill="none" :stroke="d.colour" stroke-width="1.5" stroke-dasharray="2 2" />
          <circle data-part="dot" :cx="d.cx" :cy="d.cy" :r="d.strong ? 5 : 4" :fill="d.colour" :stroke="d.colour" stroke-width="2" />
        </g>
        <g v-for="e in chart.labels" :key="e.key" data-part="end">
          <path :d="e.path" fill="none" :stroke="e.colour" stroke-width="1.5" :stroke-dasharray="e.goal ? '3 3' : undefined" />
          <text :x="e.x" :y="e.y" font-size="12.5" :font-weight="e.bold ? 700 : 600" fill="#000">{{ e.text }}</text>
        </g>
      </svg>
    </div>
  </div>
</template>
