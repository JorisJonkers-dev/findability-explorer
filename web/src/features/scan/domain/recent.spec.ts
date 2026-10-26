import { describe, expect, it } from 'vitest'
import { recentLabel } from './recent'

const now = new Date(2026, 9, 22, 15, 0)
const at = (day: number) => new Date(2026, 9, day, 9, 30).toISOString()

describe('recentLabel', () => {
  it('says a scan is still going, or failed', () => {
    expect(recentLabel({ status: 'running', createdAt: at(22) }, now)).toBe('scanning')
    expect(recentLabel({ status: 'queued', createdAt: at(20) }, now)).toBe('scanning')
    expect(recentLabel({ status: 'failed', createdAt: at(22) }, now)).toBe('failed')
    expect(recentLabel({ status: 'failed', blocked: true, createdAt: at(22) }, now)).toBe('blocked')
  })

  it('dates a finished scan by the day', () => {
    expect(recentLabel({ status: 'done', createdAt: at(22) }, now)).toBe('today')
    expect(recentLabel({ status: 'done', createdAt: at(21) }, now)).toBe('yesterday')
    expect(recentLabel({ status: 'done', createdAt: at(3) }, now)).toBe('3 Oct')
  })

  it('reads the clock when no time is given', () => {
    expect(recentLabel({ status: 'done', createdAt: new Date().toISOString() })).toBe('today')
  })
})
