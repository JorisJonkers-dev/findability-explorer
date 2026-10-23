import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import { getMeOptions } from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { Me } from '../../../infrastructure/api/types.gen'
import { describeError } from '../../../shared/describeError'

/** Server state for who is signed in. It does not change during a visit, so it is asked for once. */
export function useMe() {
  const query = useQuery({ ...getMeOptions(), staleTime: Number.POSITIVE_INFINITY })
  return {
    me: computed<Me | undefined>(() => query.data.value),
    isLoading: query.isPending,
    error: computed(() => (query.error.value ? describeError(query.error.value) : null)),
  }
}
