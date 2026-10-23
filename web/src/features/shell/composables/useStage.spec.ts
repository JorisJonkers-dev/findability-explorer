import { afterEach, describe, expect, it, vi } from 'vitest'
import { effectScope } from 'vue'
import { STAGE_QUERY } from '../domain/stage'
import { useStage } from './useStage'

/** A media query list the test can flip, as a window resized across the breakpoint would. */
function media(matches: boolean) {
  const listeners = new Set<() => void>()
  const list = {
    matches,
    media: STAGE_QUERY,
    addEventListener: (_: string, listener: () => void) => listeners.add(listener),
    removeEventListener: (_: string, listener: () => void) => listeners.delete(listener),
  }
  function flip(next: boolean): void {
    list.matches = next
    for (const listener of listeners) listener()
  }
  return { list, listeners, flip }
}

function run() {
  const scope = effectScope()
  const stage = scope.run(useStage)
  if (!stage) throw new Error('no stage')
  return { stage, scope }
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('useStage', () => {
  it('stacks the views where the window cannot answer media queries', () => {
    const { stage } = run()
    expect(stage.active.value).toBe(false)
    expect(stage.transform.value).toBeUndefined()
  })

  it('fits the stage to the window while it is large enough, following resizes and the breakpoint', () => {
    const query = media(true)
    const matchMedia = vi.fn(() => query.list)
    vi.stubGlobal('matchMedia', matchMedia)
    vi.stubGlobal('innerWidth', 1920)
    vi.stubGlobal('innerHeight', 1200)
    const { stage, scope } = run()
    expect(matchMedia).toHaveBeenCalledWith(STAGE_QUERY)
    expect(stage.active.value).toBe(true)
    expect(stage.transform.value).toBe('translate(0px, 60px) scale(0.5)')

    vi.stubGlobal('innerWidth', 3840)
    vi.stubGlobal('innerHeight', 2160)
    window.dispatchEvent(new Event('resize'))
    expect(stage.transform.value).toBe('translate(0px, 0px) scale(1)')

    query.flip(false)
    expect(stage.active.value).toBe(false)
    expect(stage.transform.value).toBeUndefined()

    scope.stop()
    expect(query.listeners.size).toBe(0)
  })
})
