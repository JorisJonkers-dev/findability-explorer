import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import type { LinesSeries } from './lines'
import LinesChart from './LinesChart.vue'

const A = '2026-10-26T06:00:00Z'
const B = '2026-10-27T06:00:00Z'
const C = '2026-10-27T09:30:00Z'
const D = '2026-10-29T06:00:00Z'
// Version 2 was published between B and C.
const BREAK = { x: '2026-10-27T08:00:00Z', label: 'Prompt set version 2' }

const series: LinesSeries[] = [
  { label: 'All', tone: 'orange', strong: true, points: [{ x: A, y: 0.4, partial: true }, { x: B, y: 0.3 }, { x: C, y: 0.25 }, { x: D, y: 0.27 }] },
  { label: 'Dutch', tone: 'blue', points: [{ x: A, y: 0.5 }, { x: B, y: 0.45 }, { x: C, y: 0.26 }] },
]

const num = (v: string | undefined): number => Number(v)

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('LinesChart', () => {
  it('draws an image with a line per segment, never across a break', () => {
    const w = mount(LinesChart, { props: { series, breaks: [BREAK], label: 'Over time' } })
    const svg = w.get('svg')
    expect(svg.attributes('role')).toBe('img')
    expect(svg.attributes('aria-label')).toBe('Over time')
    // All: A-B, then C-D. Dutch: A-B, then C alone, which is a dot without a line.
    const lines = w.findAll('polyline')
    expect(lines).toHaveLength(3)
    expect(lines.map((l) => l.attributes('points')?.split(' ').length)).toEqual([2, 2, 2])
    expect(lines[0]?.attributes('stroke-width')).toBe('3')
    expect(lines[2]?.attributes('stroke-width')).toBe('2')
  })

  it('shades from a break to the right edge, labelled with its version', () => {
    const w = mount(LinesChart, { props: { series, breaks: [BREAK] } })
    const band = w.get('[data-part="band"]')
    expect(band.text()).toBe('Prompt set version 2')
    const xs = [...new Set(w.findAll('circle[data-part="dot"]').map((d) => num(d.attributes('cx'))))]
    expect(num(band.get('rect').attributes('x'))).toBeCloseTo(((xs[1] ?? 0) + (xs[2] ?? 0)) / 2)
  })

  it('leaves out a break with no run on one of its sides', () => {
    const w = mount(LinesChart, { props: { series, breaks: [{ x: '2026-10-25T00:00:00Z', label: 'Prompt set version 1' }, { x: '2026-10-30T00:00:00Z', label: 'Prompt set version 3' }] } })
    expect(w.findAll('[data-part="band"]')).toHaveLength(0)
    expect(w.findAll('polyline')).toHaveLength(2)
  })

  it('gives every point a dot and a title, and a partial run a dashed ring', () => {
    const w = mount(LinesChart, { props: { series } })
    expect(w.findAll('circle[data-part="dot"]')).toHaveLength(7)
    const rings = w.findAll('circle[data-part="ring"]')
    expect(rings).toHaveLength(1)
    expect(rings[0]?.attributes('stroke-dasharray')).toBe('2 2')
    const titles = w.findAll('title').map((t) => t.text())
    expect(titles).toContain('All, 26 October 2026: 40% (partial run)')
    expect(titles).toContain('Dutch, 27 October 2026: 26%')
  })

  it('ends each line in a label in the right margin, at least 17px apart, joined by a line in its colour', () => {
    const w = mount(LinesChart, { props: { series, goal: 0.26 } })
    const labels = w.findAll('[data-part="end"]')
    expect(labels.map((l) => l.get('text').text())).toEqual(['All 27%', 'Dutch 26%', 'Goal 26%'])
    const ys = labels.map((l) => num(l.get('text').attributes('y'))).sort((a, b) => a - b)
    for (let i = 1; i < ys.length; i++) expect((ys[i] ?? 0) - (ys[i - 1] ?? 0)).toBeGreaterThanOrEqual(17)
    const plotRight = Math.max(...w.findAll('circle[data-part="dot"]').map((d) => num(d.attributes('cx'))))
    for (const l of labels) expect(num(l.get('text').attributes('x'))).toBeGreaterThan(plotRight)
    expect(labels[1]?.get('path').attributes('stroke')).toBe('var(--color-line-blue)')
    expect(labels[2]?.get('path').attributes('stroke-dasharray')).toBe('3 3')
  })

  it('keeps the end labels inside the plot', () => {
    const low: LinesSeries[] = ['A', 'B', 'C', 'D'].map((label) => ({ label, tone: 'ink', points: [{ x: A, y: 0 }] }))
    const w = mount(LinesChart, { props: { series: low, height: 300 } })
    const ys = w.findAll('[data-part="end"] text').map((t) => num(t.attributes('y')))
    expect(Math.max(...ys)).toBeLessThanOrEqual(300 - 44 + 4)
    expect(Math.max(...ys) - Math.min(...ys)).toBeCloseTo(3 * 17)
  })

  it('draws the goal as a dashed line with its label', () => {
    const w = mount(LinesChart, { props: { series, goal: 0.5, goalLabel: 'Goal' } })
    expect(w.get('line[data-part="goal"]').attributes('stroke-dasharray')).toBe('6 4')
    expect(w.text()).toContain('Goal 50%')
    expect(w.get('[data-part="legend"]').text()).toContain('Goal')
  })

  it('steps the scale by 5, 10, 20, or 25 points', () => {
    const steps = (top: number) => {
      const w = mount(LinesChart, { props: { series: [{ label: 'All', tone: 'orange', points: [{ x: A, y: top }] }] } })
      return w.findAll('[data-part="grid"] text').map((t) => t.text())
    }
    expect(steps(0.25)).toEqual(['0%', '5%', '10%', '15%', '20%', '25%', '30%'])
    expect(steps(0.4)).toEqual(['0%', '10%', '20%', '30%', '40%', '50%'])
    expect(steps(0.9)).toEqual(['0%', '20%', '40%', '60%', '80%', '100%', '120%'])
    expect(steps(2)).toEqual(['0%', '25%', '50%', '75%', '100%', '125%', '150%', '175%', '200%', '225%'])
    expect(steps(0)).toEqual(['0%', '5%'])
  })

  it('puts a lone run at the right edge, beside its labels', () => {
    const w = mount(LinesChart, { props: { series: [{ label: 'All', tone: 'orange', points: [{ x: A, y: 0.25 }] }] } })
    // 860 wide; the margin fits "All 25%": 7 characters at 7.4px, and 34px.
    const right = 860 - (7 * 7.4 + 34) - 18
    expect(num(w.get('circle[data-part="dot"]').attributes('cx'))).toBeCloseTo(right)
    expect(w.findAll('polyline')).toHaveLength(0)
  })

  it('writes the time under a day that has two runs', () => {
    const w = mount(LinesChart, { props: { series } })
    const ticks = w.findAll('[data-part="tick"]').map((t) => t.findAll('text').map((x) => x.text()))
    expect(ticks).toEqual([['26 October'], ['27 October', '06:00'], ['27 October', '09:30'], ['29 October']])
  })

  it('says in its legend what a dashed ring means, only when one is drawn', () => {
    expect(mount(LinesChart, { props: { series } }).get('[data-part="legend"]').text()).toContain('A dashed ring marks a partial run.')
    const whole: LinesSeries[] = series.map((s) => ({ ...s, points: s.points.map((p) => ({ x: p.x, y: p.y })) }))
    const legend = mount(LinesChart, { props: { series: whole } }).get('[data-part="legend"]').text()
    expect(legend).not.toContain('dashed ring')
    expect(legend).toContain('All')
    expect(legend).toContain('Dutch')
  })

  it('formats values as asked', () => {
    const w = mount(LinesChart, { props: { series: [{ label: 'Runs', tone: 'ink', points: [{ x: A, y: 0.2 }] }], format: (v: number) => v.toFixed(2) } })
    expect(w.text()).toContain('Runs 0.20')
  })

  it('draws nothing without a point', () => {
    const w = mount(LinesChart, { props: { series: [{ label: 'All', tone: 'orange', points: [] }] } })
    expect(w.find('svg').exists()).toBe(false)
  })

  it('redraws at the width its box has', async () => {
    let resize: ((entries: { contentRect: { width: number } }[]) => void) | undefined
    const disconnect = vi.fn()
    vi.stubGlobal(
      'ResizeObserver',
      class {
        constructor(cb: (entries: { contentRect: { width: number } }[]) => void) {
          resize = cb
        }
        observe() {}
        disconnect = disconnect
      },
    )
    const w = mount(LinesChart, { props: { series } })
    expect(w.get('svg').attributes('width')).toBe('860')
    resize?.([{ contentRect: { width: 500 } }])
    await nextTick()
    expect(w.get('svg').attributes('width')).toBe('500')
    resize?.([{ contentRect: { width: 100 } }])
    await nextTick()
    expect(w.get('svg').attributes('width')).toBe('320')
    resize?.([])
    await nextTick()
    expect(w.get('svg').attributes('width')).toBe('320')
    w.unmount()
    expect(disconnect).toHaveBeenCalled()
  })
})
