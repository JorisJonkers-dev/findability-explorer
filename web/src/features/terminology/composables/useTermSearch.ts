import { keepPreviousData, useQuery } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import { searchTermsOptions } from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { Facets, Term } from '../../../infrastructure/api/types.gen'
import { describeError } from '../../../shared/describeError'
import { type TermFilters, toSearchQuery } from '../domain/filters'

/** The most terms one search returns; the contract's maximum, so the map shows whole. */
export const SEARCH_LIMIT = 200

const NO_FACETS: Facets = { languages: [], tiers: [], families: [] }

/** Server state for the keyword search: the terms that match filters, and the facet counts. */
export function useTermSearch(filters: Ref<TermFilters>) {
  const search = useQuery(
    computed(() => ({
      ...searchTermsOptions({ query: toSearchQuery(filters.value, SEARCH_LIMIT) }),
      // The previous results stay on screen while the next search loads, so the table does not flicker.
      placeholderData: keepPreviousData,
    })),
  )

  return {
    terms: computed<Term[]>(() => (search.data.value ? search.data.value.items : [])),
    total: computed(() => (search.data.value ? search.data.value.total : 0)),
    facets: computed<Facets>(() => (search.data.value ? search.data.value.facets : NO_FACETS)),
    isLoading: search.isPending,
    error: computed(() => (search.error.value ? describeError(search.error.value) : null)),
  }
}

const NO_FILTERS: TermFilters = { q: '', language: null, tier: null, family: null, user: '' }

/**
 * The whole terminology map, unfiltered: how many terms it holds, and who uses them. It is the
 * same request as an empty search, so the two share one response.
 */
export function useTermMap() {
  const map = useQuery(searchTermsOptions({ query: toSearchQuery(NO_FILTERS, SEARCH_LIMIT) }))
  return {
    terms: computed<Term[]>(() => (map.data.value ? map.data.value.items : [])),
    total: computed(() => (map.data.value ? map.data.value.total : 0)),
  }
}
