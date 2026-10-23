import { computed, onScopeDispose, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { parseView, type TermView, type ToggleKey, toggled, toViewQuery } from '../domain/filters'

/** How long typing has to pause before the search follows it. */
export const SEARCH_DELAY_MS = 250

/**
 * The search terms view's filters, kept in the URL's query string so a view can be shared and
 * survives a reload. The search box edits a draft that reaches the URL once typing pauses.
 */
export function useTermView(delay = SEARCH_DELAY_MS) {
  const route = useRoute()
  const router = useRouter()
  const view = computed(() => parseView(route.query))
  const draft = ref(view.value.q)
  let timer: ReturnType<typeof setTimeout> | undefined

  function go(next: TermView): void {
    void router.replace({ query: toViewQuery(next) })
  }

  watch(draft, (q) => {
    clearTimeout(timer)
    if (q === view.value.q) return
    timer = setTimeout(() => {
      go({ ...view.value, q })
    }, delay)
  })

  // A link or the back button can change the query too; the search box follows it.
  watch(
    () => view.value.q,
    (q) => {
      draft.value = q
    },
  )

  onScopeDispose(() => {
    clearTimeout(timer)
  })

  return {
    view,
    draft,
    /** Sets a filter, or unsets it when it already has that value: a chart row or pill that toggles. */
    toggle: (key: ToggleKey, value: string) => {
      go(toggled(view.value, key, value))
    },
    /** Unsets one filter: a chip's remove button. */
    clear: (key: ToggleKey) => {
      go({ ...view.value, [key]: key === 'user' ? '' : null })
    },
    /** Shows only the terms Tribelt lacks, or every term again. */
    toggleGap: () => {
      go({ ...view.value, gap: !view.value.gap })
    },
  }
}
