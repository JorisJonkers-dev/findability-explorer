<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useCompanies } from '../../competitors/composables/useCompanies'
import { useRescan } from '../../competitors/composables/useCompanyAdmin'
import { rescanSummary } from '../../competitors/domain/admin'
import { useReadinessCheck, useReadinessTable } from '../../readiness/composables/useReadiness'
import StepCard from '../../claude-steps/components/StepCard.vue'
import { useClaudeStep } from '../../claude-steps/composables/useClaudeSteps'
import RunningWork from '../../scan/components/RunningWork.vue'
import ScanForm from '../../scan/components/ScanForm.vue'
import { usePageBudget } from '../../scan/composables/usePageBudget'
import { useScanQueue } from '../../scan/composables/useScanQueue'
import { useStartScan } from '../../scan/composables/useStartScan'
import { MAX_SCAN_PAGES } from '../../scan/domain/newScan'
import { lastScanned, newestScans, type RescanGroup, rescanGroups, rescanRequest, scanDay, scanEstimate } from '../../scan/domain/queue'
import BudgetMeter from '../../settings/components/BudgetMeter.vue'
import { useLlmSettings } from '../../settings/composables/useLlmSettings'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import BigFigure from './widgets/BigFigure.vue'
import IndexCard from './widgets/IndexCard.vue'
import ManageWidget from './widgets/ManageWidget.vue'
import WidgetGrid from './widgets/WidgetGrid.vue'
import { CHOICE, CHOICE_OFF, CHOICE_ON, FIELD, LABEL, PRIMARY } from './widgets/look'

/**
 * Website scans: the last scan, the budget, and the readiness check; the four ways to start; what
 * runs now; and a card per subpage.
 */
const router = useRouter()
const { companies } = useCompanies()
const { settings } = useLlmSettings()
const { rows, check } = useReadinessTable()
const { pages } = usePageBudget()
const { running, waiting } = useScanQueue()
const startScan = useStartScan()
const rescan = useRescan()
const readiness = useReadinessCheck()

// A scan's estimate is the matching step's job with the model it uses now, so it scales with it.
const { step: matching } = useClaudeStep(ref('matching'))
const budget = computed(() => {
  const b = settings.value?.budget
  return b && matching.value ? { ...b, averageScanUsd: matching.value.jobCostUsd } : b
})
const last = computed(() => lastScanned(companies.value))
const tribelt = computed(() => companies.value.find((c) => c.role === 'tribelt'))
const scanned = computed(() => newestScans(companies.value).length)
const sites = computed(() => rows.value.filter((r) => r.scan !== undefined).length)

const group = ref<RescanGroup>('stale')
const groups = computed(() => rescanGroups(companies.value, new Date()))
const chosen = computed(() => groups.value[group.value])
const GROUPS: readonly { id: RescanGroup; label: string }[] = [
  { id: 'all', label: 'All' },
  { id: 'stale', label: 'Over 30 days old' },
  { id: 'never', label: 'Never scanned' },
]
const rescanNote = computed(() => (rescan.batch.value ? rescanSummary(rescan.batch.value) : null))

function opened(id: string): void {
  void router.push(`/manage/scans/${id}`)
}

function scanTribelt(): void {
  const url = tribelt.value?.websiteUrl
  if (url) startScan.start(url, pages.value, opened)
}

function setPages(event: Event): void {
  pages.value = Number((event.target as HTMLInputElement).value)
}

const checkLine = computed(() => {
  const c = check.value
  if (!c || c.total === 0) return 'Not checked again since the app last started.'
  if (c.running) return `Checking, ${String(c.checked)} of ${String(c.total)} sites.`
  return `${String(c.checked)} of ${String(c.total)} sites checked${c.stopped ? ', then stopped by hand' : ''}.`
})
</script>

<template>
  <ManageLayout title="Website scans" purpose="Read the websites of Tribelt and its competitors." :results="{ to: '/ai-readiness', label: 'See AI readiness' }">
    <WidgetGrid>
      <ManageWidget title="Last scan" :span="4">
        <BigFigure v-if="last" :value="scanDay(last.scan.createdAt)" :label="last.company.name" />
        <p v-else class="m-0 text-ink-2">None yet.</p>
      </ManageWidget>
      <ManageWidget title="Anthropic budget" :span="4">
        <BudgetMeter v-if="budget" :budget="budget" />
        <p v-else class="m-0 text-muted">Loading the budget…</p>
      </ManageWidget>
      <ManageWidget title="AI readiness check" :span="4">
        <BigFigure :value="String(sites)" label="sites with a scan" :note="checkLine" />
      </ManageWidget>

      <ManageWidget title="Scan tribelt.nl" :span="6">
        <p class="m-0 text-ink-2">Reads up to {{ pages }} pages. Reading pages is free, and product matching is the only cost.</p>
        <p v-if="startScan.error.value" role="alert" class="m-0 font-medium text-orange-ink">{{ startScan.error.value.message }}</p>
        <template #foot>
          <button type="button" :class="PRIMARY" :disabled="!tribelt?.websiteUrl || startScan.isStarting.value" @click="scanTribelt">Scan tribelt.nl</button>
          <span class="text-sm text-ink-2">{{ scanEstimate(budget, 1) }}</span>
        </template>
      </ManageWidget>

      <ManageWidget title="Scan one site" :span="6">
        <p class="m-0 text-ink-2">Pick a competitor, or type a web address.</p>
        <label :class="LABEL">
          Pages (1 to {{ MAX_SCAN_PAGES }})
          <input type="number" min="1" :max="MAX_SCAN_PAGES" :value="pages" :class="FIELD" @change="setPages" />
        </label>
        <ScanForm submit-label="Scan this site" @started="opened" />
        <p class="m-0 text-sm text-ink-2">{{ scanEstimate(budget, 1) }}</p>
      </ManageWidget>

      <ManageWidget title="Rescan competitors" :span="6">
        <p class="m-0 text-ink-2">Reads the competitors' sites again, a few at a time.</p>
        <div role="group" aria-label="Which competitors" class="flex flex-wrap gap-2">
          <button
            v-for="g in GROUPS"
            :key="g.id"
            type="button"
            :aria-pressed="group === g.id"
            :class="[CHOICE, group === g.id ? CHOICE_ON : CHOICE_OFF]"
            @click="group = g.id"
          >
            {{ g.label }} ({{ groups[g.id].length }})
          </button>
        </div>
        <p v-if="rescanNote" class="m-0 text-sm text-ink-2" aria-live="polite">{{ rescanNote }}</p>
        <p v-if="rescan.error.value" role="alert" class="m-0 font-medium text-orange-ink">{{ rescan.error.value }}</p>
        <template #foot>
          <button
            type="button"
            :class="PRIMARY"
            :disabled="chosen.length === 0 || rescan.isQueuing.value"
            @click="rescan.rescan(rescanRequest(chosen, pages))"
          >
            Rescan these competitors
          </button>
          <span class="text-sm text-ink-2">{{ chosen.length === 0 ? 'No company is in this group.' : scanEstimate(budget, chosen.length) }}</span>
        </template>
      </ManageWidget>

      <ManageWidget title="Check AI readiness again" :span="6">
        <p class="m-0 text-ink-2">
          Checks the files that tell AI tools what they may read (robots.txt, llms.txt, llms-full.txt, and the sitemaps) again for each
          site's newest scan. It does not read the pages again.
        </p>
        <p v-if="readiness.error.value" role="alert" class="m-0 font-medium text-orange-ink">{{ readiness.error.value }}</p>
        <template #foot>
          <button type="button" :class="PRIMARY" :disabled="check?.running || readiness.isStarting.value" @click="readiness.start()">Check AI readiness again</button>
          <span class="text-sm text-ink-2">Free. {{ sites }} sites.</span>
        </template>
      </ManageWidget>

      <RunningWork />

      <IndexCard
        title="Scan queue"
        text="At most three scans run at once. The rest wait here."
        to="/manage/scans/queue"
        link="Open the queue"
        :figures="[{ value: String(running.length), label: 'running' }, { value: String(waiting.length), label: 'waiting' }]"
      />
      <IndexCard
        title="Recent scans"
        text="Each company's newest scan, and its details."
        to="/manage/scans/recent"
        link="See the scans"
        :figures="[{ value: String(scanned), label: 'companies scanned' }]"
      />
      <IndexCard
        title="Scan settings"
        text="How many pages a scan reads, and the model that matches products."
        to="/manage/scans/settings"
        link="Change the settings"
        :figures="[{ value: String(pages), label: 'pages' }]"
      />
      <StepCard step-id="matching" />
    </WidgetGrid>
  </ManageLayout>
</template>
