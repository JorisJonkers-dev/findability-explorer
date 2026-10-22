import { describe, expect, it } from 'vitest'
import { isInProgress, matchingText, POLL_INTERVAL_MS, pollInterval, progress } from './status'

describe('isInProgress and pollInterval', () => {
  it('polls while the scan is queued or running', () => {
    expect(isInProgress('queued')).toBe(true)
    expect(isInProgress('running')).toBe(true)
    expect(pollInterval('queued')).toBe(POLL_INTERVAL_MS)
    expect(pollInterval('running')).toBe(2000)
  })

  it('stops once it is done or failed, and before anything is known', () => {
    expect(isInProgress('done')).toBe(false)
    expect(pollInterval('done')).toBe(false)
    expect(pollInterval('failed')).toBe(false)
    expect(pollInterval(undefined)).toBe(false)
  })
})

describe('progress', () => {
  it('has no fraction while the scan waits to start', () => {
    expect(progress({ status: 'queued', pagesCrawled: 0, maxPages: 15 })).toEqual({ label: 'Waiting to start', fraction: null })
  })

  it('names the page being read while it runs', () => {
    expect(progress({ status: 'running', pagesCrawled: 3, maxPages: 15 })).toEqual({ label: 'Reading page 4 of 15', fraction: 0.2 })
    expect(progress({ status: 'running', pagesCrawled: 15, maxPages: 15 }).label).toBe('Reading page 15 of 15')
  })

  it('counts the pages read once it is over', () => {
    expect(progress({ status: 'done', pagesCrawled: 10, maxPages: 10 })).toEqual({ label: 'Read 10 of 10 pages', fraction: 1 })
  })
})

describe('matchingText', () => {
  it('says what became of the matching in plain words', () => {
    expect(matchingText('pending')).toBe('Matching starts when the pages are read.')
    expect(matchingText('disabled')).toBe('Product matching is off: no language model is configured on this server.')
    expect(matchingText('skipped')).toBe("Skipped: this month's language-model budget is used up, or the pages had no text.")
    expect(matchingText('failed')).toBe('The language model did not answer. The rest of the scan is complete.')
    expect(matchingText('done')).toBeNull()
  })
})
