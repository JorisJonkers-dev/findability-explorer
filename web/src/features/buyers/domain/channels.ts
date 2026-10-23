import { CHANNELS, KEY_EVENTS, type Metric, type MetricFormat, METRICS, type MetricId, USERS } from './content'

/** Seconds as the chart writes them: 22 is "0:22", 64 is "1:04". */
export function duration(seconds: number): string {
  const whole = Math.round(seconds)
  return `${String(Math.floor(whole / 60))}:${String(whole % 60).padStart(2, '0')}`
}

/** A metric's value, written as the chart shows it. */
export function formatValue(value: number, format: MetricFormat): string {
  switch (format) {
    case 'count':
      return value.toLocaleString('en-GB')
    case 'percent2':
      return `${value.toFixed(2)}%`
    case 'percent1':
      return `${value.toFixed(1)}%`
    case 'duration':
      return duration(value)
    case 'ratio':
      return value.toFixed(2)
  }
}

/** The metric with this id. */
export function metric(id: MetricId): Metric {
  return METRICS.find((m) => m.id === id) as Metric
}

export type Column = {
  channel: (typeof CHANNELS)[number]
  value: string
  /** The column's height as a share of the tallest, from 0 to 1. */
  fraction: number
}

/** The chart's three columns for a metric: the tallest is full height. */
export function channelColumns(m: Metric): Column[] {
  const max = Math.max(...m.values)
  return CHANNELS.map((channel, i) => {
    const v = m.values[i] as number
    return { channel, value: formatValue(v, m.format), fraction: max > 0 ? v / max : 0 }
  })
}

/** The chart in words, for a screen reader: "Contact rate: paid search 0.04%, organic search 1.73%, AI assistants 0.00%." */
export function chartLabel(m: Metric): string {
  const [paid, organic, ai] = channelColumns(m).map((c) => c.value) as [string, string, string]
  return `${m.label}: paid search ${paid}, organic search ${organic}, AI assistants ${ai}.`
}

/** A share as the volume-against-value bars write it; a sliver above nothing is "<0.1%". */
export function shareLabel(share: number): string {
  return share > 0 && share < 0.1 ? '<0.1%' : `${share.toFixed(1)}%`
}

export type VolumeValue = {
  channel: (typeof CHANNELS)[number]
  visits: string
  visitsFraction: number
  contacts: string
  contactsFraction: number
}

/** Of the three channels' visitors: each one's share of the visits and of the contacts. */
export function volumeAgainstValue(users: readonly number[] = USERS, events: readonly number[] = KEY_EVENTS): VolumeValue[] {
  const totalUsers = users.reduce((a, b) => a + b, 0)
  const totalEvents = events.reduce((a, b) => a + b, 0)
  const share = (part: number, total: number) => (total > 0 ? part / total : 0)
  return CHANNELS.map((channel, i) => {
    const visits = share(users[i] as number, totalUsers)
    const contacts = share(events[i] as number, totalEvents)
    return {
      channel,
      visits: shareLabel(visits * 100),
      visitsFraction: visits,
      contacts: shareLabel(contacts * 100),
      contactsFraction: contacts,
    }
  })
}
