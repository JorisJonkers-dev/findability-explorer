import type { LlmKeyUsage } from '../../../infrastructure/api/types.gen'

const DOLLARS = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', minimumFractionDigits: 2, maximumFractionDigits: 4 })
const NUMBER = new Intl.NumberFormat('en-GB')
const MONTH = new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'short', year: 'numeric', timeZone: 'UTC' })

/** Dollars as people read them: "$1.20", "$0.0042". */
export function dollars(amount: number): string {
  return DOLLARS.format(amount)
}

/** What a provider reports a key used, a line each: spend, limit, free requests, or tokens per model. */
export function usageLines(u: LlmKeyUsage): string[] {
  const lines: string[] = []
  if (u.source === 'openrouter') {
    const spend = [
      u.dailyUsd === undefined ? null : `${dollars(u.dailyUsd)} today`,
      u.weeklyUsd === undefined ? null : `${dollars(u.weeklyUsd)} this week`,
      u.monthlyUsd === undefined ? null : `${dollars(u.monthlyUsd)} this month`,
      u.totalUsd === undefined ? null : `${dollars(u.totalUsd)} in all`,
    ].filter((s) => s !== null)
    lines.push(`${spend.join(', ')}.`)
    if (u.limitUsd === undefined) {
      lines.push('No spending limit on this key.')
    } else {
      const reset = u.limitReset ? ` a ${u.limitReset.replace(/ly$/, '').replace('dai', 'day')}` : ''
      lines.push(`Limit ${dollars(u.limitUsd)}${reset}, ${dollars(u.limitRemainingUsd ?? 0)} left.`)
    }
    if (u.freeRequests) {
      lines.push(`Free models: ${NUMBER.format(u.freeRequests.used)} of ${NUMBER.format(u.freeRequests.limit)} requests today.`)
    }
  }
  if (u.source === 'anthropic') {
    const since = u.since ? ` since ${MONTH.format(new Date(u.since))}` : ''
    lines.push(u.models.length === 0 ? `No tokens${since}.` : `Tokens${since}:`)
    for (const m of u.models) {
      lines.push(`${m.model}: ${NUMBER.format(m.inputTokens)} in, ${NUMBER.format(m.outputTokens)} out, ${NUMBER.format(m.cacheReadTokens)} from the cache`)
    }
    if (u.organisationCostUsd !== undefined) lines.push(`The organisation's cost this month: ${dollars(u.organisationCostUsd)}.`)
  }
  return lines
}
