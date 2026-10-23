import { describe, expect, it } from 'vitest'
import { channelColumns, chartLabel, duration, formatValue, metric, shareLabel, volumeAgainstValue } from './channels'
import { DEFAULT_METRIC, GOAL, HOME_PAIRS, METRICS, TABLE } from './content'

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

describe('volume against value', () => {
  it("gives each channel's share of the visits and of the contacts", () => {
    expect(volumeAgainstValue()).toEqual([
      { channel: 'Paid search', visits: '96.6%', visitsFraction: 18_772 / 19_425, contacts: '47.8%', contactsFraction: 11 / 23 },
      { channel: 'Organic search', visits: '3.3%', visitsFraction: 640 / 19_425, contacts: '52.2%', contactsFraction: 12 / 23 },
      { channel: 'AI assistants', visits: '<0.1%', visitsFraction: 13 / 19_425, contacts: '0.0%', contactsFraction: 0 },
    ])
  })

  it('writes a sliver as less than a tenth, and nothing as nothing', () => {
    expect(shareLabel(0.05)).toBe('<0.1%')
    expect(shareLabel(0)).toBe('0.0%')
    expect(volumeAgainstValue([0, 0, 0], [0, 0, 0]).map((v) => v.visits)).toEqual(['0.0%', '0.0%', '0.0%'])
  })
})

describe('the content', () => {
  it("keeps the report's figures", () => {
    expect(TABLE).toHaveLength(7)
    expect(GOAL.rows.map((r) => r.fraction)).toEqual([0.08, 1])
    expect(HOME_PAIRS.map((p) => p.bars.map((b) => b.value))).toEqual([['78.9%', '2.7%'], ['0.04%', '1.73%']])
  })
})
