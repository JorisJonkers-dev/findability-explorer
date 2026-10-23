import { computed, onScopeDispose, ref } from 'vue'
import { fitStage, STAGE_QUERY, stageTransform } from '../domain/stage'

/**
 * Whether the window is large enough for the stage, and the transform that fits the stage in it,
 * kept up to date as the window is resized. Without matchMedia (a test's DOM) the views stack.
 */
export function useStage() {
  const media = typeof window.matchMedia === 'function' ? window.matchMedia(STAGE_QUERY) : null
  const active = ref(media?.matches ?? false)
  const fit = ref(fitStage(window.innerWidth, window.innerHeight))

  function update(): void {
    active.value = media?.matches ?? false
    fit.value = fitStage(window.innerWidth, window.innerHeight)
  }

  window.addEventListener('resize', update)
  media?.addEventListener('change', update)
  onScopeDispose(() => {
    window.removeEventListener('resize', update)
    media?.removeEventListener('change', update)
  })

  return {
    active,
    /** The stage's transform, or undefined while the views stack. */
    transform: computed(() => (active.value ? stageTransform(fit.value) : undefined)),
  }
}
