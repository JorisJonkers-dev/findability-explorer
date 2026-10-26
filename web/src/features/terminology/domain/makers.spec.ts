import { describe, expect, it } from 'vitest'
import { term } from '../../../testing/fixtures'
import { withScanMakers } from './makers'

describe('withScanMakers', () => {
  it('adds the companies a scan found a term at, by the short name the map credits them under', () => {
    const terms = [
      term({ term: 'Drahtösengliederband', usedBy: ['Märtens', 'Pelger'], foundBy: ['pelger', 'esfo-transportbanden', 'nobody'] }),
      term({ term: 'eyelink belt', usedBy: ['ESFO (ESFO Metal Belts)'] }),
    ]
    const companies = [
      { id: 'pelger', name: 'Pelger GmbH' },
      { id: 'esfo-transportbanden', name: 'ESFO Transportbanden B.V. (ESFO Metal Belts, Narviflex Group)' },
    ]
    const [first, second] = withScanMakers(terms, companies)
    // Pelger is named already; ESFO is added under its credited short name; an unknown id is not.
    expect(first?.usedBy).toEqual(['Märtens', 'Pelger', 'ESFO'])
    expect(first?.scanMakers).toEqual(['ESFO'])
    expect(second?.scanMakers).toEqual([])
    expect(second?.usedBy).toEqual(['ESFO (ESFO Metal Belts)'])
  })
})
