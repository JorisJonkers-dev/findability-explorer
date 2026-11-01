import type { ReadinessCheckState } from '../../../infrastructure/api/types.gen'

/** How far checking every site's AI readiness again has come, or how it ended; empty when none ran. */
export function readinessCheckLine(check: ReadinessCheckState | undefined): string {
  if (!check || check.total === 0) return ''
  const sites = (n: number) => (n === 1 ? '1 site' : `${String(n)} sites`)
  if (check.running) return `Checking ${String(check.checked)} of ${sites(check.total)} again.`
  const read = check.total - check.failed
  const done = `Checked ${sites(read)} again.`
  if (check.failed === 0) return done
  return check.failed === 1
    ? `${done} 1 site could not be read and keeps its previous result.`
    : `${done} ${String(check.failed)} sites could not be read and keep their previous result.`
}
