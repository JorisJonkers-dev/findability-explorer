import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { effectScope } from 'vue'
import { useZoom, ZOOM_FALLBACK_MS } from './useZoom'

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
})

function zoom() {
  const arrive = vi.fn<(index: number) => void>()
  const scope = effectScope()
  const state = scope.run(() => useZoom(arrive))
  if (!state) throw new Error('no zoom')
  return { state, arrive, scope }
}

describe('useZoom', () => {
  it('grows toward a tile, and arrives once its animation ends', () => {
    const { state, arrive } = zoom()
    expect(state.zoomClass.value).toBe('')
    expect(state.zoomIn(2, '10px 20px')).toBe(true)
    expect(state.zoomClass.value).toBe('zoom-in')
    expect(state.origin.value).toBe('10px 20px')
    // A second click while it moves does nothing.
    expect(state.zoomIn(3, '0px 0px')).toBe(false)
    state.finished()
    expect(arrive).toHaveBeenCalledExactlyOnceWith(2)
    // The poster stays zoomed until the section replaces it, and arrives only once.
    expect(state.zoomClass.value).toBe('zoom-in')
    state.finished()
    vi.advanceTimersByTime(ZOOM_FALLBACK_MS)
    expect(arrive).toHaveBeenCalledOnce()
  })

  it('arrives anyway when the browser never ends the animation', () => {
    const { state, arrive } = zoom()
    state.zoomIn(0, '0px 0px')
    vi.advanceTimersByTime(ZOOM_FALLBACK_MS - 1)
    expect(arrive).not.toHaveBeenCalled()
    vi.advanceTimersByTime(1)
    expect(arrive).toHaveBeenCalledWith(0)
  })

  it('shrinks back from a tile, then stands still', () => {
    const { state } = zoom()
    state.zoomOut(4, '5px 5px')
    expect(state.zoomClass.value).toBe('zoom-out')
    expect(state.origin.value).toBe('5px 5px')
    expect(state.zoomIn(1, '0px 0px')).toBe(false)
    state.finished()
    expect(state.zoomClass.value).toBe('')
  })

  it('stops its timer when the poster goes away', () => {
    const { state, arrive, scope } = zoom()
    state.zoomIn(1, '0px 0px')
    scope.stop()
    vi.advanceTimersByTime(ZOOM_FALLBACK_MS)
    expect(arrive).not.toHaveBeenCalled()
  })
})
