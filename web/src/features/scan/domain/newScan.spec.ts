import { describe, expect, it } from 'vitest'
import { checkScanUrl, describeScanUrlCheck, MAX_URL_LENGTH, PAGE_OPTIONS, placeStartError } from './newScan'

describe('checkScanUrl', () => {
  it('accepts an http or https address with a host name', () => {
    expect(checkScanUrl(' https://www.twentebelt.com/conveyor-belts/ ')).toEqual({
      kind: 'ok',
      url: 'https://www.twentebelt.com/conveyor-belts/',
    })
    expect(checkScanUrl('http://[2001:db8::1]/')).toEqual({ kind: 'ok', url: 'http://[2001:db8::1]/' })
  })

  it('reads a bare host as https', () => {
    expect(checkScanUrl('twentebelt.com')).toEqual({ kind: 'ok', url: 'https://twentebelt.com' })
  })

  it('refuses what the server would', () => {
    expect(checkScanUrl('  ').kind).toBe('empty')
    expect(checkScanUrl('not a url').kind).toBe('malformed')
    expect(checkScanUrl('ftp://example.com').kind).toBe('scheme')
    expect(checkScanUrl('https://intranet').kind).toBe('noHost')
    expect(checkScanUrl(`https://example.com/${'a'.repeat(MAX_URL_LENGTH)}`).kind).toBe('tooLong')
  })
})

describe('describeScanUrlCheck', () => {
  it('says what to fix, and nothing for a good address', () => {
    expect(describeScanUrlCheck({ kind: 'ok', url: 'https://example.com' })).toBeNull()
    expect(describeScanUrlCheck({ kind: 'empty' })).toBe('Enter the address of a website.')
    expect(describeScanUrlCheck({ kind: 'malformed' })).toBe('Enter a full address, like https://www.example.com.')
    expect(describeScanUrlCheck({ kind: 'scheme' })).toBe('Only http and https addresses can be scanned.')
    expect(describeScanUrlCheck({ kind: 'noHost' })).toContain('full host name')
    expect(describeScanUrlCheck({ kind: 'tooLong' })).toBe('Addresses can be at most 500 characters.')
  })
})

describe('the page budgets', () => {
  it('offers four, all within the contract', () => {
    expect(PAGE_OPTIONS).toEqual([5, 10, 15, 30])
  })
})

describe('placeStartError', () => {
  const problem = (status: number, detail?: string) => ({ type: 'about:blank', title: 'Refused', status, detail })

  it('puts a 422 under the address', () => {
    expect(placeStartError(problem(422, 'The host is not public.'))).toEqual({ field: 'url', message: 'The host is not public.' })
  })

  it('puts a 429 and anything else under the form', () => {
    expect(placeStartError(problem(429, 'Too many scans.'))).toEqual({ field: 'form', message: 'Too many scans.' })
    expect(placeStartError(new TypeError('Failed to fetch'))).toEqual({
      field: 'form',
      message: 'Something went wrong. Try again.',
    })
  })
})
