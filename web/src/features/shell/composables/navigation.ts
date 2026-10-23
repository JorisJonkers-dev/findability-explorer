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
 * Goes back to the poster: back through history when the poster opened this view, so the back
 * button of the browser and of the view agree; otherwise to the poster as a new page.
 */
export function useBackToOverview(): () => void {
  const router = useRouter()
  return () => {
    if (previous.value === '/') router.back()
    else void router.push('/')
  }
}
