import { useQuery } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import { getCompetitorOptions } from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { CompetitorDetail } from '../../../infrastructure/api/types.gen'
import { zProblem } from '../../../infrastructure/api/zod.gen'
import { describeError } from '../../../shared/describeError'

/** Server state for one company and its most recent scans. */
export function useCompany(id: Ref<string>) {
  const query = useQuery(computed(() => getCompetitorOptions({ path: { competitorId: id.value } })))
  return {
    detail: computed<CompetitorDetail | undefined>(() => query.data.value),
    isLoading: query.isPending,
    /** True when no company has this id, which deserves its own message. */
    notFound: computed(() => {
      const problem = zProblem.safeParse(query.error.value)
      return problem.success && problem.data.status === 404
    }),
    error: computed(() => (query.error.value ? describeError(query.error.value) : null)),
  }
}
