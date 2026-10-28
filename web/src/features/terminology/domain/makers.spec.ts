import { describe, expect, it } from 'vitest'
import { term } from '../../../testing/fixtures'
import { withMakers } from './makers'

describe('withMakers', () => {
  it("writes the server's makers by the short name the map credits them under", () => {
    const terms = [
      term({
        term: 'Drahtösengliederband',
        usedBy: ['Märtens', 'Pelger', 'Tribelt (Drahtösengliederbänder)'],
        makers: [
          { competitorId: 'pelger', fromScan: false },
          { competitorId: 'esfo-transportbanden', fromScan: true },
          { competitorId: 'nobody', fromScan: true },
        ],
      }),
      term({ term: 'eyelink belt', usedBy: ['ESFO (ESFO Metal Belts)'] }),
    ]
    const companies = [
      { id: 'pelger', name: 'Pelger GmbH' },
      { id: 'pelger-again', name: 'Pelger GmbH' },
      { id: 'esfo-transportbanden', name: 'ESFO Transportbanden B.V. (ESFO Metal Belts, Narviflex Group)' },
    ]
    const [first, second] = withMakers(terms, companies)
    // Märtens is credited but not on the list, so it is no maker; an unknown id is not either.
    expect(first?.usedBy).toEqual(['Pelger', 'ESFO'])
    expect(first?.scanMakers).toEqual(['ESFO'])
    // A term without makers has no companies.
    expect(second).toMatchObject({ usedBy: [], scanMakers: [] })
    expect(withMakers([term({ makers: [{ competitorId: 'pelger', fromScan: false }, { competitorId: 'pelger-again', fromScan: false }] })], companies)[0]?.usedBy).toEqual(['Pelger'])
  })
})
