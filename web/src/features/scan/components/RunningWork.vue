<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import type { AnalysisSummary } from '../../../infrastructure/api/types.gen'
import ProgressBar from '../../../shared/ui/ProgressBar.vue'
import { useCompanies } from '../../competitors/composables/useCompanies'
import ManageWidget from '../../manage/components/widgets/ManageWidget.vue'
import { DANGER, ROW_BUTTON } from '../../manage/components/widgets/look'
import { useReadinessTable } from '../../readiness/composables/useReadiness'
import { useScanQueue, useStopReadinessCheck } from '../composables/useScanQueue'
import { siteHost } from '../domain/results'
import { progress } from '../domain/status'

/**
 * The scans reading pages now, each with its Stop, and the AI readiness check while it runs, with
 * its own. A rescan's scans stop together with Stop the rescan; a scan started on its own runs on.
 */
const { running, waiting, isStopping, notice, stopError, stop, stopRescan } = useScanQueue()
const { check } = useReadinessTable()
const readinessStop = useStopReadinessCheck()
const { byId } = useCompanies()

const name = (scan: AnalysisSummary) => byId(scan.competitorId)?.name ?? siteHost(scan.url)
const checking = computed(() => check.value?.running === true)
const busy = computed(() => running.value.length + waiting.value.length > 0)
</script>

<template>
  <ManageWidget title="Running now" :span="12">
    <p v-if="!busy && !checking" class="m-0 text-ink-2">No scan is running.</p>
    <ul v-if="running.length > 0" aria-label="Scans running" class="m-0 flex list-none flex-col gap-4 p-0">
      <li v-for="scan in running" :key="scan.id" class="flex flex-col gap-2 rounded-2xl bg-white p-4">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <RouterLink :to="`/manage/scans/${scan.id}`" class="font-semibold">Scan of {{ name(scan) }}</RouterLink>
          <span class="text-sm text-ink-2">{{ progress(scan).label }}</span>
        </div>
        <ProgressBar thick :fraction="progress(scan).fraction" :label="`Scan of ${name(scan)}`" :text="progress(scan).label" />
        <div>
          <button type="button" :class="ROW_BUTTON" :disabled="isStopping" @click="stop(scan, name(scan))">
            Stop<span class="sr-only">{{ ` the scan of ${name(scan)}` }}</span>
          </button>
        </div>
      </li>
    </ul>
    <p v-if="waiting.length > 0" class="m-0 text-ink-2">
      {{ waiting.length === 1 ? '1 scan waits for a free place.' : `${String(waiting.length)} scans wait for a free place.` }}
      <RouterLink to="/manage/scans/queue">Open the queue</RouterLink>
    </p>
    <div v-if="checking && check" class="flex flex-col gap-2 rounded-2xl bg-white p-4" data-part="readiness-check">
      <p class="m-0 font-semibold">AI readiness check</p>
      <ProgressBar
        thick
        :fraction="check.total > 0 ? check.checked / check.total : null"
        label="AI readiness check"
        :text="`${String(check.checked)} of ${String(check.total)} sites`"
      />
      <p class="m-0 text-sm text-ink-2">{{ check.checked }} of {{ check.total }} sites. Stopping keeps the sites already checked; the others keep their older results.</p>
      <div>
        <button type="button" :class="ROW_BUTTON" :disabled="readinessStop.isStopping.value" @click="readinessStop.stop()">Stop the check</button>
      </div>
      <p v-if="readinessStop.error.value" role="alert" class="m-0 text-sm font-medium text-orange-ink">{{ readinessStop.error.value }}</p>
    </div>
    <p class="m-0 font-medium" aria-live="polite">{{ notice }}</p>
    <p v-if="stopError" role="alert" class="m-0 font-medium text-orange-ink">{{ stopError }}</p>
    <template v-if="busy" #foot>
      <button type="button" :class="DANGER" :disabled="isStopping" @click="stopRescan">Stop the rescan</button>
      <span class="text-sm text-ink-2">Stops every scan a rescan queued. A scan started on its own runs on.</span>
    </template>
  </ManageWidget>
</template>
