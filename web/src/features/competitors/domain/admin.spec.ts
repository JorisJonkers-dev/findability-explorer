import { describe, expect, it } from 'vitest'
import { checkCompanyForm, companyProblem, removeAction, rescanSummary, splitArchived } from './admin'

describe('removing a company', () => {
  it('archives a researched one, and deletes an added one with its scans', () => {
    expect(removeAction({ source: 'research', scans: 3 }).button).toBe('Archive')
    expect(removeAction({ source: 'added', scans: 1 })).toEqual({ button: 'Delete with its 1 scan', question: 'Delete this company with its 1 scan? This cannot be undone.' })
    expect(removeAction({ source: 'added', scans: 0 }).button).toBe('Delete with its 0 scans')
  })
})

describe('the company form', () => {
  it('checks the name and the website', () => {
    expect(checkCompanyForm(' Example ', 'example.com')).toEqual({ kind: 'ok', name: 'Example', url: 'https://example.com' })
    expect(checkCompanyForm('', 'example.com')).toEqual({ kind: 'invalid', field: 'name', message: 'Enter the company’s name.' })
    expect(checkCompanyForm('x'.repeat(201), 'example.com')).toMatchObject({ field: 'name', message: 'A name has at most 200 characters.' })
    expect(checkCompanyForm('Example', 'ftp://example.com')).toMatchObject({ field: 'url', message: 'Only http and https addresses can be scanned.' })
  })

  it('says why the server refused', () => {
    expect(companyProblem({ type: 'about:blank', title: 'Forbidden', status: 403 })).toBe('Only administrators can change the companies.')
    expect(companyProblem({ type: 'about:blank', title: 'Conflict', status: 409, detail: 'Twentebelt already has this website.' })).toBe(
      'Twentebelt already has this website.',
    )
    expect(companyProblem(new Error('boom'))).toBe('Something went wrong. Try again.')
  })
})

describe('a re-scan', () => {
  it('says what was queued and skipped', () => {
    expect(rescanSummary({ queued: [], skipped: [] })).toBe('Queued 0 scans.')
    const one = { id: 'a', url: 'https://a.com', status: 'queued' as const, maxPages: 60, pagesCrawled: 0, createdAt: '2026-10-23T09:30:00Z' }
    expect(rescanSummary({ queued: [one], skipped: [{ competitorId: 'b', name: 'B', reason: 'It has no website.' }] })).toBe(
      'Queued 1 scan. Skipped 1: B (it has no website).',
    )
  })

  it('keeps archived companies apart', () => {
    expect(splitArchived([{ id: 'a', archived: false }, { id: 'b', archived: true }])).toEqual({
      active: [{ id: 'a', archived: false }],
      archived: [{ id: 'b', archived: true }],
    })
  })
})
