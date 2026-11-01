import { computed, readonly, shallowRef } from 'vue'
import type { RouteLocationRaw } from 'vue-router'
import { type Router, useRouter } from 'vue-router'

const previous = shallowRef<string | null>(null)

/** A page seen in this visit: its path, its address with the query, and its title once its frame has named it. */
export type Visit = { path: string; fullPath: string; title: string }

/** The pages seen in this visit, oldest first; going back to the page before takes the current one off. */
const trail = shallowRef<Visit[]>([])
const TRAIL_MAX = 50

function follow(path: string, fullPath: string): void {
  const t = trail.value
  const current = t.at(-1)
  const before = t.at(-2)
  if (current?.path === path) trail.value = [...t.slice(0, -1), { ...current, fullPath }]
  else if (before?.path === path) trail.value = [...t.slice(0, -2), { ...before, fullPath }]
  else trail.value = [...t, { path, fullPath, title: '' }].slice(-TRAIL_MAX)
}

/**
 * Remembers the path each navigation came from, so a view knows how it was reached: the poster
 * zooms out of the section it came back from, and a section zooms in when the poster opened it.
 */
export function trackPreviousPath(router: Router): void {
  previous.value = null
  trail.value = []
  router.afterEach((to, from, failure) => {
    if (failure !== undefined) return
    previous.value = from.matched.length > 0 ? from.path : null
    follow(to.path, to.fullPath)
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

/** Names the page shown now, so a page opened from it can say "Back to" it. A page frame calls it with its title. */
export function setPageTitle(title: string): void {
  const t = trail.value
  const current = t.at(-1)
  if (current && current.title !== title) trail.value = [...t.slice(0, -1), { ...current, title }]
}

/** The page seen before the current one, or null when the current one was opened first. */
export function usePreviousPage() {
  return computed<Visit | null>(() => trail.value.at(-2) ?? null)
}

/** A page's way back: to the page it came from when that page has a title, else the page's own way back. */
export function backLink(fallback: { to: RouteLocationRaw; label: string } | undefined, before: Visit | null): { to: RouteLocationRaw; label: string } | undefined {
  if (!fallback || !before?.title) return fallback
  return { to: before.fullPath, label: `Back to ${before.title}` }
}
