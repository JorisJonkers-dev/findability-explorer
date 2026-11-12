<script setup lang="ts">
import { RouterLink } from 'vue-router'
import type { AnalysisSummary } from '../../../infrastructure/api/types.gen'
import { useCompanies } from '../../competitors/composables/useCompanies'
import BigFigure from '../../manage/components/widgets/BigFigure.vue'
import ManageWidget from '../../manage/components/widgets/ManageWidget.vue'
import WidgetGrid from '../../manage/components/widgets/WidgetGrid.vue'
import { DANGER, ROW_BUTTON } from '../../manage/components/widgets/look'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { useScanQueue } from '../composables/useScanQueue'
import { scanStamp } from '../domain/queue'
import { siteHost } from '../domain/results'
import RunningWork from './RunningWork.vue'

/** The scans that run now, each with Stop, and the ones that wait, each with Remove, and Remove all. */
const { running, waiting, isStopping, stop, removeWaiting, error } = useScanQueue()
const { byId } = useCompanies()
const name = (scan: AnalysisSummary) => byId(scan.competitorId)?.name ?? siteHost(scan.url)
</script>

<template>
  <ManageLayout
    title="Scan queue"
    purpose="The scans that run now and the ones that wait for a free place."
    :back="{ to: '/manage/scans', label: 'Back to Website scans' }"
    :results="{ to: '/ai-readiness', label: 'See AI readiness' }"
  >
    <WidgetGrid>
      <RunningWork />
      <ManageWidget title="Waiting" :span="8">
        <p v-if="error" role="alert" class="m-0 font-medium text-orange-ink">{{ error }}</p>
        <p v-else-if="waiting.length === 0" class="m-0 text-ink-2">No scan is waiting.</p>
        <ul v-else aria-label="Scans waiting" class="m-0 flex list-none flex-col p-0">
          <li v-for="scan in waiting" :key="scan.id" class="flex flex-wrap items-center justify-between gap-3 border-b border-grey py-3">
            <span class="flex min-w-0 flex-col">
              <RouterLink :to="`/manage/scans/${scan.id}`" class="font-semibold">Scan of {{ name(scan) }}</RouterLink>
              <span class="text-sm text-ink-2">Up to {{ scan.maxPages }} pages, queued {{ scanStamp(scan.createdAt) }}. Waiting for a free place.</span>
            </span>
            <button type="button" :class="ROW_BUTTON" :disabled="isStopping" @click="stop(scan, name(scan))">
              Remove<span class="sr-only">{{ ` the scan of ${name(scan)}` }}</span>
            </button>
          </li>
        </ul>
      </ManageWidget>
      <ManageWidget title="The queue" :span="4">
        <p class="m-0 text-ink-2">At most three scans run at once.</p>
        <div class="flex flex-wrap gap-8">
          <BigFigure :value="String(running.length)" label="running" />
          <BigFigure :value="String(waiting.length)" label="waiting" />
        </div>
        <template v-if="waiting.length > 0" #foot>
          <button type="button" :class="DANGER" :disabled="isStopping" @click="removeWaiting">Remove all waiting scans</button>
        </template>
      </ManageWidget>
    </WidgetGrid>
  </ManageLayout>
</template>
