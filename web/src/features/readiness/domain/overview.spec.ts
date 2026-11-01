import { describe, expect, it } from 'vitest'
import { readinessCheckLine } from './overview'

describe('checking AI readiness again', () => {
  it('says how far a check has come, and how it ended', () => {
    expect(readinessCheckLine(undefined)).toBe('')
    expect(readinessCheckLine({ running: false, checked: 0, failed: 0, total: 0 })).toBe('')
    expect(readinessCheckLine({ running: true, checked: 0, failed: 0, total: 1 })).toBe('Checking 0 of 1 site again.')
    expect(readinessCheckLine({ running: true, checked: 12, failed: 0, total: 40 })).toBe('Checking 12 of 40 sites again.')
    expect(readinessCheckLine({ running: false, checked: 40, failed: 0, total: 40 })).toBe('Checked 40 sites again.')
    expect(readinessCheckLine({ running: false, checked: 40, failed: 1, total: 40 })).toBe("Checked 39 sites again. 1 site could not be read and keeps its previous result.")
    expect(readinessCheckLine({ running: false, checked: 2, failed: 2, total: 2 })).toBe("Checked 0 sites again. 2 sites could not be read and keep their previous result.")
  })
})
