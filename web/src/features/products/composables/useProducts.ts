import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import { getProductsOptions } from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { Product } from '../../../infrastructure/api/types.gen'
import { describeError } from '../../../shared/describeError'

/** Server state for the belt families against the field: one cached request every family page shares. */
export function useProducts() {
  const query = useQuery(getProductsOptions())
  return {
    products: computed<Product[]>(() => query.data.value?.items ?? []),
    isLoading: query.isPending,
    error: computed(() => (query.error.value ? describeError(query.error.value) : null)),
  }
}
