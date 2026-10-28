<script setup lang="ts">
import { computed, useTemplateRef } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import StageColumn from '../../../shared/ui/StageColumn.vue'
import { useColumnLayout } from '../../../shared/useColumnLayout'
import SectionView from '../../shell/components/SectionView.vue'
import { COMPETITOR_TABS } from '../../shell/domain/sections'
import ScanPanel from '../../scan/components/ScanPanel.vue'
import { useTermMap } from '../../terminology/composables/useTermSearch'
import { useMe } from '../../shell/composables/useMe'
import { useCompanies } from '../composables/useCompanies'
import { splitArchived } from '../domain/admin'
import { selectedCompany } from '../domain/list'
import { useOverview } from '../../home/composables/useOverview'
import AdminTools from './AdminTools.vue'
import CompanyCard from './CompanyCard.vue'
import CompanyList from './CompanyList.vue'

defineProps<{ scanId?: string }>()
const route = useRoute()
const router = useRouter()
const { companies: all, isLoading, error } = useCompanies()
const { isAdmin } = useMe()
// Archived researched companies are hidden; administrators see them apart, to restore one. The
// mirror of tribelt.nl is no company to compare names with: it shows in the AI readiness table.
const split = computed(() => splitArchived(all.value.filter((c) => c.role !== 'mirror')))
const companies = computed(() => split.value.active)
const archived = computed(() => (isAdmin.value ? split.value.archived : []))
const map = useTermMap()
const panel = useTemplateRef<InstanceType<typeof ScanPanel>>('panel')

// Each column can be widened or minimised on the stage; the browser remembers the layout.
const COLUMNS = ['Who makes which belt', 'Company', 'Scan a competitor'] as const
const columns = useColumnLayout('competitors', [1.3, 0.95, 1])
function columnProps(i: number) {
  return { title: COLUMNS[i] ?? '', state: columns.states.value[i] ?? 'normal', canMinimise: columns.canMinimise(i) }
}

const selected = computed(() =>
  selectedCompany([...companies.value, ...archived.value], typeof route.query.company === 'string' ? route.query.company : undefined),
)
// The server counts the competitors on our list now, as every page shows them.
const { overview } = useOverview()
const title = computed(() => {
  const count = overview.value?.competitors.makers ?? 0
  return count > 0 ? `${String(count)} competitors are on our list. The same belts, other names.` : 'The same belts, other names.'
})

/** Shows a company: the address keeps it, so a link or a reload shows the same one. */
function pick(id: string): void {
  void router.replace({ query: { ...route.query, company: id } })
}

/** A removed company is gone from the list, so the view shows the first one again. */
function removed(): void {
  const query = { ...route.query }
  delete query.company
  void router.replace({ query })
}

function scan(url: string): void {
  panel.value?.scan(url)
}
</script>

<template>
  <SectionView section="competitors" :title="title" :tabs="COMPETITOR_TABS" :source="['Desk research, 22 and 26 October 2026', 'Scans: live, from this app']">
    <div class="flex flex-col gap-12 stage:grid stage:min-h-0 stage:gap-24" :style="{ gridTemplateColumns: columns.template.value }">
      <StageColumn v-bind="columnProps(0)" @widen="columns.widen(0)" @minimise="columns.minimise(0)" @restore="columns.restore(0)">
        <CompanyList :companies="companies" :archived="archived" :selected-id="selected?.id" :loading="isLoading" @pick="pick" />
      </StageColumn>
      <StageColumn v-bind="columnProps(1)" @widen="columns.widen(1)" @minimise="columns.minimise(1)" @restore="columns.restore(1)">
        <p v-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
        <CompanyCard v-else-if="selected" :company="selected" :terms="map.terms.value" :terms-from="map.tribeltFrom.value" :is-admin="isAdmin" @scan="scan" @removed="removed" />
        <p v-else class="m-0 text-muted">Loading the companies…</p>
      </StageColumn>
      <StageColumn v-bind="columnProps(2)" @widen="columns.widen(2)" @minimise="columns.minimise(2)" @restore="columns.restore(2)">
        <ScanPanel ref="panel" :scan-id="scanId" :is-admin="isAdmin">
          <AdminTools v-if="isAdmin" :companies="companies" @added="pick" />
        </ScanPanel>
      </StageColumn>
    </div>
  </SectionView>
</template>
