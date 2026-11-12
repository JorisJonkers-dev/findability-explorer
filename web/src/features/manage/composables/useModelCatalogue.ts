import { useQuery } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import { getPromptModelCatalogueOptions } from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { PromptCatalogueModel } from '../../../infrastructure/api/types.gen'
import { promptProblem } from '../../prompt-test/domain/settings'

/** How long the catalogue the page read stays fresh; the server reads OpenRouter's once a day. */
const FRESH_MS = 60 * 60 * 1000

/** Every text model on OpenRouter, read only while the Prompt test's key works. */
export function useModelCatalogue(keyWorks: Ref<boolean>) {
  const query = useQuery({ ...getPromptModelCatalogueOptions(), enabled: computed(() => keyWorks.value), staleTime: FRESH_MS })
  return {
    models: computed<PromptCatalogueModel[]>(() => query.data.value?.models ?? []),
    readAt: computed(() => query.data.value?.readAt ?? ''),
    isLoading: computed(() => keyWorks.value && query.isPending.value),
    error: computed(() => (query.error.value ? promptProblem(query.error.value) : null)),
  }
}
