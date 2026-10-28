import { CHANNELS, KEY_EVENTS, PAID_SHARE_OF_ALL_USERS, type Metric, type MetricFormat, METRICS, type MetricId, USERS } from './content'

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

/** The index of the largest value; the first wins a tie. */
function peak(values: readonly number[]): number {
  return values.indexOf(Math.max(...values))
}

/** The three figures at the top of the page, from the export: the users, the contact rate and the AI assistants. */
export function buyerFigures(users: readonly number[] = USERS, events: readonly number[] = KEY_EVENTS): { big: string; text: string }[] {
  const rate = metric('rate')
  const [paidRate, organicRate] = rate.values
  const ai = users[2] ?? 0
  const aiContacts = events[2] ?? 0
  return [
    {
      big: formatValue(users[0] ?? 0, 'count'),
      text: `users came from paid search, ${String(Math.round(PAID_SHARE_OF_ALL_USERS))}% of all users`,
    },
    {
      big: formatValue(organicRate, 'percent2'),
      text: `of organic visitors got in touch, against ${formatValue(paidRate, 'percent2')} of paid visitors`,
    },
    {
      big: formatValue(ai, 'count'),
      text: `${ai === 1 ? 'user came' : 'users came'} from AI assistants${aiContacts === 0 ? ', and none of them got in touch' : ''}`,
    },
  ]
}

/** The overview's line about the channels: which brings the most visitors, and which the visitors who get in touch. */
export function visitorsLine(users: readonly number[] = USERS, rates: readonly number[] = metric('rate').values): string {
  const most = peak(users)
  const contact = peak(rates)
  const first = CHANNELS[most] as string
  if (most === contact) return `${first} brings the most visitors, and the most who get in touch.`
  return `${first} brings the most visitors, but ${(CHANNELS[contact] as string).toLowerCase()} brings the visitors who get in touch.`
}
