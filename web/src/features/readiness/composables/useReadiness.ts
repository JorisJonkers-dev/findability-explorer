import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import {
  getAiReadinessOptions,
  getAiReadinessQueryKey,
  getReadinessFilesOptions,
  startReadinessCheckMutation,
} from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { AiReadinessRow, AiReadinessTable, ReadinessCheckState, ReadinessFiles } from '../../../infrastructure/api/types.gen'
import { zProblem } from '../../../infrastructure/api/zod.gen'
import { describeError } from '../../../shared/describeError'

/** How often the table is read again while a readiness check runs. */
export const CHECK_POLL_MS = 3000

/** Server state for every company's AI readiness: one cached request the table and the cards share. */
export function useReadinessTable() {
  const query = useQuery({
    ...getAiReadinessOptions(),
    // While a check runs, the rows change site by site.
    refetchInterval: (q: { state: { data: AiReadinessTable | undefined } }) => (q.state.data?.check.running ? CHECK_POLL_MS : false),
  })
  return {
    rows: computed<AiReadinessRow[]>(() => query.data.value?.items ?? []),
    check: computed<ReadinessCheckState | undefined>(() => query.data.value?.check),
    isLoading: query.isPending,
    error: computed(() => (query.error.value ? describeError(query.error.value) : null)),
    /** One company's row, once the table has loaded. */
    rowOf: (id: string): AiReadinessRow | undefined => query.data.value?.items.find((r) => r.competitorId === id),
  }
}

/** Starts checking every scanned site's AI readiness again, then reads the table again. */
export function useReadinessCheck() {
  const queryClient = useQueryClient()
  const start = useMutation(startReadinessCheckMutation())
  return {
    isStarting: start.isPending,
    error: computed(() => (start.error.value ? describeError(start.error.value) : null)),
    start: () => {
      start.mutate({}, { onSuccess: () => void queryClient.invalidateQueries({ queryKey: getAiReadinessQueryKey() }) })
    },
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
