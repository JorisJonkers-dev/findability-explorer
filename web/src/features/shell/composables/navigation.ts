import { readonly, shallowRef } from 'vue'
import { type Router, useRouter } from 'vue-router'

const previous = shallowRef<string | null>(null)

/**
 * Remembers the path each navigation came from, so a view knows how it was reached: the poster
 * zooms out of the section it came back from, and a section zooms in when the poster opened it.
 */
export function trackPreviousPath(router: Router): void {
  previous.value = null
  router.afterEach((_to, from, failure) => {
    if (failure === undefined) previous.value = from.matched.length > 0 ? from.path : null
  })
}

/** The path the current view was reached from, or null after a direct load. */
export function usePreviousPath() {
  return readonly(previous)
}

/**
 * Goes back to the poster the current view was zoomed into from, back through history, so the
 * back button of the browser and of the view agree. Only a view reached from the poster offers it.
 */
export function useBackToOverview(): () => void {
  const router = useRouter()
  return () => {
    router.back()
  }
}
