<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { useCompanies } from '../../competitors/composables/useCompanies'
import ManageWidget from '../../manage/components/widgets/ManageWidget.vue'
import WidgetGrid from '../../manage/components/widgets/WidgetGrid.vue'
import { DANGER, PRIMARY } from '../../manage/components/widgets/look'
import { companyPath } from '../../shell/domain/sections'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { usePageBudget } from '../composables/usePageBudget'
import { useScan } from '../composables/useScan'
import { useScanQueue } from '../composables/useScanQueue'
import { useStartScan } from '../composables/useStartScan'
import { pagesRead, scanStamp, stoppedLine } from '../domain/queue'
import { siteHost } from '../domain/results'
import { isInProgress } from '../domain/status'
import ScanProgress from './ScanProgress.vue'
import ScanResult from './ScanResult.vue'

/**
 * One scan of one website: its company, number, date, pages read, and matching model; Scan again;
 * Stop while it runs; and what the scan found once it is done.
 */
const props = defineProps<{ scanId: string }>()
const router = useRouter()
const id = computed(() => props.scanId)
const { scan, error } = useScan(id)
const { byId } = useCompanies()
const { pages } = usePageBudget()
const startScan = useStartScan()
const queue = useScanQueue()

const summary = computed(() => scan.value?.summary)
const company = computed(() => byId(summary.value?.competitorId))
const name = computed(() => company.value?.name ?? (summary.value ? siteHost(summary.value.url) : ''))
const title = computed(() => (name.value === '' ? 'Scan' : `Scan of ${name.value}`))
const inProgress = computed(() => summary.value !== undefined && isInProgress(summary.value.status))
const stopped = computed(() => (summary.value ? stoppedLine(summary.value) : null))

function again(): void {
  if (summary.value) startScan.start(summary.value.url, pages.value, (next) => void router.push(`/manage/scans/${next}`))
}
</script>

<template>
  <ManageLayout
    :title="title"
    purpose="One scan of one website, and what it found."
    :back="{ to: '/manage/scans/recent', label: 'Back to Recent scans' }"
    :results="{ to: '/ai-readiness', label: 'See AI readiness' }"
  >
    <p v-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-6 text-lg">{{ error }}</p>
    <p v-else-if="!scan || !summary" class="m-0 text-muted">Loading the scan…</p>
    <WidgetGrid v-else>
      <ManageWidget v-if="inProgress" title="Running now" :span="12">
        <ScanProgress :summary="summary" />
        <p class="m-0 font-medium" aria-live="polite">{{ queue.notice.value }}</p>
        <p v-if="queue.stopError.value" role="alert" class="m-0 font-medium text-orange-ink">{{ queue.stopError.value }}</p>
        <template #foot>
          <button type="button" :class="DANGER" :disabled="queue.isStopping.value" @click="queue.stop(summary, name)">
            {{ summary.status === 'queued' ? 'Remove it from the queue' : 'Stop the scan' }}
          </button>
          <span class="text-sm text-ink-2">A stopped scan keeps the pages it read so far.</span>
        </template>
      </ManageWidget>

      <ManageWidget title="This scan" :span="8">
        <dl class="m-0 grid grid-cols-[minmax(0,10rem)_minmax(0,1fr)] gap-x-4 gap-y-2">
          <dt class="text-ink-2">Company</dt>
          <dd class="m-0 break-words">{{ name }}</dd>
          <dt class="text-ink-2">Scan number</dt>
          <dd class="m-0">{{ summary.number ?? 1 }}</dd>
          <dt class="text-ink-2">Date</dt>
          <dd class="m-0">{{ scanStamp(summary.createdAt) }}</dd>
          <dt class="text-ink-2">Pages read</dt>
          <dd class="m-0">{{ pagesRead(summary) }}</dd>
          <dt class="text-ink-2">Matching model</dt>
          <dd class="m-0 break-words">{{ summary.matchingModel ?? 'None' }}</dd>
        </dl>
        <p v-if="stopped" class="m-0 font-medium">{{ stopped }}</p>
        <div v-else-if="summary.status === 'failed'" class="flex flex-col gap-1">
          <p class="m-0 text-sm font-semibold tracking-[0.06em] text-orange-ink uppercase">Scan failed</p>
          <p role="alert" class="m-0 text-ink-2">{{ summary.failure ?? 'The scan stopped before it read a page.' }}</p>
        </div>
        <template #foot>
          <p class="m-0 text-sm text-ink-2">
            <template v-if="company && company.role === 'competitor'">
              See the <RouterLink :to="companyPath(company.id)">competitor page for {{ company.name }}</RouterLink>.
            </template>
            See <RouterLink to="/ai-readiness">AI readiness</RouterLink>.
          </p>
        </template>
      </ManageWidget>

      <ManageWidget title="Scan again" :span="4">
        <p class="m-0 text-ink-2">Reads the site again and saves it as the next scan. The older scan stays in the history.</p>
        <p v-if="startScan.error.value" role="alert" class="m-0 font-medium text-orange-ink">{{ startScan.error.value.message }}</p>
        <template #foot>
          <button type="button" :class="PRIMARY" :disabled="inProgress || startScan.isStarting.value" @click="again">Scan again</button>
          <span class="text-sm text-ink-2">{{ pages }} pages</span>
        </template>
      </ManageWidget>

      <ManageWidget v-if="summary.status === 'done'" title="What the scan found" :span="12">
        <ScanResult :key="summary.id" :scan="scan" />
      </ManageWidget>
    </WidgetGrid>
  </ManageLayout>
</template>
