import type { Competitor, RescanBatch } from '../../../infrastructure/api/types.gen'
import { zProblem } from '../../../infrastructure/api/zod.gen'
import { describeError } from '../../../shared/describeError'
import { checkScanUrl, describeScanUrlCheck } from '../../scan/domain/newScan'

export const MAX_NAME_LENGTH = 200

/** What removing a company does, in the words its confirm button uses. */
export function removeAction(company: Pick<Competitor, 'source' | 'scans'>): { button: string; question: string } {
  if (company.source === 'research') {
    return {
      button: 'Archive',
      question: 'Archive this researched company? It leaves the list and the re-scans; its scans stay, and it can be restored.',
    }
  }
  const scans = company.scans === 1 ? '1 scan' : `${String(company.scans)} scans`
  return { button: `Delete with its ${scans}`, question: `Delete this company with its ${scans}? This cannot be undone.` }
}

/** A company form, checked: the values to send, or what is wrong with them. */
export type CompanyFormCheck = { kind: 'ok'; name: string; url: string } | { kind: 'invalid'; field: 'name' | 'url'; message: string }

/** Checks a company's name and website before they are sent. */
export function checkCompanyForm(name: string, website: string): CompanyFormCheck {
  const trimmed = name.trim()
  if (trimmed === '') return { kind: 'invalid', field: 'name', message: 'Enter the company’s name.' }
  if (trimmed.length > MAX_NAME_LENGTH) {
    return { kind: 'invalid', field: 'name', message: `A name has at most ${String(MAX_NAME_LENGTH)} characters.` }
  }
  const check = checkScanUrl(website)
  const message = describeScanUrlCheck(check)
  if (check.kind !== 'ok') return { kind: 'invalid', field: 'url', message: String(message) }
  return { kind: 'ok', name: trimmed, url: check.url }
}

/** A refused change to a company: a 409 names the company that has the website already. */
export function companyProblem(error: unknown): string {
  const problem = zProblem.safeParse(error)
  if (problem.success && problem.data.status === 403) return 'Only administrators can change the companies.'
  return describeError(error)
}

/** What a re-scan did: "Queued 12 scans. Skipped 2: Märtens (a scan of it is still running)." */
export function rescanSummary(batch: RescanBatch): string {
  const queued = `Queued ${String(batch.queued.length)} ${batch.queued.length === 1 ? 'scan' : 'scans'}.`
  if (batch.skipped.length === 0) return queued
  const skipped = batch.skipped.map((s) => `${s.name} (${s.reason.replace(/\.$/, '').toLowerCase()})`).join(', ')
  return `${queued} Skipped ${String(batch.skipped.length)}: ${skipped}.`
}

/** Archived researched companies apart from the rest, so only administrators see them. */
export function splitArchived<T extends Pick<Competitor, 'archived'>>(companies: readonly T[]): { active: T[]; archived: T[] } {
  return { active: companies.filter((c) => !c.archived), archived: companies.filter((c) => c.archived) }
}
