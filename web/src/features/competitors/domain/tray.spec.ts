import { describe, expect, it } from 'vitest'
import type { CompetitorCandidate } from '../../../infrastructure/api/types.gen'
import { addToTray, batchSummary, duplicateOf, hostIn, nameKey, parseList, pasteIntoTray, traySuggestions, websiteOf } from './tray'

const known = [
  { name: 'Wire Belt Company Ltd', host: 'wirebelt.co.uk' },
  { name: 'Märtens Transportbänder GmbH', host: 'martens-belts.com' },
  { name: 'No Site' },
]

function candidate(name: string, answers: number, address?: string): CompetitorCandidate {
  return { id: name.toLowerCase().replace(/\W/g, '').padEnd(16, '0').slice(0, 16), name, answers, providers: [], models: [], addresses: address ? [address] : [], lastSeenAt: '2026-11-01T06:00:00Z', status: 'pending' }
}

describe('hosts and websites', () => {
  it('read a host from what is typed or pasted', () => {
    expect(hostIn('https://www.Berndorfband-group.com/en')).toBe('berndorfband-group.com')
    expect(hostIn('ipco.com')).toBe('ipco.com')
    expect(hostIn('Lumsden Belting')).toBeNull()
    expect(hostIn('  ')).toBeNull()
    expect(websiteOf('ipco.com')).toBe('https://ipco.com/')
    expect(websiteOf('https://www.ipco.com/de')).toBe('https://www.ipco.com/de')
    expect(websiteOf('Lumsden')).toBe('')
  })
})

describe('duplicates', () => {
  it('match by host or by name without its legal form', () => {
    expect(nameKey('Märtens Transportbänder GmbH (Germany)')).toBe('martens transportbander')
    expect(nameKey('S.A.')).toBe('')
    expect(duplicateOf({ name: 'Other', website: 'https://www.wirebelt.co.uk/' }, known)?.name).toBe('Wire Belt Company Ltd')
    expect(duplicateOf({ name: 'Martens Transportbander', website: '' }, known)?.host).toBe('martens-belts.com')
    expect(duplicateOf({ name: 'Wire Belt Osterloh', website: 'https://wirebelt.de' }, known)).toBeUndefined()
    expect(duplicateOf({ name: 'GmbH', website: '' }, known)).toBeUndefined()
  })

  it('keep the tray free of repeats', () => {
    const tray = addToTray([], { name: 'Intralox', website: 'https://intralox.com/' })
    expect(addToTray(tray, { name: 'Intralox LLC', website: '' })).toHaveLength(1)
    expect(addToTray(tray, { name: 'Habasit', website: '' })).toHaveLength(2)
  })
})

describe('the suggestions', () => {
  const candidates = [candidate('Habasit', 14, 'https://www.habasit.com/en'), candidate('Intralox', 23, 'https://intralox.com'), candidate('Regal', 1)]

  it('show known companies as added, then named ones by how often, then a new company', () => {
    const got = traySuggestions('i', known, candidates)
    expect(got.map((s) => [s.kind, s.name, s.detail])).toEqual([
      ['known', 'Wire Belt Company Ltd', 'Already a competitor'],
      ['known', 'No Site', 'Already a competitor'],
      ['candidate', 'Intralox', 'Named in 23 AI answers'],
      ['candidate', 'Habasit', 'Named in 14 AI answers'],
      ['new', 'Add “i” as a new company', 'Add its website before adding it'],
    ])
    expect(got[2]?.item).toEqual({ name: 'Intralox', website: 'https://intralox.com/', candidateId: candidates[1]?.id })
    expect(traySuggestions('regal', [], candidates)[0]?.detail).toBe('Named in 1 AI answer')
  })

  it('offer a website as a new company named after it, and find a known one by it', () => {
    expect(traySuggestions('https://www.berndorfband-group.com', [], [])).toEqual([
      { kind: 'new', name: 'Add berndorfband-group.com', detail: 'A new company called Berndorfband Group', item: { name: 'Berndorfband Group', website: 'https://www.berndorfband-group.com' } },
    ])
    expect(traySuggestions('wirebelt.co.uk', known, [])[0]?.kind).toBe('known')
    expect(traySuggestions(' ', known, candidates)).toEqual([])
  })
})

describe('a pasted list', () => {
  it('takes one company per line, its name and website in any order', () => {
    expect(parseList('Berndorf Band, berndorfband-group.com\nhttps://www.ipco.com\n\nLumsden Belting\nfenner.com;Fenner Dunlop')).toEqual([
      { name: 'Berndorf Band', website: 'https://berndorfband-group.com/' },
      { name: 'Ipco', website: 'https://www.ipco.com' },
      { name: 'Lumsden Belting', website: '' },
      { name: 'Fenner Dunlop', website: 'https://fenner.com/' },
    ])
  })

  it('skips the lines that repeat a competitor, and names them', () => {
    const got = pasteIntoTray([{ name: 'Ipco', website: '' }], 'Wire Belt, wirebelt.co.uk\nIPCO\nIntralox', known)
    expect(got.tray.map((t) => t.name)).toEqual(['Ipco', 'Intralox'])
    expect(got.skipped).toEqual(['Wire Belt (Wire Belt Company Ltd)'])
  })
})

describe('what a batch did', () => {
  it('says what was added, skipped, refused and scanned', () => {
    expect(batchSummary({ items: [{ name: 'A', outcome: 'added' }], scans: 1 })).toBe('Added 1 competitor. Started 1 scan.')
    expect(
      batchSummary({
        items: [
          { name: 'A', outcome: 'added' },
          { name: 'B', outcome: 'added' },
          { name: 'C', outcome: 'duplicate' },
          { name: 'D', outcome: 'duplicate' },
          { name: 'E', outcome: 'rejected' },
        ],
        scans: 2,
      }),
    ).toBe('Added 2 competitors. Skipped 2 that are competitors already. Could not add 1. Started 2 scans.')
    expect(batchSummary({ items: [{ name: 'C', outcome: 'duplicate' }], scans: 0 })).toBe('Added 0 competitors. Skipped 1 that is a competitor already.')
  })
})
