// Runs are stamped in UTC, and the notes and charts write them as stamped.
const LONG = new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'long', year: 'numeric', timeZone: 'UTC' })
const DAY = new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'long', timeZone: 'UTC' })
const CLOCK = new Intl.DateTimeFormat('en-GB', { hour: '2-digit', minute: '2-digit', hourCycle: 'h23', timeZone: 'UTC' })

/** "27 October 2026". */
export function longDate(iso: string): string {
  return LONG.format(new Date(iso))
}

/** "27 October", under a point on a chart. */
export function dayMonth(iso: string): string {
  return DAY.format(new Date(iso))
}

/** "06:03", the time of day. */
export function clock(iso: string): string {
  return CLOCK.format(new Date(iso))
}
