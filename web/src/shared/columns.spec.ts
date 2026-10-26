import { describe, expect, it } from 'vitest'
import { columnTemplate, minimised, parseLayout, restored, widened } from './columns'

describe('the stage columns', () => {
  it('lays out each column as designed, widened or as a strip', () => {
    expect(columnTemplate([1, 2, 1], ['normal', 'normal', 'normal'])).toBe('minmax(0,1fr) minmax(0,2fr) minmax(0,1fr)')
    expect(columnTemplate([1, 2, 1], ['wide', 'min', 'normal'])).toBe('minmax(0,2.5fr) 150px minmax(0,1fr)')
    expect(columnTemplate([1, 1], [])).toBe('minmax(0,1fr) minmax(0,1fr)')
  })

  it('widens and puts back, minimises all but the last open column, and restores', () => {
    expect(widened(['normal', 'normal'], 0)).toEqual(['wide', 'normal'])
    expect(widened(['wide', 'normal'], 0)).toEqual(['normal', 'normal'])
    expect(widened(['min', 'normal'], 0)).toEqual(['wide', 'normal'])
    expect(minimised(['normal', 'normal'], 1)).toEqual(['normal', 'min'])
    expect(minimised(['normal', 'min'], 0)).toEqual(['normal', 'min'])
    expect(minimised(['normal', 'min'], 1)).toEqual(['normal', 'min'])
    expect(restored(['min', 'wide'], 0)).toEqual(['normal', 'wide'])
  })

  it('reads a stored layout back only when it fits the page', () => {
    expect(parseLayout('["wide","min","normal"]', 3)).toEqual(['wide', 'min', 'normal'])
    expect(parseLayout('["wide","min"]', 3)).toBeNull()
    expect(parseLayout('["min","min"]', 2)).toBeNull()
    expect(parseLayout('["big","normal"]', 2)).toBeNull()
    expect(parseLayout('{"a":1}', 2)).toBeNull()
    expect(parseLayout('not json', 2)).toBeNull()
    expect(parseLayout(null, 2)).toBeNull()
  })
})
