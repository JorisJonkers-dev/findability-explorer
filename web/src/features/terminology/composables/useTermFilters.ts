import { computed, onScopeDispose, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { type FilterKey, parseFilters, toRouteQuery } from '../domain/filters'

/** How long typing has to pause before the search follows it. */
export const SEARCH_DELAY_MS = 250

/**
 * The terminology page's filters, kept in the URL's query string so a view can be shared and
 * survives a reload. The search box edits a draft that reaches the URL once typing pauses.
 */
export function useTermFilters(delay = SEARCH_DELAY_MS) {
  const route = useRoute()
  const router = useRouter()
  const filters = computed(() => parseFilters(route.query))
  const draft = ref(filters.value.q)
  let timer: ReturnType<typeof setTimeout> | undefined

  function update(patch: Partial<Record<FilterKey, string>>): void {
    const next = parseFilters({ ...toRouteQuery(filters.value), ...patch })
    void router.replace({ query: toRouteQuery(next) })
  }

  /** Sets a filter, or unsets it when it already has that value: a pill that toggles. */
  function toggle(key: Exclude<FilterKey, 'q'>, value: string): void {
    update({ [key]: filters.value[key] === value ? '' : value })
  }

  watch(draft, (q) => {
    clearTimeout(timer)
    if (q === filters.value.q) return
    timer = setTimeout(() => {
      update({ q })
    }, delay)
  })

  // A link or the back button can change the query too; the search box follows it.
  watch(
    () => filters.value.q,
    (q) => {
      draft.value = q
    },
  )

  onScopeDispose(() => {
    clearTimeout(timer)
  })

  return {
    filters,
    draft,
    update,
    toggle,
    clear: () => {
      clearTimeout(timer)
      draft.value = ''
      void router.replace({ query: {} })
    },
  }
}
