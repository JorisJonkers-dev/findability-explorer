import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import { searchTermsOptions } from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { Term } from '../../../infrastructure/api/types.gen'
import { describeError } from '../../../shared/describeError'

/** The most terms one search returns; the contract's maximum, so the map comes whole. */
export const SEARCH_LIMIT = 200

/**
 * The whole terminology map, unfiltered: every term, how many there are, and who uses them. One
 * cached request that every view shares; the search terms view filters it in the browser.
 */
export function useTermMap() {
  const map = useQuery(searchTermsOptions({ query: { limit: SEARCH_LIMIT } }))
  return {
    terms: computed<Term[]>(() => (map.data.value ? map.data.value.items : [])),
    total: computed(() => (map.data.value ? map.data.value.total : 0)),
    isLoading: map.isPending,
    error: computed(() => (map.error.value ? describeError(map.error.value) : null)),
  }
}
