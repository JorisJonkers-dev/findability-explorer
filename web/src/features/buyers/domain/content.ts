import type { GoalRow, Recommendation } from '../../../shared/ui/types'

/**
 * Google Analytics, Tribelt's export: paid search, organic search and AI assistants side by side.
 * The figures are the report's; the period is still to be filled in there too.
 */

export const TITLE = 'Paid search buys traffic, not buyers.'
export const SOURCE = ['Google Analytics, Tribelt export', 'Period: [GA REPORTING PERIOD]'] as const

/** The three channels, in the chart's order. */
export const CHANNELS = ['Paid search', 'Organic search', 'AI assistants'] as const

/** The channels' raw figures, in the order of CHANNELS. */
export const USERS = [18_772, 640, 13] as const
export const RETURNING_USERS = [254, 166, 3] as const
/** Key events: the contacts (quotation requests, calls, e-mails) Google Analytics counted. */
export const KEY_EVENTS = [11, 12, 0] as const
/** Paid search's share of every user of the website, in percent, as the report counts it; the three channels are not all of them. */
export const PAID_SHARE_OF_ALL_USERS = 78.89

export type MetricId = 'users' | 'rate' | 'time' | 'sessions' | 'returning'

/** How a metric's value is written: a count, a percentage, minutes and seconds, or a ratio. */
export type MetricFormat = 'count' | 'percent2' | 'percent1' | 'duration' | 'ratio'

export type Metric = {
  id: MetricId
  label: string
  /** One value per channel, in the order of CHANNELS. */
  values: readonly [number, number, number]
  format: MetricFormat
  /** What the chart shows, in a sentence. */
  caption: string
}

export const METRICS: readonly Metric[] = [
  { id: 'users', label: 'Users', values: USERS, format: 'count', caption: 'Paid search brings 29 times more users than organic search.' },
  {
    id: 'rate',
    label: 'Contact rate',
    values: [0.04, 1.73, 0],
    format: 'percent2',
    caption: 'Organic visitors contact Tribelt 43 times more often than paid visitors.',
  },
  { id: 'time', label: 'Engagement time', values: [22, 64, 29], format: 'duration', caption: 'Organic visitors stay three times longer than paid visitors.' },
  {
    id: 'sessions',
    label: 'Engaged sessions',
    values: [0.34, 1.05, 1.15],
    format: 'ratio',
    caption: 'Most paid visitors leave after viewing just one page.',
  },
  {
    id: 'returning',
    label: 'Returning users',
    values: [(254 / 18_772) * 100, (166 / 640) * 100, (3 / 13) * 100],
    format: 'percent1',
    caption: 'One organic visitor in four comes back; one paid visitor in seventy.',
  },
]

/** The metric the chart opens on. */
export const DEFAULT_METRIC: MetricId = 'rate'

/** Every figure, as the table under the chart lists it. */
export const TABLE: readonly { measure: string; paid: string; organic: string; ai: string }[] = [
  { measure: 'Users', paid: '18,772', organic: '640', ai: '13' },
  { measure: 'New users', paid: '18,678', organic: '595', ai: '13' },
  { measure: 'Returning users', paid: '254', organic: '166', ai: '3' },
  { measure: 'Engagement time', paid: '22 s', organic: '1 min 04 s', ai: '29 s' },
  { measure: 'Engaged sessions per user', paid: '0.34', organic: '1.05', ai: '1.15' },
  { measure: 'Key events (contacts)', paid: '11', organic: '12', ai: '0' },
  { measure: 'User key event rate', paid: '0.04%', organic: '1.73%', ai: '0%' },
]

export const RECOMMENDATIONS: readonly Recommendation[] = [
  {
    mark: '1',
    lead: 'Count qualified leads, not visits.',
    text: 'Quotation requests and tracked calls and e-mails from product pages, fed back to Google Ads.',
  },
  { mark: '2', lead: "Restructure paid search, don't cut it.", text: 'Campaigns per market and belt family, judged on qualified leads.' },
]

export const GOAL = {
  goal: 'G2',
  text: 'Paid users who contact Tribelt',
  rows: [
    { label: 'Today', value: '0.04%', fraction: 0.04 / 0.5, fill: 'today' },
    { label: 'Target', value: '0.5%', fraction: 1, fill: 'target' },
  ] satisfies GoalRow[],
} as const
