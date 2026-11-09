/** How precisely a dollar amount is written: up to some decimals, or to some significant digits for tiny costs. */
export type Precision = { decimals?: number; significant?: number }

const formats = new Map<string, Intl.NumberFormat>()

function format(p: Precision): Intl.NumberFormat {
  const key = `${String(p.decimals)}:${String(p.significant)}`
  let f = formats.get(key)
  if (!f) {
    f = new Intl.NumberFormat(
      'en-US',
      p.significant !== undefined
        ? { style: 'currency', currency: 'USD', maximumSignificantDigits: p.significant }
        : { style: 'currency', currency: 'USD', minimumFractionDigits: 2, maximumFractionDigits: Math.max(2, p.decimals ?? 2) },
    )
    formats.set(key, f)
  }
  return f
}

/** Dollars as people read them: "$2.75" to the cent, "$0.0042" to four decimals, "$0.0014" to two significant digits. */
export function dollars(usd: number, precision: Precision = {}): string {
  return format(precision).format(usd)
}

const NUMBER = new Intl.NumberFormat('en-GB')

/** A count with its noun: "1 question", "1,200 questions", "2 companies". */
export function plural(n: number, one: string, many = `${one}s`): string {
  return `${NUMBER.format(n)} ${n === 1 ? one : many}`
}
