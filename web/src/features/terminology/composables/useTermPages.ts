import { useQuery } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import { getTermPagesOptions } from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { Language } from '../../../infrastructure/api/types.gen'
import { describeError } from '../../../shared/describeError'
import { splitTermPages } from '../domain/termPages'

/** Server state for the pages that use one search term, tribelt.nl's apart from the others'. */
export function useTermPages(term: Ref<string>, language: Ref<Language>) {
  const query = useQuery(computed(() => getTermPagesOptions({ query: { term: term.value, language: language.value } })))
  return {
    split: computed(() => splitTermPages(query.data.value?.items)),
    isLoading: query.isPending,
    error: computed(() => (query.error.value ? describeError(query.error.value) : null)),
  }
}
