import { describe, expect, it } from 'vitest'
import { buyerFigures, channelColumns, chartLabel, duration, formatValue, metric, visitorsLine } from './channels'
import { DEFAULT_METRIC, GOAL, METRICS, TABLE } from './content'

describe('the channel chart', () => {
  it('writes each kind of value as the chart shows it', () => {
    expect(formatValue(18_772, 'count')).toBe('18,772')
    expect(formatValue(1.73, 'percent2')).toBe('1.73%')
    expect(formatValue(25.9375, 'percent1')).toBe('25.9%')
    expect(formatValue(64, 'duration')).toBe('1:04')
    expect(formatValue(0.34, 'ratio')).toBe('0.34')
    expect(duration(22)).toBe('0:22')
    expect(duration(125)).toBe('2:05')
  })

  it('opens on the contact rate, organic search the tallest column', () => {
    const rate = metric(DEFAULT_METRIC)
    expect(channelColumns(rate)).toEqual([
      { channel: 'Paid search', value: '0.04%', fraction: 0.04 / 1.73 },
      { channel: 'Organic search', value: '1.73%', fraction: 1 },
      { channel: 'AI assistants', value: '0.00%', fraction: 0 },
    ])
    expect(chartLabel(rate)).toBe('Contact rate: paid search 0.04%, organic search 1.73%, AI assistants 0.00%.')
    expect(rate.caption).toBe('Organic visitors contact Tribelt 43 times more often than paid visitors.')
  })

  it('has five measures, each with its own reading', () => {
    expect(METRICS.map((m) => m.label)).toEqual(['Users', 'Contact rate', 'Engagement time', 'Engaged sessions', 'Returning users'])
    expect(channelColumns(metric('users')).map((c) => c.value)).toEqual(['18,772', '640', '13'])
    expect(channelColumns(metric('time')).map((c) => c.value)).toEqual(['0:22', '1:04', '0:29'])
    expect(channelColumns(metric('returning')).map((c) => c.value)).toEqual(['1.4%', '25.9%', '23.1%'])
  })

  it('draws no column for a measure with nothing to show', () => {
    expect(channelColumns({ ...metric('users'), values: [0, 0, 0] }).map((c) => c.fraction)).toEqual([0, 0, 0])
  })
})

describe('the figures', () => {
  it('states the three figures from the export', () => {
    expect(buyerFigures()).toEqual([
      { big: '18,772', text: 'users came from paid search, 79% of all users' },
      { big: '1.73%', text: 'of organic visitors got in touch, against 0.04% of paid visitors' },
      { big: '13', text: 'users came from AI assistants, and none of them got in touch' },
    ])
  })

  it('says which channel brings the visitors and which brings the contacts', () => {
    expect(visitorsLine()).toBe('Paid search brings the most visitors, but organic search brings the visitors who get in touch.')
  })
})

describe('the content', () => {
  it("keeps the report's figures", () => {
    expect(TABLE).toHaveLength(7)
    expect(GOAL.rows.map((r) => r.fraction)).toEqual([0.08, 1])
  })
})
