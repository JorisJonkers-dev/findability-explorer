import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getPromptTestHistoryOptions } from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { PromptTestHistory } from '../../../infrastructure/api/types.gen'
import { describeError } from '../../../shared/describeError'
import { allBreaks, chartNotes, chartSeries, forVersion, type Measure, parseVersion, tableRows, type VersionChoice, versionNote, versionOptions } from '../domain/history'

/**
 * The Prompt test's history per run, narrowed by the Prompt set filter, whose choice is kept in the
 * address (`?version=1`) so every page that shows the history shows the same version. measures are
 * the lines drawn: AI responses draws each language; a goal page draws its own measure.
 */
export function useHistory(measures: readonly Measure[]) {
  const query = useQuery(getPromptTestHistoryOptions())
  const route = useRoute()
  const router = useRouter()
  const versions = computed(() => query.data.value?.versions ?? [])
  const version = computed<VersionChoice>(() => parseVersion(route.query.version, versions.value))
  const runs = computed(() => forVersion(query.data.value?.runs ?? [], version.value))
  const breaks = computed(() => allBreaks(versions.value, runs.value))
  const series = computed(() => chartSeries(runs.value, measures))
  return {
    history: computed<PromptTestHistory | undefined>(() => query.data.value),
    isLoading: query.isPending,
    error: computed(() => (query.error.value ? describeError(query.error.value) : null)),
    version,
    options: computed(() => versionOptions(versions.value)),
    note: computed(() => versionNote(versions.value, version.value, runs.value)),
    runs,
    breaks,
    series,
    hasPoints: computed(() => series.value.some((s) => s.points.length > 0)),
    notes: computed(() => chartNotes(breaks.value, runs.value, measures)),
    rows: computed(() => tableRows(runs.value, measures)),
    choose: (v: VersionChoice) => {
      void router.replace({ query: { ...route.query, version: v === 'all' ? undefined : String(v) } })
    },
  }
}
