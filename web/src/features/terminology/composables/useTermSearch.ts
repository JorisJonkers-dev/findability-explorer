import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import { searchTermsOptions } from '../../../infrastructure/api/@tanstack/vue-query.gen'
import { describeError } from '../../../shared/describeError'
import { useCompetitors } from '../../competitors/composables/useCompetitors'
import { type MapTerm, withMakers } from '../domain/makers'

/** The most terms one search returns; the contract's maximum, so the map comes whole. */
export const SEARCH_LIMIT = 10_000

/**
 * The whole terminology map, unfiltered: every term, how many there are, and who uses them: the
 * competitors on the list that the research or the review credits, or whose newest scan found it. Two
 * cached requests that every view shares; the search terms view filters the map in the browser.
 */
export function useTermMap() {
  const map = useQuery(searchTermsOptions({ query: { limit: SEARCH_LIMIT } }))
  const { competitors } = useCompetitors()
  return {
    // The makers are the server's, by the names the map credits them under (withMakers).
    terms: computed<MapTerm[]>(() => withMakers(map.data.value ? map.data.value.items : [], competitors.value)),
    total: computed(() => (map.data.value ? map.data.value.total : 0)),
    /** What says which terms tribelt.nl uses: its newest scan, or the research before one. */
    tribeltFrom: computed(() => map.data.value?.tribeltFrom),
    isLoading: map.isPending,
    error: computed(() => (map.error.value ? describeError(map.error.value) : null)),
  }
}
