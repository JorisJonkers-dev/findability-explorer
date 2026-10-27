import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import { getProductsOptions } from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { Product, ProductStandout } from '../../../infrastructure/api/types.gen'
import { describeError } from '../../../shared/describeError'

/** Server state for the belt families against the field: one cached request every product page shares. */
export function useProducts() {
  const query = useQuery(getProductsOptions())
  return {
    products: computed<Product[]>(() => query.data.value?.items ?? []),
    standouts: computed<ProductStandout[]>(() => query.data.value?.standouts ?? []),
    isLoading: query.isPending,
    error: computed(() => (query.error.value ? describeError(query.error.value) : null)),
  }
}
