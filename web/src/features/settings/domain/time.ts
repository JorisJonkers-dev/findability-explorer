const AT = new Intl.DateTimeFormat('en-GB', {
  day: 'numeric',
  month: 'short',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
  timeZone: 'UTC',
})

/** A moment as the settings show it, in UTC like the token budget: "23 Oct 2026, 09:30 UTC". */
export function atLabel(iso: string): string {
  return `${AT.format(new Date(iso))} UTC`
}
