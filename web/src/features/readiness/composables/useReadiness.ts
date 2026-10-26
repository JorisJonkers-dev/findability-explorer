import { useQuery } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import { getAiReadinessOptions, getReadinessFilesOptions } from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { AiReadinessRow, ReadinessFiles } from '../../../infrastructure/api/types.gen'
import { zProblem } from '../../../infrastructure/api/zod.gen'
import { describeError } from '../../../shared/describeError'

/** Server state for every company's AI readiness: one cached request the table and the cards share. */
export function useReadinessTable() {
  const query = useQuery(getAiReadinessOptions())
  return {
    rows: computed<AiReadinessRow[]>(() => query.data.value?.items ?? []),
    isLoading: query.isPending,
    error: computed(() => (query.error.value ? describeError(query.error.value) : null)),
    /** One company's row, once the table has loaded. */
    rowOf: (id: string): AiReadinessRow | undefined => query.data.value?.items.find((r) => r.competitorId === id),
  }
}

/**
 * Server state for the raw files a scan read its AI readiness from; administrators only, so
 * nothing is asked until enabled is true and a scan is named. A scan without readiness is not an
 * error: it has no files.
 */
export function useReadinessFiles(scanId: Ref<string | undefined>, enabled: Ref<boolean>) {
  const query = useQuery(
    computed(() => ({
      ...getReadinessFilesOptions({ path: { analysisId: scanId.value ?? '' } }),
      enabled: enabled.value && scanId.value !== undefined,
      staleTime: Number.POSITIVE_INFINITY,
    })),
  )
  const missing = computed(() => {
    const problem = zProblem.safeParse(query.error.value)
    return problem.success && problem.data.status === 404
  })
  return {
    files: computed<ReadinessFiles | undefined>(() => query.data.value),
    isLoading: computed(() => query.isFetching.value && query.data.value === undefined),
    /** True when the scan recorded no AI readiness. */
    missing,
    error: computed(() => (query.error.value && !missing.value ? describeError(query.error.value) : null)),
  }
}
