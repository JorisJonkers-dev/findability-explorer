import { computed, onScopeDispose, ref } from 'vue'

/** How long a zoom may take before it counts as done anyway: its animation lasts 600 ms. */
export const ZOOM_FALLBACK_MS = 800

type Phase = 'idle' | 'in' | 'arrived' | 'out'

/**
 * The poster's zoom. zoomIn grows the poster toward a tile; when its animation ends (or the
 * fallback passes, should the browser skip it) arrive is called once. zoomOut shrinks the
 * poster back from a tile, as it shows again after a section.
 */
export function useZoom(arrive: (index: number) => void) {
  const phase = ref<Phase>('idle')
  const target = ref(-1)
  const origin = ref<string | undefined>(undefined)
  let timer: ReturnType<typeof setTimeout> | undefined

  function start(next: Phase, index: number, from: string): void {
    clearTimeout(timer)
    phase.value = next
    target.value = index
    origin.value = from
    timer = setTimeout(finished, ZOOM_FALLBACK_MS)
  }

  /** The animation ended. */
  function finished(): void {
    clearTimeout(timer)
    if (phase.value === 'in') {
      phase.value = 'arrived'
      arrive(target.value)
    } else if (phase.value === 'out') {
      phase.value = 'idle'
    }
  }

  onScopeDispose(() => {
    clearTimeout(timer)
  })

  return {
    /** The poster's animation class, or '' while it stands still. */
    zoomClass: computed(() => {
      if (phase.value === 'out') return 'zoom-out'
      return phase.value === 'idle' ? '' : 'zoom-in'
    }),
    origin,
    /** Grows the poster toward tile index, from origin; a second click while it moves is ignored. */
    zoomIn: (index: number, from: string): boolean => {
      if (phase.value !== 'idle') return false
      start('in', index, from)
      return true
    },
    zoomOut: (index: number, from: string): void => {
      start('out', index, from)
    },
    finished,
  }
}
