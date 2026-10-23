/** One recommendation as a view lists it: its number in the report (or an arrow), a bold lead, the rest. */
export type Recommendation = { mark: string; lead: string; text: string }

/** One bar of a goal card: today's value or the target, as a share of the bar. */
export type GoalRow = { label: string; value: string; fraction: number; fill: 'today' | 'target' }
